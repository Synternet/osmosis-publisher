package osmosis

import (
	"time"

	"github.com/syntropynet/data-layer-sdk/pkg/options"
	"github.com/syntropynet/data-layer-sdk/pkg/service"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var (
	TendermintAPIParam = "tm"
	RPCAPIParam        = "rpc"
	GRPCAPIParam       = "grpc"
	MempoolPeriodParam = "mmp"
	PoolIdsParam       = "pids"
	BlocksToIndexParam = "bti"
	PriceSubjectParam  = "prices"
)

func WithTendermintAPI(url string) options.Option {
	return func(o *options.Options) {
		service.WithParam(TendermintAPIParam, url)(o)
	}
}

func (p *Publisher) TendermintApi() string {
	return options.Param(p.Options, TendermintAPIParam, "tcp://localhost:26657")
}

func WithRPCAPI(url string) options.Option {
	return func(o *options.Options) {
		service.WithParam(RPCAPIParam, url)(o)
	}
}

func (p *Publisher) RPCApi() string {
	return options.Param(p.Options, RPCAPIParam, "http://localhost:1317")
}

func WithGRPCAPI(url string) options.Option {
	return func(o *options.Options) {
		service.WithParam(GRPCAPIParam, url)(o)
	}
}

func (p *Publisher) GRPCApi() string {
	return options.Param(p.Options, GRPCAPIParam, "localhost:9090")
}

func WithMempoolPeriod(d time.Duration) options.Option {
	return func(o *options.Options) {
		service.WithParam(MempoolPeriodParam, d)(o)
	}
}

func (p *Publisher) MempoolPeriod() time.Duration {
	return options.Param(p.Options, MempoolPeriodParam, time.Millisecond*50)
}

func WithPoolIds(ids []uint64) options.Option {
	idsMap := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		idsMap[id] = struct{}{}
	}
	ids = maps.Keys(idsMap)
	slices.SortFunc(ids, func(a uint64, b uint64) bool { return a < b })

	return func(o *options.Options) {
		service.WithParam(PoolIdsParam, ids)(o)
	}
}

func (p *Publisher) PoolIds() []uint64 {
	return options.Param(p.Options, PoolIdsParam, []uint64{})
}

func WithBlocksToIndex(blocks uint64) options.Option {
	return func(o *options.Options) {
		service.WithParam(BlocksToIndexParam, blocks)(o)
	}
}

func (p *Publisher) BlocksToIndex() uint64 {
	return options.Param(p.Options, BlocksToIndexParam, uint64(0))
}

func WithPriceSubject(url string) options.Option {
	return func(o *options.Options) {
		service.WithParam(PriceSubjectParam, url)(o)
	}
}

func (p *Publisher) PriceSubject() string {
	return options.Param(p.Options, PriceSubjectParam, "syntropy_defi.price.OSMO")
}
