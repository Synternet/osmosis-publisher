package repository

import (
	"time"

	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	_ "github.com/lib/pq"
	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/repository"
	"gorm.io/gorm/clause"
)

func (r *Repository) SaveIBCDenom(ibc IBCTypes.DenomTrace) error {
	ibcDenom := IBCDenom{
		IBC:       ibc.IBCDenom(),
		Path:      ibc.Path,
		BaseDenom: ibc.BaseDenom,
	}
	result := r.dbCon.Clauses(clause.OnConflict{DoNothing: true}).Model(&IBCDenom{}).Create(&ibcDenom)
	return result.Error
}

func (r *Repository) SaveTokenPrice(price repository.TokenPrice) error {
	ibcDenom := TokenPrice{
		LastUpdated: price.LastUpdated.UnixNano(),
		Value:       price.Value,
		Name:        price.Name,
		Base:        price.Base,
	}
	result := r.dbCon.Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"value", "last_updated"})}).Model(&TokenPrice{}).Create(&ibcDenom)
	return result.Error
}

func (r *Repository) SavePool(pool repository.Pool) error {
	newPool := Pool{
		Timestamp: pool.Timestamp,
		Height:    pool.Height,
		PoolId:    pool.PoolId,
		Liquidity: pool.Liquidity.String(),
		Volume:    pool.Volume.String(),
	}
	result := r.dbCon.Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"liquidity", "volume", "timestamp"})}).Model(&Pool{}).Create(&newPool)
	return result.Error
}

// PruneTokenPrices will remove all token prices prior timestamp.
func (r *Repository) PruneTokenPrices(timestamp time.Time) (int, error) {
	result := r.dbCon.Model(&TokenPrice{}).Delete(&TokenPrice{}, "last_updated < ?", timestamp.UnixNano())
	return int(result.RowsAffected), result.Error
}

// PrunePools will remove all pools prior block height.
func (r *Repository) PrunePools(height uint64) (int, error) {
	result := r.dbCon.Model(&Pool{}).Delete(&Pool{}, "height < ?", height)
	return int(result.RowsAffected), result.Error
}
