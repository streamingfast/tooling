package main

import (
	"fmt"
	"strings"

	"github.com/streamingfast/tooling/cli"
)

func main() {
	scanner := cli.NewOsArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(strings.ToUpper(element))
	}
}
