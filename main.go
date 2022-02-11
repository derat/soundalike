// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage %v: [flag]... <DIR>\n"+
			"Finds duplicate audio files in a directory tree.\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	fps := defaultFpcalcSettings()
	opts := scanOptions{bits: 20}
	flag.IntVar(&fps.algorithm, "algorithm", fps.algorithm, `Fingerprint algorithm (fpcalc -algorithm flag)`)
	flag.IntVar(&opts.bits, "bits", opts.bits, "Fingerprint bits to use (max is 32)")
	flag.Float64Var(&fps.chunk, "chunk", fps.chunk, `Audio chunk duration (fpcalc -chunk flag)`)
	dbPath := flag.String("db", "", `SQLite database file for storing fingerprints (empty for temp file)`)
	// TODO: I'm just guessing what should be included here. See
	// https://en.wikipedia.org/wiki/Audio_file_format#List_of_formats and
	// https://en.wikipedia.org/wiki/FFmpeg#Supported_codecs_and_formats.
	fileRegexp := flag.String("file-regexp", `\.(aiff|flac|m4a|mp3|oga|ogg|opus|wav|wma)$`,
		"Case-insensitive regular expression for audio files")
	flag.Float64Var(&fps.length, "length", fps.length, `Max audio duration to process (fpcalc -length flag)`)
	flag.BoolVar(&fps.overlap, "overlap", fps.overlap, `Overlap audio chunks (fpcalc -overlap flag)`)
	flag.Parse()

	os.Exit(func() int {
		// Perform some initial checks before creating the database file.
		if flag.NArg() != 1 {
			flag.Usage()
			return 2
		}
		dir := flag.Arg(0)
		if fi, err := os.Stat(dir); err != nil {
			fmt.Fprintln(os.Stderr, "Invalid audio dir:", err)
			return 1
		} else if !fi.IsDir() {
			fmt.Fprintln(os.Stderr, dir, "is not a directory")
			return 1
		}

		if opts.bits < 1 || opts.bits > 32 {
			fmt.Fprintln(os.Stderr, "-bits must be in the range [1, 32]")
			return 2
		}
		var err error
		if opts.re, err = regexp.Compile(*fileRegexp); err != nil {
			fmt.Fprintln(os.Stderr, "-file-regexp invalid:", err)
			return 2
		}

		if !haveFpcalc() {
			fmt.Fprintln(os.Stderr, "fpcalc not in path (install libchromaprint-tools?)")
			return 1
		}

		if *dbPath == "" {
			f, err := ioutil.TempFile("", "soundalike.db.*")
			if err != nil {
				fmt.Fprintln(os.Stderr, "Failed creating temp file for database:", err)
				return 1
			}
			f.Close()
			*dbPath = f.Name()
			defer os.Remove(*dbPath)
		}
		db, err := newAudioDB(*dbPath, fps)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed opening database:", err)
			return 1
		}
		defer func() {
			if err := db.close(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed closing database:", err)
			}
		}()

		scanFiles(dir, &opts, db, fps)

		return 0
	}())
}

type scanOptions struct {
	re   *regexp.Regexp
	bits int
}

func scanFiles(dir string, opts *scanOptions, db *audioDB, fps *fpcalcSettings) error {
	lookup := make(map[uint32][]int64) // truncated fingerprint values to file IDs

	return filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if p == dir || fi.IsDir() || !opts.re.MatchString(filepath.Base(p)) {
			return nil
		}

		rel := p[len(dir)+1:]
		id, fprint, err := db.get(rel)
		if err != nil {
			return err
		}
		if fprint == nil {
			if fprint, err = runFpcalc(p, fps); err != nil {
				return err
			}
			if id, err = db.save(rel, fprint); err != nil {
				return err
			}
		}

		hits := make(map[int64]int) // file ID to number of matching truncated values

		// TODO: Handle duplicate entries?
		for _, v := range fprint {
			key := v >> (32 - opts.bits)
			ids := lookup[key]
			for _, oid := range ids {
				if oid != id {
					hits[oid]++
				}
			}
			lookup[key] = append(ids, id)
		}

		for oid, cnt := range hits {
			fmt.Printf("%d: %d (%d/%d)\n", id, oid, cnt, len(fprint))
		}

		return nil
	})
}
