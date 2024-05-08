package osmosis

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/synternet/osmosis-publisher/pkg/types"

	cltypes "github.com/osmosis-labs/osmosis/v24/x/concentrated-liquidity/types"
	wasmtypes "github.com/osmosis-labs/osmosis/v24/x/cosmwasmpool/types"
	gammtypes "github.com/osmosis-labs/osmosis/v24/x/gamm/types"
	pmtypes "github.com/osmosis-labs/osmosis/v24/x/poolmanager/types"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
)

func (p *Publisher) subscribeOsmosisEvents() error {
	queries := []string{
		fmt.Sprintf("message.module='%s'", pmtypes.AttributeValueCategory),
		fmt.Sprintf("message.module='%s'", gammtypes.AttributeValueCategory),
		fmt.Sprintf("message.module='%s'", wasmtypes.AttributeValueCategory),
		fmt.Sprintf("message.module='%s'", cltypes.AttributeValueCategory),
	}

	for _, q := range queries {
		return p.rpc.Subscribe(q, p.handlePoolSubscriptions)
	}

	return nil
}

// combinePoolStatusesAt Fetch pool volume&liquidity at certain height and append to appropriate pool Volumes.
// The height is calculated relative to the current height depending on the duration before that height.
func (p *Publisher) combinePoolStatusesAt(height int64, before time.Duration, ps []types.PoolStatus) error {
	ids := make([]uint64, len(ps))
	mids := make(map[uint64]int, len(ps))
	for i, p := range ps {
		ids[i] = p.PoolId
		mids[p.PoolId] = i
	}

	delta := int64(before.Seconds() / p.indexer.AverageBlockTime().Seconds())
	height -= delta

	psAt, _, err := p.indexer.PoolStatusesAt(uint64(height), ids...)
	if err != nil {
		return err
	}

	for pIdx, pAt := range psAt {
		idx := mids[pAt.PoolId]
		ps[idx].Volumes = append(ps[idx].Volumes, pAt.Volumes...)
		p.Logger.Debug("SUB POOL: combinePoolStatusesAt", "poolId", pAt.PoolId, "volumes", len(ps[idx].Volumes), "pIdx", pIdx, "idx", idx)
	}

	return nil
}

// Retrieves pool volume&liquidity information at certain height as well as heights before that at regular intervals such as -1h, -4h, -12h, -24h.
// Will calculate volume prices at that height as well as volume difference between latest height and respective heights.
func (p *Publisher) getPoolsOfInterestStatuses(height int64, ids ...uint64) ([]types.PoolStatus, uint64, error) {
	errArr := make([]error, 3)

	p.Logger.Debug("SUB POOL: getPoolsOfInterestStatuses PoolStatusesAt", "len", len(ids))
	ps, h, err := p.indexer.PoolStatusesAt(uint64(height), ids...)
	if err != nil {
		errArr = append(errArr, fmt.Errorf("PoolStatusesAt failed h=%d: %w", height, err))
	}

	p.Logger.Debug("SUB POOL: getPoolsOfInterestStatuses PoolStatusesAt durations before", "len", len(ids))
	for _, before := range []time.Duration{time.Hour, time.Hour * 4, time.Hour * 12, time.Hour * 24} {
		err = p.combinePoolStatusesAt(int64(h), before, ps)
		if err != nil {
			errArr = append(errArr, fmt.Errorf("combinePoolStatusesAt failed at=%d before=%v: %w", h, before, err))
		}
	}

	p.Logger.Debug("SUB POOL: getPoolsOfInterestStatuses CalculateVolumes", "len", len(ids))
	err = p.indexer.CalculateVolumes(ps)
	if err != nil {
		errArr = append(errArr, fmt.Errorf("CalculateVolumes failed: %w", err))
	}
	p.Logger.Debug("SUB POOL: getPoolsOfInterestStatuses DONE", "len", len(ids), "errs", errArr)

	return ps, h, errors.Join(errArr...)
}

