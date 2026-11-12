package pycompat

import (
	"math"
	"testing"
)

func TestLoadsAndDumps(t *testing.T) {
	for _, tc := range jsonCases {
		value, err := Loads(tc.doc)
		if tc.err != "" {
			if err == nil {
				t.Errorf("Loads(%q) accepted it, want error %q", tc.doc, tc.err)
			} else if err.Error() != tc.err {
				t.Errorf("Loads(%q) error = %q, want %q", tc.doc, err, tc.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("Loads(%q) error = %v", tc.doc, err)
			continue
		}
		if got := Dumps(value); got != tc.ascii {
			t.Errorf("Dumps(Loads(%q)) = %q, want %q", tc.doc, got, tc.ascii)
		}
		if got := DumpsUnicode(value); got != tc.unicode {
			t.Errorf("DumpsUnicode(Loads(%q)) = %q, want %q", tc.doc, got, tc.unicode)
		}
	}
}

func TestObjectKeepsKeyOrder(t *testing.T) {
	value, err := Loads(`{"b": 1, "a": {"x": []}, "c": "é"}`)
	if err != nil {
		t.Fatal(err)
	}
	data := value.(*Object)
	data.Set("a", "changed")
	theme := NewObject()
	theme.Set("dark", "sf2-ryu")
	theme.Set("light", "sf2-ryu-light")
	data.Set("z", theme)
	data.Set("b", []any{1, 2.5, nil, false})
	if got := Dumps(data); got != objectEditWant {
		t.Errorf("Dumps = %q, want %q", got, objectEditWant)
	}
	if got, ok := data.Get("a"); !ok || got != "changed" {
		t.Errorf(`Get("a") = %v, %v`, got, ok)
	}
	if _, ok := data.Get("missing"); ok {
		t.Error(`Get("missing") reported a value`)
	}
}

func TestDumpsEscapesNonASCIIOnlyWhenAsked(t *testing.T) {
	obj := NewObject()
	obj.Set("name", "é <>&/")
	obj.Set("empty", NewObject())
	obj.Set("none", []any{})
	if got, want := Dumps(obj), "{\n  \"name\": \"\\u00e9 <>&/\",\n  \"empty\": {},\n  \"none\": []\n}"; got != want {
		t.Errorf("Dumps = %q, want %q", got, want)
	}
	if got, want := DumpsUnicode(obj), "{\n  \"name\": \"é <>&/\",\n  \"empty\": {},\n  \"none\": []\n}"; got != want {
		t.Errorf("DumpsUnicode = %q, want %q", got, want)
	}
}

func TestLoadsKeepsBigIntegersAndFloatValues(t *testing.T) {
	value, err := Loads("[123456789012345678901234567890, 1.50]")
	if err != nil {
		t.Fatal(err)
	}
	list := value.([]any)
	if list[0] != Integer("123456789012345678901234567890") {
		t.Errorf("integer = %#v", list[0])
	}
	if list[1] != 1.5 {
		t.Errorf("float = %#v", list[1])
	}
}

func TestFloatRepr(t *testing.T) {
	for _, tc := range floatReprCases {
		f := math.Float64frombits(tc.bits)
		if got := floatRepr(f); got != tc.want {
			t.Errorf("floatRepr(%v) = %q, want %q", f, got, tc.want)
		}
	}
}
