package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/dfuse-io/tooling/cli"
)

var hexRegexp = regexp.MustCompile("[a-f0-9]{1,}")
var spacesRegexp = regexp.MustCompile("\\s")

func main() {
	fi, err := os.Stdin.Stat()
	cli.NoError(err, "unable to stat stdin")

	var elements []string
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		stdin, err := ioutil.ReadAll(os.Stdin)
		cli.NoError(err, "reading from stdin")

		elements = spacesRegexp.Split(string(stdin), -1)
	} else {
		elements = os.Args[1:]
	}

	for _, element := range elements {
		fmt.Println(toDec(element))
	}
}

func toDec(element string) string {
	element = strings.TrimPrefix(element, "0x")

	if hexRegexp.MatchString(element) {
		if len(element)%2 != 0 {
			element = "0" + element
		}

		if len(element) <= 16 {
			value, err := strconv.ParseUint(element, 16, 64)
			cli.NoError(err, "invalid number %q", element)

			return strconv.FormatUint(value, 10)
		}

		value, err := hex.DecodeString(element)
		cli.NoError(err, "invalid number %q", element)

		return new(big.Int).SetBytes(value).String()
	}

	return element
}
