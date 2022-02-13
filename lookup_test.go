// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"reflect"
	"sort"
	"testing"
)

func TestLookupTable(t *testing.T) {
	table := newLookupTable()
	for _, f := range []struct {
		id     fileID
		fprint []uint32
	}{
		{1, []uint32{0x44442222, 0x44441111, 0x33332222, 0x55553333}},
		{2, []uint32{0x44442222, 0x44442222, 0x44441111, 0x55553333}},
		{3, []uint32{0x33332222, 0x33331111, 0x33334444, 0x44442222}},
	} {
		table.add(f.id, f.fprint)
	}

	for _, tc := range []struct {
		fprint []uint32
		thresh int
		want   []fileID
	}{
		{[]uint32{0x44442222, 0x44441111, 0x33332222, 0x55553333}, 4, []fileID{1}},
		{[]uint32{0x44441111, 0x44448888, 0x33331111, 0x55551111}, 4, []fileID{1}},
		{[]uint32{0x44442222, 0x44441111, 0x33332222, 0x55553333}, 3, []fileID{1, 2}},
		{[]uint32{0x44442222, 0x44441111, 0x33332222, 0x55553333}, 2, []fileID{1, 2, 3}},
		{[]uint32{0x44442222, 0x44442222, 0x44442222, 0x44442222}, 4, []fileID{}},
		{[]uint32{0x44442222, 0x44442222, 0x44442222, 0x44442222}, 3, []fileID{2}},
		{[]uint32{0x99999999, 0x99999999, 0x99999999, 0x99999999}, 1, []fileID{}},
		{[]uint32{0x33333333, 0x33333333, 0x33333333, 0x33333333}, 4, []fileID{}},
		{[]uint32{0x33333333, 0x33333333, 0x33333333, 0x33333333}, 3, []fileID{3}},
		{[]uint32{0x33333333, 0x33333333, 0x33333333, 0x33333333}, 2, []fileID{3}},
		{[]uint32{0x33333333, 0x33333333, 0x33333333, 0x33333333}, 1, []fileID{1, 3}},
	} {
		got := table.find(tc.fprint, tc.thresh)
		sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("find(%v, %d) = %v; want %v", tc.fprint, tc.thresh, got, tc.want)
		}
	}
}
