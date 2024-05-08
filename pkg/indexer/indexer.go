package indexer

import (
	"time"

	"github.com/Synternet/osmosis-publisher/pkg/types"
	ibctypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

type Indexer interface {
	// DenomTrace returns IBC Denom trace when IBC denom is provided (in the form `ibc/<hash>`)
	DenomTrace(ibc string) (ibctypes.DenomTrace, error)

	// SetLatestBlockHeight should be called at each block received
	SetLatestBlockHeight(height uint64, blockTime time.Time)

	// SetLatestPrice should be called every time a new price quote is received from price feed
	SetLatestPrice(token, base string, value float64, lastUpdated time.Time) error

	// PoolStatusesAt returns poolStatuses for a specific height given pool IDs
	PoolStatusesAt(height uint64, poolId ...uint64) ([]types.PoolStatus, uint64, error)

	// CalculateVolumes will modify poolStatuses in-place by calculating USD prices of volumes
	//
	// NOTE: volume will have two prices: actual price and price difference
	// between volume retrieved for some height and the latest height volume was retrieved for.
	CalculateVolumes(poolStatuses []types.PoolStatus) error

	// GetStatus used for telemetry and will return a map of status variables
	GetStatus() map[string]string
	AverageBlockTime() time.Duration
}
