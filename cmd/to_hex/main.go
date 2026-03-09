package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/eoscanada/eos-go/btcsuite/btcutil/base58"
	"github.com/streamingfast/tooling/cli"
)

var asBase58Flag = flag.Bool("b58", false, "Decode the input as a base58 representation")
var asBase64URLFlag = flag.Bool("b64u", false, "Decode the input as URL base64 representation")
var asBase64Flag = flag.Bool("b64", false, "Decode the input as a standard base64 representation")
var asBech32Flag = flag.String("bech32", "", "Decode the input as a standard bech32 representation with the value being the human readable part")

var asIntegerFlag = flag.Bool("i", false, "Decode the input as an integer representation")
var asStringFlag = flag.Bool("s", false, "Decode the string and not it's representation")

var fromStdIn = flag.Bool("in", false, "Decode the standard input as a bytes stream")

var withOxPrefix = flag.Bool("0x", false, "Encode back hexadecimal with 0x prefix, removing extra leading zeros (0123 becomes 0x123)")
var reversedFourFlag = flag.Bool("r4", false, "Encode back hexadecimal using reverted 4 bytes number, works only when using '-i' flag")
var reversedEightFlag = flag.Bool("r", false, "Encode back hexadecimal using reverted 8 bytes number, works only when using '-i' flag")

func main() {
	flag.Parse()

	if *reversedFourFlag || *reversedEightFlag {
		cli.Ensure(*asIntegerFlag, "Flag -r4 or -r8 can only be used when input is a integer so -i must be provided")
		// cli.Ensure(!*reversedFourFlag && !*reversedEightFlag, "Only one of -r4 or -r8 can only be used at a time")
	}

	if *fromStdIn {
		cli.Ensure(
			!*asBase58Flag && !*asBase64Flag && !*asBase64URLFlag && !*asIntegerFlag && !*asStringFlag,
			"Flag -in is exclusive and cannot be used at the same time as any of -b58, -b64, -b64u, -i nor -s",
		)

		cli.ProcessStandardInputBytes(16, func(bytes []byte) { fmt.Print(cli.EncodeHex(bytes)) })
		fmt.Println()

		return
	}

	scanner := cli.NewFlagArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toHex(element))
	}
}

func toHex(element string) string {
	if element == "" {
		return ""
	}

	encodeHex := cli.EncodeHex
	if *withOxPrefix {
		encodeHex = cli.EncodeHex0xPrefix
	}

	if *asIntegerFlag {
		var bytes []byte
		if *reversedFourFlag {
			bytes = cli.ReadReversedIntegerToBytes(element, 4)
		} else if *reversedEightFlag {
			bytes = cli.ReadReversedIntegerToBytes(element, 8)
		} else {
			bytes = cli.ReadIntegerToBytes(element)
		}

		return encodeHex(bytes)
	}

	if *asStringFlag {
		return encodeHex([]byte(element))
	}

	if *asBase58Flag {
		return encodeHex(base58.Decode(element))
	}

	if *asBase64Flag {
		return base64valueToHex(element, base64.StdEncoding, encodeHex)
	}

	if *asBase64URLFlag {
		return base64valueToHex(element, base64.RawURLEncoding, encodeHex)
	}

	if cli.IsFlagSet("bech32") {
		cli.Ensure(*asBech32Flag != "", "Flag -bech32 requires a value to be provided like '-bech32=hrp' where 'hrp' is the human readable part of the bech32 value")
		return bech32ValueToHex(element, *asBech32Flag, encodeHex)
	}

	// If wrapped with `"`, we use the string characters has the bytes value
	if element[0] == '"' && element[len(element)-1] == '"' {
		return encodeHex([]byte(element)[1 : len(element)-1])
	}

	if isInteger(element) {
		bytes := cli.ReadIntegerToBytes(element)
		return encodeHex(bytes)
	}

	cli.Quit("Unable to infer content's actual representation, specify one of -b58 (base58), -b64 (base64 std), -b64u (base64 URL), -i (integer), -s (string)")
	return ""
}

func isInteger(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func base64valueToHex(in string, encoding *base64.Encoding, encodeHex func([]byte) string) string {
	out, err := encoding.DecodeString(in)
	cli.NoError(err, "value %q is not a valid base64 value", in)

	return encodeHex(out)
}

func bech32ValueToHex(in string, expectedHrp string, encodeHex func([]byte) string) string {
	hrp, data, err := bech32.Decode(in)
	cli.NoError(err, "value %q is not a valid bech32 value", in)
	cli.Ensure(hrp == expectedHrp, "value %q is a valid bech32 value but the its human readable part %q does not match the expected part %q", in, hrp, expectedHrp)

	converted, err := bech32.ConvertBits(data, 5, 8, true)
	cli.NoError(err, "unable to convert bech32 data %q from 5 bits to 8 bits", hex.EncodeToString(data))

	return encodeHex(converted)
}
