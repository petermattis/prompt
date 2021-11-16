package prompt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVisRoundtrip(t *testing.T) {
	testCases := []string{
		`\foo`,
		" \a\b\f\n\t\vfoo",
		"\x18foo\x19",
	}
	for _, c := range testCases {
		t.Run("", func(t *testing.T) {
			e := encodeVis(c)
			d, err := decodeVis(e)
			require.NoError(t, err)
			require.Equal(t, d, c)
		})
	}
}

func TestVisDecode(t *testing.T) {
	testCases := []struct {
		encoded  string
		expected string
	}{
		{`\\`, `\`},
		{`\a`, "\a"},
		{`\b`, "\b"},
		{`\f`, "\f"},
		{`\n`, "\n"},
		{`\s`, " "},
		{`\t`, "\t"},
		{`\v`, "\v"},
		{`\E`, "\x1b"},
		{"\\\n", ""},
		{`\$`, ""},
		{`\x18`, "\x18"},
		{`\040`, " "},
		{`\^X`, "\x18"},
		{`\^Y`, "\x19"},
		// TODO(peter): Do we even need to support this meta bit stuff? It doesn't look
		// like valid utf8.
		{`\M-x`, "\xf8"},
		{`\M^x`, "\x98"},
	}
	for _, c := range testCases {
		t.Run("", func(t *testing.T) {
			d, err := decodeVis(c.encoded)
			require.NoError(t, err)
			require.Equalf(t, c.expected, d, "%q", d)
		})
	}
}

func TestVisDecodeError(t *testing.T) {
	testCases := []string{
		`\`,   // incomplete escape
		`\1`,  // insufficient octal digits
		`\12`, // insufficient octal digits
		`\^`,  // incomplete control escape
		`\M`,  // incomplete meta escape
		`\M-`, // incomplete meta escape
		`\M^`, // incomplete meta escape
		`\z`,  // unknown escape
	}
	for _, c := range testCases {
		t.Run("", func(t *testing.T) {
			_, err := decodeVis(c)
			require.Error(t, err)
		})
	}
}
