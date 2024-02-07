package osmosis

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/repository"
	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/types"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"

	tmlog "github.com/cometbft/cometbft/libs/log"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	tmtypes "github.com/cometbft/cometbft/types"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	types1 "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/osmosis-labs/osmosis/v22/app"
	"github.com/osmosis-labs/osmosis/v22/app/params"
	"github.com/osmosis-labs/osmosis/v22/x/poolmanager/client/queryproto"
	pmtypes "github.com/osmosis-labs/osmosis/v22/x/poolmanager/types"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/metadata"
)

const subscriberName = "dlosmpub"

type rpc struct {
	ctx           context.Context
	group         *errgroup.Group
	cancel        context.CancelCauseFunc
	db            repository.Repository
	tendermintUrl string
	grpcApiURL    string
	tendermint    *rpchttp.HTTP
	grpc          *grpc.ClientConn
	mempoolSet    map[string]struct{}
	enccfg        params.EncodingConfig

	pmQueryClient  queryproto.QueryClient
	ibcQueryClient IBCTypes.QueryClient

	errCounter     atomic.Uint64
	evtCounter     atomic.Uint64
	evtSkipCounter atomic.Uint64
	queueMaxSize   atomic.Uint64
	maxQueueSize   uint64

	getDenoms func(ibcTrace IBCDenomTrace) error
}

// Will add Height to gRPC call context. This will instruct the full node to return the state at that height.
// NOTE: It will ignore height values <= 0
func ContextWithHeight(ctx context.Context, height int64) context.Context {
	if height <= 0 {
		return ctx
	}
	return metadata.AppendToOutgoingContext(
		ctx,
		grpctypes.GRPCBlockHeightHeader,
		strconv.FormatInt(height, 10),
	)
}

func newRpc(ctx context.Context, cancel context.CancelCauseFunc, group *errgroup.Group, db repository.Repository, getDenoms func(ibcTrace IBCDenomTrace) error, tendermintUrl, grpcApiURL string) (*rpc, error) {
	ret := &rpc{
		ctx:           ctx,
		group:         group,
		cancel:        cancel,
		tendermintUrl: tendermintUrl,
		grpcApiURL:    grpcApiURL,
		mempoolSet:    make(map[string]struct{}),
		db:            db,
		enccfg:        app.MakeEncodingConfig(),
		getDenoms:     getDenoms,
	}

	log.Printf("Using tendermint=%s grpc=%s\n", tendermintUrl, grpcApiURL)

	grpcConn, err := grpc.Dial(
		grpcApiURL,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Println("Could not connect to Osmosis: ", err.Error())
		return nil, err
	}
	ret.grpc = grpcConn

	tmlog.AllowAll()

	client, err := rpchttp.NewWithTimeout(tendermintUrl, "/websocket", 3)
	if err != nil {
		return nil, err
	}

	err = client.Start()
	if err != nil {
		return nil, err
	}
	ret.tendermint = client

	ret.pmQueryClient = queryproto.NewQueryClient(ret.grpc)
	ret.ibcQueryClient = IBCTypes.NewQueryClient(ret.grpc)

	return ret, nil
}

func (c *rpc) Close() error {
	log.Println("Publisher.RPC.Close")
	c.cancel(nil)
	var errArr []error
	if c.tendermint != nil {
		log.Println("Publisher.RPC.UnsubscribeAll")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		errArr = append(errArr, c.tendermint.UnsubscribeAll(ctx, subscriberName))
		log.Println("Publisher.RPC.tendermint.Stop")
		errArr = append(errArr, c.tendermint.Stop())
	}
	log.Println("Publisher.RPC.group.Wait")
	errGr := c.group.Wait()
	if !errors.Is(errGr, context.Canceled) {
		errArr = append(errArr, errGr)
	}
	err := errors.Join(errArr...)
	log.Println("Publisher.RPC.Close DONE: err=", err)
	return err
}

