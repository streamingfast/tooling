package main

import (
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/tooling/cli"
	"github.com/mr-tron/base58"
)

func main() {
	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toBase58(element))
	}
}

func toBase58(element string) string {
	if cli.HexRegexp.MatchString(element) {
		bytes, err := hex.DecodeString(element)
		cli.NoError(err, "invalid hex value %q", element)

		return base58.Encode(bytes)
	}

	return base58.Encode([]byte(element))
}
