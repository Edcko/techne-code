package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
)

type SegmentType int

const (
	SegmentText SegmentType = iota
	SegmentHeader
	SegmentBold
	SegmentItalic
	SegmentCodeBlock
	SegmentInlineCode
	SegmentListItem
	SegmentNumberedItem
	SegmentBlockquote
	SegmentLink
	SegmentCodeBlockStart
	SegmentCodeBlockEnd
	SegmentHorizontalRule
)

type Segment struct {
	Type     SegmentType
	Content  string
	Language string
	Level    int
	Alt      string
	URL      string
	Number   int
}

var (
	headerColor      = lipgloss.Color("#BD93F9")
	boldColor        = lipgloss.Color("#F8F8F2")
	italicColor      = lipgloss.Color("#F1FA8C")
	inlineCodeBg     = lipgloss.Color("#44475A")
	inlineCodeFg     = lipgloss.Color("#F8F8F2")
	codeBlockBorder  = lipgloss.Color("#6272A4")
	quoteColor       = lipgloss.Color("#6272A4")
	quotePrefixColor = lipgloss.Color("#BD93F9")
	linkColor        = lipgloss.Color("#8BE9FD")
	listPrefixColor  = lipgloss.Color("#FF79C6")
	hdrColor         = lipgloss.Color("#6272A4")

	headerStyle      = lipgloss.NewStyle().Foreground(headerColor).Bold(true)
	h2Style          = lipgloss.NewStyle().Foreground(headerColor).Bold(true)
	h3Style          = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Bold(true)
	boldStyle        = lipgloss.NewStyle().Foreground(boldColor).Bold(true)
	italicStyle      = lipgloss.NewStyle().Foreground(italicColor).Italic(true)
	inlineCodeStyle  = lipgloss.NewStyle().Foreground(inlineCodeFg).Background(inlineCodeBg)
	codeBlockStyle   = lipgloss.NewStyle().Foreground(codeBlockBorder)
	quoteStyle       = lipgloss.NewStyle().Foreground(quoteColor)
	quotePrefixStyle = lipgloss.NewStyle().Foreground(quotePrefixColor)
	linkStyle        = lipgloss.NewStyle().Foreground(linkColor).Underline(true)
	listPrefixStyle  = lipgloss.NewStyle().Foreground(listPrefixColor)
	hdrStyle         = lipgloss.NewStyle().Foreground(hdrColor)
	thinkingStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Italic(true)
)

var (
	codeBlockRe  = regexp.MustCompile("(?s)(```)(\\w*)\n(.*?)```")
	inlineCodeRe = regexp.MustCompile("`([^`]+)`")
	boldRe       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe     = regexp.MustCompile(`\*(.+?)\*`)
	linkRe       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func Parse(input string) []Segment {
	var segments []Segment

	inCodeBlock := false
	var codeBlockLang string
	var codeBlockContent strings.Builder

	lines := strings.Split(input, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if inCodeBlock {
			if strings.HasPrefix(line, "```") {
				segments = append(segments, Segment{
					Type:     SegmentCodeBlock,
					Content:  strings.TrimRight(codeBlockContent.String(), "\n"),
					Language: codeBlockLang,
				})
				inCodeBlock = false
				codeBlockContent.Reset()
				continue
			}
			if codeBlockContent.Len() > 0 {
				codeBlockContent.WriteByte('\n')
			}
			codeBlockContent.WriteString(line)
			continue
		}

		if strings.HasPrefix(line, "```") {
			inCodeBlock = true
			codeBlockLang = strings.TrimPrefix(line, "```")
			continue
		}

		if strings.HasPrefix(line, "---") && len(strings.TrimLeft(line, "-")) == 0 {
			segments = append(segments, Segment{Type: SegmentHorizontalRule})
			continue
		}

		if strings.HasPrefix(line, "### ") {
			segments = append(segments, Segment{
				Type:    SegmentHeader,
				Content: line[4:],
				Level:   3,
			})
			continue
		}
		if strings.HasPrefix(line, "## ") {
			segments = append(segments, Segment{
				Type:    SegmentHeader,
				Content: line[3:],
				Level:   2,
			})
			continue
		}
		if strings.HasPrefix(line, "# ") {
			content := strings.TrimSpace(line[2:])
			if content == "" {
				continue
			}
			segments = append(segments, Segment{
				Type:    SegmentHeader,
				Content: content,
				Level:   1,
			})
			continue
		}

		if strings.HasPrefix(line, "> ") {
			segments = append(segments, Segment{
				Type:    SegmentBlockquote,
				Content: line[2:],
			})
			continue
		}

		if isUnorderedListItem(line) {
			prefixLen := unorderedListPrefixLen(line)
			segments = append(segments, Segment{
				Type:    SegmentListItem,
				Content: line[prefixLen:],
			})
			continue
		}

		if numPrefix := orderedListPrefix(line); numPrefix > 0 {
			segments = append(segments, Segment{
				Type:    SegmentNumberedItem,
				Content: line[numPrefix:],
				Number:  numPrefix,
			})
			continue
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		segments = append(segments, Segment{
			Type:    SegmentText,
			Content: line,
		})
	}

	if inCodeBlock {
		segments = append(segments, Segment{
			Type:     SegmentCodeBlock,
			Content:  strings.TrimRight(codeBlockContent.String(), "\n"),
			Language: codeBlockLang,
		})
	}

	return segments
}

func isUnorderedListItem(line string) bool {
	trimmed := line
	for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t') {
		trimmed = trimmed[1:]
	}
	return strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ")
}

func unorderedListPrefixLen(line string) int {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return i + 2
}

func orderedListPrefix(line string) int {
	trimmed := line
	space := 0
	for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t') {
		trimmed = trimmed[1:]
		space++
	}
	i := 0
	for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0
	}
	if i < len(trimmed) && trimmed[i] == '.' && i+1 < len(trimmed) && trimmed[i+1] == ' ' {
		return space + i + 2
	}
	return 0
}

