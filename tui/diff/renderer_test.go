package diff

import (
	"strings"
	"testing"
)

func TestGenerate_IdenticalContent(t *testing.T) {
	content := "line1\nline2\nline3"
	lines := Generate(content, content)

	for _, l := range lines {
		if l.Type != LineContext {
			t.Errorf("expected all context lines for identical content, got type %d", l.Type)
		}
	}
}

func TestGenerate_EmptyToContent(t *testing.T) {
	lines := Generate("", "line1\nline2\nline3")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for i, l := range lines {
		if l.Type != LineAdded {
			t.Errorf("line %d: expected LineAdded, got %d", i, l.Type)
		}
	}
}

func TestGenerate_ContentToEmpty(t *testing.T) {
	lines := Generate("line1\nline2\nline3", "")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for i, l := range lines {
		if l.Type != LineRemoved {
			t.Errorf("line %d: expected LineRemoved, got %d", i, l.Type)
		}
	}
}

func TestGenerate_EmptyToEmpty(t *testing.T) {
	lines := Generate("", "")
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}
}

func TestGenerate_SingleLineChange(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nmodified\nline3"
	lines := Generate(old, new)

	found := map[LineType]int{}
	for _, l := range lines {
		found[l.Type]++
	}

	if found[LineContext] < 2 {
		t.Errorf("expected at least 2 context lines, got %d", found[LineContext])
	}
	if found[LineRemoved] < 1 {
		t.Error("expected at least 1 removed line")
	}
	if found[LineAdded] < 1 {
		t.Error("expected at least 1 added line")
	}
}

func TestGenerate_MultipleChanges(t *testing.T) {
	old := "a\nb\nc\nd\ne"
	new := "a\nB\nc\nD\ne"
	lines := Generate(old, new)

	added := 0
	removed := 0
	for _, l := range lines {
		if l.Type == LineAdded {
			added++
		}
		if l.Type == LineRemoved {
			removed++
		}
	}
	if added < 2 {
		t.Errorf("expected at least 2 added lines, got %d", added)
	}
	if removed < 2 {
		t.Errorf("expected at least 2 removed lines, got %d", removed)
	}
}

func TestGenerate_InsertAtStart(t *testing.T) {
	old := "line2\nline3"
	new := "line1\nline2\nline3"
	lines := Generate(old, new)

	first := lines[0]
	if first.Type != LineAdded {
		t.Errorf("expected first line to be added, got type %d", first.Type)
	}
	if first.Content != "line1" {
		t.Errorf("expected 'line1', got %q", first.Content)
	}
}

func TestGenerate_InsertAtEnd(t *testing.T) {
	old := "line1\nline2"
	new := "line1\nline2\nline3"
	lines := Generate(old, new)

	last := lines[len(lines)-1]
	if last.Type != LineAdded {
		t.Errorf("expected last line to be added, got type %d", last.Type)
	}
	if last.Content != "line3" {
		t.Errorf("expected 'line3', got %q", last.Content)
	}
}

func TestGenerate_LineNumbers(t *testing.T) {
	old := "a\nb\nc"
	new := "a\nB\nc"
	lines := Generate(old, new)

	for _, l := range lines {
		switch l.Type {
		case LineContext:
			if l.OldNo == 0 || l.NewNo == 0 {
				t.Error("context line should have both line numbers")
			}
		case LineRemoved:
			if l.OldNo == 0 {
				t.Error("removed line should have old line number")
			}
		case LineAdded:
			if l.NewNo == 0 {
				t.Error("added line should have new line number")
			}
		}
	}
}

