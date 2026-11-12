package pycompat

import (
	"slices"
	"strconv"
	"strings"
)

// contextLines is unified_diff's default n.
const contextLines = 3

// UnifiedDiff is "".join(difflib.unified_diff(a, b, fromfile=fromFile, tofile=toFile)): the headers carry no
// timestamps, each line is written with the ending it already has, and a last line without a newline runs into
// whatever follows it, with no marker. It returns "" when the sequences are equal.
func UnifiedDiff(a, b []string, fromFile, toFile string) string {
	groups := groupedOpcodes(newMatcher(a, b).opcodes(), contextLines)
	if len(groups) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("--- " + fromFile + "\n")
	out.WriteString("+++ " + toFile + "\n")
	for _, group := range groups {
		first, last := group[0], group[len(group)-1]
		out.WriteString("@@ -" + unifiedRange(first.i1, last.i2) + " +" + unifiedRange(first.j1, last.j2) + " @@\n")
		for _, op := range group {
			if op.tag == opEqual {
				for _, line := range a[op.i1:op.i2] {
					out.WriteString(" " + line)
				}
				continue
			}
			if op.tag == opReplace || op.tag == opDelete {
				for _, line := range a[op.i1:op.i2] {
					out.WriteString("-" + line)
				}
			}
			if op.tag == opReplace || op.tag == opInsert {
				for _, line := range b[op.j1:op.j2] {
					out.WriteString("+" + line)
				}
			}
		}
	}
	return out.String()
}

// unifiedRange is difflib._format_range_unified.
func unifiedRange(start, stop int) string {
	beginning, length := start+1, stop-start
	if length == 1 {
		return strconv.Itoa(beginning)
	}
	if length == 0 {
		beginning--
	}
	return strconv.Itoa(beginning) + "," + strconv.Itoa(length)
}

type opTag int

const (
	opEqual opTag = iota
	opReplace
	opDelete
	opInsert
)

type opcode struct {
	tag            opTag
	i1, i2, j1, j2 int
}

// groupedOpcodes is SequenceMatcher.get_grouped_opcodes(n).
func groupedOpcodes(codes []opcode, n int) [][]opcode {
	if len(codes) == 0 {
		codes = []opcode{{opEqual, 0, 1, 0, 1}}
	}
	if c := &codes[0]; c.tag == opEqual {
		c.i1, c.j1 = max(c.i1, c.i2-n), max(c.j1, c.j2-n)
	}
	if c := &codes[len(codes)-1]; c.tag == opEqual {
		c.i2, c.j2 = min(c.i2, c.i1+n), min(c.j2, c.j1+n)
	}
	var groups [][]opcode
	var group []opcode
	for _, c := range codes {
		if c.tag == opEqual && c.i2-c.i1 > 2*n {
			group = append(group, opcode{opEqual, c.i1, min(c.i2, c.i1+n), c.j1, min(c.j2, c.j1+n)})
			groups = append(groups, group)
			group = nil
			c.i1, c.j1 = max(c.i1, c.i2-n), max(c.j1, c.j2-n)
		}
		group = append(group, c)
	}
	if len(group) > 0 && (len(group) != 1 || group[0].tag != opEqual) {
		groups = append(groups, group)
	}
	return groups
}

// matcher is difflib.SequenceMatcher(None, a, b) with autojunk on: no junk function, but when b has 200 or more
// lines, a line that appears in more than 1% of b plus one is left out of the index of b's lines.
type matcher struct {
	a, b []string
	b2j  map[string][]int
}

func newMatcher(a, b []string) *matcher {
	m := &matcher{a: a, b: b, b2j: map[string][]int{}}
	for j, line := range b {
		m.b2j[line] = append(m.b2j[line], j)
	}
	if n := len(b); n >= 200 {
		popular := n/100 + 1
		for line, indices := range m.b2j {
			if len(indices) > popular {
				delete(m.b2j, line)
			}
		}
	}
	return m
}

type match struct{ i, j, size int }

// longestMatch is SequenceMatcher.find_longest_match: the longest matching block in a[alo:ahi] and b[blo:bhi],
// the earliest in a and then in b on a tie, extended over equal lines the index left out.
func (m *matcher) longestMatch(alo, ahi, blo, bhi int) match {
	besti, bestj, bestSize := alo, blo, 0
	j2len := map[int]int{}
	for i := alo; i < ahi; i++ {
		next := map[int]int{}
		for _, j := range m.b2j[m.a[i]] {
			if j < blo {
				continue
			}
			if j >= bhi {
				break
			}
			k := j2len[j-1] + 1
			next[j] = k
			if k > bestSize {
				besti, bestj, bestSize = i-k+1, j-k+1, k
			}
		}
		j2len = next
	}
	for besti > alo && bestj > blo && m.a[besti-1] == m.b[bestj-1] {
		besti, bestj, bestSize = besti-1, bestj-1, bestSize+1
	}
	for besti+bestSize < ahi && bestj+bestSize < bhi && m.a[besti+bestSize] == m.b[bestj+bestSize] {
		bestSize++
	}
	return match{besti, bestj, bestSize}
}

// matchingBlocks is SequenceMatcher.get_matching_blocks, ending with the (len(a), len(b), 0) sentinel.
func (m *matcher) matchingBlocks() []match {
	queue := [][4]int{{0, len(m.a), 0, len(m.b)}}
	var blocks []match
	for len(queue) > 0 {
		q := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		alo, ahi, blo, bhi := q[0], q[1], q[2], q[3]
		x := m.longestMatch(alo, ahi, blo, bhi)
		if x.size == 0 {
			continue
		}
		blocks = append(blocks, x)
		if alo < x.i && blo < x.j {
			queue = append(queue, [4]int{alo, x.i, blo, x.j})
		}
		if x.i+x.size < ahi && x.j+x.size < bhi {
			queue = append(queue, [4]int{x.i + x.size, ahi, x.j + x.size, bhi})
		}
	}
	slices.SortFunc(blocks, func(p, q match) int {
		if p.i != q.i {
			return p.i - q.i
		}
		if p.j != q.j {
			return p.j - q.j
		}
		return p.size - q.size
	})
	var collapsed []match
	current := match{}
	for _, x := range blocks {
		if current.i+current.size == x.i && current.j+current.size == x.j {
			current.size += x.size
			continue
		}
		if current.size > 0 {
			collapsed = append(collapsed, current)
		}
		current = x
	}
	if current.size > 0 {
		collapsed = append(collapsed, current)
	}
	return append(collapsed, match{len(m.a), len(m.b), 0})
}

// opcodes is SequenceMatcher.get_opcodes.
func (m *matcher) opcodes() []opcode {
	var codes []opcode
	i, j := 0, 0
	for _, x := range m.matchingBlocks() {
		switch {
		case i < x.i && j < x.j:
			codes = append(codes, opcode{opReplace, i, x.i, j, x.j})
		case i < x.i:
			codes = append(codes, opcode{opDelete, i, x.i, j, x.j})
		case j < x.j:
			codes = append(codes, opcode{opInsert, i, x.i, j, x.j})
		}
		i, j = x.i+x.size, x.j+x.size
		if x.size > 0 {
			codes = append(codes, opcode{opEqual, x.i, i, x.j, j})
		}
	}
	return codes
}
