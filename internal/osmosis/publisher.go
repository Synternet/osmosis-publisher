package osmosis

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/syntropynet/data-layer-sdk/pkg/options"
	"github.com/syntropynet/data-layer-sdk/pkg/service"
	indexerimpl "github.com/syntropynet/osmosis-publisher/internal/indexer"
	"github.com/syntropynet/osmosis-publisher/pkg/indexer"
	"github.com/syntropynet/osmosis-publisher/pkg/repository"
	"github.com/syntropynet/osmosis-publisher/pkg/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// Total counters
	blocksCounter       prometheus.Counter
	transactionsCounter prometheus.Counter
	mempoolCounter      prometheus.Counter
	messagesCounter     prometheus.Counter
	pricesCounter       prometheus.Counter

	// Gauges
	blockHeight prometheus.Gauge

	// Histograms
}

func New(db repository.Repository, opts ...options.Option) (*Publisher, error) {
	ret := &Publisher{
		Service: &service.Service{},
		db:      db,
		blocksCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "osmosis_publisher_blocks",
			Help: "The total number of processed blocks",
		}),
		transactionsCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "osmosis_publisher_transactions",
			Help: "The total number of processed transactions",
		}),
		mempoolCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "osmosis_publisher_transactions_mempool",
			Help: "The total number of processed mempool transactions",
		}),
		messagesCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "osmosis_publisher_messages",
			Help: "The total number of messages published",
		}),
		pricesCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "osmosis_publisher_prices",
			Help: "The total number of processed price messages",
		}),
		blockHeight: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "osmosis_publisher_block_height",
			Help: "The latest block height as seen from the Osmosis blockchain",
		}),
	}

	ret.Configure(opts...)

	ret.Logger.Info("Tracking pools", "ids", ret.PoolIds())

	rpc, err := newRpc(ret.Context, ret.Cancel, ret.Group, ret.Logger, db, ret.getDenoms, ret.TendermintApi(), ret.GRPCApi())
	if err != nil {
		return nil, fmt.Errorf("failed connecting to Osmosis: %w", err)
	}
	ret.rpc = rpc

	indexer, err := indexerimpl.New(ret.Context, ret.Cancel, ret.Group, ret.Logger, db, rpc, ret.PoolIds(), ret.BlocksToIndex(), ret.VerboseLog)
	if err != nil {
		return nil, fmt.Errorf("failed creating an indexer: %w", err)
	}
	ret.indexer = indexer

	id, err := rpc.ChainID()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chainId: %w", err)
	}
	ret.chainId = id
	ret.Logger.Info("Chain", "id", id)

	ret.AddStatusCallback(ret.getStatus)
	ret.AddStatusCallback(ret.indexer.GetStatus)
	ret.AddStatusCallback(ret.rpc.getStatus)

	// Setup durable price stream to support at most 12h of downtime
	err = ret.AddStream(12*3600, 12*3600*92, time.Hour*12, ret.PriceSubject())
	if err != nil {
		ret.Logger.Error("AddStream failed, durable stream unavailable", "err", err, "subject", ret.PriceSubject())
	}

	return ret, nil
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

	if metricsUrl := p.MetricsURL(); metricsUrl != "" {
		p.Group.Go(
			func() error {
				http.Handle("/metrics", promhttp.Handler())
				http.ListenAndServe(metricsUrl, nil)
				return nil
			},
		)
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
						p.Logger.Warn("Mempool failed: ", "err", err)
						continue
					}
					if pool != nil {
						p.mempoolMessages.Add(uint64(len(pool)))
						p.mempoolCounter.Add(float64(len(pool)))
						p.Publish(
							&types.Mempool{
								Nonce:        p.NewNonce(),
								Transactions: pool,
							},
							"mempool",
						)
						p.messagesCounter.Add(1)
					}
				}
			}
		},
	)
	return p.Service.Start()
}

func (p *Publisher) Close() error {
	p.Logger.Info("Publisher.Close")
	p.Cancel(nil)
	var errArr []error

	p.Logger.Info("Publisher.priceFeed.Unsubscribe")
	errArr = append(errArr, fmt.Errorf("failure during priceFeed.Unsubscribe: %w", p.priceFeed.Unsubscribe()))

	p.RemoveStatusCallback(p.getStatus)
	p.RemoveStatusCallback(p.indexer.GetStatus)
	p.RemoveStatusCallback(p.rpc.getStatus)

	errArr = append(errArr, fmt.Errorf("failure during RPC Close: %w", p.rpc.Close()))

	p.Logger.Info("Publisher.Group.Wait")
	errGr := p.Group.Wait()
	if !errors.Is(errGr, context.Canceled) {
		errArr = append(errArr, errGr)
	}
	err := errors.Join(errArr...)
	p.Logger.Info("Publisher.Close DONE", "err", err)
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
		panic(err)
	}

	if minHeight < 0 {
		minHeight += lastBlock
	}
	if maxHeight <= 0 {
		maxHeight += lastBlock
	}
	if maxHeight <= minHeight {
		panic(fmt.Errorf("Nothing to do: min=%d max=%d", minHeight, maxHeight))
	}
	p.Logger.Info("Fetching blocks", "num", maxHeight-minHeight, "from", minHeight, "to", maxHeight)

	liquidity := make([]sdktypes.Coins, maxHeight-minHeight)
	for i := range liquidity {
		pl, err := p.rpc.PoolsTotalLiquidityAt(minHeight+int64(i), poolId)
		if err != nil {
			panic(fmt.Errorf("get failed: %v; %d %d %d %d", err, minHeight, maxHeight, i, lastBlock))
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
		panic(err)
	}

	if minHeight < 0 {
		minHeight += lastBlock
	}
	if maxHeight <= 0 {
		maxHeight += lastBlock
	}
	if maxHeight <= minHeight {
		panic(fmt.Errorf("Nothing to do: %d %d", minHeight, maxHeight))
	}

	p.Logger.Info("Fetching blocks", "num", maxHeight-minHeight, "from", minHeight, "to", maxHeight)

	volumes := make([]sdktypes.Coins, maxHeight-minHeight)
	for i := range volumes {
		pl, err := p.rpc.PoolsVolumeAt(minHeight+int64(i), poolId)
		if err != nil {
			panic(fmt.Errorf("get failed: %v %d %d %d %d", err, minHeight, maxHeight, i, lastBlock))
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
					p.Logger.Info("sentinel: c.Context Done")
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
			p.Logger.Error("indexer.DenomTrace failed", "denom", denom, "err", err)
		} else {
			ibcTrace[denom] = res
		}
	}
	return nil
}
