package paper

// PaperType 论文类型
type PaperType string

const (
	PaperTypeLiberalArts PaperType = "liberal_arts"
	PaperTypeScience     PaperType = "science"
)

// ChapterDefinition 章节定义
type ChapterDefinition struct {
	Type        string  `json:"type"`         // 章节类型标识
	Title       string  `json:"title"`        // 章节中文标题
	MinWords    int     `json:"min_words"`    // 最小字数
	MaxWords    int     `json:"max_words"`    // 最大字数
	Weight      float64 `json:"weight"`       // 字数权重（用于分配目标字数）
	TargetWords int     `json:"target_words"` // 目标字数（动态计算）
	SortOrder   int     `json:"sort_order"`   // 排序
	Description string  `json:"description"`  // 说明
}

// PaperTemplate 论文模板
type PaperTemplate struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Type        PaperType           `json:"type"`
	Description string              `json:"description"`
	Chapters    []ChapterDefinition `json:"chapters"`
}

// CitationStyle 引用格式
type CitationStyle string

const (
	CitationStyleChineseGB CitationStyle = "chinese-gb" // GB/T 7714 国标
	CitationStyleAPA       CitationStyle = "apa"         // APA格式
	CitationStyleMLA       CitationStyle = "mla"         // MLA格式
	CitationStyleLaTeX     CitationStyle = "latex"       // LaTeX/BibTeX格式
)

// SupportedCitationStyles 支持的引用格式列表
var SupportedCitationStyles = []CitationStyle{
	CitationStyleChineseGB,
	CitationStyleAPA,
	CitationStyleMLA,
	CitationStyleLaTeX,
}

// TemplateManager 模板管理器
type TemplateManager struct {
	templates map[string]*PaperTemplate
}

// NewTemplateManager 创建模板管理器
func NewTemplateManager() *TemplateManager {
	mgr := &TemplateManager{
		templates: make(map[string]*PaperTemplate),
	}
	mgr.registerDefaults()
	return mgr
}

// registerDefaults 注册默认模板
func (m *TemplateManager) registerDefaults() {
	m.templates["liberal_arts"] = m.liberalArtsTemplate()
	m.templates["science"] = m.scienceTemplate()
}

// GetTemplate 获取模板
func (m *TemplateManager) GetTemplate(paperType string) (*PaperTemplate, bool) {
	t, ok := m.templates[paperType]
	return t, ok
}

// GetAllTemplates 获取所有模板
func (m *TemplateManager) GetAllTemplates() []*PaperTemplate {
	result := make([]*PaperTemplate, 0, len(m.templates))
	for _, t := range m.templates {
		result = append(result, t)
	}
	return result
}

// GetChaptersForPaper 根据论文类型和目标字数获取调整后的章节定义
func (m *TemplateManager) GetChaptersForPaper(paperType string, targetWords int) ([]ChapterDefinition, error) {
	tmpl, ok := m.GetTemplate(paperType)
	if !ok {
		// 默认使用文科模板
		tmpl = m.templates["liberal_arts"]
	}
	return EstimateChapterWords(targetWords, tmpl.Chapters), nil
}

// liberalArtsTemplate 文科论文模板
func (m *TemplateManager) liberalArtsTemplate() *PaperTemplate {
	return &PaperTemplate{
		ID:          "liberal_arts",
		Name:        "文科论文模板",
		Type:        PaperTypeLiberalArts,
		Description: "适用于人文社科类论文，包含理论框架和分析论证章节",
		Chapters: []ChapterDefinition{
			{
				Type:        "abstract",
				Title:       "摘要",
				MinWords:    200,
				MaxWords:    500,
				Weight:      0.05,
				SortOrder:   1,
				Description: "研究概要，包含研究目的、方法、结论",
			},
			{
				Type:        "keywords",
				Title:       "关键词",
				MinWords:    20,
				MaxWords:    100,
				Weight:      0.01,
				SortOrder:   2,
				Description: "3-5个关键词",
			},
			{
				Type:        "intro",
				Title:       "引言",
				MinWords:    500,
				MaxWords:    1500,
				Weight:      0.10,
				SortOrder:   3,
				Description: "研究背景、问题提出、研究意义和研究方法概述",
			},
			{
				Type:        "lit_review",
				Title:       "文献综述",
				MinWords:    1500,
				MaxWords:    4000,
				Weight:      0.25,
				SortOrder:   4,
				Description: "国内外研究现状、理论梳理和文献评价",
			},
			{
				Type:        "theoretical_framework",
				Title:       "理论框架",
				MinWords:    800,
				MaxWords:    2000,
				Weight:      0.15,
				SortOrder:   5,
				Description: "核心概念界定、理论基础和分析框架",
			},
			{
				Type:        "analysis",
				Title:       "分析论证",
				MinWords:    2000,
				MaxWords:    6000,
				Weight:      0.30,
				SortOrder:   6,
				Description: "核心分析、案例论证和深入讨论",
			},
			{
				Type:        "conclusion",
				Title:       "结论",
				MinWords:    500,
				MaxWords:    1500,
				Weight:      0.10,
				SortOrder:   7,
				Description: "研究总结、主要发现、研究局限和未来展望",
			},
			{
				Type:        "reference",
				Title:       "参考文献",
				MinWords:    0,
				MaxWords:    0,
				Weight:      0.0,
				SortOrder:   8,
				Description: "引用文献列表",
			},
		},
	}
}

