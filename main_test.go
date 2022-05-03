// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func checkTestEnv() error {
	if _, err := exec.LookPath("soundalike"); err != nil {
		return errors.New("soundalike executable not in path")
	}
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		return errors.New("testdata/ should be https://github.com/derat/soundalike-testdata checkout")
	} else if err != nil {
		return err
	}
	return nil
}

func TestMain_Scan(t *testing.T) {
	if err := checkTestEnv(); err != nil {
		t.Fatal("Bad test environment: ", err)
	}

	want := strings.TrimLeft(`
64/Fanfare for Space.mp3
orig/Fanfare for Space.mp3
pad/Fanfare for Space.mp3

64/Honey Bee.mp3
orig/Honey Bee.mp3
pad/Honey Bee.mp3
`, "\n")

	db := filepath.Join(t.TempDir(), "test.db")
	scanCmd := exec.Command(
		"soundalike",
		"-db="+db,
		"-log-sec=0",
		"-print-file-info=false",
		"-fpcalc-length=45",
		"testdata",
	)
	if got, err := scanCmd.Output(); err != nil {
		t.Errorf("%s failed: %v", scanCmd, err)
	} else if string(got) != want {
		t.Errorf("%s printed unexpected output:\n got: %q\n want: %q", scanCmd, string(got), want)
	}

	// Exclude the second group.
	excludeCmd := exec.Command(
		"soundalike",
		"-db="+db,
		"-fpcalc-length=45",
		"-exclude",
		"64/Honey Bee.mp3",
		"orig/Honey Bee.mp3",
		"pad/Honey Bee.mp3",
	)
	if err := excludeCmd.Run(); err != nil {
		t.Errorf("%s failed: %v", excludeCmd, err)
	}

	// Do another scan and check that only the first group is printed.
	want2 := strings.Split(want, "\n\n")[0] + "\n"
	scanCmd = exec.Command(scanCmd.Args[0], scanCmd.Args[1:]...)
	if got, err := scanCmd.Output(); err != nil {
		t.Errorf("%s failed: %v", scanCmd, err)
	} else if string(got) != want2 {
		t.Errorf("%s printed unexpected output:\n got: %q\n want: %q", scanCmd, string(got), want2)
	}
}

func TestMain_Compare(t *testing.T) {
	if err := checkTestEnv(); err != nil {
		t.Fatal("Bad test environment: ", err)
	}

	type result int
	const (
		identical result = iota
		similar
		different
	)
	const thresh = 0.95 // threshold for "similar" songs

	const (
		file1 = "Honey Bee.mp3"
		file2 = "Fanfare for Space.mp3"
	)

	for _, tc := range []struct {
		a, b string // paths under testdata/
		res  result
	}{
		{"orig/" + file1, "orig/" + file1, identical},
		{"orig/" + file1, "orig/" + file2, different},
		{"orig/" + file1, "64/" + file1, similar},
		{"orig/" + file1, "pad/" + file1, similar},
	} {
		cmd := exec.Command(
			"soundalike",
			"-compare",
			filepath.Join("testdata", tc.a),
			filepath.Join("testdata", tc.b),
		)
		out, err := cmd.Output()
		if err != nil {
			t.Errorf("%s failed: %v", cmd, err)
			continue
		}
		got, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
		if err != nil {
			t.Errorf("%s printed bad output %q: %v", cmd, string(out), err)
			continue
		}
		if tc.res == identical && got != 1.0 {
			t.Errorf("%s returned %0.3f; want 1.0", cmd, got)
		} else if tc.res == similar && (got < thresh || got >= 1.0) {
			t.Errorf("%s returned %0.3f; want [%0.3f, 1.0)", cmd, got, thresh)
		} else if tc.res == different && (got < 0 || got >= thresh) {
			t.Errorf("%s returned %0.3f; want [0.0, %0.3f)", cmd, got, thresh)
		}
	}
}
