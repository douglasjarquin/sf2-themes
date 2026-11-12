package catalog

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// DecodeTOML is tomllib.loads on Python 3.11: it decodes a TOML 1.0 document into a generic value, or returns
// an error for every document tomllib rejects.
// Tables decode to map[string]any and arrays, including arrays of tables, to []any;
// integers are int64, or *big.Int outside int64 as Python ints have no bound;
// floats are float64, with ±Inf where a literal overflows it as in Python; date-times are the library's time values.
// It is the one TOML reader for the catalog and for the user configs adapters inspect.
//
// BurntSushi/toml v1.5 reads TOML 1.0 (TOML 1.1 forms only with BURNTSUSHI_TOML_110 set) but accepts some
// documents TOML 1.0 forbids: a table or key defined twice through headers, dotted keys, or inline tables,
// and a UTC offset minute above 59. checkTOMLStructure applies tomllib's own rules for those.
func DecodeTOML(text string) (map[string]any, error) {
	// The TOML library skips a leading byte order mark; tomllib reads it as a statement and rejects the document.
	if strings.HasPrefix(text, "\ufeff") {
		return nil, errors.New("Invalid statement (at line 1, column 1)") //nolint:staticcheck // tomllib's message
	}
	doc, replaced, err := decodeInRange(text)
	if err != nil {
		return nil, err
	}
	if err := checkTOMLStructure(strings.ReplaceAll(text, "\r\n", "\n")); err != nil {
		return nil, err
	}
	normalized, _ := normalizeTOML(doc, replaced).(map[string]any)
	return normalized, nil
}

// bigInts holds the integers outside int64 that decodeInRange replaced with stand-in strings: values[i] is the
// integer whose stand-in reads marker followed by i.
type bigInts struct {
	marker string
	values []*big.Int
}

// decodeInRange decodes text with BurntSushi/toml, which rejects an integer outside int64 or a float that
// overflows float64. Each such literal is replaced, at the same length so later positions hold, by a string
// standing in for the integer or by the float's infinity, and the document is decoded again.
func decodeInRange(text string) (map[string]any, bigInts, error) {
	var replaced bigInts
	for {
		var doc map[string]any
		_, err := toml.Decode(text, &doc)
		var parseErr toml.ParseError
		if err == nil || !errors.As(err, &parseErr) {
			return doc, replaced, err
		}
		literal, isInt := strings.CutSuffix(parseErr.Message, " is out of range for int64")
		if !isInt {
			var isFloat bool
			if literal, isFloat = strings.CutSuffix(parseErr.Message, " is out of range for float64"); !isFloat {
				return nil, bigInts{}, err
			}
		}
		// The library reports offsets after it drops a leading byte order mark.
		start := parseErr.Position.Start + len(text) - len(strings.TrimPrefix(text, "\ufeff"))
		if start > len(text) || !strings.HasPrefix(text[start:], literal) {
			return nil, bigInts{}, err
		}
		var stand string
		switch {
		case isInt:
			value, ok := new(big.Int).SetString(literal, 0)
			if !ok {
				return nil, bigInts{}, err
			}
			if replaced.marker == "" {
				control, found := unescapedControl(text)
				if !found {
					return nil, bigInts{}, err
				}
				replaced.marker = string(control) + "#"
			}
			// At least 9 bytes for the 19 or more an integer outside int64 takes.
			stand = fmt.Sprintf(`"\u%04x#%d"`, replaced.marker[0], len(replaced.values))
			replaced.values = append(replaced.values, value)
		case literal[0] == '-':
			stand = "-inf"
		default:
			stand = "inf"
		}
		if len(stand) > len(literal) {
			return nil, bigInts{}, err
		}
		text = text[:start] + stand + strings.Repeat(" ", len(literal)-len(stand)) + text[start+len(literal):]
	}
}

