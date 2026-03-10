package paper

import (
	"fmt"
	"strings"
)

// FormatCitation 根据引用格式生成引用文本
func FormatCitation(style CitationStyle, title, authors, url, doi string, year int) string {
	switch style {
	case CitationStyleChineseGB:
		return formatChineseGB(title, authors, url, doi, year)
	case CitationStyleAPA:
		return formatAPA(title, authors, url, doi, year)
	case CitationStyleMLA:
		return formatMLA(title, authors, url, doi, year)
	case CitationStyleLaTeX:
		return formatLaTeX(title, authors, url, doi, year)
	default:
		return formatChineseGB(title, authors, url, doi, year)
	}
}

// formatChineseGB GB/T 7714 国标格式
func formatChineseGB(title, authors, url, doi string, year int) string {
	var parts []string

	if authors != "" {
		parts = append(parts, authors)
	}

	if title != "" {
		parts = append(parts, title)
	}

	if year > 0 {
		parts = append(parts, fmt.Sprintf("[J]. %d", year))
	}

	if doi != "" {
		parts = append(parts, fmt.Sprintf("DOI: %s", doi))
	} else if url != "" {
		parts = append(parts, fmt.Sprintf("[EB/OL]. %s", url))
	}

	return strings.Join(parts, ". ")
}

// formatAPA APA格式
func formatAPA(title, authors, url, doi string, year int) string {
	var parts []string

	if authors != "" {
		parts = append(parts, authors)
	}

	if year > 0 {
		parts = append(parts, fmt.Sprintf("(%d)", year))
	}

	if title != "" {
		parts = append(parts, title)
	}

	if doi != "" {
		parts = append(parts, fmt.Sprintf("https://doi.org/%s", doi))
	} else if url != "" {
		parts = append(parts, fmt.Sprintf("Retrieved from %s", url))
	}

	return strings.Join(parts, ". ")
}

// formatMLA MLA格式
func formatMLA(title, authors, url, doi string, year int) string {
	var parts []string

	if authors != "" {
		parts = append(parts, authors)
	}

	if title != "" {
		parts = append(parts, fmt.Sprintf("\"%s\"", title))
	}

	if year > 0 {
		parts = append(parts, fmt.Sprintf("%d", year))
	}

	if url != "" {
		parts = append(parts, url)
	}

	return strings.Join(parts, ". ") + "."
}

// formatLaTeX LaTeX/BibTeX格式
func formatLaTeX(title, authors, url, doi string, year int) string {
	// 生成citation key
	key := "ref"
	if authors != "" {
		parts := strings.Split(authors, ",")
		key = strings.TrimSpace(parts[0])
		key = strings.ReplaceAll(key, " ", "")
	}
	if year > 0 {
		key = fmt.Sprintf("%s%d", key, year)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("@article{%s,\n", key))
	if authors != "" {
		sb.WriteString(fmt.Sprintf("  author = {%s},\n", authors))
	}
	if title != "" {
		sb.WriteString(fmt.Sprintf("  title = {%s},\n", title))
	}
	if year > 0 {
		sb.WriteString(fmt.Sprintf("  year = {%d},\n", year))
	}
	if doi != "" {
		sb.WriteString(fmt.Sprintf("  doi = {%s},\n", doi))
	}
	if url != "" {
		sb.WriteString(fmt.Sprintf("  url = {%s},\n", url))
	}
	sb.WriteString("}")

	return sb.String()
}
