package catalog

import (
	"math"
	"math/big"
	"reflect"
	"testing"
)

// Verdicts recorded from Python 3.11 tomllib.loads; the full accept/reject corpus lives with the parity harness.

func TestDecodeTOMLAcceptsWhatTomllibAccepts(t *testing.T) {
	for _, text := range []string{
		"a.b = 1\na.c = 2\n",
		"[a]\nb = 1\n[a.c]\nd = 2\n",
		"[[x]]\na = {}\n[[x]]\na.b = 1\n",
		"[a.b]\n[a]\nc = 1\n",
		"d = 1985-06-18 17:04:07+23:59\n",
		"t = { a = 1, b = { c = 2 } }\n",
		"s = \"\"\"x\"\"\"\"\"\nk = 'v'\n",
	} {
		if _, err := DecodeTOML(text); err != nil {
			t.Errorf("DecodeTOML(%q) = %v, want accepted", text, err)
		}
	}
}

func TestDecodeTOMLRejectsWhatTomllibRejects(t *testing.T) {
	for _, text := range []string{
		"arr = [1]\narr = [2]\n",
		"[fruit]\napple.color = \"red\"\n[fruit.apple]\n",
		"a.b.c = 1\na.b = 2\n",
		"a = {}\n[a.b]\n",
		"tab = { inner = { dog = \"best\" }, inner.cat = \"worst\" }\n",
		"[a.b.c]\nz = 9\n[a]\nb.c.t = 1\n",
		"d = 1985-06-18 17:04:07+12:60\n",
		"t = 12:30\n",
		"s = \"\\e\"\n",
		"t = { a = 1, }\n",
	} {
		if _, err := DecodeTOML(text); err == nil {
			t.Errorf("DecodeTOML(%q) accepted, want rejected", text)
		}
	}
}

func TestDecodeTOMLGivesArraysOfTablesOneShape(t *testing.T) {
	doc, err := DecodeTOML("[[a]]\nx = 1\n")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"a": []any{map[string]any{"x": int64(1)}}}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("DecodeTOML = %#v, want %#v", doc, want)
	}
}

// Python ints have no bound and a float literal past float64 reads as infinity; values recorded from tomllib.loads.
func TestDecodeTOMLReadsNumbersOutsideInt64AndFloat64(t *testing.T) {
	bigInt := func(digits string) *big.Int {
		value, _ := new(big.Int).SetString(digits, 10)
		return value
	}
	doc, err := DecodeTOML("a = 9223372036854775808\n" +
		"b = -9223372036854775809\n" +
		"c = 0xffffffffffffffffff\n" +
		"d = 0o7777777777777777777777\n" +
		"e = 0b11111111111111111111111111111111111111111111111111111111111111111\n" +
		"f = 9_223_372_036_854_775_808\n" +
		"g = [1, +99999999999999999999, { h = 1e309 }] # comment\n" +
		"i = -1e400\n" +
		"j = 1e-400\n" +
		"k = 9223372036854775807\n" +
		"s = \"\\u0000#0\"\n")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"a": bigInt("9223372036854775808"),
		"b": bigInt("-9223372036854775809"),
		"c": bigInt("4722366482869645213695"),
		"d": bigInt("73786976294838206463"),
		"e": bigInt("36893488147419103231"),
		"f": bigInt("9223372036854775808"),
		"g": []any{int64(1), bigInt("99999999999999999999"), map[string]any{"h": math.Inf(1)}},
		"i": math.Inf(-1),
		"j": 0.0,
		"k": int64(9223372036854775807),
		"s": "\x00#0",
	}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("DecodeTOML = %#v, want %#v", doc, want)
	}
}

func TestDecodeTOMLStillRejectsAnOutOfRangeNumberInAnInvalidDocument(t *testing.T) {
	for _, text := range []string{
		"a = 99999999999999999999\na = 1\n",
		"a = 99999999999999999999_\n",
		"a = 1e1000_\n",
	} {
		if _, err := DecodeTOML(text); err == nil {
			t.Errorf("DecodeTOML(%q) accepted, want rejected", text)
		}
	}
}
