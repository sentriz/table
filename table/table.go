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
	"regexp"
	"strings"

	"github.com/rivo/uniseg"
)

type TableWriter struct {
	*Table
	pw *io.PipeWriter
}

func (tw *TableWriter) Write(p []byte) (int, error) {
	return tw.pw.Write(p)
}

func (tw *TableWriter) Close() error {
	tw.Flush()
	return tw.pw.Close()
}

func NewWriter(out io.Writer) *TableWriter {
	pr, pw := io.Pipe()
	table := New(out, pr)
	go func() {
		for table.Scan() {
		}
		pr.Close()
	}()
	return &TableWriter{table, pw}
}

type Table struct {
	out    io.Writer
	sc     *bufio.Scanner
	widths []int
	table  [][]string
	errs   []error
	line   int

	Separator string
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
			t.widths = append(t.widths, width(p))
		}
	}
	if len(cols) != len(t.widths) {
		t.errs = append(t.errs, &RowError{Line: t.line, Want: len(t.widths), Got: len(cols)})
		return
	}
	for i := range t.widths {
		t.widths[i] = max(t.widths[i], width(cols[i]))
	}
	t.table = append(t.table, cols)
}

func (t *Table) Flush() {
	var sep string = " "
	if t.Separator != "" {
		sep = " " + t.Separator + " "
	}

	for _, row := range t.table {
		var rbuf []byte
		for i, col := range row {
			if i != 0 {
				rbuf = append(rbuf, []byte(sep)...)
			}
			rbuf = append(rbuf, []byte(col)...)
			rbuf = append(rbuf, strings.Repeat(" ", t.widths[i]-width(col))...)
		}
		fmt.Fprintln(t.out, string(rbuf))
	}

	t.table = nil
}

func (t *Table) Reset() error {
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

var ansiEscExpr = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func width(s string) int {
	s = ansiEscExpr.ReplaceAllString(s, "")
	w := uniseg.StringWidth(s)
	return w
}
