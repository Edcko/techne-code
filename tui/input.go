package tui

import "strings"

type InputBuffer struct {
	lines      []string
	cursorLine int
	cursorCol  int
	scrollY    int
}

func newInputBuffer() *InputBuffer {
	return &InputBuffer{
		lines: []string{""},
	}
}

func (b *InputBuffer) Text() string {
	return strings.Join(b.lines, "\n")
}

func (b *InputBuffer) IsEmpty() bool {
	return len(b.lines) == 1 && b.lines[0] == ""
}

func (b *InputBuffer) Clear() {
	b.lines = []string{""}
	b.cursorLine = 0
	b.cursorCol = 0
	b.scrollY = 0
}

func (b *InputBuffer) LineCount() int {
	return len(b.lines)
}

func (b *InputBuffer) CursorPos() (int, int) {
	return b.cursorLine, b.cursorCol
}

func (b *InputBuffer) SetText(text string) {
	if text == "" {
		b.lines = []string{""}
	} else {
		b.lines = strings.Split(text, "\n")
	}
	b.cursorLine = len(b.lines) - 1
	b.cursorCol = len(b.lines[b.cursorLine])
	b.scrollY = 0
}

func (b *InputBuffer) insertRune(r rune) {
	line := b.lines[b.cursorLine]
	b.lines[b.cursorLine] = line[:b.cursorCol] + string(r) + line[b.cursorCol:]
	b.cursorCol++
}

func (b *InputBuffer) InsertChar(ch string) {
	for _, r := range ch {
		b.insertRune(r)
	}
}

func (b *InputBuffer) InsertNewline() {
	line := b.lines[b.cursorLine]
	before := line[:b.cursorCol]
	after := line[b.cursorCol:]
	b.lines[b.cursorLine] = before

	newLines := make([]string, 0, len(b.lines)+1)
	newLines = append(newLines, b.lines[:b.cursorLine+1]...)
	newLines = append(newLines, after)
	newLines = append(newLines, b.lines[b.cursorLine+1:]...)
	b.lines = newLines

	b.cursorLine++
	b.cursorCol = 0
}

func (b *InputBuffer) Backspace() {
	if b.cursorCol > 0 {
		line := b.lines[b.cursorLine]
		b.lines[b.cursorLine] = line[:b.cursorCol-1] + line[b.cursorCol:]
		b.cursorCol--
	} else if b.cursorLine > 0 {
		prevLine := b.lines[b.cursorLine-1]
		currentLine := b.lines[b.cursorLine]
		b.cursorCol = len(prevLine)
		b.lines[b.cursorLine-1] = prevLine + currentLine
		b.lines = append(b.lines[:b.cursorLine], b.lines[b.cursorLine+1:]...)
		b.cursorLine--
	}
}

func (b *InputBuffer) Delete() {
	line := b.lines[b.cursorLine]
	if b.cursorCol < len(line) {
		b.lines[b.cursorLine] = line[:b.cursorCol] + line[b.cursorCol+1:]
	} else if b.cursorLine < len(b.lines)-1 {
		nextLine := b.lines[b.cursorLine+1]
		b.lines[b.cursorLine] = line + nextLine
		b.lines = append(b.lines[:b.cursorLine+1], b.lines[b.cursorLine+2:]...)
	}
}

func (b *InputBuffer) MoveLeft() {
	if b.cursorCol > 0 {
		b.cursorCol--
	} else if b.cursorLine > 0 {
		b.cursorLine--
		b.cursorCol = len(b.lines[b.cursorLine])
	}
}

func (b *InputBuffer) MoveRight() {
	if b.cursorCol < len(b.lines[b.cursorLine]) {
		b.cursorCol++
	} else if b.cursorLine < len(b.lines)-1 {
		b.cursorLine++
		b.cursorCol = 0
	}
}

func (b *InputBuffer) MoveUp() bool {
	if b.cursorLine > 0 {
		b.cursorLine--
		if b.cursorCol > len(b.lines[b.cursorLine]) {
			b.cursorCol = len(b.lines[b.cursorLine])
		}
		return true
	}
	return false
}

func (b *InputBuffer) MoveDown() bool {
	if b.cursorLine < len(b.lines)-1 {
		b.cursorLine++
		if b.cursorCol > len(b.lines[b.cursorLine]) {
			b.cursorCol = len(b.lines[b.cursorLine])
		}
		return true
	}
	return false
}

func (b *InputBuffer) MoveHome() {
	b.cursorCol = 0
}

func (b *InputBuffer) MoveEnd() {
	b.cursorCol = len(b.lines[b.cursorLine])
}

func (b *InputBuffer) CursorIsAtTop() bool {
	return b.cursorLine == 0 && b.cursorCol == 0
}

func (b *InputBuffer) CursorIsAtBottom() bool {
	return b.cursorLine == len(b.lines)-1 && b.cursorCol == len(b.lines[b.cursorLine])
}

func (b *InputBuffer) InsertPaste(text string) {
	if text == "" {
		return
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		for _, r := range line {
			b.insertRune(r)
		}
		if i < len(lines)-1 {
			b.InsertNewline()
		}
	}
}

func (b *InputBuffer) VisibleLines(maxHeight int) []string {
	total := len(b.lines)
	if total <= maxHeight {
		b.scrollY = 0
		return b.lines
	}

	if b.cursorLine < b.scrollY {
		b.scrollY = b.cursorLine
	}
	if b.cursorLine >= b.scrollY+maxHeight {
		b.scrollY = b.cursorLine - maxHeight + 1
	}
	if b.scrollY < 0 {
		b.scrollY = 0
	}

	end := b.scrollY + maxHeight
	if end > total {
		end = total
	}
	return b.lines[b.scrollY:end]
}

func (b *InputBuffer) ScrollOffset() int {
	return b.scrollY
}
