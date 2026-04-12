package markdown

import (
	"regexp"
	"strings"
	"testing"
)

func TestParse_Headers(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantType  SegmentType
		wantLevel int
		wantText  string
	}{
		{"h1", "# Hello", 1, SegmentHeader, 1, "Hello"},
		{"h2", "## World", 1, SegmentHeader, 2, "World"},
		{"h3", "### Deep", 1, SegmentHeader, 3, "Deep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segs := Parse(tt.input)
			if len(segs) != tt.wantCount {
				t.Fatalf("expected %d segments, got %d", tt.wantCount, len(segs))
			}
			seg := segs[0]
			if seg.Type != tt.wantType {
				t.Errorf("expected type %d, got %d", tt.wantType, seg.Type)
			}
			if seg.Level != tt.wantLevel {
				t.Errorf("expected level %d, got %d", tt.wantLevel, seg.Level)
			}
			if seg.Content != tt.wantText {
				t.Errorf("expected content %q, got %q", tt.wantText, seg.Content)
			}
		})
	}
}

func TestParse_EmptyHeader(t *testing.T) {
	input := "# "
	segs := Parse(input)
	if len(segs) != 0 {
		t.Fatalf("expected 0 segments for empty header, got %d", len(segs))
	}
}

func TestParse_CodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```"
	segs := Parse(input)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(segs), segs)
	}
	if segs[0].Type != SegmentCodeBlock {
		t.Errorf("expected SegmentCodeBlock, got %d", segs[0].Type)
	}
	if segs[0].Language != "go" {
		t.Errorf("expected language 'go', got %q", segs[0].Language)
	}
	if !strings.Contains(segs[0].Content, "fmt.Println") {
		t.Errorf("expected content to contain 'fmt.Println', got %q", segs[0].Content)
	}
}

func TestParse_CodeBlockNoLang(t *testing.T) {
	input := "```\nplain text\n```"
	segs := Parse(input)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if segs[0].Language != "" {
		t.Errorf("expected empty language, got %q", segs[0].Language)
	}
}

func TestParse_UnclosedCodeBlock(t *testing.T) {
	input := "```python\ndef foo():\n    pass"
	segs := Parse(input)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment for unclosed code block, got %d", len(segs))
	}
	if segs[0].Type != SegmentCodeBlock {
		t.Errorf("expected SegmentCodeBlock, got %d", segs[0].Type)
	}
	if segs[0].Language != "python" {
		t.Errorf("expected language 'python', got %q", segs[0].Language)
	}
}

func TestParse_Blockquote(t *testing.T) {
	input := "> This is a quote"
	segs := Parse(input)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if segs[0].Type != SegmentBlockquote {
		t.Errorf("expected SegmentBlockquote, got %d", segs[0].Type)
	}
	if segs[0].Content != "This is a quote" {
		t.Errorf("expected 'This is a quote', got %q", segs[0].Content)
	}
}

func TestParse_UnorderedList(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"dash", "- item one"},
		{"asterisk", "* item two"},
		{"plus", "+ item three"},
		{"indented", "  - indented item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segs := Parse(tt.input)
			if len(segs) != 1 {
				t.Fatalf("expected 1 segment, got %d", len(segs))
			}
			if segs[0].Type != SegmentListItem {
				t.Errorf("expected SegmentListItem, got %d", segs[0].Type)
			}
		})
	}
}

func TestParse_OrderedList(t *testing.T) {
	input := "1. first item\n2. second item"
	segs := Parse(input)
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0].Type != SegmentNumberedItem {
		t.Errorf("expected SegmentNumberedItem, got %d", segs[0].Type)
	}
	if segs[0].Content != "first item" {
		t.Errorf("expected 'first item', got %q", segs[0].Content)
	}
	if segs[1].Content != "second item" {
		t.Errorf("expected 'second item', got %q", segs[1].Content)
	}
}

func TestParse_HorizontalRule(t *testing.T) {
	input := "---"
	segs := Parse(input)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if segs[0].Type != SegmentHorizontalRule {
		t.Errorf("expected SegmentHorizontalRule, got %d", segs[0].Type)
	}
}

