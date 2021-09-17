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

var asStringFlag = flag.Bool("s", false, "Decode the string and not it's representation")
var asBase58Flag = flag.Bool("b58", false, "Decode the input as a base58 representation")
var asBinaryFlag = flag.Bool("in", false, "Decode the standard input as a bytes stream")

func main() {
	flag.Parse()

	if *asBinaryFlag {
		fromBinary()
		return
	}

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toHex(element))
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

func toHex(element string) string {
	if element == "" {
		return ""
	}

	if *asStringFlag {
		return hex.EncodeToString([]byte(element))
	}

	if *asBase58Flag {
		return hex.EncodeToString(base58.Decode(element))
	}

	if cli.DecRegexp.MatchString(element) {
		value := new(big.Int)
		value, success := value.SetString(element, 10)
		cli.Ensure(success, "number %q is invalid", element)

		hex := hex.EncodeToString(value.Bytes())
		if len(hex)%2 == 1 {
			hex = "0" + hex
		}

		return hex
	}

	// If wrapped with `"`, we want the hex of the string characters so AB would give 6566
	if element[0] == '"' && element[len(element)-1] == '"' {
		return hex.EncodeToString([]byte(element)[1 : len(element)-1])
	}

	base64Bytes, err := base64.StdEncoding.DecodeString(element)
	if err == nil {
		return hex.EncodeToString(base64Bytes)
	}

	return element
}
