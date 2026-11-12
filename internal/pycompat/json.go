package pycompat

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// A JSON value here is what Python's json module gives and takes: nil (None), bool, string, Integer or int (int),
// float64 (float), []any (list), or *Object (dict).
//
// A string decoded from a `\uXXXX` escape of an unpaired surrogate holds that surrogate in its three-byte UTF-8
// form, which is not valid UTF-8, so the value survives a load and dump the way it does in Python.

// Object is a dict: keys keep the order they were first set in.
type Object struct {
	keys   []string
	values map[string]any
}

// NewObject returns an empty object.
func NewObject() *Object {
	return &Object{values: map[string]any{}}
}

// Set stores a value; an existing key keeps its position and a new key goes last.
func (o *Object) Set(key string, value any) {
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

// Get returns the value stored under key and whether the key is present.
func (o *Object) Get(key string) (any, bool) {
	value, ok := o.values[key]
	return value, ok
}

// Keys returns the keys in order. The caller must not modify the slice.
func (o *Object) Keys() []string {
	return o.keys
}

// Integer is a JSON integer of any size, kept exactly as its decimal text.
type Integer string

// DecodeError is json.JSONDecodeError: what was expected, and where, counted in characters.
type DecodeError struct {
	Msg    string
	Pos    int
	Line   int
	Column int
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("%s: line %d column %d (char %d)", e.Msg, e.Line, e.Column, e.Pos)
}

// Loads is json.loads(s).
func Loads(s string) (any, error) {
	d := &decoder{doc: []rune(s)}
	if len(d.doc) > 0 && d.doc[0] == 0xfeff {
		return nil, d.fail("Unexpected UTF-8 BOM (decode using utf-8-sig)", 0)
	}
	value, end, err := d.value(d.skip(0))
	if err != nil {
		return nil, err
	}
	if end = d.skip(end); end != len(d.doc) {
		return nil, d.fail("Extra data", end)
	}
	return value, nil
}

type decoder struct {
	doc []rune
}

func (d *decoder) fail(msg string, pos int) *DecodeError {
	line, lineStart := 1, 0
	for i, r := range d.doc[:pos] {
		if r == '\n' {
			line++
			lineStart = i + 1
		}
	}
	return &DecodeError{Msg: msg, Pos: pos, Line: line, Column: pos - lineStart + 1}
}

func (d *decoder) skip(i int) int {
	for i < len(d.doc) && (d.doc[i] == ' ' || d.doc[i] == '\t' || d.doc[i] == '\n' || d.doc[i] == '\r') {
		i++
	}
	return i
}

func (d *decoder) at(i int) rune {
	if i < len(d.doc) {
		return d.doc[i]
	}
	return -1
}

func (d *decoder) literal(i int, word string) bool {
	return strings.HasPrefix(string(d.doc[i:min(i+len(word), len(d.doc))]), word)
}

// value parses the value starting at i and returns it with the index just past it.
func (d *decoder) value(i int) (any, int, error) {
	switch c := d.at(i); {
	case c == '"':
		return d.string(i + 1)
	case c == '{':
		return d.object(i + 1)
	case c == '[':
		return d.array(i + 1)
	case c == 'n' && d.literal(i, "null"):
		return nil, i + 4, nil
	case c == 't' && d.literal(i, "true"):
		return true, i + 4, nil
	case c == 'f' && d.literal(i, "false"):
		return false, i + 5, nil
	case c == 'N' && d.literal(i, "NaN"):
		return math.NaN(), i + 3, nil
	case c == 'I' && d.literal(i, "Infinity"):
		return math.Inf(1), i + 8, nil
	case c == '-' && d.literal(i, "-Infinity"):
		return math.Inf(-1), i + 9, nil
	}
	return d.number(i)
}

func isDigit(c rune) bool { return c >= '0' && c <= '9' }

func (d *decoder) number(start int) (any, int, error) {
	i := start
	if d.at(i) == '-' {
		i++
	}
	switch c := d.at(i); {
	case c >= '1' && c <= '9':
		for i++; isDigit(d.at(i)); i++ {
		}
	case c == '0':
		i++
	default:
		return nil, 0, d.fail("Expecting value", start)
	}
	isFloat := false
	if d.at(i) == '.' && isDigit(d.at(i+1)) {
		isFloat = true
		for i += 2; isDigit(d.at(i)); i++ {
		}
	}
	if c := d.at(i); c == 'e' || c == 'E' {
		j := i + 1
		if c := d.at(j); c == '-' || c == '+' {
			j++
		}
		digits := j
		for isDigit(d.at(j)) {
			j++
		}
		if j > digits {
			isFloat, i = true, j
		}
	}
	text := string(d.doc[start:i])
	if !isFloat {
		if text == "-0" {
			text = "0"
		}
		return Integer(text), i, nil
	}
	// Python's float() rounds an out-of-range literal to infinity or zero instead of failing.
	f, err := strconv.ParseFloat(text, 64)
	if err != nil && !isRangeError(err) {
		return nil, 0, err
	}
	return f, i, nil
}

func isRangeError(err error) bool {
	numErr, ok := err.(*strconv.NumError)
	return ok && numErr.Err == strconv.ErrRange
}

// string parses the rest of a string whose opening quote is just before i.
func (d *decoder) string(i int) (any, int, error) {
	begin := i - 1
	var b strings.Builder
	for {
		c := d.at(i)
		switch {
		case c < 0:
			return nil, 0, d.fail("Unterminated string starting at", begin)
		case c == '"':
			return b.String(), i + 1, nil
		case c <= 0x1f:
			return nil, 0, d.fail("Invalid control character at", i)
		case c != '\\':
			b.WriteRune(c)
			i++
			continue
		}
		i++
		switch esc := d.at(i); esc {
		case '"', '\\', '/':
			b.WriteRune(esc)
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'u':
			code, ok := d.hex4(i + 1)
			if !ok {
				return nil, 0, d.fail("Invalid \\uXXXX escape", i)
			}
			i += 4
			if code >= 0xd800 && code <= 0xdbff && d.at(i+1) == '\\' && d.at(i+2) == 'u' {
				low, ok := d.hex4(i + 3)
				if !ok {
					return nil, 0, d.fail("Invalid \\uXXXX escape", i+2)
				}
				if low >= 0xdc00 && low <= 0xdfff {
					code = 0x10000 + (code-0xd800)<<10 + (low - 0xdc00)
					i += 6
				}
			}
			writeCodePoint(&b, code)
		case -1:
			return nil, 0, d.fail("Unterminated string starting at", begin)
		default:
			return nil, 0, d.fail("Invalid \\escape", i-1)
		}
		i++
	}
}

func (d *decoder) hex4(i int) (rune, bool) {
	if i+4 > len(d.doc) {
		return 0, false
	}
	var code rune
	for _, c := range d.doc[i : i+4] {
		switch {
		case isDigit(c):
			code = code<<4 | (c - '0')
		case c >= 'a' && c <= 'f':
			code = code<<4 | (c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			code = code<<4 | (c - 'A' + 10)
		default:
			return 0, false
		}
	}
	return code, true
}

// writeCodePoint writes any code point as UTF-8, including an unpaired surrogate, which utf8 would replace.
func writeCodePoint(b *strings.Builder, c rune) {
	if c < 0xd800 || c > 0xdfff {
		b.WriteRune(c)
		return
	}
	b.WriteByte(byte(0xe0 | c>>12))
	b.WriteByte(byte(0x80 | (c>>6)&0x3f))
	b.WriteByte(byte(0x80 | c&0x3f))
}

func (d *decoder) object(i int) (any, int, error) {
	obj := NewObject()
	i = d.skip(i)
	if d.at(i) == '}' {
		return obj, i + 1, nil
	}
	for {
		if d.at(i) != '"' {
			return nil, 0, d.fail("Expecting property name enclosed in double quotes", i)
		}
		key, end, err := d.string(i + 1)
		if err != nil {
			return nil, 0, err
		}
		i = d.skip(end)
		if d.at(i) != ':' {
			return nil, 0, d.fail("Expecting ':' delimiter", i)
		}
		value, end, err := d.value(d.skip(i + 1))
		if err != nil {
			return nil, 0, err
		}
		obj.Set(key.(string), value)
		i = d.skip(end)
		if d.at(i) == '}' {
			return obj, i + 1, nil
		}
		if d.at(i) != ',' {
			return nil, 0, d.fail("Expecting ',' delimiter", i)
		}
		i = d.skip(i + 1)
	}
}

func (d *decoder) array(i int) (any, int, error) {
	list := []any{}
	i = d.skip(i)
	if d.at(i) == ']' {
		return list, i + 1, nil
	}
	for {
		value, end, err := d.value(i)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, value)
		i = d.skip(end)
		if d.at(i) == ']' {
			return list, i + 1, nil
		}
		if d.at(i) != ',' {
			return nil, 0, d.fail("Expecting ',' delimiter", i)
		}
		i = d.skip(i + 1)
	}
}

// Dumps is json.dumps(value, indent=2): non-ASCII characters are written as `\uXXXX` escapes.
// Like Python, it adds no trailing newline. It panics on a value that is not one of the JSON value types.
func Dumps(value any) string {
	var b strings.Builder
	encode(&b, value, 0, true)
	return b.String()
}

// DumpsUnicode is json.dumps(value, indent=2, ensure_ascii=False): non-ASCII characters are written as themselves.
func DumpsUnicode(value any) string {
	var b strings.Builder
	encode(&b, value, 0, false)
	return b.String()
}

func encode(b *strings.Builder, value any, level int, ascii bool) {
	switch v := value.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(v))
	case string:
		encodeString(b, v, ascii)
	case int:
		b.WriteString(strconv.Itoa(v))
	case Integer:
		b.WriteString(string(v))
	case float64:
		b.WriteString(jsonFloat(v))
	case []any:
		if len(v) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				b.WriteByte(',')
			}
			newline(b, level+1)
			encode(b, item, level+1, ascii)
		}
		newline(b, level)
		b.WriteByte(']')
	case *Object:
		if len(v.keys) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteByte('{')
		for i, key := range v.keys {
			if i > 0 {
				b.WriteByte(',')
			}
			newline(b, level+1)
			encodeString(b, key, ascii)
			b.WriteString(": ")
			encode(b, v.values[key], level+1, ascii)
		}
		newline(b, level)
		b.WriteByte('}')
	default:
		panic(fmt.Sprintf("pycompat: %T is not a JSON value", value))
	}
}

