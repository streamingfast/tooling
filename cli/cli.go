package cli

import (
	"encoding/hex"
	"fmt"
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
