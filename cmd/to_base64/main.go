package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dfuse-io/tooling/cli"
)

func main() {
	fi, err := os.Stdin.Stat()
	cli.NoError(err, "unable to stat stdin")

	var elements []string
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		stdin, err := ioutil.ReadAll(os.Stdin)
		cli.NoError(err, "reading from stdin")

		elements = cli.SpacesRegexp.Split(string(stdin), -1)
	} else {
		elements = os.Args[1:]
	}

	for _, element := range elements {
		fmt.Println(toBase64(element))
	}
}

func toBase64(element string) string {
	if cli.HexRegexp.MatchString(element) {
		bytes, err := hex.DecodeString(element)
		cli.NoError(err, "invalid hex value %q", element)

		return base64.StdEncoding.EncodeToString(bytes)
	}

	return base64.StdEncoding.EncodeToString([]byte(element))
}
