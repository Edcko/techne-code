package diff

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
	LineEllipsis
)

type DiffLine struct {
	Type    LineType
	Content string
	OldNo   int
	NewNo   int
}

var (
	addedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	removedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
	contextStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#5B6071"))
	ellipsisStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	pathStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true)
)

const maxDisplayLines = 50
const contextRadius = 3

func Generate(oldContent, newContent string) []DiffLine {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	ops := computeEditScript(oldLines, newLines)
	return buildDiffLines(ops, oldLines, newLines)
}

func Render(oldContent, newContent, filePath string, isNewFile bool) string {
	if isNewFile {
		return renderNewFile(newContent, filePath)
	}
	if oldContent == newContent {
		return pathStyle.Render(filePath) + " " + contextStyle.Render("(no changes)")
	}
	lines := Generate(oldContent, newContent)
	lines = compact(lines, maxDisplayLines, contextRadius)
	return renderDiffLines(lines, filePath)
}

func renderNewFile(content, filePath string) string {
	lines := splitLines(content)
	var sb strings.Builder
	sb.WriteString(pathStyle.Render("+++ " + filePath))
	sb.WriteString("\n")

	shown := lines
	truncated := false
	if len(shown) > maxDisplayLines {
		shown = shown[:maxDisplayLines]
		truncated = true
	}

	for _, line := range shown {
		sb.WriteString(addedStyle.Render("+" + line))
		sb.WriteString("\n")
	}

	if truncated {
		remaining := len(lines) - maxDisplayLines
		sb.WriteString(ellipsisStyle.Render(fmt.Sprintf("  ... %d more lines", remaining)))
	}

	return strings.TrimRight(sb.String(), "\n")
}

func renderDiffLines(lines []DiffLine, filePath string) string {
	var sb strings.Builder
	sb.WriteString(pathStyle.Render("--- " + filePath))
	sb.WriteString("\n")
	sb.WriteString(pathStyle.Render("+++ " + filePath))
	sb.WriteString("\n")

	for _, dl := range lines {
		switch dl.Type {
		case LineAdded:
			sb.WriteString(addedStyle.Render("+" + dl.Content))
		case LineRemoved:
			sb.WriteString(removedStyle.Render("-" + dl.Content))
		case LineContext:
			sb.WriteString(contextStyle.Render(" " + dl.Content))
		case LineEllipsis:
			sb.WriteString(ellipsisStyle.Render(dl.Content))
		}
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimRight(s, "\n")
	return strings.Split(s, "\n")
}

type editOp int

const (
	opKeep editOp = iota
	opInsert
	opDelete
)

func computeEditScript(old, new []string) []editOp {
	m, n := len(old), len(new)

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if old[i-1] == new[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var ops []editOp
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && old[i-1] == new[j-1] {
			ops = append(ops, opKeep)
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, opInsert)
			j--
		} else {
			ops = append(ops, opDelete)
			i--
		}
	}

	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}

	return ops
}

func buildDiffLines(ops []editOp, old, new []string) []DiffLine {
	var lines []DiffLine
	oldIdx, newIdx := 0, 0

	for _, op := range ops {
		switch op {
		case opKeep:
			lines = append(lines, DiffLine{
				Type:    LineContext,
				Content: old[oldIdx],
				OldNo:   oldIdx + 1,
				NewNo:   newIdx + 1,
			})
			oldIdx++
			newIdx++
		case opDelete:
			lines = append(lines, DiffLine{
				Type:    LineRemoved,
				Content: old[oldIdx],
				OldNo:   oldIdx + 1,
			})
			oldIdx++
		case opInsert:
			lines = append(lines, DiffLine{
				Type:    LineAdded,
				Content: new[newIdx],
				NewNo:   newIdx + 1,
			})
			newIdx++
		}
	}

	return lines
}

func compact(lines []DiffLine, maxLines int, context int) []DiffLine {
	if len(lines) <= maxLines {
		return lines
	}

	changed := make([]int, 0)
	for i, l := range lines {
		if l.Type != LineContext {
			changed = append(changed, i)
		}
	}

	if len(changed) == 0 {
		return lines[:maxLines]
	}

	keep := make([]bool, len(lines))
	for _, idx := range changed {
		start := idx - context
		if start < 0 {
			start = 0
		}
		end := idx + context + 1
		if end > len(lines) {
			end = len(lines)
		}
		for i := start; i < end; i++ {
			keep[i] = true
		}
	}

	var result []DiffLine
	prevKept := false
	omitted := 0

	for i, l := range lines {
		if keep[i] {
			if !prevKept && len(result) > 0 {
				result = append(result, DiffLine{
					Type:    LineEllipsis,
					Content: fmt.Sprintf("  ... %d lines omitted ...", omitted),
				})
				omitted = 0
			}
			result = append(result, l)
			prevKept = true
		} else {
			omitted++
			prevKept = false
		}
	}

	if len(result) > maxLines {
		remaining := len(result) - maxLines
		result = result[:maxLines]
		result = append(result, DiffLine{
			Type:    LineEllipsis,
			Content: fmt.Sprintf("  ... %d more lines", remaining),
		})
	}

	return result
}
