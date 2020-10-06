package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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
		fmt.Println(strings.ToLower(element))
	}
}
