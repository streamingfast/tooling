package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/dfuse-io/tooling/cli"
)

func main() {
	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
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
	if cli.DecRegexp.MatchString(element) {
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

	if cli.HexRegexp.MatchString(element) {
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
