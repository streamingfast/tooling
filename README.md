# StreamingFast Tooling
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/tooling)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A bunch of command line tools that is used by multiple developers within StreamingFast.

#### Design Principles

- Command line helper utilities
- Accepts both standard input & command line arguments
- Script friendly output (one line of input equals one line of output is a good goal)

#### Install

To install all our CLI utilities, you can use `go install .cmd/...`.

#### Usage

Most (if not all) of the tools accept their arguments through standard input
or from command line arguments directly. It's one or the other and standard input
takes precedence if found to be coming from a script.

- [bytes](#humanize-bytes-value) - Humanize bytes value
- [colmap](#map-a-specific-columns-over-rows-by-applying-a-command-to-the-columns-value) - Map a specific column(s) over rows by applying a command to the column's value
- [deltas](#compute-deltas-between-successive-lines) - Compute deltas between successive lines
- [go_replace](#go_replace) - Golang module local replace helper
- [skip](#skip-lines-at-the-beginning-or-end) - Skip line(s) at the beginning or end
- [stats](#computes-statistics-about-numbers-received) - Computes statistics about numbers received
- [to_ascii](#converts-input-to-ascii-string) - Converts input to ASCII string
- [to_hex](#converts-input-to-hexadecimal-encoded-string) - Converts input to hexadecimal encoded string
- [to_base58](#converts-input-to-base58-encoded-string) - Converts input to Base58 encoded string
- [to_base64](#converts-input-to-base64-encoded-string) - Converts input to Base64 encoded string
- [to_dec](#converts-input-to-integer-arbitrary-precision) - Converts input to integer (arbitrary precision)
- [to_date](#converts-input-to-iso-8601-string-format) - Converts input to ISO-8601 string format
- [to_duration](#converts-input-to-duration) - Converts input to duration
- [to_lower](#transforms-input-to-lower-case) - Transforms input to lower case
- [to_upper](#transforms-input-to-upper-case) - Transforms input to upper case

##### Converts input to ASCII string

```bash
# By default assumes hex
to_ascii 68656c6c6f
hello

# Invalid characters are replaced by . so can be used on binary to "see" string(s)
$ cat file.png | to_ascii -in
.PNG
.
IHDR...
       ...
          .....Vu\ç...	pHYs..

# Works with input a base58
to_ascii -b58 Cn8eVZg
hello

# Works with input a base64
to_ascii -b64 aGVsbG8=
hello
```

##### Skip line(s) at the beginning or end

> [!NOTE]
> This is a companion to `head` and `tail`, can be seen as a shortcut of using the two for some uses

```bash
# Skip at beginning
echo "1\n2\n3\n4" | skip 2
3
4

# Skip at end
echo "1\n2\n3\n4" | skip -2
1
2
```

##### Map a specific column(s) over rows by applying a command to the column's value

```bash
# Map a single column
echo "john 7171a\njane 9b5e61" | colmap -f 2 -d ' ' to_upper
john 7171A
jane 9B5E61

# Map a multiple columns
echo "john 7171a\njane 9b5e61" | colmap -f 1:2 -d ' ' to_upper
JOHN 7171A
JANE 9B5E61
```

##### Compute deltas between successive lines

```bash
# From arguments
deltas 1 2 44
1 (-)
2 (+1)
44 (+42)

# From stdin
echo "1\n2\n44" | deltas
1 (-)
2 (+1)
44 (+42)

# Works with timestamp too (a lot of formats accepted), useful for logs deltas
echo "2024-01-12T10:07:15.510-0500\n2024-01-12T10:17:20.139-0500\n2024-01-12T10:17:45.508-0500" | deltas
2024-01-12T10:07:15.510-0500 (-)
2024-01-12T10:17:20.139-0500 (+10m4.629s)
2024-01-12T10:17:45.508-0500 (+25.369s)
```

##### Converts input to hexadecimal encoded string

```bash
# As base58
to_hex -b58 5PzCau
abfe0102

# As base64
to_hex -b64 q/4BAg==
abfe0102

# As base64 URL safe encoding
to_hex -b64u q_4BAg==
abfe0102

# As integer
to_hex -i 126700
01eeec

# As string
to_hex -s myname
6d796e616d65

# Inferred as string if received value has double-quotes
to_hex '"myname"'
6d796e616d65

# Arguments from standard input (each line is an element to convert)
echo "q/4BAw==\nvv4BBA==\nDN4BBQ==" | to_hex -b64
abfe0103
befe0104
0cde0105

# Reads from standard input as bytes and convert to hexadecimal, random 16 bytes transformed to_hex here
cat /dev/random | head -c 16 | to_hex -in
bb85976f46bc1a576e141aa73268cc9a
```

##### Converts input to Base64 encoded string

Converts to Base64 Standard Encoding, use `-url` flag to convert to URL safe encoder instead.

```bash
# Inferred as hex if all characters are in the hexadecimal characters set
to_base64 abfe0102
q/4BAg==

# As hex
to_base64 -hex abfe0102
q/4BAg==

# As base58
to_base64 -b58 5PzCau
q/4BAg==

# As integer
to_base64 -i 126700
Ae7s

# As string
to_base64 -s myname
bXluYW1l

# Inferred as string if received value has double-quotes
to_base64 '"myname"'
bXluYW1l

# Arguments from standard input (each line is an element to convert)
echo "abfe0103\nbefe0104\ncde0105" | to_base64
q/4BAw==
vv4BBA==
DN4BBQ==

# Reads from standard input as bytes and convert to hexadecimal, random 16 bytes transformed to_hex here
cat /dev/random | head -c 16 | to_base64 -in
ebWxHXskW2fMMOL2QQUY8w==
```

##### Converts input to Base58 encoded string

```bash
# Infer hex is all characters
to_base58 abfe0102
5PzCau

# As hex
to_base58 -hex abfe0102
5PzCau

# As base64
to_base58 -b64 q/4BAg==
5PzCau

# As base64 URL safe encoding
to_base58 -b64u q_4BAg==
5PzCau

# As integer
to_base58 -i 126700
efV

# As string
to_base58 -s myname
wWsYQQr8

# Inferred as string if received value has double-quotes
to_base58 '"myname"'
wWsYQQr8

# Arguments from standard input (each line is an element to convert)
echo "abfe0103\nbefe0104\ncde0105" | to_base58
5PzCav
5t9xwV
L5RNL

# Reads from standard input as bytes and convert to hexadecimal, random 16 bytes transformed to_hex here
cat /dev/random | head -c 16 | to_base58 -in
5nXfEKk1UVQH2c9XXwde3g
```

##### Converts input to ISO-8601 string format

```bash
# Parse as Unix milliseconds (milliseconds inferred)
to_date 1600446733000
2020-09-18T12:32:13-04:00 (2020-09-18T16:32:13Z)

# Parse as Unix seconds (seconds inferred)
to_date 1600446733
2020-09-18T12:32:13-04:00 (2020-09-18T16:32:13Z)

# Parse as Golang date layout (multiple layouts tried one after the other)
to_date 2020-09-18T16:32:13Z
2020-09-18T12:32:13-04:00 (2020-09-18T16:32:13Z)

# Get date now (local + UTC)
to_date
2024-01-12T10:19:18-05:00 (2024-01-12T15:19:18Z)
```

##### Converts input to integer (arbitrary precision)

```bash
to_dec 01e240
123456

# Arbitrary precision
to_dec 21e19e0c9bab2400000
10000000000000000000000
```

##### Humanize bytes value

```bash
# As SI standard (base quantity is 1000) so KB, MB, etc.
bytes 100000000
100.00 MB

# As IEC standard (base quantity is 1024) so KiB, MiB, etc.
bytes -b 100000000
95.37 MiB
```

##### Converts input to duration

If the input received only contains decimal values, it's assumed to be a time unit value, time
unit is defined by one of the available flag and it we prints the humanize value of
this time duration.

Otherwise, assume it's a human duration string (loose format, multiple variants accepted)
and turn into the time unit as declared by the defined by one of the available time unit flags.

Availble flags are:

- `-ns` => Nanoseconds
- `-us` => Microseconds
- `-ms` => Milliseconds
- `-s` => Seconds
- `-m` => Minutes
- `-h` => Hours

```
# Humanized duration to time unit
to_duration -m "41h 40m 0s"
2500

# Humanized duration to time unit
to_duration -ms "45m"
2700000

# Humanized duration to time unit
to_duration -ns "10m"
600000000000

# Time unit to humanized duration
$ to_duration -h 150000
150000h 0m 0s

# Time unit to humanized duration
to_duration -s 170000000
47222h 13m 20s

# Time unit to humanized duration
to_duration -us 1500000000
1.5s
```

##### Transforms input to lower case

```
to_lower ABdg
abdg
```

##### Transforms input to upper case

```
to_upper ABdg
ABDG
```

##### Computes statistics about numbers received

```
stats 1 2 3 4 5 6 7 8 9
Count: 9
Range: Min 1.00000 - Max 9.00000
Sum: 45.00000
Average: 5.00000
Median: 5.00000
Standard Deviation: 2.73861
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
