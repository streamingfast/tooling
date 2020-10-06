package main

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/dfuse-io/tooling/cli"
)

func main() {
	fi, err := os.Stdin.Stat()
	cli.NoError(err, "unable to stat stdin")

	var elements []string
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		stdin, err := ioutil.ReadAll(os.Stdin)
		cli.NoError(err, "reading from stdin")

		elements = cli.SpacesRegexp.Split(string(stdin), -1)
	} else {
		elements = os.Args[1:]
	}

	for _, element := range elements {
		fmt.Println(toDec(element))
	}
}

func toDec(element string) string {
	if cli.HexRegexp.MatchString(element) {
		value, err := cli.DecodeHex(element)
		cli.NoError(err, "invalid number %q", element)

		return new(big.Int).SetBytes(value).String()
	}

	return element
}
