package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, "You need to provide a subcommand!\n")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "subdir":
		fmt.Printf("%s/%s/%s\n", Domain, Version, strings.ToLower(Kind))
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand %s\n", cmd)
		os.Exit(1)
	}
}
