// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

// lookupTable is used to quickly find approximate matches for a given fingerprint.
// 32-bit fingerprint values are truncated to 16 bits to conserve space.
type lookupTable struct {
	m map[uint16]map[fileID]int16 // truncated fingerprint value -> file -> count
}

func newLookupTable() *lookupTable { return &lookupTable{make(map[uint16]map[fileID]int16)} }

// add adds the supplied file to the table.
func (t *lookupTable) add(id fileID, fprint []uint32) {
	for _, v := range fprint {
		key := uint16(v >> 16)
		counts := t.m[key]
		if counts == nil {
			counts = make(map[fileID]int16)
			t.m[key] = counts
		}
		counts[id]++
	}
}

// find returns files that share at least thresh truncated values with fprint.
func (t *lookupTable) find(fprint []uint32, thresh int) []fileID {
	// For each file, maintain a map from truncated fingerprint value to the
	// number of hits we've had so far. This makes sure that we don't overcount
	// the number of hits: if fprint contains two copies of value 4 but 4 only
	// appears once in a given file, we don't want to double-count it.
	hits := make(map[fileID]map[uint16]int16)

	for _, v := range fprint {
		key := uint16(v >> 16)
		for id, cnt := range t.m[key] {
			if seen := hits[id][key]; seen < cnt {
				m := hits[id]
				if m == nil {
					m = make(map[uint16]int16)
					hits[id] = m
				}
				m[key]++
			}
		}
	}

	// Sum the hits for each file and keep the ones that reached the threshold.
	ids := make([]fileID, 0, len(hits))
	for id, m := range hits {
		var cnt int
		for _, v := range m {
			cnt += int(v)
		}
		if cnt >= thresh {
			ids = append(ids, id)
		}
	}
	return ids
}
