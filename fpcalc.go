// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

// fpcalcResult contains the result of running fpcalc against a file.
type fpcalcResult struct {
	Fingerprint []uint32 `json:"fingerprint"`
	Duration    float64  `json:"duration"`
}

// emptyFingerprintError is returned by runFpcalc when an audio file is too short
// to be fingerprinted.
var errEmptyFingerprint = errors.New("empty fingerprint")

// runFpcalc runs fpcalc to compute a fingerprint for path per settings.
func runFpcalc(path string, settings *fpcalcSettings) (*fpcalcResult, error) {
	args := []string{
		"-raw",
		"-json",
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
		// Try to get some additional info from stderr.
		if exit, ok := err.(*exec.ExitError); ok {
			if stderr := strings.SplitN(string(exit.Stderr), "\n", 2)[0]; stderr != "" {
				if strings.TrimSpace(stderr) == "ERROR: Empty fingerprint" {
					return nil, errEmptyFingerprint
				}
				err = fmt.Errorf("%v (%v)", err, stderr)
			}
		}
		return nil, err
	}
	var res fpcalcResult
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
