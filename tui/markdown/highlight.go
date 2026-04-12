package markdown

import (
	"regexp"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

type TokenType int

const (
	TokenKeyword TokenType = iota
	TokenString
	TokenComment
	TokenNumber
	TokenUserType
	TokenFunction
	TokenOperator
	TokenPlain
)

type Token struct {
	Type  TokenType
	Value string
}

type highlightSpan struct {
	start int
	end   int
	typ   TokenType
}

var (
	colorKeyword  = lipgloss.Color("#FF79C6")
	colorString   = lipgloss.Color("#F1FA8C")
	colorComment  = lipgloss.Color("#6272A4")
	colorNumber   = lipgloss.Color("#BD93F9")
	colorType     = lipgloss.Color("#8BE9FD")
	colorFunc     = lipgloss.Color("#50FA7B")
	colorOperator = lipgloss.Color("#FF79C6")
)

var keywordPatterns = map[string]*regexp.Regexp{
	"go":         regexp.MustCompile(`\b(break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go|goto|if|import|interface|map|package|range|return|select|struct|switch|type|var|true|false|nil|error|string|int|int8|int16|int32|int64|uint|uint8|uint16|uint32|uint64|float32|float64|bool|byte|rune|complex64|complex128|make|new|len|cap|append|copy|delete|close|panic|recover|print|println|complex|real|imag)\b`),
	"python":     regexp.MustCompile(`\b(and|as|assert|async|await|break|class|continue|def|del|elif|else|except|finally|for|from|global|if|import|in|is|lambda|nonlocal|not|or|pass|raise|return|try|while|with|yield|True|False|None|self|print|range|len|int|str|float|list|dict|set|tuple|type|super|isinstance|hasattr|getattr|setattr|input|open|format|abs|round|enumerate|zip|map|filter|sorted|reversed|min|max|sum|any|all|property|staticmethod|classmethod)\b`),
	"javascript": regexp.MustCompile(`\b(break|case|catch|class|const|continue|debugger|default|delete|do|else|export|extends|finally|for|from|function|if|import|in|instanceof|let|new|of|return|super|switch|this|throw|try|typeof|var|void|while|with|yield|true|false|null|undefined|async|await)\b`),
	"typescript": regexp.MustCompile(`\b(break|case|catch|class|const|continue|debugger|default|delete|do|else|export|extends|finally|for|from|function|if|import|in|instanceof|let|new|of|return|super|switch|this|throw|try|typeof|var|void|while|with|yield|true|false|null|undefined|async|await|readonly|declare|interface|type|enum|namespace|module|abstract|implements|private|protected|public|static|as|is|keyof|infer|never|unknown|any|boolean|string|number|bigint|symbol|object|constructor)\b`),
	"bash":       regexp.MustCompile(`\b(if|then|else|elif|fi|case|esac|for|while|until|do|done|in|function|select|time|return|exit|break|continue|declare|export|local|readonly|unset|source|echo|eval|exec|set|shift|test|trap|true|false|type|read|printf|cd|pwd|ls|cp|mv|rm|mkdir|cat|grep|sed|awk|find|chmod|curl|wget|npm|pip|git|docker|sudo)\b`),
	"sql":        regexp.MustCompile(`(?i)\b(SELECT|FROM|WHERE|INSERT|INTO|VALUES|UPDATE|SET|DELETE|CREATE|TABLE|ALTER|DROP|INDEX|JOIN|INNER|LEFT|RIGHT|OUTER|ON|AND|OR|NOT|IN|BETWEEN|LIKE|IS|NULL|AS|ORDER|BY|GROUP|HAVING|LIMIT|OFFSET|DISTINCT|UNION|EXISTS|CASE|WHEN|THEN|ELSE|END|COUNT|SUM|AVG|MIN|MAX|CAST|COALESCE|PRIMARY|KEY|FOREIGN|REFERENCES|CONSTRAINT|DEFAULT|CHECK|UNIQUE|WITH|RECURSIVE|OVER|PARTITION|ASC|DESC|INTO|TRUNCATE|EXPLAIN|BOOLEAN|TEXT|INTEGER|VARCHAR|SERIAL|BIGSERIAL|UUID|JSON|JSONB)\b`),
	"yaml":       regexp.MustCompile(`\b(true|false|null|yes|no|on|off)\b`),
	"json":       regexp.MustCompile(`\b(true|false|null)\b`),
}

var typePatterns = map[string]*regexp.Regexp{
	"go":         regexp.MustCompile(`\b([A-Z][a-zA-Z0-9]*)\b`),
	"python":     regexp.MustCompile(`\b([A-Z][a-zA-Z0-9]*)\b`),
	"javascript": regexp.MustCompile(`\b([A-Z][a-zA-Z0-9]*)\b`),
	"typescript": regexp.MustCompile(`\b([A-Z][a-zA-Z0-9]*)\b`),
}

var commentPatterns = map[string]*regexp.Regexp{
	"go":         regexp.MustCompile(`(//.*$|/\*[\s\S]*?\*/)`),
	"python":     regexp.MustCompile(`(#.*$|"""[\s\S]*?"""|'''[\s\S]*?''')`),
	"javascript": regexp.MustCompile(`(//.*$|/\*[\s\S]*?\*/)`),
	"typescript": regexp.MustCompile(`(//.*$|/\*[\s\S]*?\*/)`),
	"bash":       regexp.MustCompile(`(#.*$)`),
	"sql":        regexp.MustCompile(`(--.*$|/\*[\s\S]*?\*/)`),
	"yaml":       regexp.MustCompile(`(#.*$)`),
	"json":       nil,
}

var stringPattern = regexp.MustCompile(`("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|` + "`" + `(?:[^` + "`" + `\\]|\\.)*` + "`" + `)`)
var numberPattern = regexp.MustCompile(`\b(\d+\.?\d*(?:e[+-]?\d+)?)\b`)
var funcCallPattern = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)

