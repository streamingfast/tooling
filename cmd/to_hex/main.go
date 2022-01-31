package main

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/eoscanada/eos-go/btcsuite/btcutil/base58"
	"github.com/streamingfast/tooling/cli"
)

var asBase58Flag = flag.Bool("b58", false, "Decode the input as a base58 representation")
var asBase64Flag = flag.Bool("b64", false, "Decode the input as a standard base64 representation")
var asBase64URLFlag = flag.Bool("b64u", false, "Decode the input as URL base64 representation")
var asIntegerFlag = flag.Bool("i", false, "Decode the input as an integer representation")
var asStringFlag = flag.Bool("s", false, "Decode the string and not it's representation")

var fromStdIn = flag.Bool("in", false, "Decode the standard input as a bytes stream")

func main() {
	flag.Parse()

	if *fromStdIn {
		cli.Ensure(
			!*asBase58Flag && !*asBase64Flag && !*asBase64URLFlag && !*asIntegerFlag && !*asStringFlag,
			"Flag -in is exclusive and cannot be used at the same time as any of -b58, -b64, -b64u, -i nor -s",
		)

		fromStandardInput()
		return
	}

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toHex(element))
	}
}

func fromStandardInput() {
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

func toHex(element string) string {
	if element == "" {
		return ""
	}

	if *asIntegerFlag {
		return integerValueToHex(element)
	}

	if *asStringFlag {
		return hex.EncodeToString([]byte(element))
	}

	if *asBase58Flag {
		return hex.EncodeToString(base58.Decode(element))
	}

	if *asBase64Flag {
		return base64valueToHex(element, base64.StdEncoding)
	}

	if *asBase64URLFlag {
		return base64valueToHex(element, base64.URLEncoding)
	}

	if cli.DecRegexp.MatchString(element) {
		return integerValueToHex(element)
	}

	// If wrapped with `"`, we want the hex of the string characters so AB would give 6566
	if element[0] == '"' && element[len(element)-1] == '"' {
		return hex.EncodeToString([]byte(element)[1 : len(element)-1])
	}

	cli.Quit("Unable to infer content's actual representation, specify one of -b58 (base58), -b64 (base64 std), -b64u (base64 URL), -i (integer), -s (string)")
	return ""
}

func base64valueToHex(in string, encoding *base64.Encoding) string {
	out, err := encoding.DecodeString(in)
	cli.NoError(err, "value %q is not a valid base64 value", in)

	return hex.EncodeToString(out)
}

func integerValueToHex(in string) string {
	value := new(big.Int)
	value, success := value.SetString(in, 10)
	cli.Ensure(success, "number %q is invalid", in)

	hex := hex.EncodeToString(value.Bytes())
	if len(hex)%2 == 1 {
		hex = "0" + hex
	}

	return hex
}
