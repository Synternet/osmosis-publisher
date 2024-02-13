package flags

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type PoolIds struct {
	Value *[]uint64
}

func NewPoolIds(pools string) *PoolIds {
	poolsSlice := parse(pools)

	return &PoolIds{Value: &poolsSlice}
}

func (p *PoolIds) Set(pools string) error {
	if pools == "" {
		*p.Value = make([]uint64, 0)
	} else {
		*p.Value = append(*p.Value, parse(pools)...)
	}

	return nil
}

func (p *PoolIds) String() string {
	out := make([]string, len(*p.Value))
	for i, d := range *p.Value {
		out[i] = fmt.Sprintf("%d", d)
	}
	return "[" + strings.Join(out, ",") + "]"
}

func (p PoolIds) Type() string {
	return "uint64Slice"
}

func parse(pools string) []uint64 {
	cleanPools := splitAndTrimEmpty(pools, ",", " \t\r\n\b")

	out := make([]uint64, len(cleanPools))

	for i, cp := range cleanPools {
		var err error
		if out[i], err = strconv.ParseUint(cp, 10, 64); err != nil {
			log.Fatal(err)
		}
	}

	return out
}

// SplitAndTrimEmpty slices s into all subslices separated by sep and returns a
// slice of the string s with all leading and trailing Unicode code points
// contained in cutset removed. If sep is empty, SplitAndTrim splits after each
// UTF-8 sequence. First part is equivalent to strings.SplitN with a count of
// -1.  also filter out empty strings, only return non-empty strings.
func splitAndTrimEmpty(s, sep, cutset string) []string {
	if s == "" {
		return []string{}
	}

	spl := strings.Split(s, sep)
	nonEmptyStrings := make([]string, 0, len(spl))

	for i := 0; i < len(spl); i++ {
		element := strings.Trim(spl[i], cutset)
		if element != "" {
			nonEmptyStrings = append(nonEmptyStrings, element)
		}
	}

	return nonEmptyStrings
}