func normalizeLang(lang string) string {
	lower := strings.ToLower(strings.TrimSpace(lang))
	switch lower {
	case "js", "jsx", "mjs":
		return "javascript"
	case "ts", "tsx", "mts":
		return "typescript"
	case "sh", "shell", "zsh", "bash":
		return "bash"
	case "py", "python3":
		return "python"
	case "golang":
		return "go"
	default:
		return lower
	}
}

func Highlight(code, lang string) string {
	normalizedLang := normalizeLang(lang)
	if normalizedLang == "json" {
		return highlightJSON(code)
	}
	if normalizedLang == "yaml" {
		return highlightYAML(code)
	}

	lines := strings.Split(code, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, highlightLine(line, normalizedLang))
	}
	return strings.Join(result, "\n")
}

func highlightLine(line, lang string) string {
	tokens := tokenizeLine(line, lang)
	var sb strings.Builder
	for _, tok := range tokens {
		sb.WriteString(styleToken(tok))
	}
	return sb.String()
}

func tokenizeLine(line, lang string) []Token {
	if strings.TrimSpace(line) == "" {
		return []Token{{Type: TokenPlain, Value: line}}
	}

	segments := extractComments(line, lang)
	var tokens []Token
	for _, seg := range segments {
		if seg.IsComment {
			tokens = append(tokens, Token{Type: TokenComment, Value: seg.Text})
			continue
		}
		tokens = append(tokens, tokenizeCode(seg.Text, lang)...)
	}
	return tokens
}

type codeSegment struct {
	Text      string
	IsComment bool
}

func extractComments(line, lang string) []codeSegment {
	pattern, ok := commentPatterns[lang]
	if !ok || pattern == nil {
		return []codeSegment{{Text: line, IsComment: false}}
	}

	matches := pattern.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return []codeSegment{{Text: line, IsComment: false}}
	}

	var segments []codeSegment
	lastEnd := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		if start > lastEnd {
			segments = append(segments, codeSegment{Text: line[lastEnd:start], IsComment: false})
		}
		segments = append(segments, codeSegment{Text: line[start:end], IsComment: true})
		lastEnd = end
	}
	if lastEnd < len(line) {
		segments = append(segments, codeSegment{Text: line[lastEnd:], IsComment: false})
	}
	return segments
}

func tokenizeCode(code, lang string) []Token {
	var spans []highlightSpan

	stringMatches := stringPattern.FindAllStringIndex(code, -1)
	for _, m := range stringMatches {
		spans = append(spans, highlightSpan{start: m[0], end: m[1], typ: TokenString})
	}

	numberMatches := numberPattern.FindAllStringIndex(code, -1)
	for _, m := range numberMatches {
		if !overlapsAny(m[0], m[1], spans) {
			spans = append(spans, highlightSpan{start: m[0], end: m[1], typ: TokenNumber})
		}
	}

	funcMatches := funcCallPattern.FindAllStringSubmatchIndex(code, -1)
	for _, m := range funcMatches {
		fnStart, fnEnd := m[2], m[3]
		if !overlapsAny(fnStart, fnEnd, spans) {
			spans = append(spans, highlightSpan{start: fnStart, end: fnEnd, typ: TokenFunction})
		}
	}

	if kwPattern, ok := keywordPatterns[lang]; ok {
		kwMatches := kwPattern.FindAllStringIndex(code, -1)
		for _, m := range kwMatches {
			if !overlapsAny(m[0], m[1], spans) {
				spans = append(spans, highlightSpan{start: m[0], end: m[1], typ: TokenKeyword})
			}
		}
	}

	if typePattern, ok := typePatterns[lang]; ok {
		typeMatches := typePattern.FindAllStringSubmatchIndex(code, -1)
		for _, m := range typeMatches {
			ts, te := m[2], m[3]
			if !overlapsAny(ts, te, spans) {
				spans = append(spans, highlightSpan{start: ts, end: te, typ: TokenUserType})
			}
		}
	}

	spans = mergeHighlightSpans(spans, len(code))

	var tokens []Token
	for _, s := range spans {
		tokens = append(tokens, Token{Type: s.typ, Value: code[s.start:s.end]})
	}
	return tokens
}

