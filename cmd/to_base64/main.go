package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/tooling/cli"
)

var hexRegexp = regexp.MustCompile("[a-f0-9]{2,}")
var spacesRegexp = regexp.MustCompile("\\s")

var asStringFlag = flag.Bool("s", false, "Encode the string and not it's representation")

func main() {
	fi, err := os.Stdin.Stat()
	derr.Check("unable to stat stdin", err)

	var elements []string
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		stdin, err := ioutil.ReadAll(os.Stdin)
		cli.NoError(err, "reading from stdin")

		elements = spacesRegexp.Split(string(stdin), -1)
	} else {
		cli.Ensure(len(os.Args) >= 2, "You must provide at least one snapshot filename")
		elements = os.Args[1:]
	}

	for _, element := range elements {
		toBase64(element)
	}
}

func toBase64(element string) string {
	if hexRegexp.MatchString(element) {
		bytes, err := hex.DecodeString(element)
		cli.NoError(err, "invalid hex value %q", element)

		return base64.StdEncoding.EncodeToString(bytes)
	}

	return base64.StdEncoding.EncodeToString([]byte(element))
}
