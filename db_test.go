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
	const (
		path = "artist/album/01-title.mp3"
		size = 2 * 1024 * 1024
		dur  = 103.4
	)
	fprint := []uint32{
		2835786340, 2835868260, 2836164325, 2903256545, 3976998131, 3976543474,
		3980795026, 4156954754, 4135987330, 4135991426, 3532003458, 3532019842,
	}
	id, err := db.save(&fileInfo{0, path, size, dur, fprint})
	if err != nil {
		db.close()
		t.Fatal("save failed: ", err)
	}
	if err := db.close(); err != nil {
		t.Fatal("close failed: ", err)
	}

	// Reopen the database and read the file info back.
	if db, err = newAudioDB(p, settings); err != nil {
		t.Fatal("newAudioDB failed: ", err)
	}
	defer db.close()

	want := fileInfo{id, path, size, dur, fprint}
	if got, err := db.get(0, path); err != nil {
		t.Errorf("get(0, %q) failed: %v", path, err)
	} else if got == nil {
		t.Errorf("get(0, %q) returned nil", path)
	} else if !reflect.DeepEqual(*got, want) {
		t.Errorf("get(0, %q) = %+v; want %+v", path, *got, want)
	}
	if got, err := db.get(id, ""); err != nil {
		t.Errorf(`get(%d, "") failed: %v`, id, err)
	} else if got == nil {
		t.Errorf(`get(%d, "") returned nil`, id)
	} else if !reflect.DeepEqual(*got, want) {
		t.Errorf(`get(%d, "") = %+v; want %+v`, id, *got, want)
	}

	// Check that nil is returned for missing fingerprints.
	const path2 = "some-other-song.mp3"
	if got, err := db.get(0, path2); err != nil {
		t.Errorf("get(0, %q) failed: %v", path2, err)
	} else if got != nil {
		t.Errorf("get(0, %q) = %+v; want 0 nil", path2, *got)
	}
}

func TestAudioDB_ExcludedPairs(t *testing.T) {
	p := filepath.Join(t.TempDir(), "test.db")
	settings := defaultFpcalcSettings()
	db, err := newAudioDB(p, settings)
	if err != nil {
		t.Fatal("newAudioDB failed: ", err)
	}

	const (
		a = "a.mp3"
		b = "b.mp3"
		c = "c.mp3"
	)

	if ok, err := db.isExcludedPair(a, b); err != nil {
		t.Fatalf("isExcludedPair(%q, %q) failed: %v", a, b, err)
	} else if ok {
		t.Fatalf("isExcludedPair(%q, %q) = %v; want %v", a, b, ok, false)
	}
	if err := db.saveExcludedPair(a, b); err != nil {
		t.Fatalf("saveExcludedPair(%q, %q) failed: %v", a, b, err)
	}
	if ok, err := db.isExcludedPair(a, b); err != nil {
		t.Fatalf("isExcludedPair(%q, %q) failed: %v", a, b, err)
	} else if !ok {
		t.Fatalf("isExcludedPair(%q, %q) = %v; want %v", a, b, ok, true)
	}
	if ok, err := db.isExcludedPair(b, a); err != nil {
		t.Fatalf("isExcludedPair(%q, %q) failed: %v", b, a, err)
	} else if !ok {
		t.Fatalf("isExcludedPair(%q, %q) = %v; want %v", b, a, ok, true)
	}
	if ok, err := db.isExcludedPair(a, c); err != nil {
		t.Fatalf("isExcludedPair(%q, %q) failed: %v", a, c, err)
	} else if ok {
		t.Fatalf("isExcludedPair(%q, %q) = %v; want %v", a, c, ok, false)
	}
}
