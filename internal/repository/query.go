package repository

import (
	"fmt"
	"log"
	"time"

	"github.com/SyntropyNet/osmosis-publisher/pkg/repository"
	sdk "github.com/cosmos/cosmos-sdk/types"
	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	_ "github.com/lib/pq"
)

func (r *Repository) IBCDenom(ibc string) (IBCTypes.DenomTrace, bool) {
	var denom IBCDenom
	result := r.dbCon.Model(&IBCDenom{}).Limit(1).Find(&denom, "ibc = ?", ibc)
	if result.Error != nil {
		log.Println("Error fetching IBC Denom from DB:", result.Error)
	}
	if result.RowsAffected == 0 {
		return IBCTypes.DenomTrace{}, false
	}
	return IBCTypes.DenomTrace{
		Path:      denom.Path,
		BaseDenom: denom.BaseDenom,
	}, true
}

func (r *Repository) IBCDenomAll() []IBCTypes.DenomTrace {
	var denoms []IBCDenom
	result := r.dbCon.Model(&IBCDenom{}).Find(&denoms)
	if result.Error != nil {
		log.Println("Error fetching all IBC Denoms from DB:", result.Error)
		return nil
	}

	traces := make([]IBCTypes.DenomTrace, len(denoms))
	for i, d := range denoms {
		traces[i] = IBCTypes.DenomTrace{
			Path:      d.Path,
			BaseDenom: d.BaseDenom,
		}
	}

	return traces
}

func (r *Repository) TokenPrice(timestamp time.Time, denom string) (repository.TokenPrice, bool) {
	var token TokenPrice
	result := r.dbCon.Model(&TokenPrice{}).Limit(1).Find(&token, "last_updated = ? AND name = ?", timestamp.UnixNano(), denom)
	if result.Error != nil {
		log.Println("Error fetching TokenPrice from DB:", result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.TokenPrice{}, false
	}
	return repository.TokenPrice{
		LastUpdated: time.Unix(0, token.LastUpdated),
		Value:       token.Value,
		Name:        token.Name,
		Base:        token.Base,
	}, true
}

// FIXME
func (r *Repository) NearestTokenPrice(timestamp time.Time, denom string) ([]repository.TokenPrice, bool) {
	var tokens []TokenPrice
	ts := timestamp.UnixNano()
	result := r.dbCon.Raw(
		fmt.Sprintf(`
	SELECT * FROM token_prices
	WHERE name = ? AND (
		last_updated = (
			SELECT MAX(last_updated) FROM token_prices WHERE last_updated <= ?
		) OR last_updated = (
			SELECT MIN(last_updated) FROM token_prices WHERE last_updated >= ?
		)
	)
	LIMIT 2
`),
		denom, ts, ts).Scan(&tokens)

	if result.Error != nil {
		log.Println("Error fetching TokenPrices from DB:", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, false
	}

	arr := make([]repository.TokenPrice, len(tokens))
	for i, token := range tokens {
		arr[i] = repository.TokenPrice{
			LastUpdated: time.Unix(0, token.LastUpdated),
			Value:       token.Value,
			Name:        token.Name,
			Base:        token.Base,
		}
	}

	return arr, true
}

// LatestTokenPrice will return latest token price.
func (r *Repository) LatestTokenPrice(denom string) (repository.TokenPrice, bool) {
	var token TokenPrice
	result := r.dbCon.Model(&TokenPrice{}).Order("last_updated DESC").Limit(1).Find(&token, "name = ?", denom)
	if result.Error != nil {
		log.Println("Error fetching TokenPrice from DB:", result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.TokenPrice{}, false
	}
	return repository.TokenPrice{
		LastUpdated: time.Unix(0, token.LastUpdated),
		Value:       token.Value,
		Name:        token.Name,
		Base:        token.Base,
	}, true
}

// LatestPool will return latest pool
func (r *Repository) LatestPool(id uint64) (repository.Pool, bool) {
	var pool Pool
	result := r.dbCon.Model(&Pool{}).Last(&pool, "pool_id = ?", id)
	if result.Error != nil {
		log.Println("Error fetching Pool from DB:", result.Error)
		return repository.Pool{}, false
	}
	if result.RowsAffected == 0 {
		return repository.Pool{}, false
	}
	liquidity, err := sdk.ParseCoinsNormalized(pool.Liquidity)
	if err != nil {
		log.Println("Error parsing pool liquidity from DB:", err)
		return repository.Pool{}, false
	}
	volume, err := sdk.ParseCoinsNormalized(pool.Volume)
	if err != nil {
		log.Println("Error parsing pool volume from DB:", err)
		return repository.Pool{}, false
	}
	return repository.Pool{
		Height:    pool.Height,
		PoolId:    pool.PoolId,
		Liquidity: liquidity,
		Volume:    volume,
	}, true
}

// PoolsRange will return pools from min to max height
func (r *Repository) PoolsRange(min, max, poolId uint64) ([]repository.Pool, error) {
	var pools []Pool
	query := "height >= ? AND height <= ? AND pool_id = ?"
	if poolId == 0 {
		query = "height >= ? AND height <= ?"
	}
	result := r.dbCon.Model(&Pool{}).Find(&pools, query, min, max, poolId)
	if result.Error != nil {
		log.Println("Error fetching Pools from DB:", result.Error)
		return nil, result.Error
	}

	ret := make([]repository.Pool, len(pools))
	for i, p := range pools {
		liquidity, err := sdk.ParseCoinsNormalized(p.Liquidity)
		if err != nil {
			log.Printf("Error parsing pool %d liquidity from DB: %v", p.PoolId, err)
			return nil, err
		}
		volume, err := sdk.ParseCoinsNormalized(p.Volume)
		if err != nil {
			log.Printf("Error parsing pool %d volume from DB: %v", p.PoolId, err)
			return nil, err
		}
		ret[i] = repository.Pool{
			Height:    p.Height,
			PoolId:    p.PoolId,
			Liquidity: liquidity,
			Volume:    volume,
		}
	}

	return ret, nil
}

func (r *Repository) TokenPricesRange(min, max time.Time, denom string) ([]repository.TokenPrice, error) {
	var prices []TokenPrice
	query := "last_updated >= ? AND last_updated <= ? AND name = ?"
	if denom == "" {
		query = "last_updated >= ? AND last_updated <= ?"
	}
	result := r.dbCon.Model(&TokenPrice{}).Find(&prices, query, min.UnixNano(), max.UnixNano(), denom)
	if result.Error != nil {
		log.Println("Error fetching Token Prices from DB:", result.Error)
		return nil, result.Error
	}

	ret := make([]repository.TokenPrice, len(prices))
	for i, p := range prices {
		ret[i] = repository.TokenPrice{
			LastUpdated: time.Unix(0, p.LastUpdated),
			Value:       p.Value,
			Name:        p.Name,
			Base:        p.Base,
		}
	}

	return ret, nil
}
