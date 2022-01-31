package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/streamingfast/tooling/cli"
)

func main() {
	flag.Parse()

	cli.Ensure(len(os.Args) == 3, "usage: sets <fileA> <fileB>")

	fileA := os.Args[1]
	fileB := os.Args[2]

	setA := readSet(fileA)
	setB := readSet(fileB)

	fmt.Printf("Elements in %q but not in %q\n", fileA, fileB)
	inANotInB := map[string]bool{}
	for elementA := range setA {
		if _, found := setB[elementA]; !found {
			inANotInB[elementA] = true
		}
	}
	printSet(inANotInB)

	fmt.Println()
	fmt.Printf("Elements in %q but not in %q\n", fileB, fileA)
	inBNotInA := map[string]bool{}
	for elementB := range setB {
		if _, found := setA[elementB]; !found {
			inBNotInA[elementB] = true
		}
	}
	printSet(inBNotInA)

	fmt.Println()
	fmt.Printf("Elements in %q and in %q\n", fileA, fileB)
	union := map[string]bool{}
	for elementA := range setA {
		if _, found := setB[elementA]; found {
			union[elementA] = true
		}
	}
	for elementB := range setB {
		if _, found := setA[elementB]; found {
			union[elementB] = true
		}
	}
	printSet(union)
}

func readSet(file string) map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(file)
	cli.NoError(err, "unable to read file")

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out[line] = true
		}
	}

	return out
}

func printSet(elements map[string]bool) {
	i := 0
	sorted := make([]string, len(elements))
	for element := range elements {
		sorted[i] = element
		i++
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	for _, element := range sorted {
		fmt.Println(element)
	}
}
