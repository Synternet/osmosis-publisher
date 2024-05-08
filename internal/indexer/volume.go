package indexer

import (
	"errors"
	"math/big"
	"slices"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/synternet/osmosis-publisher/pkg/types"
	"golang.org/x/exp/maps"
)

const BaseVolumeDenom = "uosmo"

func (d *Indexer) CalculateVolumes(pools []types.PoolStatus) error {
	errArr := make([]error, 0, 3)
	for i := range pools {
		if err := d.calculatePoolVolumes(&pools[i]); err != nil {
			errArr = append(errArr, err)
		}
	}
	return errors.Join(errArr...)
}

// calculatePoolVolumes calculate pool value in USD based on pool total volume
// and estimated price at the point in time block where the pool was located was generated
func (d *Indexer) calculatePoolVolumes(pool *types.PoolStatus) error {
	errArr := make([]error, 0, 3)
	if len(pool.Volumes) == 0 {
		return nil
	}

	for i, v := range pool.Volumes {
		pool.Volumes[i].VolumeUSD = d.calculateVolumeValueAt(v.BlockHeight, v.Volume)
	}

	if err := d.calculateRelativeVolumeValue(pool.PoolId, pool.Volumes); err != nil {
		errArr = append(errArr, err)
	}

	return errors.Join(errArr...)
}

// calculateVolumeValueAt talculates coin value at specified block height using estimated price
// that was recorded around the same time the block was generated at.
func (d *Indexer) calculateVolumeValueAt(height int64, coins sdk.Coins) []float64 {
	timestamp := d.BlockToTimestamp(uint64(height))

	abs := func(d time.Duration) time.Duration {
		if d < 0 {
			return -d
		}
		return d
	}

	values := make([]float64, len(coins))
	for i, coin := range coins {
		value, durationError := d.prices.Estimate(timestamp, coin.Denom)
		if abs(durationError) > time.Hour*24 {
			d.logger.Debug("VOLUME: duration error too large", "denom", coin.Denom, "timestamp", timestamp, "duration", durationError)
			continue
		}

		values[i] = calculateCoinPrice(coin, value)
	}
	return values
}

func calculateCoinPrice(coin sdk.Coin, price float64) float64 {
	return calculatePrice(coin.Amount.BigInt(), price)
}

func calculatePrice(volume *big.Int, price float64) float64 {
	var c big.Float
	c.SetInt(volume)
	bigValue := c.Mul(&c, big.NewFloat(price))
	valueFloat, _ := bigValue.Float64()
	return valueFloat
}

// calculateRelativeVolumeValue will calculate relative pool volume value in USD.
// The pool volume value is calculated relative to the volume of the latest block height that is
// stored inside volumes array.
func (d *Indexer) calculateRelativeVolumeValue(poolId uint64, volumes []types.PoolStatusVolumeAt) error {
	var (
		min uint64 = d.currentBlockHeight.Load() + 10
		max uint64
	)
	for _, v := range volumes {
		if uint64(v.BlockHeight) > max {
			max = uint64(v.BlockHeight)
		}
		if uint64(v.BlockHeight) < min {
			min = uint64(v.BlockHeight)
		}
	}

	if max < min {
		return nil
	}

	vpm := d.fetchVolumeValuesPerBlockRange(min, max, poolId)

	// We assume that pool volumes are only in uosmo. However, technically, it can be a list of coins.
	uosmoVP, found := vpm[BaseVolumeDenom]
	if !found || len(uosmoVP) == 0 {
		d.logger.Warn("VOLUME: No prices were found", "denom", BaseVolumeDenom, "keys", maps.Keys(vpm))
		return nil
	}

	// Sort heights in descending order
	slices.SortFunc[[]priceAt](uosmoVP, func(a, b priceAt) int {
		return int(b.height - a.height)
	})
	// Should be sorted already, but make sure them sorted in descending order anyway.
	slices.SortFunc[[]types.PoolStatusVolumeAt](volumes, func(a, b types.PoolStatusVolumeAt) int {
		return int(b.BlockHeight - a.BlockHeight)
	})

	err := d.calculateRelativeVolumeValueCumulatively(uosmoVP, volumes)
	if err != nil {
		return err
	}

	return nil
}

