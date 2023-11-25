package main

import (
	"bufio"
	"fmt"
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

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	t := table.New(out, os.Stdin)
	t.Separator = separator

	defer func() {
		if err := t.Reset(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()

	if flushInterval == 0 {
		for t.Scan() {
		}
		t.Flush()
		return
	}

L:
	for {
		for i := 0; i < flushInterval; i++ {
			if !t.Scan() {
				break L
			}
		}
		t.Flush()
		out.Flush()
	}
	t.Flush()
}