func (c *rpc) DenomTrace(ibc string) (IBCTypes.DenomTrace, error) {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second)
	defer cancel()

	req := &IBCTypes.QueryDenomTraceRequest{
		Hash: ibc,
	}

	res, err := c.ibcQueryClient.DenomTrace(ctx, req)
	if err != nil {
		return IBCTypes.DenomTrace{}, err
	}

	return *res.DenomTrace, nil
}

func (c *rpc) DenomTraces() ([]IBCTypes.DenomTrace, error) {
	traces := make([]IBCTypes.DenomTrace, 0, 10)
	var nextPageKey []byte

	for {
		req := &IBCTypes.QueryDenomTracesRequest{
			Pagination: &query.PageRequest{
				Key:   nextPageKey,
				Limit: 100, // Adjust the limit as necessary
			},
		}
		ctx, cancel := context.WithTimeout(c.ctx, time.Second)
		res, err := c.ibcQueryClient.DenomTraces(ctx, req)
		cancel()
		if err != nil {
			c.errCounter.Add(1)
			log.Printf("Failed to fetch denom traces: %v\n", err)
			return traces, err
		}

		traces = append(traces, res.DenomTraces...)

		nextPageKey = res.Pagination.NextKey
		if nextPageKey == nil {
			break
		}
	}

	return traces, nil
}

func (c *rpc) BlockAt(height int64) (*tmtypes.Block, error) {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second*5)
	ctx = ContextWithHeight(ctx, height)
	defer cancel()
	info, err := c.tendermint.Block(ctx, nil)
	if err != nil {
		c.errCounter.Add(1)
		return nil, err
	}

	return info.Block, nil
}

func (c *rpc) ChainID() (string, error) {
	block, err := c.BlockAt(0)
	if err != nil {
		c.errCounter.Add(1)
		return "", err
	}

	return block.ChainID, nil
}

func (c *rpc) LatestBlockHeight() (int64, error) {
	block, err := c.BlockAt(0)
	if err != nil {
		c.errCounter.Add(1)
		return 0, err
	}

	return block.Height, nil
}

func (c *rpc) Subscribe(eventName string, handle func(events <-chan ctypes.ResultEvent) error) error {
	ctx, cancel := context.WithTimeout(c.ctx, time.Second*5)
	events, err := c.tendermint.Subscribe(ctx, subscriberName, eventName, 10)
	cancel()
	if err != nil {
		return err
	}
	c.group.Go(func() error {
		return handle(c.bufferChannel(eventName, events, 2048))
	})
	return nil
}

func (c *rpc) Mempool() ([]*types.Transaction, error) {
	var limit int = 1000
	ctx, cancel := context.WithTimeout(c.ctx, time.Second*5)
	defer cancel()
	res, err := c.tendermint.UnconfirmedTxs(ctx, &limit)
	if err != nil {
		c.errCounter.Add(1)
		return nil, err
	}
	if res.Count == 0 {
		return nil, nil
	}

	// NOTE: This should never be called asynchronously, therefore no need to synchronize
	currentSet := make(map[string]struct{}, res.Count)

	txs := make([]*types.Transaction, 0, res.Count)
	for _, tx := range res.Txs {
		hash := hex.EncodeToString(tx.Hash())
		currentSet[hash] = struct{}{}
		if _, ok := c.mempoolSet[hash]; ok {
			continue
		}
		c.mempoolSet[hash] = struct{}{}

		res := c.translateTransaction(tx, hash, "", nil, nil)
		txs = append(txs, res)

		log.Println("Mempool: ", hash)
	}
	// Remove hashes from mempoolSet that were not observed in the mempool this time.
	// That means that the tx was removed from the mempool.
	for k := range c.mempoolSet {
		if _, ok := currentSet[k]; ok {
			continue
		}
		delete(c.mempoolSet, k)
	}

	if len(txs) == 0 {
		return nil, nil
	}

	return txs, nil
}

