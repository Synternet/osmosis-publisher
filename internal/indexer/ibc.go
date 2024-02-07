package indexer

import (
	"log"

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
	log.Printf("SYNC: IBC Denoms loaded: %v\n", len(traces))

	return true
}

func (d *Indexer) preHeatDenomTraceCache() {
	if d.loadDenomTraceCache() {
		return
	}

	res, err := d.rpc.DenomTraces()
	if err != nil {
		d.errCounter.Add(1)
		log.Printf("Failed to fetch denom traces: %v\n", err)
		return
	}

	for _, trace := range res {
		d.ibcTraceCache[trace.IBCDenom()] = trace
	}

	log.Printf("SYNC: IBC Denoms fetched: %v\n", len(res))
}
