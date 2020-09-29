### dfuse Tooling

A bunch of command line tools that is used by multiple developers within dfuse.

#### Install

To install the Golang utilities, simply use `./scripts/install_all.sh`.

To install the Ruby utilities (slowly converting them to Golang please), first
install the required global dependencies:

```
gem install timeliness
```

Then add the folder's absolute path to your `PATH` environment variable.

#### Usage

Most (if not all) of the tools accept their arguments through standard input
or from command line arguments directly. It's one or the other and standard input
takes precedence if found to be coming from a script.

```
bytes 100000
100.00 KB

bytes 100000000
100.00 MB

to_base64 abfe0102       # <-- As hexadecimal
q/4BAg==

to_base64 myname         # <-- As string
bXluYW1l

to_bigint 11e1a300
236771705847092

to_hex 123456            # <-- As decimal
01e240

to_hex q/4BAg==          # <-- As base64
abfe0102

to_hex -s ascii          # <-- As string
6173636969

to_hex '"ascii"'         # <-- As string
6173636969

to_lower ABdg
abdg

to_upper ABdg
ABDG
```

#### Caveats

The standard input is fully consumed than split into lines and then process. So in
it's current form, this project does not supports streaming from big load of data.

PRs welcome!