func (c *rpc) PoolsAt(height int64, ids ...uint64) ([]*pmtypes.PoolI, error) {
	if ids == nil {
		ctx, cancel := context.WithTimeout(c.ctx, time.Second*5)
		ctx = ContextWithHeight(ctx, height)
		defer cancel()
		resp, err := c.pmQueryClient.AllPools(ctx, &queryproto.AllPoolsRequest{})
		if err != nil {
			c.errCounter.Add(1)
			return nil, fmt.Errorf("failed retrieving all pools: %w", err)
		}
		return c.translatePools(resp.Pools)
	}

	pools := make([]*types1.Any, len(ids))
	for i, id := range ids {
		ctx, cancel := context.WithTimeout(c.ctx, time.Second*5)
		ctx = ContextWithHeight(ctx, height)
		resp, err := c.pmQueryClient.Pool(ctx, &queryproto.PoolRequest{PoolId: id})
		cancel()
		if err != nil {
			c.errCounter.Add(1)
			return nil, fmt.Errorf("failed retrieving pool %d: %w", id, err)
		}
		pools[i] = resp.Pool
	}

	return c.translatePools(pools)
}

func (c *rpc) PoolsTotalLiquidityAt(height int64, ids ...uint64) ([]types.PoolLiquidity, error) {
	pools := make([]types.PoolLiquidity, len(ids))
	for i, id := range ids {
		ctx, cancel := context.WithTimeout(c.ctx, time.Second)
		ctx = ContextWithHeight(ctx, height)
		resp, err := c.pmQueryClient.TotalPoolLiquidity(ctx, &queryproto.TotalPoolLiquidityRequest{PoolId: id})
		cancel()
		if err != nil {
			c.errCounter.Add(1)
			return nil, fmt.Errorf("failed retrieving pool liquidity %d: %w", id, err)
		}
		pools[i] = types.PoolLiquidity{
			PoolId:    id,
			Liquidity: resp.Liquidity,
		}
	}

	return pools, nil
}

func (c *rpc) PoolsVolumeAt(height int64, ids ...uint64) ([]types.PoolVolume, error) {
	pools := make([]types.PoolVolume, len(ids))
	for i, id := range ids {
		ctx, cancel := context.WithTimeout(c.ctx, time.Second)
		ctx = ContextWithHeight(ctx, height)
		resp, err := c.pmQueryClient.TotalVolumeForPool(ctx, &queryproto.TotalVolumeForPoolRequest{PoolId: id})
		cancel()
		if err != nil {
			c.errCounter.Add(1)
			return nil, fmt.Errorf("failed retrieving pool volume %d: %w", id, err)
		}
		pools[i] = types.PoolVolume{
			PoolId: id,
			Volume: resp.Volume,
		}
	}

	return pools, nil
}

func (p *rpc) getStatus() map[string]any {
	queueSize := p.queueMaxSize.Swap(0)
	if queueSize > p.maxQueueSize {
		p.maxQueueSize = queueSize
	}

	return map[string]any{
		"errors": p.errCounter.Swap(0),
		"events": map[string]any{
			"total":     p.evtCounter.Swap(0),
			"skipped":   p.evtSkipCounter.Load(),
			"queue":     queueSize,
			"max_queue": p.maxQueueSize,
		},
	}
}

func (c *rpc) bufferChannel(name string, events <-chan ctypes.ResultEvent, size int) <-chan ctypes.ResultEvent {
	ch := make(chan ctypes.ResultEvent, size)
	c.group.Go(func() error {
		defer close(ch)
		defer log.Printf("bufferChannel %s exit", name)
		for {
			select {
			case <-c.ctx.Done():
				log.Println("bufferChannel: Context Done; queue: ", len(events))
				return nil
			case ev, ok := <-events:
				if !ok {
					log.Println("bufferChannel: events closed; queue: ", len(events))
					return nil
				}

				c.evtCounter.Add(1)

				setMaxValue(&c.queueMaxSize, uint64(len(ch)))

				select {
				case ch <- ev:
				default:
					c.evtSkipCounter.Add(1)
					log.Println("bufferChannel: Overflow! Skipping event: ", ev.Query)
				}
			}
		}
	})
	return ch
}

func setMaxValue(a *atomic.Uint64, v uint64) {
	for {
		oldValue := a.Load()
		if oldValue >= v {
			return
		}

		if a.CompareAndSwap(oldValue, v) {
			return
		}
	}
}
