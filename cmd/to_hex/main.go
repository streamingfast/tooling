package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/dfuse-io/tooling/cli"
	"github.com/eoscanada/eos-go/btcsuite/btcutil/base58"
)

var asStringFlag = flag.Bool("s", false, "Decode the string and not it's representation")
var asBase58Flag = flag.Bool("b58", false, "Decode the input as a base58 representation")

func main() {
	flag.Parse()

	fi, err := os.Stdin.Stat()
	cli.NoError(err, "unable to stat stdin")

	var elements []string
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		stdin, err := ioutil.ReadAll(os.Stdin)
		cli.NoError(err, "reading from stdin")

		elements = cli.SpacesRegexp.Split(string(stdin), -1)
	} else {
		elements = flag.Args()
	}

	for _, element := range elements {
		fmt.Println(toHex(element))
	}
}

func toHex(element string) string {
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
