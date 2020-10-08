package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/dfuse-io/tooling/cli"
)

var asStringFlag = flag.Bool("s", false, "Encode the string and not it's representation")

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
		return hex.EncodeToString([]byte(element)[1 : len(element)-1])
	}

	if cli.DecRegexp.MatchString(element) {
		number, _ := strconv.ParseInt(element, 10, 64)
		hex := strconv.FormatInt(number, 16)
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
