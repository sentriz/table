package main

import (
	"fmt"
	"os"

	"go.senan.xyz/table/table"
)

func main() {
	var separator string
	if len(os.Args) > 1 {
		separator = os.Args[1]
	}

	t := table.New(os.Stdout, os.Stdin)
	t.Separator = separator

	for t.Scan() {
	}
	if err := t.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
