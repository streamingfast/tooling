package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streamingfast/cli"
	. "github.com/streamingfast/cli"
	toolingcli "github.com/streamingfast/tooling/cli"
	"golang.org/x/exp/slices"
)

func main() {
	Run(
		"colmap -f <spec> [-d <delimiter>] <program> <args> {}",
		"Column mapper, works like 'cut' but column is mapped invoking <invoke>",
		Description(`
			This tool can be used to transform an output of the form:

			  1 john doe
			  2 jane doe

			To

			  1 JOHN doe
			  2 JANE doe

			Mapping column(s) through the external program. The <program> by default
			is invoked for each row and receives all matched column(s) as argument(s).

			The <spec> for column selection is the same as 'cut' so a single number,
			a range '1:2'.

			The {} can be used in the invoked program to exactly place the arguments.
		`),
		Example(`
			# Make upper case the second column, each column is delimited by ' '
			colmap -d ' ' -f 2 to_upper
		`),
		MinimumNArgs(3),
		PersistentFlags(func(flags *pflag.FlagSet) {
			flags.StringP("filter", "f", "", "Column filter specification, a single column or a range, multiple can be specified by separating with commas")
			flags.StringP("delimiter", "d", " ", "Column delimiter to determine how to split the row in columns")
		}),
		BeforeAllHook(func(cmd *cobra.Command) {
			cmd.DisableFlagParsing = true
		}),
		Execute(func(cmd *cobra.Command, args []string) error {
			parsed, showUsage, err := parseFlagsAndArguments(args)
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println()
				cmd.Usage()
				cli.Exit(1)
			}

			if showUsage {
				cmd.Usage()
				cli.Exit(1)
			}

			scanner, err := toolingcli.NewStdinArgumentScanner()
			NoError(err, "unable to create 'stdin' scanner")

			for line, ok := scanner.ScanArgument(); ok; line, ok = scanner.ScanArgument() {
				columns := strings.Split(line, parsed.Delimiter)
				selected, err := parsed.ColumnSelector.Select(columns)
				cli.NoError(err, "Unable to select column(s) from line")

				mapped, err := mapColumns(parsed.Command, parsed.Arguments, selected)
				cli.NoError(err, "Unable to map column(s) from line")

				replaced, err := parsed.ColumnSelector.Replace(columns, mapped)
				cli.NoError(err, "Unable to replace column(s) from line")

				fmt.Println(strings.Join(replaced, parsed.Delimiter))
			}

			return nil
		}),
	)
}

type ColumnFilter []int

func (f ColumnFilter) Select(columns []string) (selection []string, err error) {
	seen := map[int]bool{}
	for _, columnIndex := range f {
		if seen[columnIndex] {
			continue
		}

		if columnIndex >= len(columns) {
			return nil, fmt.Errorf("column index %d is out of bounds, got only %d columns", columnIndex, len(columns))
		}

		selection = append(selection, columns[columnIndex])
	}

	return
}

func (f ColumnFilter) Replace(columns []string, mapped []string) (replaced []string, err error) {
	mapping := map[int]string{}
	for i, columnIndex := range f {
		if mapping[columnIndex] != "" {
			continue
		}

		if i >= len(mapped) {
			return nil, fmt.Errorf("column index at position %d not found in mapped column of length %d", i, len(mapped))
		}

		mapping[columnIndex] = mapped[i]
	}

	replaced = make([]string, len(columns))
	for i, column := range columns {
		mapped, found := mapping[i]
		if found {
			replaced[i] = mapped
		} else {
			replaced[i] = column
		}
	}

	return
}

type CLI struct {
	ColumnSelector ColumnFilter
	Delimiter      string
	Command        string
	Arguments      []string
}

func parseFlagsAndArguments(args []string) (parsed *CLI, usage bool, err error) {
	parsed = &CLI{
		Delimiter: " ",
	}

	argumentCount := len(args)

	for i := 0; i < argumentCount; i++ {
		arg := args[i]

		switch arg {
		case "-h", "--help":
			return nil, true, nil

		case "-f", "--filter":
			if i+1 >= argumentCount {
				return nil, false, fmt.Errorf(`flag "%s <spec>", <spec> element is missing`, arg)
			}

			parsed.ColumnSelector, err = parseColumnFilter(args[i+1])
			if err != nil {
				return nil, false, fmt.Errorf(`flag "%s <spec>", invalid <spec>: %w`, arg, err)
			}

			i = i + 1

		case "-d", "--delimiter":
			if i+1 >= argumentCount {
				return nil, false, fmt.Errorf(`flag "%s <delimiter>", <delimiter> element is missing`, arg)
			}

			parsed.Delimiter = args[i+1]
			i = i + 1

		default:
			if parsed.Command == "" {
				parsed.Command = arg
			} else {
				parsed.Arguments = append(parsed.Arguments, arg)
			}
		}
	}

	if parsed.Command == "" {
		return nil, false, fmt.Errorf("The <program> argument is mandatory")
	}

	return
}

func parseColumnFilter(in string) (out ColumnFilter, err error) {
	parts := strings.Split(in, ",")
	for _, part := range parts {
		columns, err := parseColumnFilterElement(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("spec %q invalid: %w", part, err)
		}

		out = append(out, columns...)
	}

	slices.Sort(out)
	return
}

func parseColumnFilterElement(in string) (out ColumnFilter, err error) {
	before, after, isRange := strings.Cut(in, ":")

	leftElement, err := strconv.ParseInt(before, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid left element in %q: %w", in, err)
	}

	if !isRange {
		return ColumnFilter{int(leftElement)}, nil
	}

	rightElement, err := strconv.ParseInt(after, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid right element in %q: %w", in, err)
	}

	if rightElement < leftElement {
		return nil, fmt.Errorf("invalid range: element %q is lower than element %q", after, before)

	}

	if leftElement == rightElement {
		return ColumnFilter{int(leftElement)}, nil
	}

	for i := leftElement; i <= rightElement; i++ {
		out = append(out, int(i))
	}

	return out, nil
}

func mapColumns(command string, arguments []string, selected []string) (out []string, err error) {
	templated := false
	finalArguments := make([]string, 0, len(arguments)+len(selected))
	for _, argument := range arguments {
		if argument == "{}" {
			templated = true
			finalArguments = append(finalArguments, selected...)
		} else {
			finalArguments = append(finalArguments, argument)
		}
	}

	if !templated {
		finalArguments = append(finalArguments, selected...)
	}

	cmd := exec.Command(command, finalArguments...)
	output, err := cmd.CombinedOutput()
	cli.NoError(err, "Unable to invoke %q successfully", cmd)

	for _, line := range strings.Split(string(output), "\n") {
		if line := strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}

	return
}