func Render(input string) string {
	segments := Parse(input)
	var sb strings.Builder
	for _, seg := range segments {
		sb.WriteString(renderSegment(seg))
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderSegment(seg Segment) string {
	switch seg.Type {
	case SegmentHeader:
		return renderHeader(seg)
	case SegmentBold:
		return boldStyle.Render(seg.Content)
	case SegmentItalic:
		return italicStyle.Render(seg.Content)
	case SegmentCodeBlock:
		return renderCodeBlock(seg)
	case SegmentInlineCode:
		return inlineCodeStyle.Render(seg.Content)
	case SegmentListItem:
		prefix := listPrefixStyle.Render("  • ")
		return prefix + renderInline(seg.Content)
	case SegmentNumberedItem:
		prefix := listPrefixStyle.Render(fmt.Sprintf("  %d. ", seg.Number))
		return prefix + renderInline(seg.Content)
	case SegmentBlockquote:
		prefix := quotePrefixStyle.Render("  │ ")
		return prefix + quoteStyle.Render(renderInline(seg.Content))
	case SegmentLink:
		return linkStyle.Render(seg.Content)
	case SegmentHorizontalRule:
		return hdrStyle.Render("────────────────────────────────────────")
	case SegmentText:
		return renderInline(seg.Content)
	default:
		return seg.Content
	}
}

func renderHeader(seg Segment) string {
	var style lipgloss.Style
	switch seg.Level {
	case 1:
		style = headerStyle
	case 2:
		style = h2Style
	case 3:
		style = h3Style
	default:
		style = headerStyle
	}
	content := renderInline(seg.Content)
	prefix := ""
	switch seg.Level {
	case 1:
		prefix = "▓ "
	case 2:
		prefix = "▒ "
	case 3:
		prefix = "░ "
	}
	return style.Render(prefix + stripInlineANSI(content))
}

func renderCodeBlock(seg Segment) string {
	highlighted := Highlight(seg.Content, seg.Language)

	var sb strings.Builder
	langLabel := ""
	if seg.Language != "" {
		langLabel = " " + seg.Language
	}
	sb.WriteString(codeBlockStyle.Render("┌" + langLabel + strings.Repeat("─", max(40-len(langLabel), 1))))
	lines := strings.Split(highlighted, "\n")
	for _, line := range lines {
		sb.WriteByte('\n')
		sb.WriteString(codeBlockStyle.Render("│ "))
		sb.WriteString(line)
	}
	sb.WriteByte('\n')
	sb.WriteString(codeBlockStyle.Render("└" + strings.Repeat("─", 40)))
	return sb.String()
}

func renderInline(text string) string {
	text = inlineCodeRe.ReplaceAllStringFunc(text, func(match string) string {
		inner := inlineCodeRe.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		return inlineCodeStyle.Render(inner[1])
	})

	text = boldRe.ReplaceAllStringFunc(text, func(match string) string {
		inner := boldRe.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		return boldStyle.Render(inner[1])
	})

	text = italicRe.ReplaceAllStringFunc(text, func(match string) string {
		inner := italicRe.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		return italicStyle.Render(inner[1])
	})

	text = linkRe.ReplaceAllStringFunc(text, func(match string) string {
		inner := linkRe.FindStringSubmatch(match)
		if len(inner) < 3 {
			return match
		}
		return linkStyle.Render(inner[1]) + " (" + linkStyle.Render(inner[2]) + ")"
	})

	return text
}

func RenderThinking(text string) string {
	if text == "" {
		return ""
	}
	return thinkingStyle.Render("💭 " + text)
}

func stripInlineANSI(s string) string {
	var sb strings.Builder
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
		sb.WriteByte(s[i])
	}
	return sb.String()
}
