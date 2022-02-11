// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func main() {
	fps := defaultFpcalcSettings()
	opts := defaultScanOptions()

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage %v: [flag]... <DIR>\n"+
			"Finds duplicate audio files within a directory.\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	dbPath := flag.String("db", "", `SQLite database file for storing file info (empty for temp file)`)
	flag.StringVar(&opts.fileString, "file-regexp", opts.fileString, "Case-insensitive regular expression for audio files")
	flag.IntVar(&fps.algorithm, "fpcalc-algorithm", fps.algorithm, `Fingerprint algorithm`)
	flag.Float64Var(&fps.chunk, "fpcalc-chunk", fps.chunk, `Audio chunk duration in seconds`)
	flag.Float64Var(&fps.length, "fpcalc-length", fps.length, `Max audio duration in seconds to process`)
	flag.BoolVar(&fps.overlap, "fpcalc-overlap", fps.overlap, `Overlap audio chunks in fingerprints`)
	flag.IntVar(&opts.logSec, "log-sec", opts.logSec, `Logging frequency in seconds (0 or negative to disable logging)`)
	flag.Float64Var(&opts.lookupThresh, "lookup-threshold", opts.lookupThresh, `Match threshold for lookup table in (0.0, 1.0]`)
	printFileInfo := flag.Bool("print-file-info", true, `Print file sizes and durations`)
	printFullPaths := flag.Bool("print-full-paths", false, `Print absolute file paths (rather than relative to dir)`)
	flag.BoolVar(&opts.skipBadFiles, "skip-bad-files", true, `Skip files that can't be fingerprinted by fpcalc`)
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

		var pre string
		if *printFullPaths {
			pre = opts.dir + "/"
		}
		for i, infos := range groups {
			if i != 0 {
				fmt.Println()
			}
			if *printFileInfo {
				for _, ln := range formatFiles(infos, pre) {
					fmt.Println(ln)
				}
			} else {
				for _, info := range infos {
					fmt.Println(pre + info.path)
				}
			}
		}

		return 0
	}())
}

func formatFiles(infos []*fileInfo, pathPrefix string) []string {
	if len(infos) == 0 {
		return nil
	}

	var rows [][]string
	lens := make([]int, 3)
	for _, info := range infos {
		row := []string{
			pathPrefix + info.path,
			strconv.FormatFloat(float64(info.size)/(1024*1024), 'f', 2, 64),
			strconv.FormatFloat(info.duration, 'f', 2, 64),
		}
		rows = append(rows, row)
		for i, max := range lens {
			if ln := len(row[i]); ln > max {
				lens[i] = ln
			}
		}
	}
	lines := make([]string, len(rows))
	fs := strings.Join([]string{
		"%" + strconv.Itoa(-lens[0]) + "s",    // path
		"%" + strconv.Itoa(lens[1]) + "s MB",  // size
		"%" + strconv.Itoa(lens[2]) + "s sec", // duration
	}, "  ")
	for i, row := range rows {
		lines[i] = fmt.Sprintf(fs, row[0], row[1], row[2])
	}
	return lines
}
