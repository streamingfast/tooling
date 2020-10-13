package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/dfuse-io/tooling/cli"
)

var reversedFlag = flag.Bool("r", false, "Decode assuming the input value is a reverted number")

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
		fmt.Println(toDec(element))
	}
}

func toDec(element string) string {
	if cli.HexRegexp.MatchString(element) {
		value, err := cli.DecodeHex(element)
		cli.NoError(err, "invalid number %q", element)

		bigValue := new(big.Int).SetBytes(value)

		if *reversedFlag && bigValue.BitLen() > 0 {
			max := new(big.Int).Lsh(big.NewInt(1), uint(bigValue.BitLen()-1))
			for i := 0; i < bigValue.BitLen(); i++ {
				max.SetBit(max, i, 1)
			}

			bigValue = new(big.Int).Sub(max, bigValue)
		}

		return bigValue.String()
	}

	return element
}
