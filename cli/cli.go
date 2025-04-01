package cli

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeNow = time.Now

var Base64URLRegexp = regexp.MustCompile(`^[a-zA-z0-9\-_]+(=){0,2}$`)
var Base64StdRegexp = regexp.MustCompile(`^[a-zA-z0-9\+\/]+(=){0,2}$`)
var DecRegexp = regexp.MustCompile(`^[0-9]+$`)
var HexRegexp = regexp.MustCompile(`^(0(x|X))?[a-fA-F0-9]+$`)
var SpacesRegexp = regexp.MustCompile(`\s`)

var ErrNoStdin = errors.New("stdin with mode is not a char device")

func Ensure(condition bool, message string, args ...interface{}) {
	if !condition {
		NoError(fmt.Errorf(message, args...), "invalid arguments")
	}
}

func NoError(err error, message string, args ...interface{}) {
	if err != nil {
		Quit(message+": "+err.Error(), args...)
	}
}

func Quit(message string, args ...interface{}) {
	fmt.Printf(message+"\n", args...)
	os.Exit(1)
}

func End(message string, args ...interface{}) {
	fmt.Printf(message+"\n", args...)
	os.Exit(0)
}

// IsFlagSet checks if a flag is set in the [flag] package
// by using [flag.Visit] to walk over *set* flags and return
// true if one of them matches the provided flag name.
func IsFlagSet(flagName string) bool {
	isSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == flagName {
			isSet = true
		}
	})

	return isSet
}

func EncodeHex(in []byte) string {
	hex := hex.EncodeToString(in)
	if len(hex) == 0 {
		return "00"
	}

	if len(hex)%2 == 1 {
		hex = "0" + hex
	}

	return hex
}

func DecodeHex(in string) ([]byte, error) {
	out := strings.TrimPrefix(strings.ToLower(in), "0x")
	if len(out)%2 != 0 {
		out = "0" + out
	}

	return hex.DecodeString(out)
}

type ArgumentScanner interface {
	ScanArgument() (value string, ok bool)
}

func NewOsArgumentScanner() ArgumentScanner {
	return NewArgumentScanner(os.Args[1:])
}

func NewFlagArgumentScanner() ArgumentScanner {
	if flag.Parsed() {
		return NewArgumentScanner(flag.Args())
	}

	return NewOsArgumentScanner()
}

func NewArgumentScanner(args []string) ArgumentScanner {
	fi, err := os.Stdin.Stat()
	NoError(err, "unable to stat stdin")

	if (fi.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		// Let's allow token as long as 50MiB
		scanner.Buffer(nil, 50*1024*1024)

		return (*bufioArgumentScanner)(scanner)
	}

	slice := stringSliceArgumentScanner(args)
	return &slice
}

func NewStdinArgumentScanner() (ArgumentScanner, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat stdin: %w", err)
	}

	if (fi.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		// Let's allow token as long as 50MiB
		scanner.Buffer(nil, 50*1024*1024)

		return (*bufioArgumentScanner)(scanner), nil
	}

	return nil, ErrNoStdin
}

func NewFileArgumentScanner(path string) (scaner ArgumentScanner, close func() error, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open file: %w", err)
	}

	scanner := bufio.NewScanner(file)
	// Let's allow token as long as 50MiB
	scanner.Buffer(nil, 50*1024*1024)

	return (*bufioArgumentScanner)(scanner), file.Close, nil
}

type bufioArgumentScanner bufio.Scanner

func (s *bufioArgumentScanner) ScanArgument() (string, bool) {
	scanner := (*bufio.Scanner)(s)
	ok := scanner.Scan()
	if ok {
		return scanner.Text(), true
	}

	if err := scanner.Err(); err != nil {
		if err == io.EOF {
			return "", false
		}

		NoError(err, "unable to scan argument from reader")
		return "", false
	}

	return "", false
}

type stringSliceArgumentScanner []string

func (s *stringSliceArgumentScanner) ScanArgument() (string, bool) {
	slice := *s
	if len(*s) == 0 {
		return "", false
	}

	value := slice[0]
	*s = slice[1:]
	return value, true
}

