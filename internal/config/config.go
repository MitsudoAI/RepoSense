package config

import (
	"time"

	"reposense/pkg/reporter"
)

// Config holds the application configuration
type Config struct {
	// Global settings
	WorkerCount  int                   `json:"worker_count"`
	Timeout      time.Duration         `json:"timeout"`
	Verbose      bool                  `json:"verbose"`
	DryRun       bool                  `json:"dry_run"`
	OutputFormat reporter.ReportFormat `json:"output_format"`
	
	// Filtering options
	IncludePatterns []string `json:"include_patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
	
	// Sorting options
	SortByTime bool `json:"sort_by_time"`
	Reverse    bool `json:"reverse"`
	
	// LLM options
	EnableLLM     bool   `json:"enable_llm"`
	LLMProvider   string `json:"llm_provider"`
	LLMModel      string `json:"llm_model"`
	LLMAPIKey     string `json:"llm_api_key"`
	LLMBaseURL    string `json:"llm_base_url"`
	LLMLanguage   string `json:"llm_language"`
	LLMTimeout    time.Duration `json:"llm_timeout"`
	
	// Output options
	SaveReport   bool   `json:"save_report"`
	ReportFile   string `json:"report_file"`
	LogLevel     string `json:"log_level"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		WorkerCount:     10,
		Timeout:         30 * time.Second,
		Verbose:         false,
		DryRun:          false,
		OutputFormat:    reporter.FormatText,
		IncludePatterns: []string{},
		ExcludePatterns: []string{},
		SortByTime:      false,
		Reverse:         false,
		EnableLLM:       false,
		LLMProvider:     "openai",
		LLMModel:        "gpt-4o-mini",
		LLMAPIKey:       "",
		LLMBaseURL:      "",
		LLMLanguage:     "zh",
		LLMTimeout:      10 * time.Second,
		SaveReport:      false,
		ReportFile:      "",
		LogLevel:        "info",
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.WorkerCount <= 0 {
		c.WorkerCount = 10
	}
	
	if c.WorkerCount > 50 {
		c.WorkerCount = 50
	}
	
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	
	// 验证输出格式
	switch c.OutputFormat {
	case reporter.FormatTable, reporter.FormatJSON, reporter.FormatText:
		// 有效格式
	default:
		c.OutputFormat = reporter.FormatText
	}
	
	return nil
}