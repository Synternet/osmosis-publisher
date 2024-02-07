package indexer

import (
	"context"
	"sync/atomic"
	"time"

	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/indexer"
	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/repository"
	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/types"
	"golang.org/x/sync/errgroup"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	pmtypes "github.com/osmosis-labs/osmosis/v22/x/poolmanager/types"
)

var _ indexer.Indexer = (*Indexer)(nil)

const (
	DefaultBlocksPerHour = 720
)

type ExpectedRPC interface {
	DenomTrace(ibc string) (IBCTypes.DenomTrace, error)
	DenomTraces() ([]IBCTypes.DenomTrace, error)
	BlockAt(height int64) (*tmtypes.Block, error)
	ChainID() (string, error)
	Close() error
	Mempool() ([]*types.Transaction, error)
	PoolsAt(height int64, ids ...uint64) ([]*pmtypes.PoolI, error)
	PoolsTotalLiquidityAt(height int64, ids ...uint64) ([]types.PoolLiquidity, error)
	PoolsVolumeAt(height int64, ids ...uint64) ([]types.PoolVolume, error)
	Subscribe(eventName string, handle func(events <-chan ctypes.ResultEvent) error) error
}

type Indexer struct {
	ctx    context.Context
	group  *errgroup.Group
	cancel context.CancelCauseFunc
	repo   repository.Repository
	rpc    ExpectedRPC

	ibcTraceCache map[string]IBCTypes.DenomTrace
	ibcMisses     atomic.Uint64

	errCounter atomic.Uint64

	syncHeights        chan uint64
	poolIdsToMonitor   []uint64
	pools              PoolMap
	prices             PriceMap
	currentBlockHeight atomic.Uint64
	currentBlockTime   atomic.Int64
	blocksPerHour      atomic.Int64
	lastBlockHeight    uint64
	lastBlockTimestamp atomic.Int64

	verbose bool
}

func New(ctx context.Context, cancel context.CancelCauseFunc, group *errgroup.Group, repo repository.Repository, rpc ExpectedRPC, poolIds []uint64, blocks uint64, verbose bool) (*Indexer, error) {
	ret := &Indexer{
		ctx:    ctx,
		group:  group,
		cancel: cancel,
		repo:   repo,
		rpc:    rpc,
		pools: PoolMap{
			pools: make(map[uint64]map[uint64]repository.Pool, blocks),
		},
		prices: PriceMap{
			prices: make(map[string][]repository.TokenPrice),
		},
		syncHeights:      make(chan uint64, DefaultBlocksPerHour),
		poolIdsToMonitor: make([]uint64, len(poolIds)),
		ibcTraceCache:    make(map[string]IBCTypes.DenomTrace),
		verbose:          verbose,
	}
	ret.blocksPerHour.Store(DefaultBlocksPerHour)
	block, err := rpc.BlockAt(0)
	if err != nil {
		return nil, err
	}
	ret.currentBlockHeight.Store(uint64(block.Height))
	ret.currentBlockTime.Store(block.Time.UnixNano())

	copy(ret.poolIdsToMonitor, poolIds)
	ret.preHeatDenomTraceCache()
	ret.preHeatPools(blocks)
	ret.preHeatPrices(blocks)

	group.Go(func() error {
		return ret.handleSyncing(blocks)
	})

	return ret, nil
}

func (d *Indexer) GetStatus() map[string]any {
	return map[string]any{
		"indexer": map[string]any{
			"errors":          d.errCounter.Load(),
			"blocks_per_hour": d.blocksPerHour.Load(),
			"ibc": map[string]any{
				"tokens":       len(d.ibcTraceCache),
				"cache_misses": d.ibcMisses.Load(),
			},
			"pool": map[string]any{
				"current_height": d.currentBlockHeight.Load(),
				"sync_count":     len(d.syncHeights),
				// "errors":          d.poolErrors.Load(),
				// "misses":          d.poolMisses.Load(),
			},
		},
	}
}

func (d *Indexer) SetLatestBlockHeight(height uint64, blockTime time.Time) {
	oldHeight := setMaxValue(&d.currentBlockHeight, height)

	if oldHeight >= height {
		return
	}

	d.currentBlockTime.Store(blockTime.UnixNano())

	// Assumes SetLatestBlockHeight is called at the moment a new block is received
	now := time.Now()
	if d.lastBlockHeight == 0 {
		d.lastBlockHeight = height
		d.lastBlockTimestamp.Store(now.UnixNano())
	}

	if height-d.lastBlockHeight >= uint64(d.blocksPerHour.Load()) {
		duration := time.Duration(now.UnixNano() - d.lastBlockTimestamp.Load())
		d.blocksPerHour.Store(int64(float64(height-d.lastBlockHeight) / duration.Hours()))
	}
}

func (d *Indexer) AverageBlockTime() time.Duration {
	bph := d.blocksPerHour.Load()
	if bph == 0 {
		bph = DefaultBlocksPerHour
	}

	return time.Hour / time.Duration(bph)
}
