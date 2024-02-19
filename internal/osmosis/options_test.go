package osmosis

import (
	"reflect"
	"testing"

	"github.com/syntropynet/data-layer-sdk/pkg/options"
)

func TestWithPoolIds(t *testing.T) {
	tests := []struct {
		name string
		ids  []uint64
		want []uint64
	}{
		{
			"duplicates",
			[]uint64{3, 1, 2, 2},
			[]uint64{1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opt options.Options

			err := opt.Parse(WithPoolIds(tt.ids))
			if err != nil {
				t.Errorf("opt.Parse failed: %v", err)
			}

			got := options.Param(opt, PoolIdsParam, []uint64{})

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithPoolIds() = %v, want %v", got, tt.want)
			}
		})
	}
}
