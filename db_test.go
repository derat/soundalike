// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewAudioDB(t *testing.T) {
	p := filepath.Join(t.TempDir(), "test.db")
	settings := defaultFpcalcSettings()
	if db, err := newAudioDB(p, settings); err != nil {
		t.Fatal("newAudioDB with new file failed: ", err)
	} else if err := db.close(); err != nil {
		t.Fatal("close failed: ", err)
	}

	if db, err := newAudioDB(p, settings); err != nil {
		t.Fatal("newAudioDB with existing file failed: ", err)
	} else if err := db.close(); err != nil {
		t.Fatal("close failed: ", err)
	}

	settings.length *= 2
	if db, err := newAudioDB(p, settings); err == nil {
		db.close()
		t.Fatal("newAudioDB with different settings unexpectedly succeeded")
	}
}

func TestAudioDB_Save_Get(t *testing.T) {
	// Save a fingerprint to the database.
	p := filepath.Join(t.TempDir(), "test.db")
	settings := defaultFpcalcSettings()
	db, err := newAudioDB(p, settings)
	if err != nil {
		t.Fatal("newAudioDB failed: ", err)
	}
	const file = "artist/album/01-title.mp3"
	fprint := []uint32{
		2835786340, 2835868260, 2836164325, 2903256545, 3976998131, 3976543474, 3980795026,
		4156954754, 4135987330, 4135991426, 3532003458, 3532019842, 3532061074, 4068982706,
		4069310099, 4052258449, 4039675520, 4030369408, 4029333056, 3761155584, 3761219872,
		3761219872, 3760630048, 3760626465, 3760639537, 3777418769, 3810980355, 3861312019,
		3861169907, 3865333155, 3865331875, 3873491121, 4007905457, 4003649712, 4003654736,
		4007846912, 4007911681, 3999524627, 3999428374, 2921470742, 2921536278, 2921405222,
		2926578470, 2867858726, 2861565230, 2860515614, 3128963150, 3128959069, 3145753692,
		2676026452, 2642712788, 2512685268, 2504820948, 3041773748, 2970273957, 2971326631,
	}
	if err := db.save(file, fprint); err != nil {
		db.close()
		t.Fatalf("save(%q, ...) failed: %v", file, err)
	}
	if err := db.close(); err != nil {
		t.Fatal("close failed: ", err)
	}

	// Reopen the database and read the fingerprint back.
	if db, err = newAudioDB(p, settings); err != nil {
		t.Fatal("newAudioDB failed: ", err)
	}
	defer db.close()
	got, err := db.get(file)
	if err != nil {
		t.Errorf("get(%q) failed: %v", file, err)
	} else if !reflect.DeepEqual(got, fprint) {
		t.Errorf("get(%q) returned wrong fingerprint", file)
	}

	// Update the fingerprint.
	fprint2 := fprint[:10]
	if err := db.save(file, fprint2); err != nil {
		t.Errorf("save(%q, ...) failed: %v", file, err)
	} else if got, err := db.get(file); err != nil {
		t.Errorf("get(%q) failed: %v", file, err)
	} else if !reflect.DeepEqual(got, fprint2) {
		t.Errorf("get(%q) returned wrong fingerprint", file)
	}

	// Check that nil is returned for missing fingerprints.
	const file2 = "some-other-song.mp3"
	if got, err := db.get(file2); err != nil {
		t.Errorf("get(%q) failed: %v", file2, err)
	} else if got != nil {
		t.Errorf("get(%q) = %v; want nil", file2, got)
	}
}
