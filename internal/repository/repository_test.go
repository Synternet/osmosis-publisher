package repository_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/syntropynet/osmosis-publisher/internal/repository"
	repotypes "github.com/syntropynet/osmosis-publisher/pkg/repository"
	sdk "github.com/cosmos/cosmos-sdk/types"
	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	_ "github.com/lib/pq"
)

func TestRepository_Latest(t *testing.T) {
	tests := []struct {
		name    string
		f       func(db *repository.Repository, t *testing.T) error
		wantErr bool
	}{
		{
			name: "token price",
			f: func(db *repository.Repository, t *testing.T) error {
				price, found := db.LatestTokenPrice("OSMO")
				if !found {
					return fmt.Errorf("Did not find")
				}
				if price.LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second*6)) != 0 {
					return fmt.Errorf("wrong token: %v", price)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "token price now",
			f: func(db *repository.Repository, t *testing.T) error {
				now := time.Now()
				db.SaveTokenPrice(repotypes.TokenPrice{
					LastUpdated: now,
					Value:       10050,
					Name:        "AMBER",
					Base:        "EUR",
				})

				price, found := db.LatestTokenPrice("AMBER")
				if !found {
					return fmt.Errorf("Did not find")
				}
				if price.LastUpdated.Compare(now) != 0 {
					return fmt.Errorf("wrong token: %v", price)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "token price out of order",
			f: func(db *repository.Repository, t *testing.T) error {
				now := time.Now()
				db.SaveTokenPrice(repotypes.TokenPrice{
					LastUpdated: now.Add(time.Second),
					Value:       10050,
					Name:        "AMBER",
					Base:        "EUR",
				})
				db.SaveTokenPrice(repotypes.TokenPrice{
					LastUpdated: now,
					Value:       123,
					Name:        "AMBER",
					Base:        "EUR",
				})

				price, found := db.LatestTokenPrice("AMBER")
				if !found {
					return fmt.Errorf("Did not find")
				}
				if price.LastUpdated.Compare(now.Add(time.Second)) != 0 {
					return fmt.Errorf("wrong token: %v", price)
				}
				if price.Value != 10050 {
					return fmt.Errorf("wrong token: %v", price)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "pool",
			f: func(db *repository.Repository, t *testing.T) error {
				pool, found := db.LatestPool(1)
				if !found {
					return fmt.Errorf("Did not find")
				}
				if pool.Height != 3 {
					return fmt.Errorf("wrong pool: %v", pool)
				}
				return nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := makeDB()
			addIBCDenoms(db)
			addTokenPrices(db)
			addPools(db)

			err := tt.f(db, t)
			if (tt.wantErr && err == nil) || (!tt.wantErr && err != nil) {
				t.Errorf("latest X test wantErr = %v, err %v", tt.wantErr, err)
			}
		})
	}
}

func TestRepository_IBCDenom(t *testing.T) {
	tests := []struct {
		name    string
		f       func(db *repository.Repository, t *testing.T) error
		wantErr bool
	}{
		{
			name: "get",
			f: func(db *repository.Repository, t *testing.T) error {
				ibc := IBCTypes.DenomTrace{Path: "transfer/channel-229", BaseDenom: "A"}
				denom, found := db.IBCDenom(ibc.IBCDenom())
				if !found {
					return fmt.Errorf("Did not find")
				}
				if denom.BaseDenom != "A" {
					return fmt.Errorf("wrong denom: %s", denom)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "all",
			f: func(db *repository.Repository, t *testing.T) error {
				denoms := db.IBCDenomAll()
				truth := []IBCTypes.DenomTrace{
					{
						Path:      "transfer/channel-229",
						BaseDenom: "A",
					},
					{
						Path:      "transfer/channel-229",
						BaseDenom: "B",
					},
					{
						Path:      "transfer/channel-229",
						BaseDenom: "C",
					},
				}
				if !reflect.DeepEqual(denoms, truth) {
					return fmt.Errorf("unexpected denoms: %v", denoms)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "404",
			f: func(db *repository.Repository, t *testing.T) error {
				ibc := IBCTypes.DenomTrace{Path: "transfer/channel-230", BaseDenom: "A"}
				denom, found := db.IBCDenom(ibc.IBCDenom())
				if found {
					return fmt.Errorf("found %s", denom)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "add same",
			f: func(db *repository.Repository, t *testing.T) error {
				ibc := IBCTypes.DenomTrace{Path: "transfer/channel-229", BaseDenom: "A"}
				err := db.SaveIBCDenom(ibc)
				if err != nil {
					return err
				}
				return nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := makeDB()
			addIBCDenoms(db)

			err := tt.f(db, t)
			if (tt.wantErr && err == nil) || (!tt.wantErr && err != nil) {
				t.Errorf("IBCDenom test wantErr = %v, err %v", tt.wantErr, err)
			}
		})
	}
}

func TestRepository_TokenPrice(t *testing.T) {
	tests := []struct {
		name    string
		f       func(db *repository.Repository, t *testing.T) error
		wantErr bool
	}{
		{
			name: "get",
			f: func(db *repository.Repository, t *testing.T) error {
				price, found := db.TokenPrice(time.Unix(TimestampBaseOsmo, 0), "OSMO")
				if !found {
					return fmt.Errorf("Did not find")
				}
				if price.Name != "OSMO" {
					return fmt.Errorf("wrong record: %v v.s. %v", price, time.Unix(TimestampBaseOsmo, 0))
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "nearest",
			f: func(db *repository.Repository, t *testing.T) error {
				prices, found := db.NearestTokenPrice(time.Unix(TimestampBaseOsmo, 0).Add(time.Second*5), "OSMO")
				if !found {
					return fmt.Errorf("Did not find")
				}
				if len(prices) != 2 {
					return fmt.Errorf("wrong number of records: %v", prices)
				}
				if prices[0].LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second*2)) != 0 || prices[1].LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second*6)) != 0 {
					return fmt.Errorf("wrong records: %v", prices)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "range",
			f: func(db *repository.Repository, t *testing.T) error {
				prices, err := db.TokenPricesRange(time.Unix(TimestampBaseOsmo, 0).Add(time.Second), time.Unix(TimestampBaseOsmo, 0).Add(time.Second*5), "OSMO")
				if err != nil {
					return fmt.Errorf("TokenPricesRange failed: %w", err)
				}
				if len(prices) != 2 {
					return fmt.Errorf("wrong number of records: %v", prices)
				}
				if prices[0].LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second)) != 0 || prices[1].LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second*2)) != 0 {
					return fmt.Errorf("wrong records: %v", prices)
				}
				if prices[0].Name != "OSMO" || prices[1].Name != "OSMO" {
					return fmt.Errorf("wrong records: %v", prices)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "range all",
			f: func(db *repository.Repository, t *testing.T) error {
				prices, err := db.TokenPricesRange(time.Unix(TimestampBaseOsmo, 0).Add(time.Second), time.Unix(TimestampBaseOsmo, 0).Add(time.Second*5), "")
				if err != nil {
					return fmt.Errorf("TokenPricesRange failed: %w", err)
				}
				if len(prices) != 4 {
					return fmt.Errorf("wrong number of records: %v", prices)
				}
				if prices[0].LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second)) != 0 || prices[1].LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second)) != 0 {
					return fmt.Errorf("wrong records: %v", prices)
				}
				if prices[2].LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second*2)) != 0 || prices[3].LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second*2)) != 0 {
					return fmt.Errorf("wrong records: %v", prices)
				}
				if prices[0].Name != "ATOM" || prices[1].Name != "OSMO" || prices[2].Name != "ATOM" || prices[3].Name != "OSMO" {
					return fmt.Errorf("wrong records: %v", prices)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "nearest last",
			f: func(db *repository.Repository, t *testing.T) error {
				prices, found := db.NearestTokenPrice(time.Unix(TimestampBaseOsmo, 0).Add(time.Minute), "OSMO")
				if !found {
					return fmt.Errorf("Did not find")
				}
				if len(prices) != 1 {
					return fmt.Errorf("wrong number of records: %v", prices)
				}
				if prices[0].LastUpdated.Compare(time.Unix(TimestampBaseOsmo, 0).Add(time.Second*6)) != 0 {
					return fmt.Errorf("wrong records: %v", prices)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "404 height",
			f: func(db *repository.Repository, t *testing.T) error {
				price, found := db.TokenPrice(time.Now(), "OSMO")
				if found {
					return fmt.Errorf("found %v", price)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "404 token",
			f: func(db *repository.Repository, t *testing.T) error {
				price, found := db.TokenPrice(time.Now(), "ATOM")
				if found {
					return fmt.Errorf("found %v", price)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "add same",
			f: func(db *repository.Repository, t *testing.T) error {
				now := time.Now()
				err := db.SaveTokenPrice(repotypes.TokenPrice{
					LastUpdated: now,
					Name:        "OSMO",
					Base:        "USD",
					Value:       100500,
				})
				if err != nil {
					return err
				}

				price, found := db.TokenPrice(now, "OSMO")
				if !found {
					return fmt.Errorf("not found")
				}
				if price.Value != 100500 {
					return fmt.Errorf("found %v instead", price)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "prune",
			f: func(db *repository.Repository, t *testing.T) error {
				numDeleted, err := db.PruneTokenPrices(time.Unix(TimestampBaseOsmo, 0).Add(time.Second * 5))
				if err != nil {
					return fmt.Errorf("PruneTokenPrices failed: %w", err)
				}
				if numDeleted != 6 {
					return fmt.Errorf("unexpected PruneTokenPrices deleted rows want=%d got %d", 6, numDeleted)
				}
				prices, err := db.TokenPricesRange(time.Unix(TimestampBaseOsmo, 0).Add(time.Second), time.Unix(TimestampBaseOsmo, 0).Add(time.Second*6), "")
				if err != nil {
					return fmt.Errorf("TokenPricesRange failed: %w", err)
				}
				if len(prices) != 2 {
					return fmt.Errorf("wrong number of records: %v", prices)
				}
				if prices[0].Name != "ATOM" && prices[1].Name != "OSMO" {
					return fmt.Errorf("unexpected prices: %v", prices)
				}

				return nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := makeDB()
			addTokenPrices(db)

			err := tt.f(db, t)
			if (tt.wantErr && err == nil) || (!tt.wantErr && err != nil) {
				t.Errorf("TokenPrice test wantErr = %v, err %v", tt.wantErr, err)
			}
		})
	}
}

func TestRepository_Pools(t *testing.T) {
	tests := []struct {
		name    string
		f       func(db *repository.Repository, t *testing.T) error
		wantErr bool
	}{
		{
			name: "range",
			f: func(db *repository.Repository, t *testing.T) error {
				pools, err := db.PoolsRange(2, 3, 1)
				if err != nil {
					return fmt.Errorf("PoolsRange failed: %w", err)
				}
				if len(pools) != 2 {
					return fmt.Errorf("wrong number of records: %v", pools)
				}
				if pools[0].Height != 2 || pools[1].Height != 3 {
					return fmt.Errorf("wrong records: %v", pools)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "range all",
			f: func(db *repository.Repository, t *testing.T) error {
				err := db.SavePool(
					repotypes.Pool{
						Height:    2,
						PoolId:    15,
						Liquidity: must(sdk.ParseCoinsNormalized("10stake")),
						Volume:    must(sdk.ParseCoinsNormalized("100500uosmo")),
					},
				)
				if err != nil {
					return err
				}
				pools, err := db.PoolsRange(0, 3, 0)
				if err != nil {
					return fmt.Errorf("PoolsRange failed: %w", err)
				}
				if len(pools) != 4 {
					return fmt.Errorf("wrong number of records: %v", pools)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "404 pool",
			f: func(db *repository.Repository, t *testing.T) error {
				pool, found := db.LatestPool(7)
				if found {
					return fmt.Errorf("found %v", pool)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "add same",
			f: func(db *repository.Repository, t *testing.T) error {
				err := db.SavePool(
					repotypes.Pool{
						Height:    3,
						PoolId:    1,
						Liquidity: must(sdk.ParseCoinsNormalized("101stake")),
						Volume:    must(sdk.ParseCoinsNormalized("15uosmo")),
					},
				)
				if err != nil {
					return err
				}

				pool, found := db.LatestPool(1)
				if !found {
					return fmt.Errorf("not found")
				}
				if pool.Height != 3 {
					return fmt.Errorf("found %v instead", pool)
				}
				if pool.Volume.String() != "15uosmo" {
					return fmt.Errorf("found %v instead", pool)
				}
				if pool.Liquidity.String() != "101stake" {
					return fmt.Errorf("found %v instead", pool)
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "prune",
			f: func(db *repository.Repository, t *testing.T) error {
				_, err := db.PrunePools(3)
				if err != nil {
					return fmt.Errorf("PrunePools failed: %w", err)
				}
				pools, err := db.PoolsRange(0, 10, 1)
				if err != nil {
					return fmt.Errorf("PoolsRange failed: %w", err)
				}
				if len(pools) != 1 {
					return fmt.Errorf("wrong number of records: %v", pools)
				}
				if pools[0].PoolId != 1 {
					return fmt.Errorf("unexpected pools: %v", pools)
				}

				return nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := makeDB()
			addPools(db)

			err := tt.f(db, t)
			if (tt.wantErr && err == nil) || (!tt.wantErr && err != nil) {
				t.Errorf("Pools test wantErr = %v, err %v", tt.wantErr, err)
			}
		})
	}
}
