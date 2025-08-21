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
	var separator, flushIntervalStr, prefix, suffix string
	parseArgs(os.Args[1:], &separator, &flushIntervalStr, &prefix, &suffix)

	flushInterval, _ := strconv.Atoi(flushIntervalStr)

	w := table.New(os.Stdout)
	w.SetFormat(prefix, separator, suffix)

	// No flush interval, just copy
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

func parseArgs(args []string, ptrs ...*string) {
	for i := 0; i < min(len(args), len(ptrs)); i++ {
		*ptrs[i] = args[i]
	}
}
