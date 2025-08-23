// Package table provides a Writer that formats tab-separated data into aligned columns.
// Column widths account for ANSI escape sequences and grapheme cluster widths (via uniseg).
package table

import (
	"bufio"
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
	err            error // first error (e.g., RowError) recorded during Write
	lineNum        int
}

// New constructs a new Writer that will emit formatted rows to out on Flush.
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
// Parsed lines are buffered; call Flush to write formatted output to out.
// Column-count errors are recorded and surfaced on Flush; subsequent lines are still processed.
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
		w.lineNum++
		if err := w.addLine(line, w.lineNum); err != nil && w.err == nil {
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

	for _, row := range w.rows {
		line := formatRow(row, w.widths, w.pre, w.sep, w.suff)
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

// addLine parses, trims, validates column count, updates widths, and buffers the row.
func (w *Writer) addLine(line string, lineNum int) error {
	cols := strings.Split(line, "\t")
	for i := range cols {
		cols[i] = strings.TrimSpace(cols[i])
	}
	if w.widths == nil {
		w.widths = make([]int, len(cols))
	}
	if len(cols) != len(w.widths) {
		return &RowError{Want: len(w.widths), Got: len(cols), Line: lineNum}
	}
	for i, c := range cols {
		if cw := strWidth(c); cw > w.widths[i] {
			w.widths[i] = cw
		}
	}
	w.rows = append(w.rows, cols)
	return nil
}

func (w *Writer) reset() {
	w.rows = nil
	w.widths = nil
	w.err = nil
	w.lineNum = 0
}

func formatRow(row []string, widths []int, pre, sep, suff string) string {
	var sb strings.Builder
	sb.WriteString(pre)
	for i, col := range row {
		if i != 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(col)
		if i < len(widths)-1 {
			if pad := widths[i] - strWidth(col); pad > 0 {
				sb.WriteString(strings.Repeat(" ", pad))
			}
		}
	}
	if suff != "" {
		if pad := widths[len(widths)-1] - strWidth(row[len(row)-1]); pad > 0 {
			sb.WriteString(strings.Repeat(" ", pad))
		}
		sb.WriteString(suff)
	}
	return sb.String()
}

type RowError struct {
	Want, Got int
	Line      int
}

func (re *RowError) Error() string {
	return fmt.Sprintf("line %d, want %d cols got %d", re.Line, re.Want, re.Got)
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
	rowIndices := make([]int, 0, len(lines)) // track which line index each row corresponds to

	for i, line := range lines {
		cols := strings.Split(line, "\t")
		for j := range cols {
			cols[j] = strings.TrimSpace(cols[j])
		}

		if widths == nil {
			widths = make([]int, len(cols))
			for j, c := range cols {
				widths[j] = strWidth(c)
			}
			rows = append(rows, cols)
			rowIndices = append(rowIndices, i)
			continue
		}

		if len(cols) != len(widths) {
			// ignore mismatched rows (keep original line content)
			continue
		}

		for j, c := range cols {
			if w := strWidth(c); w > widths[j] {
				widths[j] = w
			}
		}
		rows = append(rows, cols)
		rowIndices = append(rowIndices, i)
	}

	if len(rows) == 0 {
		return
	}

	// Format and apply only the matched rows
	for i, row := range rows {
		if i < len(rowIndices) {
			lines[rowIndices[i]] = formatRow(row, widths, "", " ", "")
		}
	}
}

// FormatReader reads tab-separated data and returns formatted lines in one pass
func FormatReader(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)

	var widths []int
	var result []string
	var rows [][]string
	var rowIndices []int // track which result index each formatted row should go to

	for scanner.Scan() {
		line := scanner.Text()
		cols := strings.Split(line, "\t")

		for j := range cols {
			cols[j] = strings.TrimSpace(cols[j])
		}

		if widths == nil {
			widths = make([]int, len(cols))
			for j, c := range cols {
				widths[j] = strWidth(c)
			}
			rows = append(rows, cols)
			rowIndices = append(rowIndices, len(result))
			result = append(result, "") // placeholder
			continue
		}

		if len(cols) != len(widths) {
			// ignore mismatched rows (keep original line content)
			result = append(result, line)
			continue
		}

		for j, c := range cols {
			if w := strWidth(c); w > widths[j] {
				widths[j] = w
			}
		}
		rows = append(rows, cols)
		rowIndices = append(rowIndices, len(result))
		result = append(result, "") // placeholder
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// format and apply only the matched rows
	for i, row := range rows {
		if i < len(rowIndices) {
			result[rowIndices[i]] = formatRow(row, widths, "", " ", "")
		}
	}

	return result, nil
}

var ansiEscExpr = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func strWidth(str string) int {
	str = ansiEscExpr.ReplaceAllString(str, "")
	width := uniseg.StringWidth(str)
	return width
}
