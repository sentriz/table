// Package table provides a Writer for formatting tab-separated data into aligned columns.
package table

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/rivo/uniseg"
)

type Buffer struct {
	formatter  formatter
	buffer     []byte
	readBuffer []byte
	readPos    int
}

func (b *Buffer) SetSeparator(sep string) {
	b.formatter.setSeperator(sep)
}

func (b *Buffer) Write(p []byte) (int, error) {
	b.readBuffer = nil
	b.readPos = 0

	if b.buffer == nil {
		b.buffer = make([]byte, 0, 1024)
	}

	b.buffer = append(b.buffer, p...)
	for {
		if i := bytes.IndexByte(b.buffer, '\n'); i >= 0 {
			line := string(b.buffer[:i])
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1] // remove \r for \r\n
			}
			b.formatter.addLine(line) // ignore errors in Write
			b.buffer = b.buffer[i+1:]
		} else {
			break
		}
	}
	return len(p), nil
}

func (b *Buffer) Lines() []string {
	b.flush()

	return b.formatter.lines()
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	if b.readBuffer == nil {
		b.flush()
		lines := b.formatter.lines()
		if len(lines) == 0 {
			return 0, io.EOF
		}
		b.readBuffer = []byte(strings.Join(lines, "\n") + "\n")
		b.readPos = 0
		// clear the formatter after generating read buffer so next write/read cycle is fresh
		b.formatter.reset()
	}

	if b.readPos >= len(b.readBuffer) {

		// read is complete, can reset for next cycle
		b.readBuffer = nil
		b.readPos = 0
		return 0, io.EOF
	}

	n = copy(p, b.readBuffer[b.readPos:])
	b.readPos += n

	if b.readPos >= len(b.readBuffer) {
		// read is complete, can reset for next cycle
		b.readBuffer = nil
		b.readPos = 0
		return n, io.EOF
	}
	return n, nil
}

func (b *Buffer) flush() {
	if len(b.buffer) > 0 {
		line := string(b.buffer)
		b.formatter.addLine(line)
		b.buffer = b.buffer[:0]
	}
}

type formatter struct {
	widths    []int
	table     [][]string
	separator string
}

func (f *formatter) setSeperator(sep string) {
	f.separator = sep
}

func (f *formatter) addLine(line string) error {
	cols := strings.Split(line, "\t")
	for i := range cols {
		cols[i] = strings.TrimSpace(cols[i])
	}
	if f.widths == nil {
		for _, p := range cols {
			f.widths = append(f.widths, strWidth(p))
		}
	}
	if len(cols) != len(f.widths) {
		return &ColumnCountError{Want: len(f.widths), Got: len(cols)}
	}
	for i := range f.widths {
		f.widths[i] = max(f.widths[i], strWidth(cols[i]))
	}
	f.table = append(f.table, cols)
	return nil
}

func (f *formatter) lines() []string {
	result := make([]string, len(f.table))
	for i, row := range f.table {
		result[i] = f.formatRow(row)
	}
	return result
}

func (f *formatter) formatRow(row []string) string {
	var sep string = " "
	if f.separator != "" {
		sep = " " + f.separator + " "
	}

	var rbuf []byte
	for i, col := range row {
		if i != 0 {
			rbuf = append(rbuf, []byte(sep)...)
		}
		rbuf = append(rbuf, []byte(col)...)
		if i < len(f.widths) {
			rbuf = append(rbuf, strings.Repeat(" ", f.widths[i]-strWidth(col))...)
		}
	}
	return string(rbuf)
}

func (f *formatter) reset() {
	f.widths = nil
	f.table = nil
}

type RowError struct {
	Line, Want, Got int
}

func (re *RowError) Error() string {
	return fmt.Sprintf("line %d: want %d cols got %d", re.Line, re.Want, re.Got)
}

type ColumnCountError struct {
	Want, Got int
}

func (ce *ColumnCountError) Error() string {
	return fmt.Sprintf("want %d cols got %d", ce.Want, ce.Got)
}

var ansiEscExpr = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func strWidth(s string) int {
	s = ansiEscExpr.ReplaceAllString(s, "")
	w := uniseg.StringWidth(s)
	return w
}

func FormatLines(lines []string) {
	if len(lines) == 0 {
		return
	}

	var f formatter
	for _, line := range lines {
		f.addLine(line)
	}

	formatted := f.lines()
	for i := range formatted {
		if i < len(lines) {
			lines[i] = formatted[i]
		}
	}
}
