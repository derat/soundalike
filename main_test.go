// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	if _, err := exec.LookPath("soundalike"); err != nil {
		t.Fatal("soundalike executable not in path")
	}
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		t.Fatal("testdata/ should be https://github.com/derat/soundalike-testdata checkout")
	} else if err != nil {
		t.Fatal(err)
	}

	want := strings.TrimLeft(`
64/Fanfare for Space.mp3
orig/Fanfare for Space.mp3
pad/Fanfare for Space.mp3

64/Honey Bee.mp3
orig/Honey Bee.mp3
pad/Honey Bee.mp3
`, "\n")

	cmd := exec.Command(
		"soundalike",
		"-log-sec=0",
		"-print-file-info=false",
		"-fpcalc-length=45",
		"testdata")
	if got, err := cmd.Output(); err != nil {
		t.Fatal("soundalike failed: ", err)
	} else if string(got) != want {
		t.Errorf("soundalike printed unexpected output:\n got: %q\n want: %q", string(got), want)
	}
}
