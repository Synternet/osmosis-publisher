package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type PoolIds struct {
	value *[]uint64
}

func NewPoolIds(pools string) *PoolIds {
	poolsSlice := parse(pools)

	return &PoolIds{value: &poolsSlice}
}

func (p *PoolIds) Set(pools string) error {
	*p.value = parse(pools)
	//*p.value = append(*p.value, out...)

	return nil
}

func (p *PoolIds) String() string {
	out := make([]string, len(*p.value))
	for i, d := range *p.value {
		out[i] = fmt.Sprintf("%d", d)
	}
	return "[" + strings.Join(out, ",") + "]"
}

func (p PoolIds) Type() string {
	return "uint64Slice"
}

func parse(pools string) []uint64 {
	cleanPools := SplitAndTrimEmpty(pools, ",", " \t\r\n\b")

	out := make([]uint64, len(cleanPools))

	for i, cp := range cleanPools {
		var err error
		if out[i], err = strconv.ParseUint(cp, 10, 64); err != nil {
			log.Fatal(err)
		}
	}

	return out
}