// scienceTemplate 理科论文模板
func (m *TemplateManager) scienceTemplate() *PaperTemplate {
	return &PaperTemplate{
		ID:          "science",
		Name:        "理科论文模板",
		Type:        PaperTypeScience,
		Description: "适用于自然科学和工程技术类论文，包含实验方法和结果讨论章节",
		Chapters: []ChapterDefinition{
			{
				Type:        "abstract",
				Title:       "摘要",
				MinWords:    200,
				MaxWords:    500,
				Weight:      0.05,
				SortOrder:   1,
				Description: "研究概要，包含研究目的、方法、结果和结论",
			},
			{
				Type:        "keywords",
				Title:       "关键词",
				MinWords:    20,
				MaxWords:    100,
				Weight:      0.01,
				SortOrder:   2,
				Description: "3-5个关键词",
			},
			{
				Type:        "intro",
				Title:       "引言",
				MinWords:    500,
				MaxWords:    1500,
				Weight:      0.10,
				SortOrder:   3,
				Description: "研究背景、文献回顾和研究目标",
			},
			{
				Type:        "lit_review",
				Title:       "文献综述",
				MinWords:    1000,
				MaxWords:    3000,
				Weight:      0.18,
				SortOrder:   4,
				Description: "相关研究综述和技术发展现状",
			},
			{
				Type:        "method",
				Title:       "研究方法",
				MinWords:    800,
				MaxWords:    2500,
				Weight:      0.15,
				SortOrder:   5,
				Description: "实验设计、数据来源、研究方法和技术路线",
			},
			{
				Type:        "result",
				Title:       "研究结果",
				MinWords:    1500,
				MaxWords:    4000,
				Weight:      0.22,
				SortOrder:   6,
				Description: "实验数据呈现、结果分析和图表说明",
			},
			{
				Type:        "discussion",
				Title:       "讨论",
				MinWords:    1000,
				MaxWords:    3000,
				Weight:      0.18,
				SortOrder:   7,
				Description: "结果解释、与已有研究对比和研究意义讨论",
			},
			{
				Type:        "conclusion",
				Title:       "结论",
				MinWords:    300,
				MaxWords:    1000,
				Weight:      0.08,
				SortOrder:   8,
				Description: "研究总结、主要贡献和未来工作展望",
			},
			{
				Type:        "reference",
				Title:       "参考文献",
				MinWords:    0,
				MaxWords:    0,
				Weight:      0.0,
				SortOrder:   9,
				Description: "引用文献列表",
			},
		},
	}
}

// GetSearchStrategy 获取章节的搜索策略
func GetSearchStrategy(chapterType string) []string {
	switch chapterType {
	case "lit_review":
		return []string{"arxiv_search", "web_search"}
	case "method":
		return []string{"arxiv_search"}
	case "intro":
		return []string{"wikipedia", "web_search"}
	case "analysis", "result":
		return []string{"web_search", "web_reader"}
	case "theoretical_framework":
		return []string{"web_search", "arxiv_search"}
	case "discussion":
		return []string{"web_search", "arxiv_search"}
	default:
		return []string{"web_search"}
	}
}
