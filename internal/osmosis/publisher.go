package osmosis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/syntropynet/data-layer-sdk/pkg/options"
	"github.com/syntropynet/data-layer-sdk/pkg/service"
	indexerimpl "gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/internal/indexer"
	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/indexer"
	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/repository"
	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

type Publisher struct {
	*service.Service
	rpc       *rpc
	db        repository.Repository
	indexer   indexer.Indexer
	chainId   string
	priceFeed *nats.Subscription

	mempoolMessages   atomic.Uint64
	publishedMessages atomic.Uint64
	counter           atomic.Int64
	blockCounter      atomic.Uint64
	txCounter         atomic.Uint64
	poolCounter       atomic.Uint64
	errCounter        atomic.Uint64
	evtOtherCounter   atomic.Uint64
}

func New(db repository.Repository, opts ...options.Option) *Publisher {
	ret := &Publisher{
		Service: &service.Service{},
		db:      db,
	}

	ret.Configure(opts...)

	rpc, err := newRpc(ret.Context, ret.Cancel, ret.Group, db, ret.getDenoms, ret.TendermintApi(), ret.GRPCApi())
	if err != nil {
		log.Println("Could not connect to Osmosis: ", err.Error())
		return nil
	}
	ret.rpc = rpc

	indexer, err := indexerimpl.New(ret.Context, ret.Cancel, ret.Group, db, rpc, ret.PoolIds(), ret.BlocksToIndex(), ret.VerboseLog)
	if err != nil {
		log.Println("Could not create an indexer: ", err.Error())
		return nil
	}
	ret.indexer = indexer

	id, err := rpc.ChainID()
	if err != nil {
		log.Println("Failed to retrieve chain ID: ", err.Error())
		return nil
	}
	ret.chainId = id
	log.Println("Chain ID:", id)

	ret.AddStatusCallback(ret.getStatus)
	ret.AddStatusCallback(ret.indexer.GetStatus)
	ret.AddStatusCallback(ret.rpc.getStatus)

	return ret
}

func (p *Publisher) NewNonce() string {
	return fmt.Sprint(p.counter.Add(1))
}

func (p *Publisher) SubscribeAll() error {
	if err := p.subscribePriceFeed(); err != nil {
		return fmt.Errorf("failed subscribing to price feed: %w", err)
	}
	if err := p.subscribeBlocks(); err != nil {
		return fmt.Errorf("failed subscribing to blocks: %w", err)
	}
	if err := p.subscribeTransactions(); err != nil {
		return fmt.Errorf("failed subscribing to txs: %w", err)
	}
	if err := p.subscribeOsmosisEvents(); err != nil {
		return fmt.Errorf("failed subscribing to osmosis events: %w", err)
	}
	return nil
}

func (p *Publisher) Start() context.Context {
	p.mempoolMessages.Store(0)

	err := p.SubscribeAll()
	if err != nil {
		p.Fail(err)
		return p.Context
	}

	mempoolTicker := time.NewTicker(p.MempoolPeriod())
	p.Group.Go(
		func() error {
			for {
				select {
				case <-p.Context.Done():
					return nil
				case <-mempoolTicker.C:
					if p.rpc == nil {
						continue
					}
					pool, err := p.rpc.Mempool()
					if err != nil {
						log.Println("Mempool failed: ", err.Error())
						continue
					}
					if pool != nil {
						p.mempoolMessages.Add(uint64(len(pool)))
						p.Publish(
							&types.Mempool{
								Nonce:        p.NewNonce(),
								Transactions: pool,
							},
							"mempool",
						)
					}
				}
			}
		},
	)
	return p.Service.Start()
}

func (p *Publisher) Close() error {
	log.Println("Publisher.Close")
	p.Cancel(nil)
	var errArr []error

	log.Println("Publisher.priceFeed.Unsubscribe")
	errArr = append(errArr, fmt.Errorf("failure during priceFeed.Unsubscribe: %w", p.priceFeed.Unsubscribe()))

	p.RemoveStatusCallback(p.getStatus)
	p.RemoveStatusCallback(p.indexer.GetStatus)
	p.RemoveStatusCallback(p.rpc.getStatus)

	errArr = append(errArr, fmt.Errorf("failure during RPC Close: %w", p.rpc.Close()))

	log.Println("Publisher.Group.Wait")
	errGr := p.Group.Wait()
	if !errors.Is(errGr, context.Canceled) {
		errArr = append(errArr, errGr)
	}
	err := errors.Join(errArr...)
	log.Println("Publisher.Close DONE: err=", err)
	return err
}

