package indexer

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/syntropynet/osmosis-publisher/pkg/repository"
	"github.com/syntropynet/osmosis-publisher/pkg/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestIndexer_fetchVolumeValuesPerBlockRange(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name             string
		min, max, poolId uint64
		init             func() *Indexer
		want             map[string][]priceAt
	}{
		{
			"normal",
			10, 12, 13,
			func() *Indexer {
				ret := &Indexer{
					verbose: true,
					prices: PriceMap{
						prices: make(map[string][]repository.TokenPrice),
					},
					pools: PoolMap{
						pools: make(map[uint64]map[uint64]repository.Pool),
					},
				}
				ret.blocksPerHour.Store(1000)
				ret.currentBlockHeight.Store(12)
				ret.currentBlockTime.Store(now.UnixNano())

				ret.prices.Set(repository.TokenPrice{
					LastUpdated: now.Add(-2 * time.Hour / 1000),
					Value:       4,
					Name:        "uatom",
					Base:        "USD",
				})
				ret.prices.Set(repository.TokenPrice{
					LastUpdated: now.Add(-time.Hour / 1000),
					Value:       3,
					Name:        "uatom",
					Base:        "USD",
				})
				ret.prices.Set(repository.TokenPrice{
					LastUpdated: now,
					Value:       2,
					Name:        "uatom",
					Base:        "USD",
				})

				ret.pools.Set(repository.Pool{
					Height: 10,
					PoolId: 13,
					Volume: sdk.NewCoins(sdk.NewCoin("uatom", sdk.NewInt(3))),
				})
				ret.pools.Set(repository.Pool{
					Height: 11,
					PoolId: 13,
					Volume: sdk.NewCoins(sdk.NewCoin("uatom", sdk.NewInt(4))),
				})
				ret.pools.Set(repository.Pool{
					Height: 12,
					PoolId: 13,
					Volume: sdk.NewCoins(sdk.NewCoin("uatom", sdk.NewInt(5))),
				})
				return ret
			},
			map[string][]priceAt{
				"uatom": []priceAt{
					priceAt{
						height: 10,
						volume: *big.NewInt(3),
						price:  4,
					},
					priceAt{
						height: 11,
						volume: *big.NewInt(4),
						price:  3,
					},
					priceAt{
						height: 12,
						volume: *big.NewInt(5),
						price:  2,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.init()

			if got := d.fetchVolumeValuesPerBlockRange(tt.min, tt.max, tt.poolId); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Indexer.fetchVolumeValuesPerBlockRange() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexer_BlockToTimestamp(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		init   func() *Indexer
		height uint64
		want   time.Time
	}{
		{
			"now",
			func() *Indexer {
				ret := &Indexer{}
				ret.blocksPerHour.Store(1)
				ret.currentBlockHeight.Store(100)
				ret.currentBlockTime.Store(now.UnixNano())
				return ret
			},
			100,
			now,
		},
		{
			"before",
			func() *Indexer {
				ret := &Indexer{}
				ret.blocksPerHour.Store(1)
				ret.currentBlockHeight.Store(100)
				ret.currentBlockTime.Store(now.UnixNano())
				return ret
			},
			10,
			now.Add(-time.Hour * 90),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.init()
			if got := d.BlockToTimestamp(tt.height); got.Compare(tt.want) != 0 {
				t.Errorf("Indexer.BlockToTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexer_calculateRelativeVolumeValueCumulatively(t *testing.T) {
	tests := []struct {
		name         string
		init         func() *Indexer
		volumePrices []priceAt
		volumes      []types.PoolStatusVolumeAt
		inspect      func(volumes []types.PoolStatusVolumeAt) error
		wantErr      bool
	}{
		{
			"normal",
			func() *Indexer { return &Indexer{} },
			[]priceAt{
				priceAt{
					height: 12,
					volume: *big.NewInt(1230),
					price:  2,
				},
				priceAt{
					height: 11,
					volume: *big.NewInt(500),
					price:  3,
				},
				priceAt{
					height: 10,
					volume: *big.NewInt(123),
					price:  4,
				},
			},
			[]types.PoolStatusVolumeAt{
				types.PoolStatusVolumeAt{
					BlockHeight: 12,
					Volume:      sdk.NewCoins(sdk.NewCoin("uosmo", sdk.NewInt(1230))),
					VolumeUSD:   []float64{1230 * 2},
				},
				types.PoolStatusVolumeAt{
					BlockHeight: 10,
					Volume:      sdk.NewCoins(sdk.NewCoin("uosmo", sdk.NewInt(123))),
					VolumeUSD:   []float64{123 * 4},
				},
			},
			func(volumes []types.PoolStatusVolumeAt) error {
				if len(volumes[0].RelativeVolumeUSD) != 1 {
					return fmt.Errorf("len(volumes[0].RelativeVolumeUSD) != 1")
				}
				if len(volumes[1].RelativeVolumeUSD) != 1 {
					return fmt.Errorf("len(volumes[1].RelativeVolumeUSD) != 1")
				}

				if volumes[0].RelativeVolumeUSD[0] != 0 {
					return fmt.Errorf("volumes[0].RelativeVolumeUSD == %v, wanted = 0", volumes[0].RelativeVolumeUSD[0])
				}
				want := (1230.0-500.0)*(2.0+3.0)/2.0 + (500.0-123.0)*(3.0+4.0)/2.0
				if math.Abs(volumes[1].RelativeVolumeUSD[0]-want) > 1e-12 {
					return fmt.Errorf("volumes[1].RelativeVolumeUSD == %v, wanted = %v", volumes[1].RelativeVolumeUSD[0], want)
				}
				return nil
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.init()
			if err := d.calculateRelativeVolumeValueCumulatively(tt.volumePrices, tt.volumes); (err != nil) != tt.wantErr {
				t.Errorf("Indexer.calculateRelativeVolumeValueCumulatively() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := tt.inspect(tt.volumes); err != nil {
				t.Errorf("Indexer.calculateRelativeVolumeValueCumulatively() inspect error = %v", err)
			}
		})
	}
}

func Test_calculateCoinPrice(t *testing.T) {
	tests := []struct {
		name  string
		coin  sdk.Coin
		price float64
		want  float64
	}{
		{
			"normal",
			sdk.NewCoin("denom", sdk.NewInt(123)),
			.00003,
			123 * 0.00003,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateCoinPrice(tt.coin, tt.price); math.Abs(got-tt.want) > 1e-12 {
				t.Errorf("calculateCoinPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexer_calculateVolumeValueAt(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		init   func() *Indexer
		height int64
		coins  sdk.Coins
		want   []float64
	}{
		{
			"normal",
			func() *Indexer {
				ret := &Indexer{
					prices: PriceMap{
						prices: make(map[string][]repository.TokenPrice),
					},
				}
				ret.blocksPerHour.Store(1)
				ret.currentBlockHeight.Store(2)
				ret.currentBlockTime.Store(now.UnixNano())
				ret.prices.Set(repository.TokenPrice{
					LastUpdated: now.Add(-time.Hour),
					Value:       2,
					Name:        "uatom",
					Base:        "USD",
				})
				ret.prices.Set(repository.TokenPrice{
					LastUpdated: now.Add(-time.Hour),
					Value:       10,
					Name:        "uosmo",
					Base:        "USD",
				})
				ret.prices.Set(repository.TokenPrice{
					LastUpdated: now,
					Value:       200,
					Name:        "uatom",
					Base:        "USD",
				})
				ret.prices.Set(repository.TokenPrice{
					LastUpdated: now,
					Value:       1000,
					Name:        "uosmo",
					Base:        "USD",
				})
				ret.prices.Set(repository.TokenPrice{
					LastUpdated: now.Add(time.Hour),
					Value:       32,
					Name:        "uatom",
					Base:        "USD",
				})
				ret.prices.Set(repository.TokenPrice{
					LastUpdated: now.Add(time.Hour),
					Value:       54,
					Name:        "uosmo",
					Base:        "USD",
				})
				return ret
			},
			1,
			sdk.NewCoins(sdk.NewCoin("uatom", sdk.NewInt(123)), sdk.NewCoin("uosmo", sdk.NewInt(3)), sdk.NewCoin("none", sdk.NewInt(345))),
			[]float64{0, 246, 30},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.init()
			if got := d.calculateVolumeValueAt(tt.height, tt.coins); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Indexer.calculateVolumeValueAt() = %v, want %v for %v", got, tt.want, tt.coins)
			}
		})
	}
}