// handleMonitoredPools will retrieve pools for different heights configured to be monitored,
// calculate volume prices and send as a message.
func (p *Publisher) handleMonitoredPools(height int64, blockTime time.Time, hash string) {
	poolStatus := types.PoolOfInterest{
		Nonce:        p.NewNonce(),
		BlockHeight:  height,
		AvgBlockTime: p.indexer.AverageBlockTime().Seconds(),
		BlockHash:    hash,
	}

	now := time.Now()
	ps, _, err := p.getPoolsOfInterestStatuses(height, p.PoolIds()...)
	if err != nil {
		p.Logger.Warn("Failed getting pools of interest", "err", err)
	}
	poolStatus.Pools = ps

	ibcMap := make(IBCDenomTrace)
	for _, p := range poolStatus.Pools {
		for _, d := range p.TotalLiquidity {
			ibcMap.Add(d.Denom)
		}
		for _, v := range p.Volumes {
			for _, d := range v.Volume {
				ibcMap.Add(d.Denom)
			}
		}
	}
	err = p.getDenoms(ibcMap)
	if err != nil {
		p.Logger.Warn("Extracting denoms failed", "err", err)
	}
	poolStatus.Metadata = ibcMap

	p.Logger.Info("Pool Volumes", "num_pools", len(poolStatus.Pools), "height", height, "blockTime", blockTime, "duration", time.Since(now))
	p.Publish(
		&poolStatus,
		"volume",
		"pool",
	)
	p.messagesCounter.Add(1)
}

// handlePoolSubscriptions will parse events, determine what pools were involved,
// retrieve states/volumes/liquidity for such pools, calculate volume prices, and send the message.
func (p *Publisher) handlePoolSubscriptions(events <-chan ctypes.ResultEvent) error {
	for {
		select {
		case <-p.Context.Done():
			p.Logger.Info("handlePoolSubscriptions: c.Context Done")
			return nil
		case ev, ok := <-events:
			if !ok {
				p.Logger.Info("handlePoolSubscriptions: events closed")
				return nil
			}

			ibcMap := make(IBCDenomTrace)
			poolIds := ExtractUniquePoolIds(ev)

			p.Logger.Debug("Pool", "query", ev.Query, "poolIds", poolIds)

			if poolIds == nil {
				continue
			}

			// TODO: Fetch in parallel
			poolResults, err := p.rpc.PoolsAt(0, poolIds...)
			if err != nil {
				p.Logger.Warn("Failed to fetch pools", "poolIds", poolIds, "err", err)
			}
			poolStatuses, height, err := p.getPoolsOfInterestStatuses(0, poolIds...)
			if err != nil {
				p.Logger.Warn("Failed getting pools of interest", "err", err)
			}

			pools := make([]any, 0, len(poolResults))
			for idx, pool := range poolResults {
				if pool == nil {
					p.Logger.Debug("Pool is nil", "idx", idx)
					continue
				}
				pools = append(pools, (*pool).AsSerializablePool())
			}

			if len(pools) == 0 {
				continue
			}

			for _, ps := range poolStatuses {
				for _, d := range ps.TotalLiquidity {
					ibcMap.Add(d.Denom)
				}
				for _, v := range ps.Volumes {
					for _, d := range v.Volume {
						ibcMap.Add(d.Denom)
					}
				}
			}

			p.poolCounter.Add(uint64(len(pools)))

			hash := ""
			block, err := p.rpc.BlockAt(int64(height))
			if err != nil {
				p.Logger.Warn("Failed getting block", "err", err)
			} else {
				height = uint64(block.Height)
				hash = block.Hash().String()
			}

			err = p.rpc.getDenoms(ibcMap)
			if err != nil {
				p.Logger.Warn("Extracting denoms failed", "err", err)
			}

			msg := types.Pools{
				Nonce:        p.NewNonce(),
				BlockHeight:  int64(height),
				BlockHash:    hash,
				AvgBlockTime: p.indexer.AverageBlockTime().Seconds(),
				Events:       ev.Events,
				Pools:        pools,
				PoolStatus:   poolStatuses,
				Metadata:     ibcMap,
			}

			p.Publish(
				msg,
				"state",
				"pools",
			)
			p.messagesCounter.Add(1)
		}
	}
}

func ExtractUniquePoolIds(ev ctypes.ResultEvent) []uint64 {
	poolIdSet := make(map[uint64]struct{}, 0)
	for k, v := range ev.Events {
		if !strings.Contains(k, "pool_id") {
			continue
		}
		for _, id := range v {
			idInt, err := strconv.ParseUint(id, 10, 64)
			if err != nil {
				continue
			}
			poolIdSet[idInt] = struct{}{}
		}
	}
	if len(poolIdSet) == 0 {
		return nil
	}

	poolIds := make([]uint64, 0, len(poolIdSet))
	for id := range poolIdSet {
		poolIds = append(poolIds, id)
	}
	return poolIds
}
