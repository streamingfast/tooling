package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"regexp"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/tooling/cli"
)

var digitsRegexp = regexp.MustCompile("^[0-9]+$")
var spacesRegexp = regexp.MustCompile("\\s")

var asStringFlag = flag.Bool("s", false, "Encode the string and not it's representation")

func main() {
	fi, err := os.Stdin.Stat()
	derr.Check("unable to stat stdin", err)

	var elements []string
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		stdin, err := ioutil.ReadAll(os.Stdin)
		cli.NoError(err, "reading from stdin")

		elements = spacesRegexp.Split(string(stdin), -1)
	} else {
		elements = os.Args[1:]
	}

	for _, element := range elements {
		fmt.Println(toBigInt(element))
	}
}

func toBigInt(element string) string {
	if digitsRegexp.MatchString(element) {
		return element
	}

	base64Bytes, err := base64.StdEncoding.DecodeString(element)
	if err == nil {
		return new(big.Int).SetBytes(base64Bytes).String()
	}

	return element
}
