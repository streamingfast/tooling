package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/tooling/cli"
)

var digitsRegexp = regexp.MustCompile("^[0-9]+$")
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
		elements = os.Args[1:]
	}

	for _, element := range elements {
		fmt.Println(strings.ToUpper(element))
	}
}
