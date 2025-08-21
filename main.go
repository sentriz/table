package main

import (
	"bufio"
	"io"
	"os"
	"strconv"

	"go.senan.xyz/table/table"
)

// $ stream | table [ <separator> [ <flush interval> [ <prefix> [ <suffix> ] ] ] ]

func main() {
	var prefix, separator, suffix = "", " ", ""
	var flushInterval int

	if n := 1; len(os.Args) > n {
		separator = os.Args[n]
	}
	if n := 2; len(os.Args) > n {
		flushInterval, _ = strconv.Atoi(os.Args[n])
	}
	if n := 3; len(os.Args) > n {
		prefix = os.Args[n]
	}
	if n := 4; len(os.Args) > n {
		suffix = os.Args[n]
	}

	w := table.New(os.Stdout)
	w.SetFormat(prefix, separator, suffix)

	// no flush interval, just copy
	if flushInterval == 0 {
		if _, err := io.Copy(w, os.Stdin); err != nil {
			panic(err)
		}
		if err := w.Flush(); err != nil {
			panic(err)
		}
		return
	}

	sc := bufio.NewScanner(os.Stdin)
L:
	for {
		for i := 0; i < flushInterval; i++ {
			if !sc.Scan() {
				break L
			}
			if _, err := w.Write(sc.Bytes()); err != nil {
				panic(err)
			}
			if _, err := w.Write([]byte{'\n'}); err != nil {
				panic(err)
			}
		}

		if err := w.Flush(); err != nil {
			panic(err)
		}
	}

	if err := sc.Err(); err != nil {
		panic(err)
	}
	if err := w.Flush(); err != nil {
		panic(err)
	}
}
