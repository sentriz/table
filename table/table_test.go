package table_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"go.senan.xyz/table/table"
)

func TestTable(t *testing.T) {
	var in, out bytes.Buffer
	tbl := table.NewReader(&out, &in)

	fmt.Fprintf(&in, "%s\t%s\t%s\n", "", "b", "c!")
	fmt.Fprintf(&in, "%s\t%s\t%s\n", "aaaa", "b", "c")
	tbl.Scan()
	tbl.Scan()
	tbl.Flush()
	tNoErr(t, tbl.Reset())
	tEq(t, tRead(t, &out), "     b c!\naaaa b c \n")

	fmt.Fprintf(&in, "%s\t%s\t%s\n", "a", "bbbbbbbbbbb", "c")
	fmt.Fprintf(&in, "%s\t%s\t%s\n", "", "", "c")
	tbl.Scan()
	tbl.Scan()
	tbl.Flush()
	tNoErr(t, tbl.Reset())
	tEq(t, tRead(t, &out), "a bbbbbbbbbbb c\n              c\n")
}

func TestTableError(t *testing.T) {
	var in, out bytes.Buffer
	tbl := table.NewReader(&out, &in)

	fmt.Fprintf(&in, "%s\t%s\t%s\n", "", "b", "c!")
	fmt.Fprintf(&in, "%s\t%s\n", "1", "2")
	fmt.Fprintf(&in, "%s\t%s\n", "3", "4")
	for tbl.Scan() {
	}

	var re *table.RowError
	tbl.Flush()
	if !errors.As(tbl.Reset(), &re) {
		t.Fatal("didn't get row error")
	}
	tEq(t, re.Line, 2)
	tEq(t, re.Want, 3)
	tEq(t, re.Got, 2)

}

func tNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func tEq[T comparable](t *testing.T, a, b T) {
	t.Helper()
	if a != b {
		t.Fatalf("%v != %v", a, b)
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
