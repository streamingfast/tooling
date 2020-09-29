package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/tooling/cli"
)

var decRegexp = regexp.MustCompile("[0-9]+")
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
		elements = os.Args[1:]
	}

	for _, element := range elements {
		fmt.Println(humanize(element))
	}
}

const (
	Bytes     int64 = 1
	KiloBytes       = 1000 * Bytes
	MegaBytes       = 1000 * KiloBytes
	GigaBytes       = 1000 * MegaBytes
	TeraBytes       = 1000 * GigaBytes
)

const (
	KibiBytes = 1024 * Bytes
	MebiBytes = 1024 * KiloBytes
	GibiBytes = 1024 * MegaBytes
	TebiBytes = 1024 * GigaBytes
)

func humanize(element string) string {
	if decRegexp.MatchString(element) {
		value, err := strconv.ParseInt(element, 10, 64)
		cli.NoError(err, "invalid dec value %q", element)

		if value > TeraBytes {
			return format(value, TeraBytes, TebiBytes, "TB")
		}

		if value > GigaBytes {
			return format(value, GigaBytes, GibiBytes, "GB")
		}

		if value > MegaBytes {
			return format(value, MegaBytes, MebiBytes, "MB")
		}

		if value > KiloBytes {
			return format(value, KiloBytes, KibiBytes, "KB")
		}

		return format(value, Bytes, Bytes, "bytes")
	}

	if hexRegexp.MatchString(element) {
		bytes, err := hex.DecodeString(element)
		cli.NoError(err, "invalid hex value %q", element)

		return base64.StdEncoding.EncodeToString(bytes)
	}

	return base64.StdEncoding.EncodeToString([]byte(element))
}

func format(value int64, decimalConversion int64, binaryConversion int64, unit string) string {
	converted := float64(value) / float64(decimalConversion)

	return fmt.Sprintf("%.2f %s", converted, unit)
}
