// Package table defines a type for reading tab separated lines from the input. When Flush is called the
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
package table

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Table struct {
	out    io.Writer
	sc     *bufio.Scanner
	widths []int
	table  [][]string
	errs   []error
	line   int
}

func New(out io.Writer, in io.Reader) *Table {
	return &Table{out: out, sc: bufio.NewScanner(in)}
}

func (t *Table) Scan() bool {
	if t.sc.Scan() {
		t.line++
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
		t.errs = append(t.errs, &RowError{Line: t.line, Want: len(t.widths), Got: len(cols)})
		return
	}
	for i := range t.widths {
		t.widths[i] = max(t.widths[i], len(cols[i]))
	}
	t.table = append(t.table, cols)
}

func (t *Table) Flush() error {
	for _, row := range t.table {
		var rbuf []byte
		for i, col := range row {
			if i != 0 {
				rbuf = fmt.Append(rbuf, " ")
			}
			rbuf = fmt.Appendf(rbuf, "%-*s", t.widths[i], col)
		}
		fmt.Fprintln(t.out, string(rbuf))
	}

	t.table = nil
	t.widths = nil

	err := errors.Join(t.errs...)
	t.errs = nil
	return err
}

type RowError struct {
	Line, Want, Got int
}

func (re *RowError) Error() string {
	return fmt.Sprintf("line %d: want %d cols got %d", re.Line, re.Want, re.Got)
}