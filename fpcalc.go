// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

// fpcalcSettings contains command-line settings for the fpcalc utility.
type fpcalcSettings struct {
	length    float64 // "-length SECS   Restrict the duration of the processed input audio (default 120)"
	chunk     float64 // "-chunk SECS    Split the input audio into chunks of this duration"
	algorithm int     // "-algorithm NUM Set the algorithm method (default 2)"
	overlap   bool    // "-overlap       Overlap the chunks slightly to make sure audio on the edges is fingerprinted"
}

func defaultFpcalcSettings() *fpcalcSettings {
	return &fpcalcSettings{
		length:    15,
		chunk:     0,
		algorithm: 2,
		overlap:   false,
	}
}

func (s *fpcalcSettings) String() string {
	return fmt.Sprintf("length=%0.3f,chunk=%0.3f,algorithm=%d,overlap=%v",
		s.length, s.chunk, s.algorithm, s.overlap)
}

// haveFpcalc returns false if fpcalc isn't in $PATH.
func haveFpcalc() bool {
	_, err := exec.LookPath("fpcalc")
	return err == nil
}

// runFpcalc runs fpcalc to compute a fingerprint for path per settings.
func runFpcalc(path string, settings *fpcalcSettings) ([]uint32, error) {
	args := []string{
		"-raw",
		"-plain",
		"-length", strconv.FormatFloat(settings.length, 'f', 3, 64),
		"-algorithm", strconv.Itoa(settings.algorithm),
	}
	if settings.chunk > 0 {
		args = append(args, "-chunk", strconv.FormatFloat(settings.chunk, 'f', 3, 64))
	}
	if settings.overlap {
		args = append(args, "-overlap")
	}
	args = append(args, path)

	out, err := exec.Command("fpcalc", args...).Output()
	if err != nil {
		return nil, err
	}

	var fprint []uint32
	for _, s := range strings.Split(strings.TrimSpace(string(out)), ",") {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		if v < 0 || v > math.MaxUint32 {
			return nil, fmt.Errorf("non-uint32 value %v", v)
		}
		fprint = append(fprint, uint32(v))
	}
	return fprint, nil
}
