package repository

import (
	"time"

	types "github.com/cosmos/cosmos-sdk/types"
)

type TokenPrice struct {
	LastUpdated time.Time
	Value       float64
	Name        string
	Base        string
}

type Pool struct {
	Timestamp time.Time
	Height    uint64
	PoolId    uint64
	Liquidity types.Coins
	Volume    types.Coins
}

type CalculatedVolume struct {
	HeightStart uint64
	HeightEnd   uint64
	PoolId      uint64
	Volume      types.Coins
	Value       []float64
}

type MissingPool struct {
	Height uint64
	PoolId uint64
}
