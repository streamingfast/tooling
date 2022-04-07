package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"strings"
	"unicode"

	"github.com/eoscanada/eos-go/btcsuite/btcutil/base58"
	"github.com/streamingfast/tooling/cli"
)

var asBinaryFlag = flag.Bool("in", false, "Decode the standard input as a binary representation")
var asBase58Flag = flag.Bool("b58", false, "Decode the input as a base58 representation")
var asBase64Flag = flag.Bool("b64", false, "Decode the input as a base64 representation")

func main() {
	flag.Parse()

	if *asBinaryFlag {
		cli.ProcessStandardInputBytes(16, func(bytes []byte) {
			fmt.Print(bytesToAscii(bytes))
		})
		fmt.Println()

		return
	}

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toAscii(element))
	}
}

func toAscii(element string) string {
	if element == "" {
		return ""
	}

	if *asBase58Flag {
		return bytesToAscii(base58.Decode(element))
	}

	if *asBase64Flag {
		base64Bytes, err := base64.StdEncoding.DecodeString(element)
		cli.NoError(err, "unable to decode %q as base64", element)

		return bytesToAscii(base64Bytes)
	}

	if cli.HexRegexp.MatchString(element) {
		hexBytes, err := hex.DecodeString(element)
		cli.NoError(err, "unable to decode %q as hexadecimal", element)

		return bytesToAscii(hexBytes)
	}

	return element
}

func bytesToAscii(bytes []byte) string {
	builder := strings.Builder{}

	for _, byteValue := range bytes {
		character := rune(byteValue)

		switch {
		case unicode.IsPrint(character):
			builder.WriteRune(character)

		case unicode.IsSpace(character):
			builder.WriteRune(character)

		default:
			builder.WriteString(".")
		}
	}

	return builder.String()
}
