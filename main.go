package main

import (
	"fmt"
	"os"

	"go.senan.xyz/table/table"
)

func main() {
	t := table.New(os.Stdout, os.Stdin)
	for t.Scan() {
	}
	if err := t.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
