package indexer

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/syntropynet/osmosis-publisher/pkg/repository"
)

var tokenMapping = map[string]float64{
	"OSMO": 1e-6,
	"ATOM": 1e-6,
	// TODO
}

type PriceMap struct {
	sync.Mutex
	// Mapping by token name
	// the array of prices should be sorted by lastUpdated
	prices map[string][]repository.TokenPrice
}

func compareFunction(a, b repository.TokenPrice) int {
	return a.LastUpdated.Compare(b.LastUpdated)
}

// Set will add a token price if such does not exist (must not match LastUpdated).
// Will return true if it results in unsorted array.
func (p *PriceMap) Set(price repository.TokenPrice) bool {
	p.Lock()
	defer p.Unlock()

	arr, exists := p.prices[price.Name]
	if !exists {
		// Preallocate
		arr = make([]repository.TokenPrice, 0, DefaultBlocksPerHour*12)
	}

	index, found := slices.BinarySearchFunc[[]repository.TokenPrice, repository.TokenPrice](arr, price, compareFunction)
	if found {
		arr[index] = price
		return false
	}

	if index == len(arr) {
		arr = append(arr, price)
	} else {
		arr = slices.Insert(arr, index, price)
	}
	p.prices[price.Name] = arr
	return false
}

func (p *PriceMap) Nearest(timestamp time.Time, name string) []repository.TokenPrice {
	p.Lock()
	defer p.Unlock()

	arr, exists := p.prices[name]
	if !exists {
		return nil
	}

	if len(arr) == 0 {
		return nil
	}

	index, found := slices.BinarySearchFunc[[]repository.TokenPrice, repository.TokenPrice](arr, repository.TokenPrice{LastUpdated: timestamp, Name: name}, compareFunction)

	if index == 0 || found {
		return []repository.TokenPrice{arr[index]}
	}

	N := len(arr)
	if index >= N {
		return []repository.TokenPrice{arr[N-1]}
	}

	return []repository.TokenPrice{
		arr[index-1],
		arr[index],
	}
}

// Estimate will extrapolate or interpolate(depending on cache state and lastUpdated param) the price.
//   - If the price is outside of cache dates - final price will be the same as closest price available.
//   - If the price is inside cache dates - final price will be the average of two prices(unless exact match is found)
//     Estimation time error will be returned.
func (p *PriceMap) Estimate(lastUpdated time.Time, denom string) (float64, time.Duration) {
	prices := p.Nearest(lastUpdated, denom)
	switch len(prices) {
	case 0:
		return 0, time.Hour * 48
	case 1:
		return prices[0].Value, lastUpdated.Sub(prices[0].LastUpdated)
	default:
		d := lastUpdated.Sub(prices[0].LastUpdated)
		deltaT := prices[1].LastUpdated.Sub(prices[0].LastUpdated)
		if deltaT == 0 {
			return prices[0].Value, deltaT
		}
		deltaP := prices[1].Value - prices[0].Value
		return prices[0].Value + d.Seconds()*deltaP/deltaT.Seconds(), deltaT
	}
}

func (p *PriceMap) Prune(minLastUpdated time.Time) int {
	p.Lock()
	defer p.Unlock()
	counter := 0

	return counter
}

func (p *PriceMap) Sort() {
	for tokenName := range p.prices {
		p.SortToken(tokenName)
	}
}

func (p *PriceMap) SortToken(denom string) {
	arr, found := p.prices[denom]
	if !found {
		return
	}
	slices.SortFunc[[]repository.TokenPrice, repository.TokenPrice](
		arr,
		compareFunction,
	)
}

func (d *Indexer) preHeatPrices(blocks uint64) {
	min, max := time.Now().Add(-(time.Hour*time.Duration(blocks))/time.Duration(d.blocksPerHour.Load())), time.Now()
	prices, err := d.repo.TokenPricesRange(min, max, "")
	if err != nil {
		log.Printf("Failed fetching prices from %v till %v: %v", min, max, err)
		return
	}

	first_lastUpdated := max
	last_lastUpdated := min
	for _, p := range prices {
		if first_lastUpdated.Compare(p.LastUpdated) < 0 {
			first_lastUpdated = p.LastUpdated
		}
		if last_lastUpdated.Compare(p.LastUpdated) > 0 {
			last_lastUpdated = p.LastUpdated
		}
		d.prices.Set(p)
	}

	log.Printf("SYNC: Prices loaded: %d for min_lastUpdated=%v and max_LastUpdated=%v; first_lastUpdated=%v last_LastUpdated=%v\n", len(prices), min, max, first_lastUpdated, last_lastUpdated)

	d.prices.Sort()
}

func (d *Indexer) pricesPrune(minHeight uint64) {
	current := d.currentBlockHeight.Load()
	delta := int64(current - minHeight)
	minLastUpdated := time.Unix(0, d.lastBlockTimestamp.Load()-(int64(time.Hour)*delta)/d.blocksPerHour.Load())
	d.prices.Prune(minLastUpdated)

	d.repo.PruneTokenPrices(minLastUpdated)
}

func convertToMicroToken(token string, value float64) (string, float64, bool) {
	if factor, found := tokenMapping[token]; found {
		uToken := fmt.Sprintf("u%s", strings.ToLower(token))
		uValue := value * factor
		return uToken, uValue, true
	}
	return token, value, false
}

func (d *Indexer) SetLatestPrice(token, base string, value float64, lastUpdated time.Time) error {
	if uToken, uValue, converted := convertToMicroToken(token, value); converted {
		if d.verbose {
			log.Printf("PRICE: Formatted %s=%f to %s=%f", token, value, uToken, uValue)
		}
		token = uToken
		value = uValue
	}

	tokenPrice := repository.TokenPrice{
		LastUpdated: lastUpdated,
		Value:       value,
		Name:        token,
		Base:        base,
	}
	needSort := d.prices.Set(tokenPrice)
	err := d.repo.SaveTokenPrice(tokenPrice)
	if err != nil {
		log.Printf("Failed saving tokenPrice to DB: %v", err)
	}

	// More efficient could be Sorted Insert, but we do it once in say 5 seconds and we only practically store 48h worth of records for a token.
	// The array is always sorted except for some rare occasions(e.g. resync)
	// This translates to roughly 3600 * 48 / 5.5 ~= 31418 records per token.
	// Currently we only store one token.
	if needSort {
		d.prices.SortToken(token)
	}

	return nil
}
