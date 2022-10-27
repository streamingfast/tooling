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

	setA, _, fileADuplicateCount := readSet(fileA)
	setB, _, fileBDuplicateCount := readSet(fileB)

	// In A but not in B

	inANotInB := map[string]bool{}
	for elementA := range setA {
		if _, found := setB[elementA]; !found {
			inANotInB[elementA] = true
		}
	}

	if len(inANotInB) == 0 {
		fmt.Println(writeHeader("All elements", "from", fileA, fileADuplicateCount, "are also contained", "in", fileB, fileBDuplicateCount))
	} else {
		fmt.Println(writeHeader("Elements", "in", fileA, fileADuplicateCount, "but", "not in", fileB, fileBDuplicateCount))
		printSet(inANotInB)
	}

	// In B but not in A

	inBNotInA := map[string]bool{}
	for elementB := range setB {
		if _, found := setA[elementB]; !found {
			inBNotInA[elementB] = true
		}
	}

	fmt.Println()
	if len(inBNotInA) == 0 {
		fmt.Println(writeHeader("All elements", "from", fileB, fileBDuplicateCount, "are also contained", "in", fileA, fileADuplicateCount))
	} else {
		fmt.Println(writeHeader("Elements", "in", fileB, fileBDuplicateCount, "but", "not in", fileA, fileADuplicateCount))
		printSet(inBNotInA)
	}

	// Union

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

	fmt.Println()
	if len(union) == 0 {
		fmt.Println(writeHeader("No elements in common", "in", fileA, fileADuplicateCount, "and", "in", fileB, fileBDuplicateCount))
	} else {
		fmt.Println(writeHeader("Elements", "in", fileA, fileADuplicateCount, "and", "in", fileB, fileBDuplicateCount))
		printSet(union)
	}
}

func writeHeader(prefix string, leftIn string, left string, leftDuplicateCount uint64, operator string, rightIn string, right string, rightDuplicateCount uint64, suffixes ...string) string {
	header := strings.Builder{}
	header.WriteString(prefix)

	header.WriteString(" " + leftIn + " ")
	header.WriteString(`"` + left + `"`)
	if leftDuplicateCount > 0 {
		header.WriteString(fmt.Sprintf(" (set contained %d duplicates)", leftDuplicateCount))
	}

	header.WriteString(" " + operator)

	header.WriteString(" " + rightIn + " ")
	header.WriteString(`"` + right + `"`)
	if rightDuplicateCount > 0 {
		header.WriteString(fmt.Sprintf(" (set contained %d duplicates)", rightDuplicateCount))
	}

	for _, suffix := range suffixes {
		header.WriteString(" " + suffix)
	}

	return header.String()
}

// This was not working as expected
// func normalizePath(in string, side string) string {
// 	if strings.HasPrefix(in, "/dev/fd/") {
// 		// FIXME: Specially deal with /dev/fd/1 (stdin)?
// 		if side == "left" {
// 			return "<(Left)"
// 		} else if side == "right" {
// 			return "<(Right)"
// 		}
// 	}

// 	return in
// }

func readSet(file string) (set map[string]bool, duplicates map[string]uint64, duplicateCount uint64) {
	set = map[string]bool{}
	data, err := os.ReadFile(file)
	cli.NoError(err, "unable to read file")

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			existsAlready := set[line]
			if existsAlready {
				if duplicates == nil {
					duplicates = make(map[string]uint64)
				}

				duplicates[line] = duplicates[line] + 1
				duplicateCount++
			}

			set[line] = true
		}
	}

	return
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
