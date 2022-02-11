// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	fps := defaultFpcalcSettings()
	opts := defaultScanOptions()

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage %v: [flag]... <DIR>\n"+
			"Finds duplicate audio files in a directory tree.\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.IntVar(&fps.algorithm, "algorithm", fps.algorithm, `Fingerprint algorithm (fpcalc -algorithm flag)`)
	flag.IntVar(&opts.bits, "bits", opts.bits, "Fingerprint bits to use (max is 32)")
	flag.Float64Var(&fps.chunk, "chunk", fps.chunk, `Audio chunk duration (fpcalc -chunk flag)`)
	dbPath := flag.String("db", "", `SQLite database file for storing fingerprints (empty for temp file)`)
	flag.StringVar(&opts.fileString, "file-regexp", opts.fileString, "Case-insensitive regular expression for audio files")
	flag.Float64Var(&fps.length, "length", fps.length, `Max audio duration to process (fpcalc -length flag)`)
	flag.Float64Var(&opts.lookupThresh, "lookup-threshold", opts.lookupThresh, `Match threshold for lookup table in (0.0, 1.0]`)
	flag.BoolVar(&fps.overlap, "overlap", fps.overlap, `Overlap audio chunks (fpcalc -overlap flag)`)
	flag.Parse()

	os.Exit(func() int {
		// Perform some initial checks before creating the database file.
		if flag.NArg() != 1 {
			flag.Usage()
			return 2
		}
		opts.dir = flag.Arg(0)
		if err := opts.finish(); err != nil {
			fmt.Fprintln(os.Stderr, err)
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

		groups, err := scanFiles(opts, db, fps)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed scanning files:", err)
			return 1
		}
		for i, group := range groups {
			if i != 0 {
				fmt.Println()
			}
			for _, p := range group {
				fmt.Println(filepath.Join(opts.dir, p))
			}
		}

		return 0
	}())
}

type scanOptions struct {
	dir          string         // directory containing audio files
	fileString   string         // uncompiled fileRegexp
	fileRegexp   *regexp.Regexp // matches files to scan
	bits         int            // bits to use from 32-bit fingerprint values
	lookupThresh float64        // match threshold for lookup table in (0.0, 1.0]
}

func defaultScanOptions() *scanOptions {
	return &scanOptions{
		// TODO: I'm just guessing what should be included here. See
		// https://en.wikipedia.org/wiki/Audio_file_format#List_of_formats and
		// https://en.wikipedia.org/wiki/FFmpeg#Supported_codecs_and_formats.
		fileString: `\.(aiff|flac|m4a|mp3|oga|ogg|opus|wav|wma)$`,
		bits:       20,
		// TODO: I have no idea what this should be.
		lookupThresh: 0.25,
	}
}

func (o *scanOptions) finish() error {
	if fi, err := os.Stat(o.dir); err != nil {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("%v is not a directory", o.dir)
	}
	if o.bits < 1 || o.bits > 32 {
		return errors.New("bits must be in the range [1, 32]")
	}
	var err error
	if o.fileRegexp, err = regexp.Compile(o.fileString); err != nil {
		return fmt.Errorf("bad file regexp: %v", err)
	}
	return nil
}

func scanFiles(opts *scanOptions, db *audioDB, fps *fpcalcSettings) ([][]string, error) {
	lookup := newLookupTable()
	edges := make(map[fileID][]fileID)

	if err := filepath.Walk(opts.dir, func(p string, fi os.FileInfo, err error) error {
		if p == opts.dir || fi.IsDir() || !opts.fileRegexp.MatchString(filepath.Base(p)) {
			return nil
		}

		rel := p[len(opts.dir)+1:]
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

		// TODO: This probably produces lots of false positives.
		// I should probably do additional comparisons using the full fingerprints.
		thresh := int(float64(len(fprint)) * opts.lookupThresh)
		for _, oid := range lookup.find(fprint, thresh) {
			edges[id] = append(edges[id], oid)
			edges[oid] = append(edges[oid], id)
		}

		lookup.add(id, fprint)
		return nil
	}); err != nil {
		return nil, err
	}

	var err error
	var groups [][]string
	for _, comp := range components(edges) {
		group := make([]string, len(comp))
		for i, id := range comp {
			if group[i], err = db.path(id); err != nil {
				return nil, fmt.Errorf("getting path for %d: %v", id, err)
			}
		}
		groups = append(groups, group)
	}
	return groups, nil
}

// components returns all components from the undirected graph described by edges.
func components(edges map[fileID][]fileID) [][]fileID {
	visited := make(map[fileID]struct{})

	var search func(fileID) []fileID
	search = func(src fileID) []fileID {
		if _, ok := visited[src]; ok {
			return nil
		}
		visited[src] = struct{}{}
		comp := []fileID{src}
		for _, dst := range edges[src] {
			comp = append(comp, search(dst)...)
		}
		return comp
	}

	var comps [][]fileID
	for src := range edges {
		if _, ok := visited[src]; !ok {
			comps = append(comps, search(src))
		}
	}
	return comps
}