func newline(b *strings.Builder, level int) {
	b.WriteByte('\n')
	b.WriteString(strings.Repeat("  ", level))
}

func encodeString(b *strings.Builder, s string, ascii bool) {
	b.WriteByte('"')
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 && i+2 < len(s) && s[i] == 0xed && s[i+1] >= 0xa0 && s[i+1] <= 0xbf {
			// An unpaired surrogate, written by writeCodePoint.
			r, size = rune(s[i]&0x0f)<<12|rune(s[i+1]&0x3f)<<6|rune(s[i+2]&0x3f), 3
		}
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || (ascii && r >= 0x7f && r < 0x10000):
			fmt.Fprintf(b, `\u%04x`, r)
		case ascii && r >= 0x10000:
			r -= 0x10000
			fmt.Fprintf(b, `\u%04x\u%04x`, 0xd800|r>>10, 0xdc00|r&0x3ff)
		default:
			b.WriteString(s[i : i+size])
		}
		i += size
	}
	b.WriteByte('"')
}

// jsonFloat is how json.dumps writes a float: repr(), with NaN and the infinities spelled as JavaScript does.
func jsonFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	return floatRepr(f)
}

// floatRepr is repr() of a finite float: the shortest digits that read back as the same float, in positional
// notation with at least one fractional digit when the decimal exponent is from -4 to 15, and as `1.5e+16` otherwise.
func floatRepr(f float64) string {
	sci := strconv.FormatFloat(f, 'e', -1, 64) // -d.ddde±dd
	sign := ""
	if sci[0] == '-' {
		sign, sci = "-", sci[1:]
	}
	mantissa, expText, _ := strings.Cut(sci, "e")
	exp, _ := strconv.Atoi(expText)
	digits := strings.Replace(mantissa, ".", "", 1)
	if exp < -4 || exp >= 16 {
		if len(digits) > 1 {
			digits = digits[:1] + "." + digits[1:]
		}
		expSign := "+"
		if exp < 0 {
			expSign, exp = "-", -exp
		}
		return fmt.Sprintf("%s%se%s%02d", sign, digits, expSign, exp)
	}
	point := exp + 1
	switch {
	case point <= 0:
		return sign + "0." + strings.Repeat("0", -point) + digits
	case point >= len(digits):
		return sign + digits + strings.Repeat("0", point-len(digits)) + ".0"
	default:
		return sign + digits[:point] + "." + digits[point:]
	}
}
