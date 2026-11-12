package catalog

import (
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
