package cli

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var DecRegexp = regexp.MustCompile("^[0-9]+$")
var HexRegexp = regexp.MustCompile("^(0(x|X))?[a-fA-F0-9]+$")
var SpacesRegexp = regexp.MustCompile("\\s")

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

func NewArgumentScanner() ArgumentScanner {
	fi, err := os.Stdin.Stat()
	NoError(err, "unable to stat stdin")

	if (fi.Mode() & os.ModeCharDevice) == 0 {
		return (*bufioArgumentScanner)(bufio.NewScanner(os.Stdin))
	}

	args := os.Args[1:]
	if flag.Parsed() {
		args = flag.Args()
	}

	slice := stringSliceArgumentScanner(args)
	return &slice
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
