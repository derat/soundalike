# soundalike

`soundalike` is a command-line program written in Go that tries to find
duplicate songs in a music collection by comparing acoustic fingerprints.
Fingerprints are generated using the `fpcalc` utility from the the [Chromaprint]
library.

[Chromaprint]: https://github.com/acoustid/chromaprint

`fpcalc` must be in your path. On Debian systems, it can be installed by
running `sudo apt install libchromaprint-utils`.

The following pages contain background information that may be of interest:

*   [Acoustid's Chromaprint page](https://acoustid.org/chromaprint)
*   [Lukáš Lalinský's "How does Chromaprint work?" post](https://oxygene.sk/2011/01/how-does-chromaprint-work/)
*   [This "question about using chromaprint to identify the same tracks" thread from the Acoustid mailing list](https://groups.google.com/g/acoustid/c/C3EHIkZVpZI/m/Zd2qdOKRNzkJ)
