// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"reflect"
	"sort"
	"testing"
)

func TestCompareFingerprints(t *testing.T) {
	for _, tc := range []struct {
		a, b      []uint32
		minLength bool
		want      float64
	}{
		{
			[]uint32{0x0000ffe4},
			[]uint32{0xffff0f14},
			false, 8.0 / 32,
		},
		{
			[]uint32{0xfffffffe, 0x80000001},
			[]uint32{0x7fffffff, 0xf0000001},
			false, 59.0 / 64,
		},
		{
			[]uint32{0x00000000, 0x01010101, 0xffffffff, 0xcafebeef},
			[]uint32{0x01010101, 0xffffffff, 0xcafebeef, 0x00000000},
			false, 96.0 / 128,
		},
		{
			[]uint32{0x00000000, 0xffffffff, 0x01010101},
			[]uint32{0xffffffff, 0x01010101},
			false, 64.0 / 96,
		},
		{
			[]uint32{0x00000000, 0xffffffff, 0x01010101},
			[]uint32{0xffffffff, 0x01010101},
			true, 64.0 / 64,
		},
	} {
		if got := compareFingerprints(tc.a, tc.b, tc.minLength); got != tc.want {
			t.Errorf("compareFingerprints(%v, %v, %v) = %0.3f; want %0.3f",
				tc.a, tc.b, tc.minLength, got, tc.want)
		}
	}
}

func TestComponents(t *testing.T) {
	edges := make(map[fileID][]fileID)
	add := func(a, b fileID) {
		edges[a] = append(edges[a], b)
		edges[b] = append(edges[b], a)
	}
	add(1, 2)
	add(1, 3)
	add(2, 3)
	add(3, 4)
	add(5, 6)
	add(5, 7)

	got := components(edges)
	for i := range got {
		sort.Slice(got[i], func(a, b int) bool { return got[i][a] < got[i][b] })
	}
	sort.Slice(got, func(a, b int) bool { return got[a][0] < got[b][0] })
	if want := [][]fileID{{1, 2, 3, 4}, {5, 6, 7}}; !reflect.DeepEqual(got, want) {
		t.Errorf("components(...) = %v; want %v", got, want)
	}
}