func overlapsAny(start, end int, spans []highlightSpan) bool {
	for _, s := range spans {
		if start < s.end && end > s.start {
			return true
		}
	}
	return false
}

func mergeHighlightSpans(spans []highlightSpan, codeLen int) []highlightSpan {
	if len(spans) == 0 {
		return []highlightSpan{{start: 0, end: codeLen, typ: TokenPlain}}
	}

	sort.Slice(spans, func(i, j int) bool {
		return spans[i].start < spans[j].start
	})

	var merged []highlightSpan
	cur := 0
	for _, s := range spans {
		if s.start > cur {
			merged = append(merged, highlightSpan{start: cur, end: s.start, typ: TokenPlain})
		}
		merged = append(merged, s)
		cur = s.end
	}
	if cur < codeLen {
		merged = append(merged, highlightSpan{start: cur, end: codeLen, typ: TokenPlain})
	}
	return merged
}

func styleToken(tok Token) string {
	switch tok.Type {
	case TokenKeyword:
		return lipgloss.NewStyle().Foreground(colorKeyword).Render(tok.Value)
	case TokenString:
		return lipgloss.NewStyle().Foreground(colorString).Render(tok.Value)
	case TokenComment:
		return lipgloss.NewStyle().Foreground(colorComment).Render(tok.Value)
	case TokenNumber:
		return lipgloss.NewStyle().Foreground(colorNumber).Render(tok.Value)
	case TokenUserType:
		return lipgloss.NewStyle().Foreground(colorType).Render(tok.Value)
	case TokenFunction:
		return lipgloss.NewStyle().Foreground(colorFunc).Render(tok.Value)
	case TokenOperator:
		return lipgloss.NewStyle().Foreground(colorOperator).Render(tok.Value)
	default:
		return tok.Value
	}
}

func highlightJSON(code string) string {
	var sb strings.Builder
	i := 0
	for i < len(code) {
		ch := code[i]
		switch ch {
		case '"':
			j := i + 1
			for j < len(code) {
				if code[j] == '\\' {
					j += 2
					continue
				}
				if code[j] == '"' {
					j++
					break
				}
				j++
			}
			raw := code[i:j]
			if isJSONKey(code, i) {
				sb.WriteString(lipgloss.NewStyle().Foreground(colorType).Render(raw))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(colorString).Render(raw))
			}
			i = j
		case ':', '{', '}', '[', ']', ',':
			sb.WriteString(lipgloss.NewStyle().Foreground(colorOperator).Render(string(ch)))
			i++
		default:
			if ch >= '0' && ch <= '9' || ch == '-' {
				j := i + 1
				for j < len(code) && ((code[j] >= '0' && code[j] <= '9') || code[j] == '.' || code[j] == 'e' || code[j] == 'E' || code[j] == '+' || code[j] == '-') {
					j++
				}
				sb.WriteString(lipgloss.NewStyle().Foreground(colorNumber).Render(code[i:j]))
				i = j
			} else if isWordChar(ch) {
				j := i + 1
				for j < len(code) && isWordChar(code[j]) {
					j++
				}
				word := code[i:j]
				switch word {
				case "true", "false", "null":
					sb.WriteString(lipgloss.NewStyle().Foreground(colorKeyword).Render(word))
				default:
					sb.WriteString(word)
				}
				i = j
			} else {
				sb.WriteByte(ch)
				i++
			}
		}
	}
	return sb.String()
}

func isJSONKey(code string, quoteIdx int) bool {
	j := quoteIdx + 1
	for j < len(code) {
		if code[j] == '\\' {
			j += 2
			continue
		}
		if code[j] == '"' {
			j++
			break
		}
		j++
	}
	for j < len(code) {
		if code[j] == ' ' || code[j] == '\t' || code[j] == '\n' || code[j] == '\r' {
			j++
			continue
		}
		return code[j] == ':'
	}
	return false
}

func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func highlightYAML(code string) string {
	lines := strings.Split(code, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, highlightYAMLLine(line))
	}
	return strings.Join(result, "\n")
}

var yamlKeyPattern = regexp.MustCompile(`^(\s*)([\w][\w.-]*)\s*:`)

func highlightYAMLLine(line string) string {
	m := yamlKeyPattern.FindStringSubmatchIndex(line)
	if m != nil {
		indent := line[m[2]:m[3]]
		key := line[m[4]:m[5]]
		colonAndRest := line[m[5]:]

		var sb strings.Builder
		sb.WriteString(indent)
		sb.WriteString(lipgloss.NewStyle().Foreground(colorType).Render(key))

		rest := colonAndRest
		commentIdx := strings.Index(rest, " #")
		if commentIdx >= 0 {
			sb.WriteString(rest[:commentIdx])
			sb.WriteString(lipgloss.NewStyle().Foreground(colorComment).Render(rest[commentIdx:]))
		} else {
			sb.WriteString(rest)
		}
		return sb.String()
	}

	commentIdx := strings.Index(line, " #")
	if commentIdx >= 0 {
		return line[:commentIdx] + lipgloss.NewStyle().Foreground(colorComment).Render(line[commentIdx:])
	}
	return line
}
