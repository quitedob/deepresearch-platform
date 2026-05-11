package paper

import (
	"unicode"
	"unicode/utf8"
)

// CountWords 计算中文字数（排除空白字符，保留汉字、字母、数字）
// 对于中文学术论文，标点符号和空格不计入字数
func CountWords(content string) int {
	count := 0
	for _, r := range content {
		// 跳过空白字符
		if unicode.IsSpace(r) {
			continue
		}
		// 跳过 Markdown 标记字符（#、*、_、`、>、-、|、~）
		if r == '#' || r == '*' || r == '_' || r == '`' || r == '>' || r == '-' || r == '|' || r == '~' {
			continue
		}
		// 跳过标点符号（中英文标点）
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		count++
	}
	return count
}

// CountWordsRaw 计算原始字符数（含标点和空格，用于兼容旧逻辑）
func CountWordsRaw(content string) int {
	return utf8.RuneCountInString(content)
}

// CountWordsExcludingSpaces 计算字数（排除空白字符）
func CountWordsExcludingSpaces(content string) int {
	count := 0
	for _, r := range content {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			count++
		}
	}
	return count
}

// EstimateChapterWords 根据总目标字数和章节权重估算每个章节的目标字数
func EstimateChapterWords(totalTargetWords int, chapters []ChapterDefinition) []ChapterDefinition {
	// 计算参考文献外的章节总权重
	totalWeight := 0.0
	for _, ch := range chapters {
		if ch.Type != "reference" && ch.Type != "keywords" {
			totalWeight += ch.Weight
		}
	}

	result := make([]ChapterDefinition, len(chapters))
	copy(result, chapters)

	for i := range result {
		if result[i].Type == "reference" || result[i].Type == "keywords" {
			continue
		}
		// 按权重分配字数
		estimated := int(float64(totalTargetWords) * (result[i].Weight / totalWeight))

		// 确保在最小最大范围内
		if estimated < result[i].MinWords {
			estimated = result[i].MinWords
		}
		if result[i].MaxWords > 0 && estimated > result[i].MaxWords {
			estimated = result[i].MaxWords
		}
		result[i].TargetWords = estimated
	}

	return result
}