// unescapedControl returns a control character that no \u or \U escape in text names. TOML allows no other
// control character in a string than tab, newline, and the \b, \f, \n, \r, and \t escapes, so no string of the
// document can equal a stand-in that starts with it.
func unescapedControl(text string) (rune, bool) {
	lower := strings.ToLower(text)
	for control := rune(0); control < 0x20; control++ {
		if strings.ContainsRune("\b\t\n\f\r", control) {
			continue
		}
		if !strings.Contains(lower, fmt.Sprintf(`\u%04x`, control)) && !strings.Contains(lower, fmt.Sprintf(`\u%08x`, control)) {
			return control, true
		}
	}
	return 0, false
}

// normalizeTOML turns the library's []map[string]any arrays of tables into []any so every array has one shape,
// as every array is a list in Python, and puts each integer outside int64 back in place of its stand-in.
func normalizeTOML(value any, replaced bigInts) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			typed[key] = normalizeTOML(item, replaced)
		}
		return typed
	case []map[string]any:
		items := make([]any, len(typed))
		for i, item := range typed {
			items[i] = normalizeTOML(item, replaced)
		}
		return items
	case []any:
		for i, item := range typed {
			typed[i] = normalizeTOML(item, replaced)
		}
		return typed
	case string:
		if index, ok := strings.CutPrefix(typed, replaced.marker); ok && replaced.marker != "" {
			if i, err := strconv.Atoi(index); err == nil && i < len(replaced.values) {
				return replaced.values[i]
			}
		}
		return typed
	default:
		return value
	}
}

// tomlNode mirrors one value of tomllib's NestedDict: a table, a list (an array or an array of tables),
// or a scalar.
type tomlNode struct {
	table map[string]*tomlNode
	list  []*tomlNode
	isArr bool
}

func newTable() *tomlNode { return &tomlNode{table: map[string]*tomlNode{}} }

// getOrCreateNest is NestedDict.get_or_create_nest.
func (n *tomlNode) getOrCreateNest(key []string, accessLists bool) (*tomlNode, bool) {
	cont := n
	for _, k := range key {
		child, ok := cont.table[k]
		if !ok {
			child = newTable()
			cont.table[k] = child
		}
		cont = child
		if accessLists && cont.isArr {
			if len(cont.list) == 0 {
				return nil, false
			}
			cont = cont.list[len(cont.list)-1]
		}
		if cont.table == nil {
			return nil, false
		}
	}
	return cont, true
}

// tomlFlags mirrors tomllib's Flags: FROZEN marks an inline table or array, EXPLICIT_NEST a table that a
// header or a dotted key already opened.
type tomlFlags struct {
	children map[string]*tomlFlags
	flags    [2]bool
	rflags   [2]bool
}

const (
	flagFrozen = iota
	flagExplicitNest
)

func (f *tomlFlags) child(k string) *tomlFlags {
	if f.children == nil {
		f.children = map[string]*tomlFlags{}
	}
	c, ok := f.children[k]
	if !ok {
		c = &tomlFlags{}
		f.children[k] = c
	}
	return c
}

func (f *tomlFlags) set(key []string, flag int, recursive bool) {
	cont := f
	for _, k := range key {
		cont = cont.child(k)
	}
	if recursive {
		cont.rflags[flag] = true
	} else {
		cont.flags[flag] = true
	}
}

func (f *tomlFlags) unsetAll(key []string) {
	cont := f
	for _, k := range key[:len(key)-1] {
		next, ok := cont.children[k]
		if !ok {
			return
		}
		cont = next
	}
	delete(cont.children, key[len(key)-1])
}

func (f *tomlFlags) is(key []string, flag int) bool {
	if len(key) == 0 {
		return false
	}
	cont := f
	for _, k := range key[:len(key)-1] {
		next, ok := cont.children[k]
		if !ok {
			return false
		}
		if next.rflags[flag] {
			return true
		}
		cont = next
	}
	last, ok := cont.children[key[len(key)-1]]
	return ok && (last.flags[flag] || last.rflags[flag])
}

var errTOMLStructure = errors.New("toml: a table or key is defined twice")

