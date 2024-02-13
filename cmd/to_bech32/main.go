package main

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/mr-tron/base58"
	"github.com/streamingfast/tooling/cli"
)

var asHexFlag = flag.Bool("hex", false, "Decode the input as an hexadecimal representation")
var asBase58Flag = flag.Bool("b58", false, "Decode the input as a standard base58 representation")
var asIntegerFlag = flag.Bool("i", false, "Decode the input as an integer representation")
var asStringFlag = flag.Bool("s", false, "Decode the string and not it's representation")
var fromStdIn = flag.Bool("in", false, "Decode the standard input as a bytes stream")
var toUrlFlag = flag.Bool("url", false, "If true, used base64 URL encoder instead of the standard non-URL safe one")
var hrpFlag = flag.String("hrp", "sei", "The human-readable part that should be appended at the front of the address")

func main() {
	flag.Parse()

	if *fromStdIn {
		cli.Ensure(
			!*asHexFlag && !*asBase58Flag && !*asIntegerFlag && !*asStringFlag,
			"Flag -in is exclusive and cannot be used at the same time as any of -hex, -b58, -i nor -s",
		)

		cli.ProcessStandardInputBytes(-1, func(bytes []byte) { fmt.Print(bech32Encode(bytes)) })
		fmt.Println()

		return
	}

	scanner := cli.NewFlagArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toBech32(element))
	}
}

func toBech32(element string) string {
	if element == "" {
		return ""
	}

	if *asIntegerFlag {
		return bech32Encode(cli.ReadIntegerToBytes(element))
	}

	if *asStringFlag {
		return bech32Encode([]byte(element))
	}

	if *asBase58Flag {
		return base58valueToBech32(element)
	}

	// If wrapped with `"`, we use the string characters has the bytes value
	if element[0] == '"' && element[len(element)-1] == '"' {
		return bech32Encode([]byte(element)[1 : len(element)-1])
	}

	if cli.HexRegexp.MatchString(element) {
		bytes, err := cli.DecodeHex(element)
		cli.NoError(err, "invalid hex value %q", element)

		return bech32Encode(bytes)
	}

	cli.Quit("Unable to infer content's actual representation, specify one of -hex (hexadecimal), -b58 (base58), -i (integer), -s (string)")
	return ""
}

func base58valueToBech32(in string) string {
	out, err := base58.Decode(in)
	cli.NoError(err, "value %q is not a valid base58 value", in)

	return bech32Encode(out)
}

func bech32Encode(in []byte) string {
	data, err := bech32.ConvertBits(in, 8, 5, true)
	cli.NoError(err, "unable to convert bits")

	out, err := bech32.Encode(*hrpFlag, data)
	cli.NoError(err, "value %q is not a valid bech32 value", hex.EncodeToString(data))

	return out
}