// calculateRelativeVolumeValueCumulatively will accumulate pool price difference between two adjacent
// records in sorted volumePrices array. volumePrices and volumes must be sorted in descending order by block height.
func (d *Indexer) calculateRelativeVolumeValueCumulatively(volumePrices []priceAt, volumes []types.PoolStatusVolumeAt) error {
	prevPrice := volumePrices[0].price
	priceSum := float64(0)
	volumeSum := big.NewInt(0)

	prevVolume := volumePrices[0].volume

	rangeIndex := 0
	pricesNum := len(volumePrices)
	volumesNum := len(volumes)
	idx := 0

	// We accumulate differences of adjacent total volume prices in USD. When we reach volume block height
	// of a snapshot indicated in volumes array, we update it inside volumes array. We also log the differences of accumulated volumes and
	// expected in the mentioned array.
	// priceSum is a monotonic function of volume price in USD difference versus block height:
	//   priceSum[i] = price[i]*(volume[i] - volume[i-1])
	// where i is block height. We implicitly ignore block heights that do not exist in the respective arrays.
	for rangeIndex < volumesNum && idx < pricesNum {
		var deltaVolume = big.NewInt(0)
		deltaVolume = deltaVolume.Sub(&prevVolume, &volumePrices[idx].volume)
		avgPrice := (prevPrice + volumePrices[idx].price) / 2.0
		priceSum += calculatePrice(deltaVolume, avgPrice)

		if volumePrices[idx].height <= uint64(volumes[rangeIndex].BlockHeight) {
			volumes[rangeIndex].RelativeVolumeUSD = []float64{priceSum}
			rangeIndex++
		}

		volumeSum = volumeSum.Add(volumeSum, deltaVolume)
		prevVolume = volumePrices[idx].volume
		prevPrice = volumePrices[idx].price
		idx++
	}

	if rangeIndex < volumesNum {
		volumes[rangeIndex].RelativeVolumeUSD = []float64{priceSum}
	}

	return nil
}

type priceAt struct {
	height uint64
	volume big.Int
	price  float64
}

func (d *Indexer) fetchVolumeValuesPerBlockRange(min, max, poolId uint64) map[string][]priceAt {
	d.logger.Debug("VOLUME: fetchVolumeValuesPerBlockRange", "poolId", poolId, "min", min, "max", max, "range", max-min)
	vm := make(map[string][]priceAt)
	for blockHeight := min; blockHeight <= max; blockHeight++ {
		poolState, found := d.pools.Get(blockHeight, poolId)
		if !found {
			continue
		}
		blockTime := d.BlockToTimestamp(uint64(blockHeight))

		for _, coin := range poolState.Volume {
			vml, found := vm[coin.Denom]
			if !found {
				vml = make([]priceAt, 0, max-min)
			}
			price, durationError := d.prices.Estimate(blockTime, coin.Denom)
			if durationError > time.Hour*24 {
				d.logger.Debug("VOLUME: duration error too large", "denom", coin.Denom, "blockTime", blockTime, "duration", durationError)
				continue
			}

			vm[coin.Denom] = append(
				vml,
				priceAt{
					height: blockHeight,
					price:  price,
					volume: *coin.Amount.BigInt(),
				},
			)
		}
	}

	return vm
}

// Calculates block timestamp based on latest block height and latest block timestamp
// FIXME: This is approximation. Better way would be to record each block timestamp and find that block's timestamp and return it instead.
// If no block exists with that height, find a closest block and interpolate/extrapolate from that.
func (d *Indexer) BlockToTimestamp(height uint64) time.Time {
	bph := d.blocksPerHour.Load()
	if bph == 0 {
		// Should not happen
		bph = DefaultBlocksPerHour
		d.logger.Warn("VOLUME: BlockToTimestamp Blocks Per Hour = 0!")
	}

	current := d.currentBlockHeight.Load()
	now := time.Unix(0, d.currentBlockTime.Load())
	if current < height {
		delta := height - current
		return now.Add((time.Duration(delta) * time.Hour) / time.Duration(bph))
	}

	delta := current - height
	return now.Add(-(time.Duration(delta) * time.Hour) / time.Duration(bph))
}
