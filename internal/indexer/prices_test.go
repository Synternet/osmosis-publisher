package indexer

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/Synternet/osmosis-publisher/pkg/repository"
)

func TestPriceMap_Set(t *testing.T) {
	tests := []struct {
		name string
		init func() *PriceMap
		test func(pm *PriceMap) error
	}{
		{
			"in order",
			func() *PriceMap {
				return &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
			},
			func(pm *PriceMap) error {
				now := time.Now()
				pm.Set(repository.TokenPrice{LastUpdated: now, Value: 1, Name: "uosmo"})
				pm.Set(repository.TokenPrice{LastUpdated: now.Add(1), Value: 2, Name: "uosmo"})
				pm.Set(repository.TokenPrice{LastUpdated: now.Add(2), Value: 3, Name: "uosmo"})

				want := []repository.TokenPrice{
					repository.TokenPrice{
						LastUpdated: now,
						Value:       1,
						Name:        "uosmo",
					},
					repository.TokenPrice{
						LastUpdated: now.Add(1),
						Value:       2,
						Name:        "uosmo",
					},
					repository.TokenPrice{
						LastUpdated: now.Add(2),
						Value:       3,
						Name:        "uosmo",
					},
				}
				pl, ok := pm.prices["uosmo"]
				if !ok {
					return fmt.Errorf("uosmo not found")
				}
				if !reflect.DeepEqual(want, pl) {
					return fmt.Errorf("not equal want=%v got=%v", want, pl)
				}
				return nil
			},
		},
		{
			"reverse order",
			func() *PriceMap {
				return &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
			},
			func(pm *PriceMap) error {
				now := time.Now()
				pm.Set(repository.TokenPrice{LastUpdated: now.Add(2), Value: 3, Name: "uosmo"})
				pm.Set(repository.TokenPrice{LastUpdated: now.Add(1), Value: 2, Name: "uosmo"})
				pm.Set(repository.TokenPrice{LastUpdated: now, Value: 1, Name: "uosmo"})

				want := []repository.TokenPrice{
					repository.TokenPrice{
						LastUpdated: now,
						Value:       1,
						Name:        "uosmo",
					},
					repository.TokenPrice{
						LastUpdated: now.Add(1),
						Value:       2,
						Name:        "uosmo",
					},
					repository.TokenPrice{
						LastUpdated: now.Add(2),
						Value:       3,
						Name:        "uosmo",
					},
				}
				pl, ok := pm.prices["uosmo"]
				if !ok {
					return fmt.Errorf("uosmo not found")
				}
				if !reflect.DeepEqual(want, pl) {
					return fmt.Errorf("not equal want=%v got=%v", want, pl)
				}
				return nil
			},
		},
		{
			"overwrite",
			func() *PriceMap {
				return &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
			},
			func(pm *PriceMap) error {
				now := time.Now()
				pm.Set(repository.TokenPrice{LastUpdated: now.Add(1), Value: 3, Name: "uosmo"})
				pm.Set(repository.TokenPrice{LastUpdated: now.Add(1), Value: 2, Name: "uosmo"})

				want := []repository.TokenPrice{
					repository.TokenPrice{
						LastUpdated: now.Add(1),
						Value:       2,
						Name:        "uosmo",
					},
				}
				pl, ok := pm.prices["uosmo"]
				if !ok {
					return fmt.Errorf("uosmo not found")
				}
				if !reflect.DeepEqual(want, pl) {
					return fmt.Errorf("not equal want=%v got=%v", want, pl)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.init()
			if err := tt.test(p); err != nil {
				t.Errorf("PriceMap.Set() error = %v", err)
			}
		})
	}
}

func TestPriceMap_Nearest(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		init func() *PriceMap
		test func(pm *PriceMap) error
	}{
		{
			"exact",
			func() *PriceMap {
				ret := &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(1), Value: 3, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(2), Value: 2, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now, Value: 10, Name: "uosmo"})

				return ret
			},
			func(pm *PriceMap) error {
				want := []repository.TokenPrice{
					repository.TokenPrice{
						LastUpdated: now,
						Value:       10,
						Name:        "uosmo",
					},
				}

				got := pm.Nearest(now, "uosmo")
				if !reflect.DeepEqual(want, got) {
					return fmt.Errorf("not equal want=%v got=%v", want, got)
				}
				return nil
			},
		},
		{
			"outside before",
			func() *PriceMap {
				ret := &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(1), Value: 3, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(2), Value: 2, Name: "uosmo"})

				return ret
			},
			func(pm *PriceMap) error {
				want := []repository.TokenPrice{
					repository.TokenPrice{
						LastUpdated: now.Add(1),
						Value:       3,
						Name:        "uosmo",
					},
				}

				got := pm.Nearest(now, "uosmo")
				if !reflect.DeepEqual(want, got) {
					return fmt.Errorf("not equal want=%v got=%v", want, got)
				}
				return nil
			},
		},
		{
			"outside after",
			func() *PriceMap {
				ret := &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(-1), Value: 3, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(-2), Value: 2, Name: "uosmo"})

				return ret
			},
			func(pm *PriceMap) error {
				want := []repository.TokenPrice{
					repository.TokenPrice{
						LastUpdated: now.Add(-1),
						Value:       3,
						Name:        "uosmo",
					},
				}

				got := pm.Nearest(now, "uosmo")
				if !reflect.DeepEqual(want, got) {
					return fmt.Errorf("not equal want=%v got=%v", want, got)
				}
				return nil
			},
		},
		{
			"gap",
			func() *PriceMap {
				ret := &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(-1), Value: 3, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(2), Value: 2, Name: "uosmo"})

				return ret
			},
			func(pm *PriceMap) error {
				want := []repository.TokenPrice{
					repository.TokenPrice{
						LastUpdated: now.Add(-1),
						Value:       3,
						Name:        "uosmo",
					},
					repository.TokenPrice{
						LastUpdated: now.Add(2),
						Value:       2,
						Name:        "uosmo",
					},
				}

				got := pm.Nearest(now, "uosmo")
				if !reflect.DeepEqual(want, got) {
					return fmt.Errorf("not equal want=%v got=%v", want, got)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.init()
			if err := tt.test(p); err != nil {
				t.Errorf("PriceMap.Nearest() error = %v", err)
			}
		})
	}
}