func TestGenerate_LineNumbersSequential(t *testing.T) {
	lines := Generate("a\nb", "a\nc")

	var contextOld, contextNew []int
	var addedNew []int
	var removedOld []int

	for _, l := range lines {
		switch l.Type {
		case LineContext:
			contextOld = append(contextOld, l.OldNo)
			contextNew = append(contextNew, l.NewNo)
		case LineAdded:
			addedNew = append(addedNew, l.NewNo)
		case LineRemoved:
			removedOld = append(removedOld, l.OldNo)
		}
	}

	for i, n := range contextOld {
		if n != i+1 && n <= 0 {
			t.Errorf("context old line number unexpected: %d at index %d", n, i)
		}
	}
	for i, n := range contextNew {
		if n != i+1 && n <= 0 {
			t.Errorf("context new line number unexpected: %d at index %d", n, i)
		}
	}
}

func TestRender_NoChanges(t *testing.T) {
	result := Render("hello\nworld", "hello\nworld", "test.txt", false)
	if !strings.Contains(result, "no changes") {
		t.Errorf("expected 'no changes' message, got %q", result)
	}
}

func TestRender_EmptyToEmpty(t *testing.T) {
	result := Render("", "", "empty.txt", false)
	if !strings.Contains(result, "no changes") {
		t.Errorf("expected 'no changes' for empty to empty, got %q", result)
	}
}

func TestRender_NewFile(t *testing.T) {
	result := Render("", "hello\nworld", "new.txt", true)
	if !strings.Contains(result, "+++ new.txt") {
		t.Error("expected '+++ new.txt' header")
	}
	if !strings.Contains(result, "hello") {
		t.Error("expected 'hello' in new file diff")
	}
}

func TestRender_NewFileEmpty(t *testing.T) {
	result := Render("", "", "empty.txt", true)
	if !strings.Contains(result, "+++ empty.txt") {
		t.Error("expected '+++ empty.txt' header for empty new file")
	}
}

func TestRender_WithChanges(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nmodified\nline3"
	result := Render(old, new, "test.go", false)

	if !strings.Contains(result, "--- test.go") {
		t.Error("expected '--- test.go' header")
	}
	if !strings.Contains(result, "+++ test.go") {
		t.Error("expected '+++ test.go' header")
	}
	if !strings.Contains(result, "modified") {
		t.Error("expected 'modified' in diff")
	}
}

func TestRender_ShowsFilePath(t *testing.T) {
	result := Render("a", "b", "path/to/file.go", false)
	if !strings.Contains(result, "path/to/file.go") {
		t.Error("expected file path in diff output")
	}
}

func TestRender_LargeDiffTruncates(t *testing.T) {
	var oldLines []string
	for i := 0; i < 200; i++ {
		oldLines = append(oldLines, "original line")
	}
	old := strings.Join(oldLines, "\n")

	var newLines []string
	for i := 0; i < 200; i++ {
		newLines = append(newLines, "changed line")
	}
	new := strings.Join(newLines, "\n")

	result := Render(old, new, "big.txt", false)
	resultLines := strings.Split(result, "\n")

	if len(resultLines) > maxDisplayLines+5 {
		t.Errorf("expected output to be truncated (~%d lines), got %d lines", maxDisplayLines+5, len(resultLines))
	}
}

func TestRender_NewFileTruncates(t *testing.T) {
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, "new content line")
	}
	content := strings.Join(lines, "\n")

	result := Render("", content, "big.txt", true)
	if !strings.Contains(result, "more lines") {
		t.Error("expected truncation indicator for large new file")
	}
}

func TestRender_NewFileSmallDoesNotTruncate(t *testing.T) {
	content := "line1\nline2\nline3"
	result := Render("", content, "small.txt", true)
	if strings.Contains(result, "more lines") {
		t.Error("expected no truncation for small new file")
	}
	if !strings.Contains(result, "line1") {
		t.Error("expected 'line1' in new file diff")
	}
}

func TestCompact_ShortDiffUnchanged(t *testing.T) {
	lines := []DiffLine{
		{Type: LineContext, Content: "a"},
		{Type: LineRemoved, Content: "b"},
		{Type: LineAdded, Content: "B"},
		{Type: LineContext, Content: "c"},
	}

	result := compact(lines, 50, 3)
	if len(result) != len(lines) {
		t.Errorf("expected %d lines (no compaction needed), got %d", len(lines), len(result))
	}
}

