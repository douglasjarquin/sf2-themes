// Code generated from Python 3.11 at the parity reference commit; DO NOT EDIT.

package pycompat

var reprCases = []struct{ in, want string }{
	{"", "''"},
	{"abc", "'abc'"},
	{"it's", "\"it's\""},
	{"say \"hi\"", "'say \"hi\"'"},
	{"both ' and \"", "'both \\' and \"'"},
	{"back\\slash", "'back\\\\slash'"},
	{"tab\tnl\ncr\x0d", "'tab\\tnl\\ncr\\r'"},
	{"\x00\x1b[0m\x7f", "'\\x00\\x1b[0m\\x7f'"},
	{"\xc2\x85\xc2\xa0\xc2\xad", "'\\x85\\xa0\\xad'"},
	{"café 名前", "'café 名前'"},
	{"\xe2\x80\x8b\xe2\x80\xa8\xe2\x80\xa9\xe3\x80\x80", "'\\u200b\\u2028\\u2029\\u3000'"},
	{"\xee\x80\x80\xf4\x8f\xbf\xbf", "'\\ue000\\U0010ffff'"},
	{"😀\xf3\xa0\x80\x81", "'😀\\U000e0001'"},
	{"#GGGGGG", "'#GGGGGG'"},
	{"\xef\xbb\xbf", "'\\ufeff'"},
}
