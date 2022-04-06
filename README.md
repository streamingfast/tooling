# StreamingFast Tooling
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/tooling)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A bunch of command line tools that is used by multiple developers within StreamingFast.

#### Design Principles

- Command line helper utilities
- Accepts both standard input & command line arguments
- Script friendly output (one line of input equals one line of output is a good goal)

#### Install

To install the Golang utilities, simply use `./scripts/install_all.sh`.

To install the Bash & Ruby utilities (slowly converting them to Golang please), first
install the required global dependencies:

```
gem install timeliness
```

Then add the `bin` folder's absolute path to your `PATH` environment variable:

```
export PATH=`pwd`/bin:$PATH
```

#### Usage

Most (if not all) of the tools accept their arguments through standard input
or from command line arguments directly. It's one or the other and standard input
takes precedence if found to be coming from a script.

```
bytes 100000
100.00 KB

bytes 100000000
100.00 MB

bytes -b 100000000                           # <-- As IEC standard (base 2) so KiB, MiB, etc.
95.37 MiB

stats 1 2 3 4 5 6 7 8 9                      # <-- Computes statistics about numbers received
Count: 9
Range: Min 1.00000 - Max 9.00000
Sum: 45.00000
Average: 5.00000
Median: 5.00000
Standard Deviation: 2.73861

to_base58 abfe0102                           # <-- As hexadecimal
5PzCau

to_base58 myname                             # <-- As string
wWsYQQr8

to_base64 abfe0102                           # <-- As hexadecimal
q/4BAg==

to_base64 myname                             # <-- As string
bXluYW1l

to_date 1600446733000                        # <-- Parse as unix milliseconds (milliseconds inferred)
2020-09-18T12:32:13-04:00 (2020-09-18T16:32:13Z)

to_date 1600446733                           # <-- Parse as unix seconds (seconds inferred)
2020-09-18T12:32:13-04:00 (2020-09-18T16:32:13Z)

to_date 2020-09-18T16:32:13Z                 # <-- Parse as Golang date layout (multiple layouts tried one after the other)
2020-09-18T12:32:13-04:00 (2020-09-18T16:32:13Z)

to_dec 01e240
123456

to_dec 21e19e0c9bab2400000                   # <-- Arbitrary precision
10000000000000000000000

to_duration -ms 24h                          # <-- Parse as Golang duration, returned as selected unit (here milliseconds)
ABDG

to_duration -h 124                           # <-- Parse as selected unit (here hours) and returned as Golang humanized duration
124h 0m 0s

to_duration <unit>                           # <-- Available units: -ns (Nanoseconds), -us (Microseconds), -ms (Milliseconds), -s (Seconds), -m (Minutes) and -h (Hours)

to_hex 123456                                # <-- As decimal
01e240

to_hex -b64 q/4BAg==                         # <-- As base64
abfe0102

to_hex -b58 25JnwSn7                         # <-- As base58
02284274561c

to_hex -s ascii                              # <-- As string
6173636969

to_hex '"ascii"'                             # <-- As string
6173636969

cat /dev/random | head -c 16 | to_hex -in    # <-- Random 16 bytes transformed to_hex
abfe0102

to_lower ABdg
abdg

to_upper ABdg
ABDG
```

##### `go_replace`

The `go_replace` command can be used to quickly replace go dependencies of your organization
by automatically filling the repository + organization part (`github.com/organization/`) and
resolves to a location on your disk.

With a config file located at `$HOME/.config/go_replace/default.yaml` with the follwing content:

```
default_work_dir: $HOME/work
default_repo_shortcut: github.com
default_project_shortcut: github.com/streamingfast
```

One could do

```
go_replace merger
```

In a project's root and would get a replacement statement in it's `go.mod` file that would
look like

```
go_replace github.com/streamingfast/merger => /home/john/work/merger
```

It can also be easily dropped with

```
go_replace -d merger
```

Finally, it comes with a Git hook support to ensure you do not commit
local replacement.

Install the hooks to all Git repository found from working directory:

```
go_replace hook install
```

> Runs in dry-run by default, use `-f` to actually write the hooks

This installs a `pre-push` hook that will prevent the push from happing
if commits touched any `go.mod` file and it appears that the working
directory contains some local replacement.

#### Caveats

The standard input is fully consumed then split into lines and then processed. So in
its current form, this project does not support streaming from big load of data.

PRs welcome!

## Contributing

**Issues and PR in this repo related strictly to the tooling library.**

Report any protocol-specific issues in their
[respective repositories](https://github.com/streamingfast/streamingfast#protocols)

**Please first refer to the general
[StreamingFast contribution guide](https://github.com/streamingfast/streamingfast/blob/master/CONTRIBUTING.md)**,
if you wish to contribute to this code base.

This codebase uses unit tests extensively, please write and run tests.

## License

[Apache 2.0](LICENSE)