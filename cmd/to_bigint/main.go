package main

import (
	"encoding/base64"
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
		fmt.Println(toBigInt(element))
	}
}

func toBigInt(element string) string {
	if cli.DecRegexp.MatchString(element) {
		return element
	}

	if cli.HexRegexp.MatchString(element) {
		out, err := cli.DecodeHex(element)
		cli.NoError(err, "invalid hex")

		return new(big.Int).SetBytes(out).String()
	}

	base64Bytes, err := base64.StdEncoding.DecodeString(element)
	if err == nil {
		return new(big.Int).SetBytes(base64Bytes).String()
	}

	return element
}
