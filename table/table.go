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
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/rivo/uniseg"
)

type StringWriter struct {
	*Writer
	buff *bytes.Buffer
}

func NewStringWriter() *StringWriter {
	var buff bytes.Buffer
	return &StringWriter{Writer: NewWriter(&buff), buff: &buff}
}

func (s *StringWriter) String() string {
	s.Close()
	return s.buff.String()
}

type Writer struct {
	*Reader
	pw   *io.PipeWriter
	done chan struct{}
}

func NewWriter(out io.Writer) *Writer {
	done := make(chan struct{})
	pr, pw := io.Pipe()
	r := NewReader(out, pr)
	go func() {
		for r.Scan() {
		}
		r.Flush()
		close(done)
	}()
	return &Writer{r, pw, done}
}

func (tw *Writer) Write(p []byte) (int, error) {
	return tw.pw.Write(p)
}

func (tw *Writer) Close() error {
	err := tw.pw.Close()
	<-tw.done
	return err
}

type Reader struct {
	out    io.Writer
	sc     *bufio.Scanner
	widths []int
	table  [][]string
	errs   []error
	line   int

	Separator string
}

func NewReader(out io.Writer, in io.Reader) *Reader {
	return &Reader{out: out, sc: bufio.NewScanner(in)}
}

func (r *Reader) Scan() bool {
	if r.sc.Scan() {
		r.line++
		r.parseLine(r.sc.Text())
		return true
	}
	return false
}

func (r *Reader) parseLine(line string) {
	cols := strings.Split(line, "\t")
	for i := range cols {
		cols[i] = strings.TrimSpace(cols[i])
	}
	if r.widths == nil {
		for _, p := range cols {
			r.widths = append(r.widths, strWidth(p))
		}
	}
	if len(cols) != len(r.widths) {
		r.errs = append(r.errs, &RowError{Line: r.line, Want: len(r.widths), Got: len(cols)})
		return
	}
	for i := range r.widths {
		r.widths[i] = max(r.widths[i], strWidth(cols[i]))
	}
	r.table = append(r.table, cols)
}

func (r *Reader) Flush() {
	var sep string = " "
	if r.Separator != "" {
		sep = " " + r.Separator + " "
	}

	for _, row := range r.table {
		var rbuf []byte
		for i, col := range row {
			if i != 0 {
				rbuf = append(rbuf, []byte(sep)...)
			}
			rbuf = append(rbuf, []byte(col)...)
			rbuf = append(rbuf, strings.Repeat(" ", r.widths[i]-strWidth(col))...)
		}
		fmt.Fprintln(r.out, string(rbuf))
	}

	r.table = nil
}

func (r *Reader) Reset() error {
	r.widths = nil

	err := errors.Join(r.errs...)
	r.errs = nil
	return err
}

type RowError struct {
	Line, Want, Got int
}

func (re *RowError) Error() string {
	return fmt.Sprintf("line %d: want %d cols got %d", re.Line, re.Want, re.Got)
}

var ansiEscExpr = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func strWidth(s string) int {
	s = ansiEscExpr.ReplaceAllString(s, "")
	w := uniseg.StringWidth(s)
	return w
}
