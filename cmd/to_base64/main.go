package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/streamingfast/tooling/cli"
)

func main() {
	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toBase64(element))
	}
}

func toBase64(element string) string {
	if cli.HexRegexp.MatchString(element) {
		if strings.HasPrefix(element, "0x") {
			element = element[2:]
		}

		bytes, err := hex.DecodeString(element)

		cli.NoError(err, "invalid hex value %q", element)

		return base64.StdEncoding.EncodeToString(bytes)
	}

	return base64.StdEncoding.EncodeToString([]byte(element))
}
