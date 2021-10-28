package main

import (
	"flag"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/streamingfast/tooling/cli"
)

var humanizeFlag = flag.Bool("h", false, "Humanize the output number")
var reversedFlag = flag.Bool("r", false, "Decode assuming the input value is a reverted number")

func main() {
	flag.Parse()

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toDec(element))
	}
}

var scientificNotationRegexp = regexp.MustCompile(`^([0-9]+)?\.[0-9]+(e|E)\+[0-9]+$`)

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

		return formatNumber(bigValue)
	}

	// So we handle humanize for decimal number correctly
	if cli.DecRegexp.MatchString(element) {
		bigValue, _ := new(big.Int).SetString(element, 10)
		return formatNumber(bigValue)
	}

	if scientificNotationRegexp.MatchString(element) {
		flt, _, err := big.ParseFloat(element, 10, 0, big.ToNearestEven)
		cli.NoError(err, "invalid scientific notation %q", element)

		bigValue, _ := flt.Int(new(big.Int))
		return formatNumber(bigValue)
	}

	return element
}

func formatNumber(number *big.Int) string {
	if *humanizeFlag {
		return humanize(number)
	}

	return number.String()
}

var b1000 = big.NewInt(1000)

// humanize copied from https://github.com/dustin/go-humanize/blob/master/comma.go#L89 (MIT license)
func humanize(b *big.Int) string {
	sign := ""
	if b.Sign() < 0 {
		sign = "-"
		b.Abs(b)
	}

	c := (&big.Int{}).Set(b)
	_, m := orderOfMagnitude(c, b1000)
	parts := make([]string, m+1)
	j := len(parts) - 1

	mod := &big.Int{}
	for b.Cmp(b1000) >= 0 {
		b.DivMod(b, b1000, mod)
		parts[j] = strconv.FormatInt(mod.Int64(), 10)
		switch len(parts[j]) {
		case 2:
			parts[j] = "0" + parts[j]
		case 1:
			parts[j] = "00" + parts[j]
		}
		j--
	}

	parts[j] = strconv.Itoa(int(b.Int64()))
	return sign + strings.Join(parts[j:], " ")
}

func orderOfMagnitude(n, b *big.Int) (float64, int) {
	mag := 0
	m := &big.Int{}
	for n.Cmp(b) >= 0 {
		n.DivMod(n, b, m)
		mag++
	}
	return float64(n.Int64()) + (float64(m.Int64()) / float64(b.Int64())), mag
}
