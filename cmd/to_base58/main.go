package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/streamingfast/tooling/cli"
)

var asHexFlag = flag.Bool("hex", false, "Decodes the input as a hex representation")
var asBase64Flag = flag.Bool("b64", false, "Decodes the input as a standard base64 representation")

func main() {
	flag.Parse()

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toBase58(element))
	}
}

func toBase58(element string) string {
	if element == "" {
		return ""
	}

	var bytes []byte
	var err error
	if *asHexFlag {
		bytes, err = hex.DecodeString(element)
		cli.NoError(err, "invalid hex value %q", element)
	}

	if *asBase64Flag {
		bytes, err = base64.StdEncoding.DecodeString(element)
		cli.NoError(err, "invalid base64 value %q", element)
	}

	return base58.Encode(bytes)
}
