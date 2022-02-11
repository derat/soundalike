# soundalike

`soundalike` is a Go command-line program that tries to find similar audio files
by comparing acoustic fingerprints. Its main focus is identifying duplicate
songs in music collections.

Fingerprints are generated using the `fpcalc` utility from the [Chromaprint]
library. No network requests are made to [AcoustID] or other APIs.

[Chromaprint]: https://github.com/acoustid/chromaprint
[AcoustID]: https://acoustid.org/

## Usage

To compile and install the `soundalike` executable, run `go install` from the
root of this repository. You will need to have [Go] installed.

[Go]: https://go.dev/

`soundalike` scans all of the audio files that it finds in the supplied
directory and then prints groups of similar files.

```
Usage soundalike: [flag]... <DIR>
Finds duplicate audio files within a directory.

  -algorithm int
        Fingerprint algorithm (fpcalc -algorithm flag) (default 2)
  -chunk float
        Audio chunk duration (fpcalc -chunk flag)
  -db string
        SQLite database file for storing file info (empty for temp file)
  -file-regexp string
        Case-insensitive regular expression for audio files (default "\\.(aiff|flac|m4a|mp3|oga|ogg|opus|wav|wma)$")
  -length float
        Max audio duration to process (fpcalc -length flag) (default 15)
  -log-sec int
        Logging frequency in seconds (0 or negative to disable logging) (default 10)
  -lookup-threshold float
        Match threshold for lookup table in (0.0, 1.0] (default 0.25)
  -overlap
        Overlap audio chunks (fpcalc -overlap flag)
  -print-file-info
        Print file sizes and durations (default true)
  -print-full-paths
        Print absolute file paths (rather than relative to dir)
  -skip-bad-files
        Skip files that can't be fingerprinted by fpcalc (default true)
```

`fpcalc` must be in your path. On a Debian system, it can be installed by
running:

```
sudo apt install libchromaprint-utils
```

## Performance

Performance is largely dependent on the `-length` flag's value.

On a laptop with an Intel Core i5-8250U CPU 1.60GHz processor, `soundalike`
takes about 10 seconds to scan 99 MP3 and WAV files totalling 266 MB using the
default flags (i.e. fingerprinting up to 15 seconds of each file).

A much slower system with an Intel Celeron 2955U 1.40GHz processor takes roughly
3 minutes to scan 1,000 "song-length" MP3 files with default flags (so, about
20,000 songs per hour).

When running against a large music collection, the `-db` flag can be passed to
save fingerprints and other file information for future runs. Note that the
database will not be reusable if you pass different `-algorithm`, `-chunk`,
`-length`, or `-overlap` flags in the future, since those change how `fpcalc`
computes fingerprints.

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