// ProcessStandardInputBytes reads standard input using a buffer as big as `bufferSize`
// and pass the read bytes to `processor` function. The number of bytes received by the
// `processor` function might be lower than buffer size but will never be bigger than it.
//
// If you pass -1 to bufferSize, the whole standard input will be read and passed to the
// `processor` function in one shot.
func ProcessStandardInputBytes(bufferSize int, processor func(bytes []byte)) {
	fi, err := os.Stdin.Stat()
	NoError(err, "unable to stat stdin")
	Ensure((fi.Mode()&os.ModeCharDevice) == 0, "Standard input must be piped when from stdin mode is used")

	reader := bufio.NewReader(os.Stdin)

	full := bytes.NewBuffer(nil)

	size := bufferSize
	if size < 0 {
		size = 64 * 1024
	}

	buf := make([]byte, size)
	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			Ensure(n == 0, "Byte count should be 0 when getting EOF")
			break
		}

		NoError(err, "unable to read %d bytes stdin", size)

		if bufferSize < 0 {
			full.Write(buf[0:n])
		} else {
			processor(buf[0:n])
		}
	}

	if bufferSize < 0 {
		processor(full.Bytes())
	}
}

func ErrorUsage(usage func() string, message string, args ...interface{}) string {
	return fmt.Sprintf(message+"\n\n"+usage(), args...)
}

func SetupFlag(usage func() string) {
	flag.CommandLine.Usage = func() {
		fmt.Print(usage())
	}
	flag.Parse()
}

func FlagUsage() string {
	buf := bytes.NewBuffer(nil)
	oldOutput := flag.CommandLine.Output()
	defer func() { flag.CommandLine.SetOutput(oldOutput) }()

	flag.CommandLine.SetOutput(buf)
	flag.CommandLine.PrintDefaults()

	return buf.String()
}

func AskForConfirmation(message string, args ...interface{}) bool {
	for {
		fmt.Printf(message+" ", args...)
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			panic(fmt.Errorf("unable to read user confirmation, are you in an interactive terminal: %w", err))
		}

		response = strings.ToLower(strings.TrimSpace(response))
		for _, yesResponse := range []string{"y", "yes"} {
			if response == yesResponse {
				return true
			}
		}

		for _, noResponse := range []string{"n", "no"} {
			if response == noResponse {
				return false
			}
		}

		fmt.Println("Only Yes or No accepted, please retry!")
		fmt.Println()
	}
}

func ReadInteger(in string) *big.Int {
	value := new(big.Int)
	value, success := value.SetString(in, 10)
	Ensure(success, "number %q is invalid", in)

	return value
}

func ReadIntegerToBytes(in string) []byte {
	return ReadInteger(in).Bytes()
}

func ReadReversedInteger(in string, count int) *big.Int {
	value := new(big.Int)
	value.SetBytes(ReadReversedIntegerToBytes(in, count))

	return value
}

func ReadReversedIntegerToBytes(in string, count int) []byte {
	bytes := ReadIntegerToBytes(in)

	reversed := make([]byte, count)
	for i := count - 1; i >= 0; i-- {
		if len(bytes)-1 >= i {
			reversed[count-1-i] = 0xFF ^ bytes[len(bytes)-1-i]
		} else {
			reversed[count-1-i] = 0xFF
		}
	}

	return reversed
}

//go:generate go-enum -f=$GOFILE --marshal --names

// ENUM(
//
//	None
//	UnixSeconds
//	UnixMilliseconds
//
// )
type DateLikeHint uint

// ENUM(
//
//	Layout
//	Timestamp
//
// )
type DateParsedFrom uint

func ParseDateLikeInput(element string, hint DateLikeHint, timezoneIfUnset *time.Location) (out time.Time, parsedFrom DateParsedFrom, ok bool) {
	if element == "" {
		return out, 0, false
	}

	if element == "now" {
		return timeNow(), DateParsedFromLayout, true
	}

	if DecRegexp.MatchString(element) {
		value, _ := strconv.ParseUint(element, 10, 64)

		if hint == DateLikeHintUnixMilliseconds {
			return fromUnixMilliseconds(value), DateParsedFromTimestamp, true
		}

		if hint == DateLikeHintUnixSeconds {
			return fromUnixSeconds(value), DateParsedFromTimestamp, true
		}

		// If the value is lower than this Unix seconds timestamp representing 3000-01-01, we assume it's a Unix seconds value
		if value <= 32503683661 {
			return fromUnixSeconds(value), DateParsedFromTimestamp, true
		}

		// In all other cases, we assume it's a Unix milliseconds
		return fromUnixMilliseconds(value), DateParsedFromTimestamp, true
	}
	// Try all layouts we support
	return fromLayouts(element, timezoneIfUnset)
}