func TestCompact_LongDiffAddsEllipsis(t *testing.T) {
	var lines []DiffLine
	for i := 0; i < 200; i++ {
		lines = append(lines, DiffLine{Type: LineContext, Content: "context"})
	}
	lines[20] = DiffLine{Type: LineRemoved, Content: "old1"}
	lines[21] = DiffLine{Type: LineAdded, Content: "new1"}
	lines[180] = DiffLine{Type: LineRemoved, Content: "old2"}
	lines[181] = DiffLine{Type: LineAdded, Content: "new2"}

	result := compact(lines, 50, 3)
	if len(result) > 55 {
		t.Errorf("expected compacted output, got %d lines", len(result))
	}

	hasEllipsis := false
	for _, l := range result {
		if l.Type == LineEllipsis {
			hasEllipsis = true
			break
		}
	}
	if !hasEllipsis {
		t.Error("expected ellipsis in compacted output")
	}
}

func TestCompact_AllChanges(t *testing.T) {
	var lines []DiffLine
	for i := 0; i < 60; i++ {
		lines = append(lines, DiffLine{Type: LineRemoved, Content: "old"})
		lines = append(lines, DiffLine{Type: LineAdded, Content: "new"})
	}

	result := compact(lines, 50, 3)
	if len(result) > maxDisplayLines+1 {
		t.Errorf("expected max ~%d lines, got %d", maxDisplayLines+1, len(result))
	}
}

func TestCompact_PreservesChangedLines(t *testing.T) {
	var lines []DiffLine
	for i := 0; i < 100; i++ {
		lines = append(lines, DiffLine{Type: LineContext, Content: "ctx"})
	}
	lines[30] = DiffLine{Type: LineRemoved, Content: "removed_line"}
	lines[70] = DiffLine{Type: LineAdded, Content: "added_line"}

	result := compact(lines, 50, 3)

	hasRemoved := false
	hasAdded := false
	for _, l := range result {
		if l.Content == "removed_line" {
			hasRemoved = true
		}
		if l.Content == "added_line" {
			hasAdded = true
		}
	}
	if !hasRemoved {
		t.Error("expected compacted output to preserve removed line")
	}
	if !hasAdded {
		t.Error("expected compacted output to preserve added line")
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single line no newline", "hello", 1},
		{"single line with newline", "hello\n", 1},
		{"two lines", "hello\nworld", 2},
		{"trailing newline", "hello\nworld\n", 2},
		{"multiple trailing newlines", "hello\nworld\n\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			if len(result) != tt.want {
				t.Errorf("expected %d lines, got %d", tt.want, len(result))
			}
		})
	}
}

func TestRender_OutputContainsANSI(t *testing.T) {
	old := "line1\nline2"
	new := "line1\nmodified"
	result := Render(old, new, "test.go", false)

	if !strings.Contains(result, "\x1b[") {
		t.Error("expected ANSI color codes in rendered output")
	}
}

func TestRender_NewFileOutputContainsANSI(t *testing.T) {
	result := Render("", "hello\nworld", "new.go", true)
	if !strings.Contains(result, "\x1b[") {
		t.Error("expected ANSI color codes in new file rendered output")
	}
}

func TestRender_PlainTextContentPreserved(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		filePath string
		isNew    bool
		want     string
	}{
		{"change preserves text", "foo\nbar", "foo\nbaz", "f.txt", false, "baz"},
		{"new file preserves text", "", "hello world", "f.txt", true, "hello world"},
		{"multi-line change", "a\nb\nc", "a\nx\nc", "f.go", false, "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Render(tt.old, tt.new, tt.filePath, tt.isNew)
			plain := stripANSI(result)
			if !strings.Contains(plain, tt.want) {
				t.Errorf("expected rendered output to contain %q, got %q", tt.want, plain)
			}
		})
	}
}

func stripANSI(s string) string {
	var result strings.Builder
	esc := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			esc = true
			continue
		}
		if esc {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				esc = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}
