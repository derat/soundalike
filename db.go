// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var dbByteOrder = binary.LittleEndian

// audioDB holds previously-computed audio fingerprints.
type audioDB struct{ db *sql.DB }

// newAudioDB opens or creates a audioDB at path with the supplied settings.
// An error is returned if an existing database was created with different settings.
func newAudioDB(path string, settings *fpcalcSettings) (*audioDB, error) {
	if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS Settings (Desc STRING PRIMARY KEY NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS Files (Path STRING PRIMARY KEY NOT NULL, Fingerprint BLOB)`,
	} {
		if _, err = db.Exec(q); err != nil {
			return nil, err
		}
	}

	// Check that the database wasn't created with different settings from what we're using now.
	var dbSettings string
	if err := db.QueryRow(`SELECT Desc FROM Settings`).Scan(&dbSettings); err == nil {
		if s := settings.String(); dbSettings != s {
			return nil, fmt.Errorf("database settings (%v) don't match current settings (%v)", dbSettings, s)
		}
	} else if err == sql.ErrNoRows {
		if _, err := db.Exec(`INSERT INTO Settings (Desc) VALUES(?)`, settings.String()); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	adb := &audioDB{db}
	db = nil // disarm Close() call
	return adb, nil
}

func (adb *audioDB) close() error { return adb.db.Close() }

// fileID uniquely identifies a file in audioDB.
type fileID int32

// get returns the ID and saved fingerprint corresponding to the file at path.
// If the file is not present in the database, 0 and a nil slice are returned.
func (adb *audioDB) get(path string) (id fileID, fprint []uint32, err error) {
	// ROWID is automatically assigned by SQLite: https://www.sqlite.org/autoinc.html
	row := adb.db.QueryRow(`SELECT ROWID, Fingerprint FROM Files WHERE Path = ?`, path)
	var b []byte
	if err := row.Scan(&id, &b); err == sql.ErrNoRows {
		return 0, nil, nil
	} else if err != nil {
		return 0, nil, err
	}

	if len(b)%4 != 0 {
		return 0, nil, fmt.Errorf("invalid fingerprint size %v", len(b))
	}
	for i := 0; i < len(b); i += 4 {
		fprint = append(fprint, dbByteOrder.Uint32(b[i:i+4]))
	}
	return id, fprint, nil
}

// save saves the supplied fingerprint for the file at path.
func (adb *audioDB) save(path string, fprint []uint32) (id fileID, err error) {
	var b bytes.Buffer
	if err := binary.Write(&b, dbByteOrder, fprint); err != nil {
		return 0, err
	}
	res, err := adb.db.Exec(`INSERT INTO Files (Path, Fingerprint) VALUES(?, ?)`, path, b.Bytes())
	if err != nil {
		return 0, err
	}
	id64, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	// This is a hack to save space. ROWID is really an int64, but int32 seems like
	// more than enough here since IDs are apparently assigned in increasing order.
	if id64 <= 0 || id64 > math.MaxInt32 {
		return 0, fmt.Errorf("invalid id %v", id64)
	}
	return fileID(id64), nil
}

// path returns the path of the file with the specified ID.
func (adb *audioDB) path(id fileID) (path string, err error) {
	row := adb.db.QueryRow(`SELECT Path FROM Files WHERE ROWID = ?`, int64(id))
	err = row.Scan(&path)
	return path, err
}