func fromLayouts(element string, timezone *time.Location) (out time.Time, parsedFrom DateParsedFrom, ok bool) {
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, element)
		if err == nil {
			// Fixe the timezone if it's offset is 0 which happens when using time
			// zone abbreviations like "CET" which are not recognized by Go. Those are ambiguous,
			// so they cannot be relied on to be parsed correctly in all cases, specific offset
			// should be used instead.
			name, offset := parsed.Zone()
			if offset == 0 {
				// Reload the location, unsure what Golang does when dealing with ambiguous
				// timezone abbreviations like MST (Mountain Standard Time or Malaysian Standard Time).
				// Our rules would be to use the closest timezone from America.
				location, err := ParseTimezone(name)
				if err != nil {
					// Panic for now, so I get aware of cases we would need to handle differently
					panic(fmt.Errorf("unable to load location %q: %w", name, err))
				}

				// Reload the time with the correct location
				parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), location)
			}

			return addMissingDateComponents(parsed), DateParsedFromLayout, true
		}
	}

	for _, layout := range localLayouts {
		parsed, err := time.Parse(layout, element)
		if err == nil {
			return adjustBackToTimezone(parsed, timezone), DateParsedFromLayout, true
		}
	}

	return
}

func addMissingDateComponents(in time.Time) time.Time {
	if in.Year() == 0 && in.Month() == 1 && in.Day() == 1 {
		now := timeNow()

		return time.Date(now.Year(), now.Month(), now.Day(), in.Hour(), in.Minute(), in.Second(), in.Nanosecond(), in.Location())
	}

	if in.Year() == 0 {
		in = in.AddDate(timeNow().Year(), 0, 0)
		return in
	}

	return in
}

func fromUnixSeconds(value uint64) time.Time {
	return time.Unix(int64(value), 0).UTC()
}

func fromUnixMilliseconds(value uint64) time.Time {
	ns := (int64(value) % 1000) * int64(time.Millisecond)

	return time.Unix(int64(value)/1000, ns).UTC()
}

func adjustBackToTimezone(in time.Time, timezone *time.Location) time.Time {
	in = addMissingDateComponents(in)

	if in.Location() == time.UTC {
		adjusted := in.In(timezone)

		_, offset := adjusted.Zone()
		if adjusted.IsDST() {
			offset -= 3600
		}

		return adjusted.Add(-1 * time.Duration(int64(offset)) * time.Second)
	}

	return in
}

// ParseTimezone returns the timezone from the provided string. If the string is empty, it returns the local timezone.
// If the string is "local", it returns the local timezone. If the string is "utc" or "z", it returns the UTC timezone.
// Otherwise, it tries to load the timezone from the provided string.
func ParseTimezone(value string) (*time.Location, error) {
	if value == "" {
		return time.UTC, nil
	}

	if strings.ToLower(value) == "local" {
		return time.Local, nil
	}

	if strings.ToLower(value) == "utc" || strings.ToLower(value) == "z" {
		return time.UTC, nil
	}

	location, err := time.LoadLocation(value)
	if err != nil {
		// Check if it's a location abbreviation we know about
		if location, found := GetTimeZoneAbbreviationLocation(value); found {
			return location, nil
		}

		return nil, fmt.Errorf("invalid timezone %q: %w", value, err)
	}

	return location, nil
}

var layouts = []string{
	// Sorted from most probably to less probably
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999-0700",
	"2006-01-02 15:04:05.999999999 -0700 UTC",
	"2006-01-02T15:04:05 MST",
	"2006-01-02T15:04:05.999999999 MST",
	time.UnixDate,
	time.RFC850,
	time.RubyDate,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC822,
	time.RFC822Z,
	time.ANSIC,
	time.Stamp,
	time.StampMilli,
	time.StampMicro,
	time.StampNano,

	// Not sure where seen
	"Mon Jan 02 2006 15:04:05 GMT-0700",

	// Not sure where seen
	"Mon Jan 02 15:04:05 2006 -0700",

	// Seen in go-ethereum release notes
	"Mon, Jan 02 at 15:04:05 UTC",

	// Seen in Lighthouse release notes
	"Mon 02 Jan 2006 15:04:05 UTC",

	// Found in `zap-pretty` output
	"2006-01-02 15:04:05.999999999 MST",

	// Found in some Telegram data reporting
	"2006-01-02 15:04:05UTC",

	// Found in BNB releases
	"2006-01-02 15:04:05 PM UTC",

	"15:04 MST",

	// Found on Sei releases notes
	"2006-01-02 15:04:05 UTC",
}

var localLayouts = []string{
	// Seen on some websites
	"Jan-02-2006 15:04:05 PM",

	// Seen on Polygon logs
	"01-02|15:04:05.999999999",

	// Variation of non-local version, see in `layouts` list
	"Mon Jan 02 2006 15:04:05",

	// Variation of non-local version, see in `layouts` list
	"Mon Jan 02 15:04:05 2006",
}