func (p *Publisher) getStatus() map[string]any {
	return map[string]any{
		"blocks":         p.blockCounter.Swap(0),
		"unknown_events": p.evtOtherCounter.Swap(0),
		"txs":            p.txCounter.Swap(0),
		"pools":          p.poolCounter.Swap(0),
		"errors":         p.errCounter.Swap(0),
		"mempool.txs":    p.mempoolMessages.Swap(0),
		"published":      p.publishedMessages.Swap(0),
	}

}

// DiagnosticsObtainLiquidity sequentially retrieves pool Liquidity from minHeight till maxHeight.
// It will panic on error.
func (p *Publisher) DiagnosticsObtainLiquidity(poolId uint64, minHeight, maxHeight int64) []sdktypes.Coins {
	lastBlock, err := p.rpc.LatestBlockHeight()
	if err != nil {
		log.Fatal(err)
	}

	if minHeight < 0 {
		minHeight += lastBlock
	}
	if maxHeight <= 0 {
		maxHeight += lastBlock
	}
	if maxHeight <= minHeight {
		log.Fatal("Nothing to do: ", minHeight, maxHeight)
	}
	log.Printf("Fetching %d blocks from %d to %d", maxHeight-minHeight, minHeight, maxHeight)

	liquidity := make([]sdktypes.Coins, maxHeight-minHeight)
	for i := range liquidity {
		pl, err := p.rpc.PoolsTotalLiquidityAt(minHeight+int64(i), poolId)
		if err != nil {
			log.Fatal("get failed: ", err, "; ", minHeight, maxHeight, i, lastBlock)
		}
		liquidity[i] = pl[0].Liquidity
	}
	return liquidity
}

// DiagnosticsObtainVolume sequentially retrieves pool volume from minHeight till maxHeight.
// It will panic on error.
func (p *Publisher) DiagnosticsObtainVolume(poolId uint64, minHeight, maxHeight int64) []sdktypes.Coins {
	lastBlock, err := p.rpc.LatestBlockHeight()
	if err != nil {
		log.Fatal(err)
	}

	if minHeight < 0 {
		minHeight += lastBlock
	}
	if maxHeight <= 0 {
		maxHeight += lastBlock
	}
	if maxHeight <= minHeight {
		log.Fatal("Nothing to do: ", minHeight, maxHeight)
	}

	log.Printf("Fetching %d blocks from %d to %d", maxHeight-minHeight, minHeight, maxHeight)

	volumes := make([]sdktypes.Coins, maxHeight-minHeight)
	for i := range volumes {
		pl, err := p.rpc.PoolsVolumeAt(minHeight+int64(i), poolId)
		if err != nil {
			log.Fatal("get failed: ", err, "; ", minHeight, maxHeight, i, lastBlock)
		}
		volumes[i] = pl[0].Volume
	}
	return volumes
}

func (p *Publisher) MakeSentinel(timeout time.Duration) func() error {
	sentinel := time.NewTimer(timeout)
	lastEvent := time.Now()
	p.Group.Go(
		func() error {
			defer sentinel.Stop()
			for {
				select {
				case <-p.Context.Done():
					log.Println("sentinel: c.Context Done")
					return nil
				case <-sentinel.C:
					err := fmt.Errorf("event subscription timed out, last seen: %s", time.Since(lastEvent))
					return err
				}
			}
		},
	)

	return func() error {
		if !sentinel.Stop() {
			return fmt.Errorf("event subscription timed out while resetting, last seen: %s", time.Since(lastEvent))
		}
		sentinel.Reset(timeout)
		lastEvent = time.Now()
		return nil
	}
}

func (p *Publisher) getDenoms(ibcTrace IBCDenomTrace) error {
	for denom := range ibcTrace {
		res, err := p.indexer.DenomTrace(denom)
		if err != nil {
			log.Printf("indexer.DenomTrace failed for denom %s: \n %s", denom, err.Error())
		} else {
			ibcTrace[denom] = res
		}
	}
	return nil
}
