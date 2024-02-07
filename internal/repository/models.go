package repository

import (
	"time"
)

type IBCDenom struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	IBC       string `gorm:"index:idx_ibc,unique"`
	Path      string
	BaseDenom string
}

type TokenPrice struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastUpdated int64 `gorm:"index:idx_token_price,unique"`
	Value       float64
	Name        string `gorm:"index:idx_token_price,unique"`
	Base        string `gorm:"index:idx_token_price,unique"`
}

type Pool struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Timestamp time.Time
	Height    uint64 `gorm:"index:idx_pool_id,unique"`
	PoolId    uint64 `gorm:"index:idx_pool_id,unique"`
	Liquidity string
	Volume    string
}