// datetimeOffset finds a date-time's UTC offset; tomllib allows hours 00-23 and minutes 00-59.
var datetimeOffset = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[Tt ].*[+-](\d{2}):(\d{2})$`)

var localDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// tomlChecker walks a document BurntSushi/toml has already accepted, so it can assume valid TOML syntax.
type tomlChecker struct {
	src string
	pos int
}

// checkTOMLStructure applies the parts of tomllib.loads that BurntSushi/toml does not enforce.
func checkTOMLStructure(src string) error {
	c := &tomlChecker{src: src}
	data, flags := newTable(), &tomlFlags{}
	var header []string
	var pending [][]string
	finalize := func() {
		for _, key := range pending {
			flags.set(key, flagExplicitNest, false)
		}
		pending = nil
	}
	for {
		c.skipWS()
		if c.eof() {
			return nil
		}
		switch c.peek() {
		case '\n':
			c.pos++
			continue
		case '#':
			c.skipComment()
			continue
		case '[':
			finalize()
			if strings.HasPrefix(c.src[c.pos:], "[[") {
				c.pos += 2
				key := c.parseKey()
				c.pos += 2
				if flags.is(key, flagFrozen) {
					return errTOMLStructure
				}
				flags.unsetAll(key)
				flags.set(key, flagExplicitNest, false)
				parent, ok := data.getOrCreateNest(key[:len(key)-1], true)
				if !ok {
					return errTOMLStructure
				}
				last := key[len(key)-1]
				if existing, ok := parent.table[last]; ok {
					if !existing.isArr {
						return errTOMLStructure
					}
					existing.list = append(existing.list, newTable())
				} else {
					parent.table[last] = &tomlNode{isArr: true, list: []*tomlNode{newTable()}}
				}
				header = key
				continue
			}
			c.pos++
			key := c.parseKey()
			c.pos++
			if flags.is(key, flagExplicitNest) || flags.is(key, flagFrozen) {
				return errTOMLStructure
			}
			flags.set(key, flagExplicitNest, false)
			if _, ok := data.getOrCreateNest(key, true); !ok {
				return errTOMLStructure
			}
			header = key
			continue
		}
		key, value, err := c.parseKeyValue()
		if err != nil {
			return err
		}
		absParent := join(header, key[:len(key)-1])
		for i := 1; i < len(key); i++ {
			contKey := join(header, key[:i])
			if flags.is(contKey, flagExplicitNest) {
				return errTOMLStructure
			}
			pending = append(pending, contKey)
		}
		if flags.is(absParent, flagFrozen) {
			return errTOMLStructure
		}
		nest, ok := data.getOrCreateNest(absParent, true)
		if !ok {
			return errTOMLStructure
		}
		stem := key[len(key)-1]
		if _, exists := nest.table[stem]; exists {
			return errTOMLStructure
		}
		if value.table != nil || value.isArr {
			flags.set(join(header, key), flagFrozen, true)
		}
		nest.table[stem] = value
	}
}

func join(a, b []string) []string {
	return append(append([]string{}, a...), b...)
}

func (c *tomlChecker) eof() bool  { return c.pos >= len(c.src) }
func (c *tomlChecker) peek() byte { return c.src[c.pos] }

func (c *tomlChecker) skipWS() {
	for !c.eof() && (c.peek() == ' ' || c.peek() == '\t') {
		c.pos++
	}
}

func (c *tomlChecker) skipComment() {
	for !c.eof() && c.peek() != '\n' {
		c.pos++
	}
}

// skipWSNewlinesComments skips what may separate array items.
func (c *tomlChecker) skipWSNewlinesComments() {
	for !c.eof() {
		switch c.peek() {
		case ' ', '\t', '\n':
			c.pos++
		case '#':
			c.skipComment()
		default:
			return
		}
	}
}

func (c *tomlChecker) parseKey() []string {
	var key []string
	for {
		c.skipWS()
		switch c.peek() {
		case '"':
			start := c.pos
			c.skipBasicString()
			part, _ := strconv.Unquote(c.src[start:c.pos])
			key = append(key, part)
		case '\'':
			end := strings.IndexByte(c.src[c.pos+1:], '\'')
			key = append(key, c.src[c.pos+1:c.pos+1+end])
			c.pos += end + 2
		default:
			start := c.pos
			for !c.eof() && strings.IndexByte(" \t.=]", c.peek()) < 0 {
				c.pos++
			}
			key = append(key, c.src[start:c.pos])
		}
		c.skipWS()
		if c.eof() || c.peek() != '.' {
			return key
		}
		c.pos++
	}
}

func (c *tomlChecker) parseKeyValue() ([]string, *tomlNode, error) {
	key := c.parseKey()
	c.pos++ // =
	c.skipWS()
	value, err := c.parseValue()
	return key, value, err
}

func (c *tomlChecker) skipBasicString() {
	for c.pos++; c.peek() != '"'; c.pos++ {
		if c.peek() == '\\' {
			c.pos++
		}
	}
	c.pos++
}

// skipMultiline skips a """ or ”' string, whose closing delimiter may carry up to two more quotes.
func (c *tomlChecker) skipMultiline(delim string, escapes bool) {
	c.pos += 3
	for !strings.HasPrefix(c.src[c.pos:], delim) {
		if escapes && c.peek() == '\\' {
			c.pos++
		}
		c.pos++
	}
	c.pos += 3
	for extra := 0; extra < 2 && !c.eof() && c.peek() == delim[0]; extra++ {
		c.pos++
	}
}

func (c *tomlChecker) parseValue() (*tomlNode, error) {
	rest := c.src[c.pos:]
	switch {
	case strings.HasPrefix(rest, `"""`):
		c.skipMultiline(`"""`, true)
	case strings.HasPrefix(rest, `'''`):
		c.skipMultiline(`'''`, false)
	case rest[0] == '"':
		c.skipBasicString()
	case rest[0] == '\'':
		c.pos += strings.IndexByte(rest[1:], '\'') + 2
	case rest[0] == '[':
		return c.parseArray()
	case rest[0] == '{':
		return c.parseInlineTable()
	default:
		return &tomlNode{}, c.checkScalar()
	}
	return &tomlNode{}, nil
}

func (c *tomlChecker) parseArray() (*tomlNode, error) {
	array := &tomlNode{isArr: true}
	c.pos++
	for {
		c.skipWSNewlinesComments()
		if c.peek() == ']' {
			c.pos++
			return array, nil
		}
		item, err := c.parseValue()
		if err != nil {
			return nil, err
		}
		array.list = append(array.list, item)
		c.skipWSNewlinesComments()
		if c.peek() == ',' {
			c.pos++
		}
	}
}

// parseInlineTable is tomllib's parse_inline_table, which keeps its own flags.
func (c *tomlChecker) parseInlineTable() (*tomlNode, error) {
	data, flags := newTable(), &tomlFlags{}
	c.pos++
	c.skipWS()
	if c.peek() == '}' {
		c.pos++
		return data, nil
	}
	for {
		key, value, err := c.parseKeyValue()
		if err != nil {
			return nil, err
		}
		if flags.is(key, flagFrozen) {
			return nil, errTOMLStructure
		}
		nest, ok := data.getOrCreateNest(key[:len(key)-1], false)
		if !ok {
			return nil, errTOMLStructure
		}
		stem := key[len(key)-1]
		if _, exists := nest.table[stem]; exists {
			return nil, errTOMLStructure
		}
		nest.table[stem] = value
		c.skipWS()
		if c.peek() == '}' {
			c.pos++
			return data, nil
		}
		if value.table != nil || value.isArr {
			flags.set(key, flagFrozen, true)
		}
		c.pos++ // ,
		c.skipWS()
	}
}

// checkScalar reads a number, boolean, or date-time and rejects a UTC offset tomllib would not read.
func (c *tomlChecker) checkScalar() error {
	start := c.pos
	c.readScalar()
	if localDate.MatchString(c.src[start:c.pos]) && strings.HasPrefix(c.src[c.pos:], " ") &&
		len(c.src) > c.pos+3 && c.src[c.pos+3] == ':' {
		c.pos++
		c.readScalar()
	}
	if m := datetimeOffset.FindStringSubmatch(c.src[start:c.pos]); m != nil && (m[1] > "23" || m[2] > "59") {
		return errors.New("toml: invalid UTC offset in " + c.src[start:c.pos])
	}
	return nil
}

func (c *tomlChecker) readScalar() {
	for !c.eof() && strings.IndexByte(" \t\n,]}#", c.peek()) < 0 {
		c.pos++
	}
}
