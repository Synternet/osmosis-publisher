package repository

import (
	"time"

	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

type Repository interface {
	// IBCDenom will return a denom mapped from IBC denom
	IBCDenom(ibcDenom string) (IBCTypes.DenomTrace, bool)
	// IBCDenomAll will return all ibc trace denoms
	IBCDenomAll() []IBCTypes.DenomTrace
	// TokenPrice will return token price record at a fiven timestamp
	TokenPrice(timestamp time.Time, denom string) (TokenPrice, bool)
	// NearestTokenPrice will return:
	//
	//   - Same as TokenPrice if the denom exists at specified timestamp
	//   - Nearest token price from timestamp < specified timestamp
	//   - Nearest token price from timestamp > specified timestamp
	NearestTokenPrice(timestamp time.Time, denom string) ([]TokenPrice, bool)
	// LatestTokenPrice will return latest token price
	LatestTokenPrice(denom string) (TokenPrice, bool)
	// LatestPool will return latest pool
	LatestPool(id uint64) (Pool, bool)

	// PoolsRange will return a list of available pools from minimum to maximum heights
	PoolsRange(minHeight, maxHeight, poolId uint64) ([]Pool, error)

	TokenPricesRange(min, max time.Time, denom string) ([]TokenPrice, error)

	// CalculateVolume(poolId uint64, latestHeight uint64, period time.Duration) (CalculatedVolume, bool)
	// MissingPools(heightStart, heightEnd uint64) []MissingPool

	SaveIBCDenom(IBCTypes.DenomTrace) error
	SaveTokenPrice(TokenPrice) error
	SavePool(Pool) error
}
