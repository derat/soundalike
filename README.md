# soundalike

[![Build Status](https://storage.googleapis.com/derat-build-badges/94f3a3fa-5be1-4aee-883b-7907fb50a7fa.svg)](https://storage.googleapis.com/derat-build-badges/94f3a3fa-5be1-4aee-883b-7907fb50a7fa.html)

`soundalike` is a command-line program that tries to find similar audio files by
comparing acoustic fingerprints. Its main focus is identifying duplicate songs
in music collections.

Fingerprints are generated using the `fpcalc` utility from the [Chromaprint]
library (which does basically all of the heavy lifting). No network requests
are made to [AcoustID] or other APIs.

[Chromaprint]: https://github.com/acoustid/chromaprint
[AcoustID]: https://acoustid.org/

## Usage

To compile and install the `soundalike` executable, run `go install` from the
root of this repository. You will need to have [Go] installed.

[Go]: https://go.dev/

`fpcalc` must be in your path. On a Debian system, it can be installed by
running:

```
sudo apt install libchromaprint-utils
```

`soundalike` scans all of the audio files that it finds in the supplied
directory and then prints groups of similar files.

```
Usage: soundalike [flag]... <DIR>
Find duplicate audio files within a directory.

  -compare
    	Compare two files given via positional args instead of scanning directory
    	(increases -fpcalc-length by default)
  -compare-interval int
    	Score interval for -compare (0 to print overall score)
  -db string
    	SQLite database file for storing file info (temp file if unset)
  -exclude
    	Update database to exclude files in positional args from being grouped together
  -file-regexp string
    	Regular expression for audio files (default "(?i)\\.(aiff|flac|m4a|mp3|oga|ogg|opus|wav|wma)$")
  -fpcalc-algorithm int
    	Fingerprint algorithm (default 2)
  -fpcalc-chunk float
    	Audio chunk duration in seconds
  -fpcalc-length float
    	Max audio duration in seconds to process (default 15)
  -fpcalc-overlap
    	Overlap audio chunks in fingerprints
  -log-sec int
    	Logging frequency in seconds (0 or negative to disable logging) (default 10)
  -lookup-threshold float
    	Threshold for lookup table in (0.0, 1.0] (default 0.25)
  -match-min-length
    	Use shorter fingerprint length when scoring bitwise comparisons
  -match-threshold float
    	Threshold for bitwise comparisons in (0.0, 1.0] (default 0.95)
  -print-file-info
    	Print file sizes and durations (default true)
  -print-full-paths
    	Print absolute file paths (rather than relative to dir)
  -skip-bad-files
    	Skip files that can't be fingerprinted by fpcalc (default true)
  -skip-new-files
    	Skip files not already in database given via -db
  -version
    	Print version and exit
```

Example output when scanning a directory:

```
% soundalike .
2022/02/12 08:49:49 Finished scanning 67 files
eva02.mp3    3.07 MB  120.45 sec
Eva_Two.mp3  1.84 MB  120.46 sec

hedgehogs_dilemma.mp3  3.82 MB  167.37 sec
Hedgehogs_Dilemma.mp3  2.55 MB  167.35 sec

...
```

`-compare` can also be used to compare two files:

```
% soundalike -compare \
  fly_me_to_the_moon_instrumental_version.mp3 \
  Fly_Me_To_The_Moon_Instrumental.mp3
0.972
% soundalike -compare \
  fly_me_to_the_moon_instrumental_version.mp3 \
  Fly_Me_To_The_Moon_Instrumental_2.mp3
0.202
```

Add `-compare-interval 100` to instruct `-compare` to print a score after
comparing every 100 value pairs within the two fingerprints instead of printing
an overall score. This can help approximate the point at which differences
between two songs occur:

```
% soundalike -compare -compare-interval 100 instrumental.mp3 vocals.mp3
 100: 0.983
 200: 0.984
 300: 0.983
 400: 0.989
 500: 0.819
 600: 0.757
 700: 0.751
 800: 0.753
...
```

The `-match-min-length` flag may be helpful for detecting songs that have been
truncated or that contain added silence at their beginnings or ends.

## How it works

[Chromaprint]'s `fpcalc` utility splits audio into overlapping frames and uses
[Fourier transforms] to identify (Western) notes, eventually generating
fingerprints consisting of sequences of 32-bit unsigned integers (in my
experience, ~100 for a 15-second sample and ~462 for a minute).

`soundalike` runs `fpcalc` on each file and maintains a lookup table from
values from fingerprints (truncated from 32 to 16 bits) to files. Fingerprint
values (irrespective of order) are compared between files. If
`-lookup-threshold` or more of the values match, the file pair moves on to the
next phase.

The two fingerprints are then compared in their original order, with all
possible alignments, to see how many bits match between the two. If more than
`-match-threshold` of the bits are identical, the file pair is retained.

Pairs are treated as edges in an undirected graph, and files are grouped into
components. Finally, each group is printed.

When `-compare` is passed, only the final bitwise comparison is performed.

[Fourier transforms]: https://en.wikipedia.org/wiki/Fourier_transform

## Accuracy

Vocal and instrumental versions of the same song can be problematic. With the
`-fpcalc-length` flag's default value only the first 15 seconds of each file are
fingerprinted, so if the vocals only come in later in the song the files will
appear to be identical. A larger value can be passed at the the expense of
performance and memory consumption.

I've also sometimes seen false positives between tracks that start with silence,
or between electronic tracks that start with the same chords or with similar
drumbeats. Passing a higher `-match-threshold` value may prevent some of these.

When running `soundalike` repeatedly with the `-db` flag, `-exclude` can be used
to specify false positives that should be excluded from future runs.

`fpcalc` isn't able to generate fingerprints for very short files, so they're
silently skipped. The cutoff seems to be somewhere around 2 to 3 seconds.

## Performance

Performance is largely dependent on the `-fpcalc-length` flag's value.

On a laptop with an Intel Core i5-8250U CPU 1.60GHz processor, `soundalike`
takes about 10 seconds to scan 99 MP3 and WAV files totalling 266 MB using the
default flags (i.e. fingerprinting up to 15 seconds of each file).

A much slower system with an Intel Celeron 2955U 1.40GHz processor takes roughly
3 minutes to scan 1,000 "song-length" MP3 files with default flags (so, about
20,000 songs per hour).

When running against a large music collection, the `-db` flag can be passed to
save fingerprints and other file information for future runs. Note that the
database will not be reusable if you pass different `-fpcalc-*` flags in the
future, since those change how `fpcalc` computes fingerprints.

When running `soundalike` repeatedly, `-skip-new-files` can be used to avoid
repeatedly trying to fingerprint bad/corrupted files.

## Memory usage

Memory usage grows with the number of files that are scanned and the fingerprint
length, since (truncated) values from fingerprints are stored in memory to
enable searching for collisions. I saw the resident set size (`RSS` in `ps`,
`RES` in `top`) grow to 78,116 KB while scanning a bit over 20,000 MP3 files
with 15-second fingerprints.

## More information

The following pages contain additional technical details that may be of
interest:

*   [Acoustid's Chromaprint page](https://acoustid.org/chromaprint)
*   [Lukáš Lalinský's "How does Chromaprint work?" post](https://oxygene.sk/2011/01/how-does-chromaprint-work/)
*   ["question about using chromaprint to identify the same tracks" from Acoustid mailing list](https://groups.google.com/g/acoustid/c/C3EHIkZVpZI/m/Zd2qdOKRNzkJ)

See also the [Picard] music tagger from the [MusicBrainz] project.

[Picard]: https://picard.musicbrainz.org/
[MusicBrainz]: https://musicbrainz.org/
