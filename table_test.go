package table_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"go.senan.xyz/table"
)

func TestTable(t *testing.T) {
	var buff bytes.Buffer
	tbl := table.New(&buff)
	tbl.SetFormat("", " ", "")
	fmt.Fprintf(tbl, "%s\t%s\t%s\n", "", "b", "c!")
	fmt.Fprintf(tbl, "%s\t%s\t%s\n", "aaaa", "b", "c")
	tNoErr(t, tbl.Flush())
	tEq(t, tRead(t, &buff), "     b c!\naaaa b c\n")

	fmt.Fprintf(tbl, "%s\t%s\t%s\n", "a", "bbbbbbbbbbb", "c")
	fmt.Fprintf(tbl, "%s\t%s\t%s\n", "", "", "c")
	tNoErr(t, tbl.Flush())
	tEq(t, tRead(t, &buff), "a bbbbbbbbbbb c\n              c\n")

	fmt.Fprintf(tbl, "%s\t%s\t%s\n", "a", "b", "c")
	fmt.Fprintf(tbl, "%s\t%s\t%s\n", "aa", "bb", "cc")
	tNoErr(t, tbl.Flush())
	tEq(t, tRead(t, &buff), "a  b  c\naa bb cc\n") // no trailing space on first line

	tbl.SetFormat("", " ", "|")
	fmt.Fprintf(tbl, "%s\t%s\t%s\n", "a", "b", "c ")
	fmt.Fprintf(tbl, "%s\t%s\t%s\n", "aa", "bb", "cc")
	tNoErr(t, tbl.Flush())
	tEq(t, tRead(t, &buff), "a  b  c |\naa bb cc|\n")
}

func TestTableError(t *testing.T) {
	var buff bytes.Buffer
	tbl := table.New(&buff)
	tbl.SetFormat("", " ", "")
	fmt.Fprintf(tbl, "%s\t%s\t%s\n", "", "b", "c!")
	fmt.Fprintf(tbl, "%s\t%s\n", "1", "2")
	fmt.Fprintf(tbl, "%s\t%s\n", "3", "4")

	var re *table.RowError
	if !errors.As(tbl.Flush(), &re) {
		t.Fatal("didn't get row error")
	}
	tEq(t, re.Line, 2)
	tEq(t, re.Want, 3)
	tEq(t, re.Got, 2)
}

func TestFormatLines(t *testing.T) {
	lines := []string{
		fmt.Sprintf("%s\t%s\t%s\n", "a", "bbbbbbbbbbb", "c"),
		fmt.Sprintf("%s\t%s\t%s\n", "", "", "c"),
	}
	table.FormatLines(lines)
	tEq(t, lines[0], "a bbbbbbbbbbb c")
	tEq(t, lines[1], "              c")

	lines = []string{
		fmt.Sprintf("%s\t%s\t%s\n", "a", "bbbbbbbbbbb", "c"),
		"hello hello", // malformatted
	}
	table.FormatLines(lines)
	tEq(t, lines[0], "a bbbbbbbbbbb c")
	tEq(t, lines[1], "hello hello")
}

func TestFormatReader(t *testing.T) {
	var buff bytes.Buffer
	fmt.Fprintf(&buff, "%s\t%s\t%s\n", "a", "bbbbbbbbbbb", "c")
	fmt.Fprintf(&buff, "%s\t%s\t%s\n", "", "", "c")

	lines, err := table.FormatReader(&buff)
	tNoErr(t, err)

	tEq(t, lines[0], "a bbbbbbbbbbb c")
	tEq(t, lines[1], "              c")

	buff.Reset()
	fmt.Fprintf(&buff, "%s\t%s\t%s\n", "a", "bbbbbbbbbbb", "c")
	fmt.Fprintln(&buff, "hello hello")

	lines, err = table.FormatReader(&buff)
	tNoErr(t, err)

	tEq(t, lines[0], "a bbbbbbbbbbb c")
	tEq(t, lines[1], "hello hello")
}

func tNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func tEq[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if want != got {
		fmt.Println("want:")
		fmt.Println(want)
		fmt.Println("got:")
		fmt.Println(got)
		t.Fatal()
	}
}

func tRead(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
