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

type Writer struct {
	out            io.Writer
	buf            []byte
	pre, sep, suff string
	widths         []int
	rows           [][]string
	err            error // first error (e.g., ColumnCountError) recorded during Write
}

// New constructs a new Writer that will emit formatted rows to out on Flush/Close.
func New(out io.Writer) *Writer {
	return &Writer{
		out: out,
		buf: make([]byte, 0, 1024),
	}
}

// SetFormat controls formatting for the output including prefix, column separator, and suffix.
func (w *Writer) SetFormat(pre, sep, suff string) {
	w.pre = pre
	w.sep = sep
	w.suff = suff
}

// Write ingests bytes, splitting on '\n' (handles optional trailing '\r').
// Parsed lines are buffered; call Flush or Close to write formatted output to out.
// Column-count errors are recorded and surfaced on Flush/Close; subsequent lines are still processed.
func (w *Writer) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)

	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := string(w.buf[:i])
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1] // handle \r\n
		}
		if err := w.addLine(line); err != nil && w.err == nil {
			w.err = err
		}
		w.buf = w.buf[i+1:]
	}

	return len(p), nil
}

// Flush formats all buffered rows and writes them to out, then resets rows/widths.
// Returns the first error encountered during Write/addLine or any write error.
func (w *Writer) Flush() error {
	if len(w.rows) == 0 {
		// nothing to emit; still report any earlier error
		err := w.err
		w.reset()
		return err
	}

	var sep = " "
	if w.sep != "" {
		sep = " " + w.sep + " "
	}

	for _, row := range w.rows {
		line := formatRow(row, w.widths, w.pre, sep, w.suff)
		if _, err := io.WriteString(w.out, line+"\n"); err != nil {
			// preserve original write-time error if it existed; otherwise, set this write error
			if w.err == nil {
				w.err = err
			}
			break
		}
	}

	err := w.err
	w.reset()
	return err
}

func (w *Writer) reset() {
	w.rows = nil
	w.widths = nil
	w.err = nil
}

// addLine parses, trims, validates column count, updates widths, and buffers the row.
func (w *Writer) addLine(line string) error {
	cols := strings.Split(line, "\t")
	for i := range cols {
		cols[i] = strings.TrimSpace(cols[i])
	}

	if w.widths == nil {
		// initialize widths to number of columns in the first row
		w.widths = make([]int, len(cols))
	}

	if len(cols) != len(w.widths) {
		return &ColumnCountError{Want: len(w.widths), Got: len(cols)}
	}

	for i, c := range cols {
		if cw := strWidth(c); cw > w.widths[i] {
			w.widths[i] = cw
		}
	}
	w.rows = append(w.rows, cols)
	return nil
}

func formatRow(row []string, widths []int, pre, sep, suff string) string {
	var sb strings.Builder
	sb.WriteString(pre)

	for i, col := range row {
		if i != 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(col)
		if i < len(widths) {
			pad := widths[i] - strWidth(col)
			if pad > 0 {
				sb.WriteString(strings.Repeat(" ", pad))
			}
		}
	}

	sb.WriteString(suff)
	return sb.String()
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
	return uniseg.StringWidth(s)
}

// FormatLines formats the provided lines in-place.
// Lines with a mismatched column count (relative to the first line) are ignored,
// matching the behavior of the original code (they remain unchanged).
func FormatLines(lines []string) {
	if len(lines) == 0 {
		return
	}

	var widths []int
	rows := make([][]string, 0, len(lines))

	for _, line := range lines {
		cols := strings.Split(line, "\t")
		for i := range cols {
			cols[i] = strings.TrimSpace(cols[i])
		}

		if widths == nil {
			widths = make([]int, len(cols))
			for i, c := range cols {
				widths[i] = strWidth(c)
			}
			rows = append(rows, cols)
			continue
		}

		if len(cols) != len(widths) {
			// ignore mismatched rows (keep original line content)
			continue
		}

		for i, c := range cols {
			w := strWidth(c)
			if w > widths[i] {
				widths[i] = w
			}
		}
		rows = append(rows, cols)
	}

	if len(rows) == 0 {
		return
	}

	formatted := make([]string, len(rows))
	for i, r := range rows {
		formatted[i] = formatRow(r, widths, "", " ", "")
	}

	for i := range formatted {
		if i < len(lines) {
			lines[i] = formatted[i]
		}
	}
}
