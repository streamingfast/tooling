package main

import (
	"encoding/base64"
	"flag"
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/streamingfast/tooling/cli"
)

var asHexFlag = flag.Bool("hex", false, "Decode the input as an hexadecimal representation")
var asBase64Flag = flag.Bool("b64", false, "Decode the input as a standard base64 representation")
var asBase64URLFlag = flag.Bool("b64u", false, "Decode the input as URL base64 representation")
var asIntegerFlag = flag.Bool("i", false, "Decode the input as an integer representation")
var asStringFlag = flag.Bool("s", false, "Decode the string and not it's representation")

var fromStdIn = flag.Bool("in", false, "Decode the standard input as a bytes stream")

func main() {
	flag.Parse()

	if *fromStdIn {
		cli.Ensure(
			!*asHexFlag && !*asBase64Flag && !*asBase64URLFlag && !*asIntegerFlag && !*asStringFlag,
			"Flag -in is exclusive and cannot be used at the same time as any of -hex, -b64, -b64u, -i nor -s",
		)

		cli.ProcessStandardInputBytes(16, func(bytes []byte) { fmt.Print(base58.Encode(bytes)) })
		fmt.Println()

		return
	}

	scanner := cli.NewFlagArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toBase58(element))
	}
}

func toBase58(element string) string {
	if element == "" {
		return ""
	}

	if *asIntegerFlag {
		return base58.Encode(cli.ReadIntegerToBytes(element))
	}

	if *asStringFlag {
		return base58.Encode([]byte(element))
	}

	if *asBase64Flag {
		return base64valueToBase58(element, base64.StdEncoding)
	}

	if *asBase64URLFlag {
		return base64valueToBase58(element, base64.RawURLEncoding)
	}

	// If wrapped with `"`, we use the string characters has the bytes value
	if element[0] == '"' && element[len(element)-1] == '"' {
		return base58.Encode([]byte(element)[1 : len(element)-1])
	}

	if cli.HexRegexp.MatchString(element) {
		bytes, err := cli.DecodeHex(element)
		cli.NoError(err, "invalid hex value %q", element)

		return base58.Encode(bytes)
	}

	cli.Quit("Unable to infer content's actual representation, specify one of -hex (hexadecimal), -b64 (base64 std), -b64u (base64 URL), -i (integer), -s (string)")
	return ""
}

func base64valueToBase58(in string, encoding *base64.Encoding) string {
	out, err := encoding.DecodeString(in)
	cli.NoError(err, "value %q is not a valid base64 value", in)

	return base58.Encode(out)
}
