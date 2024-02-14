package indexer

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/syntropynet/osmosis-publisher/pkg/repository"
	"github.com/syntropynet/osmosis-publisher/pkg/types"
)

type PoolMap struct {
	sync.Mutex
	pools map[uint64]map[uint64]repository.Pool
}

func (p *PoolMap) Set(pool repository.Pool) {
	p.Lock()
	defer p.Unlock()

	hMap, ok := p.pools[pool.Height]
	if !ok {
		hMap = make(map[uint64]repository.Pool)
		p.pools[pool.Height] = hMap
	}
	hMap[pool.PoolId] = pool
}

func (p *PoolMap) Has(height, id uint64) bool {
	p.Lock()
	defer p.Unlock()

	hMap, ok := p.pools[height]
	if !ok {
		return false
	}

	_, ok = hMap[id]

	return ok
}

func (p *PoolMap) Get(height, id uint64) (repository.Pool, bool) {
	p.Lock()
	defer p.Unlock()

	hMap, ok := p.pools[height]
	if !ok {
		return repository.Pool{}, false
	}

	pool, ok := hMap[id]
	return pool, ok
}

func (p *PoolMap) Prune(minHeight uint64) int {
	p.Lock()
	defer p.Unlock()
	counter := 0
	for h := range p.pools {
		if h < minHeight {
			delete(p.pools, h)
			counter++
		}
	}
	return counter
}

func setMaxValue(a *atomic.Uint64, v uint64) uint64 {
	for {
		oldValue := a.Load()
		if oldValue >= v {
			return oldValue
		}

		if a.CompareAndSwap(oldValue, v) {
			return oldValue
		}
	}
}

// preHeatPools will retrieve  pools for a range of heights from the database
func (d *Indexer) preHeatPools(blocks uint64) {
	height := d.currentBlockHeight.Load()
	pools, err := d.repo.PoolsRange(height-blocks, height, 0)
	if err != nil {
		d.logger.Error("Failed fetching pools for blocks", "from", height-blocks, "to", height, "error", err)
		return
	}

	var (
		last_height  uint64
		first_height uint64 = height + 10
	)
	for _, p := range pools {
		d.pools.Set(p)
		setMaxValue(&d.currentBlockHeight, p.Height)
		if p.Height > last_height {
			last_height = p.Height
		}
		if p.Height < first_height {
			first_height = p.Height
		}
	}

	d.logger.Info("SYNC: Pools loaded", "len(pools)", len(pools), "from", height-blocks, "to", height, "first_height", first_height, "last_height", last_height)
}

func (d *Indexer) poolsPrune(minHeight uint64) {
	d.pools.Prune(minHeight)

	d.repo.PrunePools(minHeight)
}

func (d *Indexer) PoolStatusesAt(height uint64, poolId ...uint64) ([]types.PoolStatus, uint64, error) {
	if height == 0 {
		height = d.currentBlockHeight.Load()
	}

	poolStatuses := make([]types.PoolStatus, len(poolId))
	errArr := make([]error, 0, len(poolId))
	for i, id := range poolId {
		ps, _, err := d.PoolStatusAt(height, id)
		if err != nil {
			errArr = append(errArr, err)
			continue
		}
		poolStatuses[i] = ps
	}

	return poolStatuses, height, errors.Join(errArr...)
}

func (d *Indexer) PoolStatusAt(height, poolId uint64) (types.PoolStatus, uint64, error) {
	if height == 0 {
		height = d.currentBlockHeight.Load()
	}

	poolStatus := types.PoolStatus{
		PoolId: poolId,
	}

	pool, err := d.getPool(height, poolId)
	if err != nil {
		d.logger.Error("SYNC: PoolStatusAt failed", "poolId", poolId, "height", height, "err", err)
		return poolStatus, height, err
	}

	poolStatus.TotalLiquidity = pool.Liquidity
	poolStatus.Volumes = []types.PoolStatusVolumeAt{
		{
			BlockHeight: int64(height),
			Volume:      pool.Volume,
		},
	}

	return poolStatus, height, nil
}

func (d *Indexer) getPool(height, poolId uint64) (repository.Pool, error) {
	if height == 0 {
		height = d.currentBlockHeight.Load()
	}
	pool, found := d.pools.Get(height, poolId)
	if found {
		return pool, nil
	}

	liquidity, err := d.rpc.PoolsTotalLiquidityAt(int64(height), poolId)
	if err != nil {
		return pool, err
	}
	volume, err := d.rpc.PoolsVolumeAt(int64(height), poolId)
	if err != nil {
		return pool, err
	}

	pool = repository.Pool{
		Height:    height,
		PoolId:    poolId,
		Liquidity: liquidity[0].Liquidity,
		Volume:    volume[0].Volume,
	}

	d.pools.Set(pool)
	err = d.repo.SavePool(pool)
	if err != nil {
		return pool, err
	}

	return pool, nil
}
