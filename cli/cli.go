package cli

import (
	"bufio"
	"bytes"
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
