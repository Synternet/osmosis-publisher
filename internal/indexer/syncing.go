package indexer

import (
	"time"
)

const (
	pruneDatabaseDuration = time.Hour * 24 * 7
	pruneDatabaseHeights  = DefaultBlocksPerHour * 24 * 7
)

func (d *Indexer) handleSyncing(blocks uint64) error {
	d.group.Go(func() error {
		return d.monitorHeights(blocks)
	})

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Info("indexer.handleSyncing: c.Context Done")
			return nil
		case height := <-d.syncHeights:
			d.logger.Info("SYNC", "height", height, "current_height", d.currentBlockHeight.Load(), "queue_heights", len(d.syncHeights))
			err := d.syncHeight(height)
			if err != nil {
				d.logger.Error("SYNC: failed syncing", "height", height, "err", err)
			}
		}
	}
}

// monitorHeights periodically checks for missing heights and queue them for syncing.
// After missing height check data is pruned from cache and possibly database.
func (d *Indexer) monitorHeights(blocks uint64) error {
	ticker := time.NewTicker(time.Minute)
	for {
		err := d.queueMissingHeights(blocks)
		if err != nil {
			return err
		}
		d.poolsPrune(d.currentBlockHeight.Load() - (blocks*3)/2)
		d.pricesPrune(d.currentBlockHeight.Load() - (blocks*3)/2)

		select {
		case <-d.ctx.Done():
			d.logger.Info("indexer.monitorHeights: c.Context Done")
			return nil
		case <-ticker.C:
		}
	}
}

// queueMissingHeights will observe memory cache for missing blocks and queue them for
// syncing.
func (d *Indexer) queueMissingHeights(blocks uint64) error {
	if len(d.syncHeights) > 0 {
		return nil
	}
	heightEnd := d.currentBlockHeight.Load()
	heightStart := heightEnd - blocks

	for i := heightStart; i < heightEnd; i++ {
		// NOTE: Here we assume that if one pool is missing from the height, then all pools are missing most likely.
		// However, this is not a problem since we still look up the cache before doing any fetching.
		for _, id := range d.poolIdsToMonitor {
			if d.pools.Has(i, id) {
				continue
			}

			// TODO: Add backoff for failed fetches

			select {
			case <-d.ctx.Done():
				d.logger.Info("indexer.queueMissingHeights: c.Context Done")
				return d.ctx.Err()
			case d.syncHeights <- i:
			}
			break
		}
	}

	return nil
}

// syncHeight retrieves relevant data from blockchain for specific block height
func (d *Indexer) syncHeight(height uint64) error {
	// Fetch missing data by looking into the cache
	_, _, err := d.PoolStatusesAt(height, d.poolIdsToMonitor...)
	if err != nil {
		// TODO: Add backoff for failed fetches
		return err
	}

	return nil
}
