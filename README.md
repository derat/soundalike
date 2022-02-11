# soundalike

`soundalike` is a command-line program written in Go that tries to find
duplicate songs in a music collection by comparing acoustic fingerprints.
Fingerprints are generated using the `fpcalc` utility from the the [Chromaprint]
library.

[Chromaprint]: https://github.com/acoustid/chromaprint

## Usage

```
Usage soundalike: [flag]... <DIR>
Finds duplicate audio files in a directory tree.

  -algorithm int
        Fingerprint algorithm (fpcalc -algorithm flag) (default 2)
  -bits int
        Fingerprint bits to use (max is 32) (default 20)
  -chunk float
        Audio chunk duration (fpcalc -chunk flag)
  -db string
        SQLite database file for storing fingerprints (empty for temp file)
  -file-regexp string
        Case-insensitive regular expression for audio files (default "\\.(aiff|flac|m4a|mp3|oga|ogg|opus|wav|wma)$")
  -length float
        Max audio duration to process (fpcalc -length flag) (default 15)
  -lookup-threshold float
        Match threshold for lookup table in (0.0, 1.0] (default 0.25)
  -overlap
        Overlap audio chunks (fpcalc -overlap flag)
```

`fpcalc` must be in your path. On Debian systems, it can be installed by running
```
sudo apt install libchromaprint-utils
```

## More info

The following pages contain background information that may be of interest:

*   [Acoustid's Chromaprint page](https://acoustid.org/chromaprint)
*   [Lukáš Lalinský's "How does Chromaprint work?" post](https://oxygene.sk/2011/01/how-does-chromaprint-work/)
*   [This "question about using chromaprint to identify the same tracks" thread from the Acoustid mailing list](https://groups.google.com/g/acoustid/c/C3EHIkZVpZI/m/Zd2qdOKRNzkJ)
