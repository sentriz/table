package main

import (
	"io"
	"os"
	"strconv"

	"go.senan.xyz/table/table"
)

// $ stream | table [ sep ]
// $ stream | table [ sep [ flush interval ] ]

func main() {
	var separator string
	var flushInterval int

	args := os.Args[1:]
	if len(args) > 0 {
		separator = args[0]
		args = args[1:]
	}
	if len(args) > 0 {
		if i, _ := strconv.Atoi(args[0]); i > 0 {
			flushInterval = i
		}
	}

	// TODO: use
	_ = flushInterval

	var b table.Buffer
	b.SetSeparator(separator)

	io.CopyN(&b, os.Stdin, 1000)
	io.CopyN(os.Stdout, &b, 1000)
}