func TestParse_PlainText(t *testing.T) {
	input := "Just some regular text"
	segs := Parse(input)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if segs[0].Type != SegmentText {
		t.Errorf("expected SegmentText, got %d", segs[0].Type)
	}
	if segs[0].Content != "Just some regular text" {
		t.Errorf("expected 'Just some regular text', got %q", segs[0].Content)
	}
}

func TestParse_MixedContent(t *testing.T) {
	input := `# Title

This is **bold** and *italic* text.

- item one
- item two

` + "```go" + `
fmt.Println("hello")
` + "```" + `

> A blockquote`

	segs := Parse(input)
	if len(segs) < 6 {
		t.Fatalf("expected at least 6 segments, got %d", len(segs))
	}

	found := map[SegmentType]bool{}
	for _, seg := range segs {
		found[seg.Type] = true
	}

	for _, expected := range []SegmentType{SegmentHeader, SegmentText, SegmentListItem, SegmentCodeBlock, SegmentBlockquote} {
		if !found[expected] {
			t.Errorf("expected to find segment type %d in parsed output", expected)
		}
	}
}

func TestParse_BlankLinesSkipped(t *testing.T) {
	input := "hello\n\n\nworld"
	segs := Parse(input)
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments (blank lines skipped), got %d", len(segs))
	}
}

func TestRender_HeaderProducesOutput(t *testing.T) {
	result := Render("# Hello World")
	if !strings.Contains(result, "Hello World") {
		t.Errorf("expected rendered output to contain 'Hello World', got %q", result)
	}
}

func TestRender_CodeBlockWithBorder(t *testing.T) {
	input := "```go\nfmt.Println(\"hi\")\n```"
	result := Render(input)
	if !strings.Contains(result, "┌") {
		t.Error("expected code block to have top border")
	}
	if !strings.Contains(result, "└") {
		t.Error("expected code block to have bottom border")
	}
	if !strings.Contains(result, "│") {
		t.Error("expected code block to have line prefix")
	}
	if !strings.Contains(result, "go") {
		t.Error("expected code block to show language label")
	}
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestRender_InlineFormatting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bold", "This is **important** text", "important"},
		{"italic", "This is *emphasized* text", "emphasized"},
		{"inline code", "Use the `fmt` package", "fmt"},
		{"link", "Visit [Google](https://google.com)", "Google"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Render(tt.input)
			plain := stripANSI(result)
			if !strings.Contains(plain, tt.want) {
				t.Errorf("expected rendered output to contain %q, got %q", tt.want, plain)
			}
		})
	}
}

func TestRender_ListItems(t *testing.T) {
	input := "- first\n- second\n- third"
	result := Render(input)
	if !strings.Contains(result, "first") {
		t.Error("expected 'first' in rendered output")
	}
	if !strings.Contains(result, "second") {
		t.Error("expected 'second' in rendered output")
	}
}

func TestRender_Blockquote(t *testing.T) {
	input := "> Words of wisdom"
	result := Render(input)
	if !strings.Contains(result, "Words of wisdom") {
		t.Errorf("expected blockquote content in output, got %q", result)
	}
}

func TestRender_Thinking(t *testing.T) {
	result := RenderThinking("hmm let me think")
	if !strings.Contains(result, "💭") {
		t.Error("expected thinking indicator")
	}
	if !strings.Contains(result, "hmm let me think") {
		t.Error("expected thinking content")
	}
}

func TestRender_ThinkingEmpty(t *testing.T) {
	result := RenderThinking("")
	if result != "" {
		t.Errorf("expected empty string for empty thinking, got %q", result)
	}
}

func TestHighlight_GoCode(t *testing.T) {
	code := "func main() {\n\tfmt.Println(\"hello\")\n}"
	result := Highlight(code, "go")
	if !strings.Contains(result, "func") {
		t.Error("expected 'func' in highlighted output")
	}
	if !strings.Contains(result, "hello") {
		t.Error("expected string content in highlighted output")
	}
}

func TestHighlight_PythonCode(t *testing.T) {
	code := "def foo():\n    # comment\n    return 42"
	result := Highlight(code, "python")
	if !strings.Contains(result, "def") {
		t.Error("expected 'def' in highlighted output")
	}
	if !strings.Contains(result, "foo") {
		t.Error("expected function name in highlighted output")
	}
}

