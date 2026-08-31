package main

import (
	. "github.com/streamingfast/cli"
)

func main() {
	Run(
		"dockerx",
		"Docker helper commands",
		Description(`
			A collection of helper commands for working with Docker and Docker Compose.
		`),
		ComposeGroup,
	)
}
