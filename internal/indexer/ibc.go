package indexer

import (
	ibctypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

func (d *Indexer) DenomTrace(ibc string) (ibctypes.DenomTrace, error) {
	// Check if the denomStr is in the cache
	if trace, found := d.ibcTraceCache[ibc]; found {
		// If found, return the trace
		return trace, nil
	}

	// If not found in the cache, call GetDenomTrace
	trace, err := d.queryDenomTrace(ibc)
	if err != nil {
		return ibctypes.DenomTrace{}, err
	}

	// Update the cache with the new denom trace
	d.ibcTraceCache[ibc] = trace
	d.repo.SaveIBCDenom(trace)

	return trace, nil
}

func (d *Indexer) queryDenomTrace(denomStr string) (ibctypes.DenomTrace, error) {
	d.ibcMisses.Add(1)
	res, err := d.rpc.DenomTrace(denomStr)
	if err != nil {
		d.errCounter.Add(1)
		return ibctypes.DenomTrace{}, err
	}

	return res, nil
}

func (d *Indexer) loadDenomTraceCache() bool {
	traces := d.repo.IBCDenomAll()
	if len(traces) == 0 {
		return false
	}

	for _, trace := range traces {
		d.ibcTraceCache[trace.IBCDenom()] = trace
	}
	d.logger.Info("SYNC: IBC Denoms loaded", "len(traces)", len(traces))

	return true
}

func (d *Indexer) preHeatDenomTraceCache() {
	if d.loadDenomTraceCache() {
		return
	}

	traces, err := d.rpc.DenomTraces()
	if err != nil {
		d.errCounter.Add(1)
		d.logger.Warn("SYNC: Failed to fetch denom traces", "err", err)
		return
	}

	for _, trace := range traces {
		d.ibcTraceCache[trace.IBCDenom()] = trace
	}

	d.logger.Info("SYNC: IBC Denoms fetched", "len(traces)", len(traces))
}
