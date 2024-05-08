package types

import (
	"github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Transaction struct {
	Nonce    string `json:"nonce"`
	Raw      string `json:"raw"`
	Code     uint32 `json:"code"`
	TxID     string `json:"tx_id"`
	Tx       any    `json:"tx"`
	TxResult any    `json:"tx_result"`
	Metadata any    `json:"metadata"`
}

func (*Transaction) ProtoReflect() protoreflect.Message { return nil }

type Block struct {
	Nonce string `json:"nonce"`
	Block any    `json:"block"`
}

func (*Block) ProtoReflect() protoreflect.Message { return nil }

type PoolOfInterest struct {
	Nonce        string       `json:"nonce"`
	BlockHeight  int64        `json:"block_height"`
	AvgBlockTime float64      `json:"avg_block_time"`
	BlockHash    string       `json:"block_hash"`
	Pools        []PoolStatus `json:"pools"`
	Metadata     any          `json:"metadata"`
}

func (*PoolOfInterest) ProtoReflect() protoreflect.Message { return nil }

type Mempool struct {
	Nonce        string         `json:"nonce"`
	Transactions []*Transaction `json:"txs"`
}

func (*Mempool) ProtoReflect() protoreflect.Message { return nil }

type Pools struct {
	Nonce        string       `json:"nonce"`
	BlockHeight  int64        `json:"block_height"`
	AvgBlockTime float64      `json:"avg_block_time"`
	BlockHash    string       `json:"block_hash"`
	Pools        []any        `json:"pools"`
	PoolStatus   []PoolStatus `json:"pools_status"`
	Events       any          `json:"events"`
	Metadata     any          `json:"metadata"`
}

func (Pools) ProtoReflect() protoreflect.Message { return nil }

type PoolStatusVolumeAt struct {
	BlockHeight       int64       `json:"block_height"`
	Volume            types.Coins `json:"volume"`
	VolumeUSD         []float64   `json:"volume_usd"`
	RelativeVolumeUSD []float64   `json:"relative_volume_usd"`
}

type PoolStatus struct {
	PoolId         uint64               `json:"pool_id"`
	TotalLiquidity types.Coins          `json:"total_liquidity"`
	Volumes        []PoolStatusVolumeAt `json:"total_volume"`
}

// RPC types used by Indexer
type PoolLiquidity struct {
	PoolId    uint64      `json:"pool_id"`
	Liquidity types.Coins `json:"liquidity"`
}

type PoolVolume struct {
	PoolId uint64      `json:"pool_id"`
	Volume types.Coins `json:"volume"`
}
