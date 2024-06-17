package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/mr-tron/base58"
	"github.com/streamingfast/tooling/cli"
)

var asHexFlag = flag.Bool("hex", false, "Decode the input as an hexadecimal representation")
var asBase64Flag = flag.Bool("b64", false, "Decode the input as a standard base64 representation")
var asBase64URLFlag = flag.Bool("b64u", false, "Decode the input as URL base64 representation")
var asBech32Flag = flag.String("bech32", "", "Decode the input as a standard bech32 representation with the value being the human readable part")
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

		cli.ProcessStandardInputBytes(-1, func(bytes []byte) { fmt.Print(base58.Encode(bytes)) })
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

	if cli.IsFlagSet("bech32") {
		cli.Ensure(*asBech32Flag != "", "Flag -bech32 requires a value to be provided like '-bech32=hrp' where 'hrp' is the human readable part of the bech32 value")
		return bech32ValueToBase58(element, *asBech32Flag)
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

func bech32ValueToBase58(in string, expectedHrp string) string {
	hrp, data, err := bech32.Decode(in)
	cli.NoError(err, "value %q is not a valid bech32 value", in)
	cli.Ensure(hrp == expectedHrp, "value %q is a valid bech32 value but the its human readable part %q does not match the expected part %q", in, hrp, expectedHrp)

	converted, err := bech32.ConvertBits(data, 5, 8, true)
	cli.NoError(err, "unable to convert bech32 data %q from 5 bits to 8 bits", hex.EncodeToString(data))

	return base58.Encode(converted)
}

func base64valueToBase58(in string, encoding *base64.Encoding) string {
	out, err := encoding.DecodeString(in)
	cli.NoError(err, "value %q is not a valid base64 value", in)

	return base58.Encode(out)
}
