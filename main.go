package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	t := NewTable(os.Stdout, os.Stdin)
	for t.Scan() {
	}
	if err := t.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Table reads and buffers tab separated lines from the input. When Flush is called the
// lines are reformatted according to the longest column seen in the lines so far.
//
// It's also suitable for use in a long running stream by Flushing at set intervals.
//
//	t := NewTable(out, in)
//	t.Scan()
//	t.Scan()
//	t.Flush()
//	t.Scan()
//	t.Scan()
//	t.Flush()
//
// Each Flush will be formatted to the widest columns in the previous set of lines.
type Table struct {
	out    io.Writer
	sc     *bufio.Scanner
	widths []int
	table  [][]string
	err    error
}

func NewTable(out io.Writer, in io.Reader) *Table {
	return &Table{out: out, sc: bufio.NewScanner(in)}
}

func (t *Table) Scan() bool {
	if t.sc.Scan() {
		t.parseLine(t.sc.Text())
		return true
	}
	return false
}

func (t *Table) parseLine(line string) {
	cols := strings.Split(line, "\t")
	for i := range cols {
		cols[i] = strings.TrimSpace(cols[i])
	}
	if t.widths == nil {
		for _, p := range cols {
			t.widths = append(t.widths, len(p))
		}
	}
	if len(cols) != len(t.widths) {
		t.err = errors.Join(t.err, fmt.Errorf("%q: want %d cols got %d", line, len(t.widths), len(cols)))
		return
	}
	for i := range t.widths {
		t.widths[i] = max(t.widths[i], len(cols[i]))
	}
	t.table = append(t.table, cols)
}

func (t *Table) Flush() error {
	for _, row := range t.table {
		var rowbuff strings.Builder
		for i, col := range row {
			if i != 0 {
				fmt.Fprint(&rowbuff, " ")
			}
			fmt.Fprintf(&rowbuff, "%-*s", t.widths[i], col)
		}
		fmt.Fprintln(t.out, rowbuff.String())
	}

	err := t.err
	t.table = nil
	t.widths = nil
	t.err = nil
	return err
}