func TestHighlight_JavaScriptCode(t *testing.T) {
	code := "const x = 42;\nconsole.log(x);"
	result := Highlight(code, "javascript")
	if !strings.Contains(result, "const") {
		t.Error("expected 'const' in highlighted output")
	}
}

func TestHighlight_JSON(t *testing.T) {
	code := `{"name": "test", "value": 42, "active": true}`
	result := Highlight(code, "json")
	if !strings.Contains(result, "name") {
		t.Error("expected key 'name' in highlighted output")
	}
	if !strings.Contains(result, "test") {
		t.Error("expected string value in highlighted output")
	}
	if !strings.Contains(result, "42") {
		t.Error("expected number in highlighted output")
	}
}

func TestHighlight_YAML(t *testing.T) {
	code := "name: test\nport: 8080\n# comment"
	result := Highlight(code, "yaml")
	if !strings.Contains(result, "name") {
		t.Error("expected key 'name' in highlighted output")
	}
}

func TestHighlight_BashCode(t *testing.T) {
	code := "#!/bin/bash\necho hello\nfor i in 1 2 3; do\n  echo $i\ndone"
	result := Highlight(code, "bash")
	if !strings.Contains(result, "echo") {
		t.Error("expected 'echo' in highlighted output")
	}
}

func TestHighlight_SQLCode(t *testing.T) {
	code := "SELECT * FROM users WHERE id = 1;"
	result := Highlight(code, "sql")
	if !strings.Contains(result, "SELECT") {
		t.Error("expected 'SELECT' in highlighted output")
	}
}

func TestHighlight_LangAliases(t *testing.T) {
	tests := []struct {
		alias string
	}{
		{"js"}, {"jsx"}, {"ts"}, {"tsx"}, {"sh"}, {"shell"},
		{"py"}, {"golang"}, {"zsh"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			result := Highlight("hello", tt.alias)
			if result != "hello" {
				t.Errorf("expected plain text passthrough, got %q", result)
			}
		})
	}
}

func TestHighlight_EmptyCode(t *testing.T) {
	result := Highlight("", "go")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestHighlight_UnknownLang(t *testing.T) {
	code := "some plain text"
	result := Highlight(code, "brainfuck")
	if !strings.Contains(result, "some plain text") {
		t.Error("expected plain text passthrough for unknown language")
	}
}

func TestNormalizeLang(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Go", "go"},
		{"JS", "javascript"},
		{"TypeScript", "typescript"},
		{"Python", "python"},
		{"Bash", "bash"},
		{"SH", "bash"},
		{"py", "python"},
		{"golang", "go"},
		{"unknown", "unknown"},
		{"", ""},
		{"  go  ", "go"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeLang(tt.input)
			if got != tt.want {
				t.Errorf("normalizeLang(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRender_FullMarkdownDocument(t *testing.T) {
	input := `# Project Title

This is a **description** with *emphasis* and ` + "`code`" + `.

## Features

- Feature one
- Feature two

1. First
2. Second

> A notable quote

` + "```python" + `
def hello():
    print("world")
    return 42
` + "```" + `

---

Check out [this link](https://example.com).`

	result := Render(input)
	plain := stripANSI(result)
	indicators := []string{
		"Project Title",
		"description",
		"Features",
		"Feature one",
		"First",
		"print",
		"example.com",
	}
	for _, want := range indicators {
		if !strings.Contains(plain, want) {
			t.Errorf("expected rendered output to contain %q", want)
		}
	}
}

func TestRender_MultipleCodeBlocks(t *testing.T) {
	input := "```go\nfmt.Println(\"a\")\n```\n\ntext between\n\n```js\nconsole.log(\"b\");\n```"
	segs := Parse(input)
	codeBlocks := 0
	for _, seg := range segs {
		if seg.Type == SegmentCodeBlock {
			codeBlocks++
		}
	}
	if codeBlocks != 2 {
		t.Errorf("expected 2 code blocks, got %d", codeBlocks)
	}
}
