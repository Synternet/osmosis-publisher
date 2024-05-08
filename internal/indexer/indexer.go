package indexer

import (
	"context"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Synternet/osmosis-publisher/pkg/indexer"
	"github.com/Synternet/osmosis-publisher/pkg/repository"
	"github.com/Synternet/osmosis-publisher/pkg/types"
	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"golang.org/x/sync/errgroup"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	pmtypes "github.com/osmosis-labs/osmosis/v24/x/poolmanager/types"
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
	logger *slog.Logger

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

func New(ctx context.Context, cancel context.CancelCauseFunc, group *errgroup.Group, logger *slog.Logger, repo repository.Repository, rpc ExpectedRPC, poolIds []uint64, blocks uint64, verbose bool) (*Indexer, error) {
	ret := &Indexer{
		ctx:    ctx,
		group:  group,
		cancel: cancel,
		logger: logger.With("module", "indexer"),
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

func (d *Indexer) GetStatus() map[string]string {
	return map[string]string{
		"indexer_errors":              strconv.FormatUint(d.errCounter.Load(), 10),
		"indexer_blocks_per_hour":     strconv.FormatInt(d.blocksPerHour.Load(), 10),
		"indexer_ibc_tokens":          strconv.Itoa(len(d.ibcTraceCache)),
		"indexer_ibc_cache_misses":    strconv.FormatUint(d.ibcMisses.Load(), 10),
		"indexer_pool_current_height": strconv.FormatUint(d.currentBlockHeight.Load(), 10),
		"indexer_pool_sync_count":     strconv.Itoa(len(d.syncHeights)),
		// "indexer_pool_errors":       strconv.FormatUint(d.poolErrors.Load(), 10),
		// "indexer_pool_misses":       strconv.FormatUint(d.poolMisses.Load(), 10),
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