func TestPriceMap_Estimate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		init      func() *PriceMap
		wantPrice float64
		wantSpan  time.Duration
	}{
		{
			"exact",
			func() *PriceMap {
				ret := &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(1), Value: 3, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(2), Value: 2, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now, Value: 10, Name: "uosmo"})

				return ret
			},
			10,
			0,
		},
		{
			"outside before",
			func() *PriceMap {
				ret := &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(1), Value: 3, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(2), Value: 2, Name: "uosmo"})

				return ret
			},
			3,
			-1,
		},
		{
			"outside after",
			func() *PriceMap {
				ret := &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(-1), Value: 3, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(-2), Value: 2, Name: "uosmo"})

				return ret
			},
			3,
			1,
		},
		{
			"gap",
			func() *PriceMap {
				ret := &PriceMap{
					prices: make(map[string][]repository.TokenPrice, 10),
				}
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(-1), Value: 3, Name: "uosmo"})
				ret.Set(repository.TokenPrice{LastUpdated: now.Add(2), Value: 2, Name: "uosmo"})

				return ret
			},
			3.0 + (2.0-3.0)/(2.0+1.0),
			3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.init()
			price, span := p.Estimate(now, "uosmo")
			if math.Abs(price-tt.wantPrice) > 1e-6 {
				t.Errorf("PriceMap.Estimate() price want=%v got %v", tt.wantPrice, price)
			}
			if span != tt.wantSpan {
				t.Errorf("PriceMap.Estimate() span want=%v got %v", tt.wantSpan, span)
			}
		})
	}
}
