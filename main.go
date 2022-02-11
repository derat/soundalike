// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"flag"
	"fmt"
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
	settings := defaultFpcalcSettings()
	flag.IntVar(&settings.algorithm, "algorithm", settings.algorithm, `Fingerprint algorithm (fpcalc -algorithm flag)`)
	bits := flag.Int("bits", 20, "Fingerprint bits to use (max is 32)")
	flag.Float64Var(&settings.chunk, "chunk", settings.chunk, `Audio chunk duration (fpcalc -chunk flag)`)
	dbPath := flag.String("db", "", `SQLite database file for storing fingerprints`)
	// TODO: I'm just guessing what should be included here. See
	// https://en.wikipedia.org/wiki/Audio_file_format#List_of_formats and
	// https://en.wikipedia.org/wiki/FFmpeg#Supported_codecs_and_formats.
	fileRegexp := flag.String("file-regexp", `\.(aiff|flac|m4a|mp3|oga|ogg|opus|wav|wma)$`,
		"Case-insensitive regular expression for audio files")
	flag.Float64Var(&settings.length, "length", settings.length, `Max audio duration to process (fpcalc -length flag)`)
	flag.BoolVar(&settings.overlap, "overlap", settings.overlap, `Overlap audio chunks (fpcalc -overlap flag)`)
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

		if *bits < 1 || *bits > 32 {
			fmt.Fprintln(os.Stderr, "-bits must be in the range [1, 32]")
			return 2
		}
		re, err := regexp.Compile(*fileRegexp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "-file-regexp invalid:", err)
			return 2
		}

		if !haveFpcalc() {
			fmt.Fprintln(os.Stderr, "fpcalc not in path (install libchromaprint-tools?)")
			return 1
		}

		var db *audioDB
		if *dbPath != "" {
			var err error
			if db, err = newAudioDB(*dbPath, settings); err != nil {
				fmt.Fprintln(os.Stderr, "Failed opening database:", err)
				return 1
			}
			defer func() {
				if err := db.close(); err != nil {
					fmt.Fprintln(os.Stderr, "Failed closing database:", err)
				}
			}()
		}

		processFiles(dir, re, *bits, settings, db)

		return 0
	}())
}

func processFiles(dir string, re *regexp.Regexp, bits int, settings *fpcalcSettings, db *audioDB) error {
	if err := filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if p == dir || fi.IsDir() || !re.MatchString(filepath.Base(p)) {
			return nil
		}

		rel := p[len(dir)+1:]
		var fprint []uint32
		if db != nil {
			if fprint, err = db.get(rel); err != nil {
				return err
			}
		}
		if fprint == nil {
			if fprint, err = runFpcalc(p, settings); err != nil {
				return err
			}
		}
		if db != nil {
			if err := db.save(rel, fprint); err != nil {
				return err
			}
		}

		fprint = fprint[:bits]

		// TODO: Save fingerprint for later lookup.

		return nil
	}); err != nil {
		return err
	}

	// TODO: Find duplicates.

	return nil
}
