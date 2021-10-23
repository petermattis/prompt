package prompt

import (
	"bytes"
	"strconv"
)

const (
	attrBold      = "\x1b[1m"
	attrDim       = "\x1b[2m"
	attrReset     = "\x1b[0m"
	attrReverse   = "\x1b[7m"
	attrUnderline = "\x1b[4m"
)

const (
	fgDefault   = "\x1b[39m"
	fgBlack     = "\x1b[30m"
	fgBlue      = "\x1b[94m"
	fgBrown     = "\x1b[33m"
	fgCyan      = "\x1b[36m"
	fgDarkBlue  = "\x1b[34m"
	fgDarkGray  = "\x1b[90m"
	fgDarkGreen = "\x1b[32m"
	fgDarkRed   = "\x1b[31m"
	fgFuchsia   = "\x1b[95m"
	fgGreen     = "\x1b[92m"
	fgLightGray = "\x1b[37m"
	fgPurple    = "\x1b[35m"
	fgRed       = "\x1b[91m"
	fgTurquoise = "\x1b[96m"
	fgWhite     = "\x1b[97m"
	fgYellow    = "\x1b[93m"
)

const (
	bgDefault   = "\x1b[49m"
	bgBlack     = "\x1b[40m"
	bgBlue      = "\x1b[104m"
	bgBrown     = "\x1b[43m"
	bgCyan      = "\x1b[46m"
	bgDarkBlue  = "\x1b[44m"
	bgDarkGray  = "\x1b[100m"
	bgDarkGreen = "\x1b[42m"
	bgDarkRed   = "\x1b[41m"
	bgFuchsia   = "\x1b[105m"
	bgGreen     = "\x1b[102m"
	bgLightGray = "\x1b[47m"
	bgPurple    = "\x1b[45m"
	bgRed       = "\x1b[101m"
	bgTurquoise = "\x1b[106m"
	bgWhite     = "\x1b[107m"
	bgYellow    = "\x1b[103m"
)

// eraseLineToRight generates the escape sequence to erase the line from the
// current cursor position to the end of the line.
func eraseLineToRight(buf *bytes.Buffer) {
	const seq = "\x1b[K"
	_, _ = buf.WriteString(seq)
}

// eraseScreen generates the escape sequence to move the cursor to the top left
// of the screen and to erase the contents of the screen.
func eraseScreen(buf *bytes.Buffer) {
	const seq = "\x1b[H\x1b[2J"
	_, _ = buf.WriteString(seq)
}

// cursorMove generates the escape sequences to move the cursor relative to its
// current position. Moving by one character (a common case) is slightly more
// efficient.
func cursorMove(buf *bytes.Buffer, up, down, left, right int) {
	const (
		csi             = "\x1b[" // csi = Control Sequence Introducer
		moveUpSuffix    = "A"
		moveDownSuffix  = "B"
		moveRightSuffix = "C"
		moveLeftSuffix  = "D"
	)

	if up == 1 {
		_, _ = buf.WriteString(csi)
		_, _ = buf.WriteString(moveUpSuffix)
	} else if up > 1 {
		_, _ = buf.WriteString(csi)
		_, _ = buf.WriteString(strconv.Itoa(up))
		_, _ = buf.WriteString(moveUpSuffix)
	}

	if down == 1 {
		_, _ = buf.WriteString(csi)
		_, _ = buf.WriteString(moveDownSuffix)
	} else if down > 1 {
		_, _ = buf.WriteString(csi)
		_, _ = buf.WriteString(strconv.Itoa(down))
		_, _ = buf.WriteString(moveDownSuffix)
	}

	if right == 1 {
		_, _ = buf.WriteString(csi)
		_, _ = buf.WriteString(moveRightSuffix)
	} else if right > 1 {
		_, _ = buf.WriteString(csi)
		_, _ = buf.WriteString(strconv.Itoa(right))
		_, _ = buf.WriteString(moveRightSuffix)
	}

	if left == 1 {
		_, _ = buf.WriteString(csi)
		_, _ = buf.WriteString(moveLeftSuffix)
	} else if left > 1 {
		_, _ = buf.WriteString(csi)
		_, _ = buf.WriteString(strconv.Itoa(left))
		_, _ = buf.WriteString(moveLeftSuffix)
	}
}
