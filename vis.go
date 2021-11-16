package prompt

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// encodeVis encodes a string using the visual encoding used by libedit for
// entries in the history file.
func encodeVis(s string) string {
	var buf strings.Builder
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		s = s[size:]

		switch {
		case unicode.IsSpace(r) || r == '\\':
			fmt.Fprintf(&buf, "\\%03o", int(r))
		case unicode.IsControl(r):
			buf.WriteByte('\\')
			buf.WriteByte('^')
			buf.WriteRune(r + 0x40)
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// decodeVis decodes the visual encoding used by libedit for entries in the
// history file. This function does not handle the "%<hex>", "&<amp>", or
// "=<mime>" escape sequences which are not used in the history file.
func decodeVis(s string) (string, error) {
	var buf strings.Builder

	for len(s) > 0 {
		meta := byte(0)
		t, ch := s, s[0]
		s = s[1:]

		switch ch {
		case '\\':
			if len(s) == 0 {
				return "", fmt.Errorf("invalid syntax")
			}
			ch, s = s[0], s[1:]
			switch ch {
			case '0', '1', '2', '3', '4', '5', '6', '7', 'x', '\\', 'a', 'b', 'f', 'n', 'r', 't', 'v':
				// Octal or hex.
				r, _, rem, err := strconv.UnquoteChar(t, 0)
				if err != nil {
					return "", err
				}
				buf.WriteRune(r)
				s = rem
			case 'M':
				// Meta
				if len(s) == 0 {
					return "", fmt.Errorf("invalid syntax 1")
				}
				meta = 0200
				ch, s = s[0], s[1:]
				switch ch {
				case '-':
					if len(s) == 0 {
						return "", fmt.Errorf("invalid syntax 2")
					}
					ch, s = s[0], s[1:]
					buf.WriteByte(ch | meta)
					continue
				case '^':
					// Meta+control
					break // fall-through to Control
				default:
					return "", fmt.Errorf("invalid syntax 3")
				}
				fallthrough
			case '^':
				// Control
				if len(s) == 0 {
					return "", fmt.Errorf("invalid syntax 4")
				}
				ch, s = s[0], s[1:]
				switch ch {
				case '?':
					buf.WriteByte(0177 | meta)
				default:
					buf.WriteByte((ch & 037) | meta)
				}
			case 's':
				// Space
				buf.WriteByte(' ')
			case 'E':
				// Escape
				buf.WriteByte('\x1b')
			case '\n', '$':
				// Hidden newline or marker, skip.
			default:
				return "", fmt.Errorf("invalid syntax")
			}

		default:
			r, size := utf8.DecodeRuneInString(t)
			buf.WriteRune(r)
			s = t[size:]
		}
	}

	return buf.String(), nil
}
