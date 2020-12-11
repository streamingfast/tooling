package main

import (
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/dfuse-io/tooling/cli"
)

func main() {
	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
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
