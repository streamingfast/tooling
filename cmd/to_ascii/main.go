package main

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
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
		fromBinary()
		return
	}

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toAscii(element))
	}
}

func fromBinary() {
	fi, err := os.Stdin.Stat()
	cli.NoError(err, "unable to stat stdin")
	cli.Ensure((fi.Mode()&os.ModeCharDevice) == 0, "Standard input must be piped when from stdin when using -bin")

	reader := bufio.NewReader(os.Stdin)

	buf := make([]byte, 16)
	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			cli.Ensure(n == 0, "Byte count should be 0 when getting EOF")
			break
		}

		cli.NoError(err, "unable to read 16 bytes stdin")
		fmt.Print(hex.EncodeToString(buf[0:n]))
	}

	fmt.Println()
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
