package analyzer

import "time"

// LanguageInfo represents programming language information
type LanguageInfo struct {
	Name        string  `json:"name"`
	Percentage  float64 `json:"percentage"`
	LinesOfCode int     `json:"lines_of_code"`
	FileCount   int     `json:"file_count"`
	BytesCount  int     `json:"bytes_count"`
}

// FrameworkInfo represents framework information
type FrameworkInfo struct {
	Name             string  `json:"name"`
	Version          string  `json:"version"`
	Category         string  `json:"category"`       // frontend, backend, mobile, desktop, testing等
	Confidence       float64 `json:"confidence"`     // 检测置信度 0-1
	DetectionMethod  string  `json:"detection_method"` // 检测方法说明
}

// LicenseInfo represents license information
type LicenseInfo struct {
	Name        string  `json:"name"`
	Key         string  `json:"key"`          // SPDX标识符
	Type        string  `json:"type"`         // permissive, copyleft, proprietary等
	SourceFile  string  `json:"source_file"`  // 检测到的文件路径
	Confidence  float64 `json:"confidence"`   // 检测置信度 0-1
}

// DependencyInfo represents dependency information
type DependencyInfo struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Type           string `json:"type"`            // production, development, peer等
	PackageManager string `json:"package_manager"` // npm, pip, maven等
	SourceFile     string `json:"source_file"`     // 来源文件
}

// ProjectMetadata represents comprehensive project metadata
type ProjectMetadata struct {
	ProjectType       string           `json:"project_type"`        // library, application, cli-tool等
	MainLanguage      string           `json:"main_language"`       // 主要编程语言
	Languages         []LanguageInfo   `json:"languages"`           // 所有语言信息
	Frameworks        []FrameworkInfo  `json:"frameworks"`          // 框架信息
	Licenses          []LicenseInfo    `json:"licenses"`            // 许可证信息
	Dependencies      []DependencyInfo `json:"dependencies"`        // 依赖信息
	TotalLinesOfCode  int              `json:"total_lines_of_code"` // 总代码行数
	FileCount         int              `json:"file_count"`          // 文件总数
	DirectoryCount    int              `json:"directory_count"`     // 目录总数
	RepositorySize    int64            `json:"repository_size"`     // 仓库大小（字节）
	HasReadme         bool             `json:"has_readme"`          // 是否有README
	HasLicense        bool             `json:"has_license"`         // 是否有LICENSE
	HasTests          bool             `json:"has_tests"`           // 是否有测试
	HasCI             bool             `json:"has_ci"`              // 是否有CI配置
	HasDocs           bool             `json:"has_docs"`            // 是否有文档
	ComplexityScore   float64          `json:"complexity_score"`    // 复杂度评分
	QualityScore      float64          `json:"quality_score"`       // 质量评分
	StructureHash     string           `json:"structure_hash"`      // 项目结构哈希值
	Description       string           `json:"description"`         // 项目描述
	EnhancedDescription string         `json:"enhanced_description"` // LLM增强的项目描述
	AnalyzedAt        time.Time        `json:"analyzed_at"`         // 分析时间
}

// DetectionResult represents the result of a detection operation
type DetectionResult struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Confidence float64 `json:"confidence"`
}

// AnalysisConfig represents configuration for analysis
type AnalysisConfig struct {
	IncludeLanguages     bool     `json:"include_languages"`      // 是否包含语言检测
	IncludeFrameworks    bool     `json:"include_frameworks"`     // 是否包含框架检测
	IncludeLicenses      bool     `json:"include_licenses"`       // 是否包含许可证检测
	IncludeDependencies  bool     `json:"include_dependencies"`   // 是否包含依赖检测
	IgnorePatterns       []string `json:"ignore_patterns"`        // 忽略的文件模式
	MaxFileSize          int64    `json:"max_file_size"`          // 最大文件大小（字节）
	MaxFiles             int      `json:"max_files"`              // 最大文件数量
	DeepAnalysis         bool     `json:"deep_analysis"`          // 是否进行深度分析
}

// DefaultAnalysisConfig returns default analysis configuration
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		IncludeLanguages:    true,
		IncludeFrameworks:   true,
		IncludeLicenses:     true,
		IncludeDependencies: true,
		IgnorePatterns: []string{
			"node_modules/*",
			".git/*",
			"vendor/*",
			"build/*",
			"dist/*",
			"target/*",
			"*.min.js",
			"*.min.css",
		},
		MaxFileSize:  1024 * 1024, // 1MB
		MaxFiles:     10000,
		DeepAnalysis: false,
	}
}