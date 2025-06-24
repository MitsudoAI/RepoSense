package changelog

import (
	"time"

	"reposense/pkg/reporter"
	"reposense/pkg/scanner"
)

// AnalysisMode 分析模式
type AnalysisMode string

const (
	ModeFast AnalysisMode = "fast" // 仅 commit message
	ModeDeep AnalysisMode = "deep" // commit + 关键文件
	ModeFull AnalysisMode = "full" // 完整差分分析
)

// Commit 提交信息（兼容git log输出）
type Commit struct {
	Hash     string    `json:"hash"`
	Author   string    `json:"author"`
	Date     time.Time `json:"date"`
	Message  string    `json:"message"`
	ShortMsg string    `json:"short_message"`
}

// Summary 变更总结
type Summary struct {
	Title       string              `json:"title"`        // 一句话总结
	Highlights  []string            `json:"highlights"`   // 要点列表
	Categories  map[string][]string `json:"categories"`   // 分类总结（features, fixes, docs等）
	Language    string              `json:"language"`     // 总结语言
	GeneratedBy string              `json:"generated_by"` // 生成方式（rule-based或llm）
}

// ChangeStats 变更统计
type ChangeStats struct {
	CommitCount  int      `json:"commit_count"`
	AuthorCount  int      `json:"author_count"`
	FilesChanged int      `json:"files_changed"`
	Insertions   int      `json:"insertions"`
	Deletions    int      `json:"deletions"`
	MajorChanges []string `json:"major_changes"` // 重大变更标识
}

// TimeRange 时间范围
type TimeRange struct {
	Since time.Time `json:"since"`
	Until time.Time `json:"until"`
}

// ChangelogEntry 单个仓库的变更记录
type ChangelogEntry struct {
	Repository scanner.Repository `json:"repository"` // 复用现有的Repository结构
	TimeRange  TimeRange          `json:"time_range"`
	Commits    []Commit           `json:"commits"`
	Summary    Summary            `json:"summary"`
	Stats      ChangeStats        `json:"stats"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// ChangelogConfig 分析配置
type ChangelogConfig struct {
	Mode        AnalysisMode `json:"mode"`
	Language    string       `json:"language"`
	EnableLLM   bool         `json:"enable_llm"`
	LLMProvider string       `json:"llm_provider,omitempty"`
	LLMModel    string       `json:"llm_model,omitempty"`
}

// ChangelogReport 完整的变更日志报告
type ChangelogReport struct {
	TimeRange    TimeRange         `json:"time_range"`
	TotalRepos   int               `json:"total_repos"`
	UpdatedRepos int               `json:"updated_repos"`
	Entries      []ChangelogEntry  `json:"entries"`
	GeneratedAt  time.Time         `json:"generated_at"`
	Config       ChangelogConfig   `json:"config"`
}

// ChangelogOptions 分析选项
type ChangelogOptions struct {
	Directory       string
	Mode            AnalysisMode
	TimeRange       TimeRange
	Language        string
	EnableLLM       bool
	LLMProvider     string
	LLMModel        string
	LLMAPIKey       string
	LLMBaseURL      string
	LLMTimeout      time.Duration
	IncludePatterns []string
	ExcludePatterns []string
	OutputFormat    reporter.ReportFormat
	SaveReport      bool
	ReportFile      string
	WorkerCount     int
	Timeout         time.Duration
	Verbose         bool
}