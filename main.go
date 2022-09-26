// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/bits"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

var buildVersion = "non-release" // injected by create_release.sh

func main() {
	fps := defaultFpcalcSettings()
	opts := defaultScanOptions()

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %v [flag]... <DIR>\n"+
			"Find duplicate audio files within a directory.\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	compare := flag.Bool("compare", false, `Compare two files given via positional args instead of scanning directory`+
		"\n(increases -fpcalc-length by default)")
	compareInterval := flag.Int("compare-interval", 0, `Score interval for -compare (0 to print overall score)`)
	dbPath := flag.String("db", "", `SQLite database file for storing file info (temp file if unset)`)
	exclude := flag.Bool("exclude", false, `Update database to exclude files in positional args from being grouped together`)
	flag.StringVar(&opts.fileString, "file-regexp", opts.fileString, "Regular expression for audio files")
	flag.IntVar(&fps.algorithm, "fpcalc-algorithm", fps.algorithm, `Fingerprint algorithm`)
	flag.Float64Var(&fps.chunk, "fpcalc-chunk", fps.chunk, `Audio chunk duration in seconds`)
	flag.Float64Var(&fps.length, "fpcalc-length", fps.length, `Max audio duration in seconds to process`)
	flag.BoolVar(&fps.overlap, "fpcalc-overlap", fps.overlap, `Overlap audio chunks in fingerprints`)
	flag.IntVar(&opts.logSec, "log-sec", opts.logSec, `Logging frequency in seconds (0 or negative to disable logging)`)
	flag.Float64Var(&opts.lookupThresh, "lookup-threshold", opts.lookupThresh, `Threshold for lookup table in (0.0, 1.0]`)
	flag.Float64Var(&opts.matchThresh, "match-threshold", opts.matchThresh, `Threshold for bitwise comparisons in (0.0, 1.0]`)
	flag.BoolVar(&opts.matchMinLength, "match-min-length", opts.matchMinLength,
		`Use shorter fingerprint length when scoring bitwise comparisons`)
	printFileInfo := flag.Bool("print-file-info", true, `Print file sizes and durations`)
	printFullPaths := flag.Bool("print-full-paths", false, `Print absolute file paths (rather than relative to dir)`)
	flag.BoolVar(&opts.skipBadFiles, "skip-bad-files", opts.skipBadFiles, `Skip files that can't be fingerprinted by fpcalc`)
	flag.BoolVar(&opts.skipNewFiles, "skip-new-files", opts.skipNewFiles, `Skip files not already in database given via -db`)
	printVersion := flag.Bool("version", false, `Print version and exit`)
	flag.Parse()

	os.Exit(func() int {
		if *printVersion {
			doVersion()
			return 0
		}

		// Perform some initial checks before creating the database file.
		if *compare {
			if flag.NArg() != 2 {
				flag.Usage()
				return 2
			}
		} else if *exclude {
			if flag.NArg() < 2 {
				flag.Usage()
				return 2
			}
			if *dbPath == "" {
				fmt.Fprintln(os.Stderr, "-exclude requires -db")
				return 2
			}
		} else {
			if flag.NArg() != 1 {
				flag.Usage()
				return 2
			}
			opts.dir = flag.Arg(0)
		}
		if err := opts.finish(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}

		if !haveFpcalc() {
			advice := "install from https://github.com/acoustid/chromaprint/releases"
			if _, err := exec.LookPath("apt"); err == nil {
				advice = "apt install libchromaprint-tools"
			}
			fmt.Fprintf(os.Stderr, "fpcalc not in path (%v)\n", advice)
			return 1
		}

		if *compare {
			// If -fpcalc-length wasn't specified, make it default to a larger
			// value so we'll fingerprint the files in their entirety.
			if !flagWasSet("fpcalc-length") {
				fps.length = 7200
			}
			return doCompare(flag.Arg(0), flag.Arg(1), opts, fps, *compareInterval)
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

		if *exclude {
			// Save all possible pairs within the group.
			for i := 0; i < flag.NArg()-1; i++ {
				for j := i + 1; j < flag.NArg(); j++ {
					if err := db.saveExcludedPair(flag.Arg(i), flag.Arg(j)); err != nil {
						fmt.Fprintln(os.Stderr, "Failed saving excluded pair:", err)
						return 1
					}
				}
			}
			return 0
		}

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

// flagWasSet returns true if the specified flag was passed on the command line.
func flagWasSet(name string) bool {
	var found bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// doVersion prints the soundalike and fpcalc versions to stdout.
func doVersion() {
	fmt.Printf("soundalike %v compiled with %v for %v/%v\n",
		buildVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if ver, err := getFpcalcVersion(); err != nil {
		fmt.Printf("Failed getting fpcalc version: %v\n", err)
	} else {
		fmt.Println(ver)
	}
}

// doCompare compares the files at pa and pb on behalf of the -compare flag.
func doCompare(pa, pb string, opts *scanOptions, fps *fpcalcSettings, interval int) int {
	ra, err := runFpcalc(pa, fps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed fingerprinting %v: %v\n", pa, err)
		return 1
	}
	rb, err := runFpcalc(pb, fps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed fingerprinting %v: %v\n", pb, err)
		return 1
	}
	score, aoff, boff := compareFingerprints(ra.Fingerprint, rb.Fingerprint, opts.matchMinLength)
	if interval <= 0 {
		fmt.Printf("%0.3f\n", score)
	} else {
		if aoff > boff {
			fmt.Printf("[%d only in b]\n", aoff-boff)
		} else if boff > aoff {
			fmt.Printf("[%d only in a]\n", boff-aoff)
		}
		a := ra.Fingerprint[aoff:]
		b := rb.Fingerprint[boff:]
		var i, ncmp, nbits int
		for ; i < len(a) && i < len(b); i++ {
			if i%interval == 0 && ncmp > 0 {
				fmt.Printf("%4d: %0.3f\n", i, float64(nbits)/float64(32*ncmp))
				nbits = 0
				ncmp = 0
			}
			nbits += 32 - bits.OnesCount32(a[i]^b[i])
			ncmp++
		}
		if ncmp > 0 {
			fmt.Printf("%4d: %0.3f\n", i, float64(nbits)/float64(32*ncmp))
		}
		if na, nb := len(a), len(b); na > nb {
			fmt.Printf("[%d only in a]\n", na-nb)
		} else if nb > na {
			fmt.Printf("[%d only in b]\n", nb-na)
		}
	}
	return 0
}

// formatFiles returns column-aligned lines describing each supplied file.
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
