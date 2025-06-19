package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
		OutputFormat:    reporter.FormatTable,
		IncludePatterns: []string{},
		ExcludePatterns: []string{},
		SortByTime:      false,
		Reverse:         false,
		EnableLLM:       true,
		LLMProvider:     "gemini",
		LLMModel:        "gemini-2.5-flash",
		LLMAPIKey:       "",
		LLMBaseURL:      "",
		LLMLanguage:     "zh",
		LLMTimeout:      30 * time.Second,
		SaveReport:      false,
		ReportFile:      "",
		LogLevel:        "info",
	}
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".reposense.json"
	}
	return filepath.Join(homeDir, ".reposense.json")
}

// LoadConfig loads configuration from file and merges with defaults
func LoadConfig() *Config {
	cfg := DefaultConfig()
	
	configPath := GetConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		var fileConfig Config
		if err := json.Unmarshal(data, &fileConfig); err == nil {
			mergeConfig(cfg, &fileConfig)
		}
	}
	
	return cfg
}

// SaveConfig saves configuration to file
func (c *Config) SaveConfig() error {
	configPath := GetConfigPath()
	
	// 创建目录
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}
	
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}
	
	return nil
}

// mergeConfig merges file config into default config, only overriding non-zero values
func mergeConfig(dst, src *Config) {
	if src.WorkerCount != 0 {
		dst.WorkerCount = src.WorkerCount
	}
	if src.Timeout != 0 {
		dst.Timeout = src.Timeout
	}
	if src.Verbose {
		dst.Verbose = src.Verbose
	}
	if src.DryRun {
		dst.DryRun = src.DryRun
	}
	if src.OutputFormat != "" {
		dst.OutputFormat = src.OutputFormat
	}
	if len(src.IncludePatterns) > 0 {
		dst.IncludePatterns = src.IncludePatterns
	}
	if len(src.ExcludePatterns) > 0 {
		dst.ExcludePatterns = src.ExcludePatterns
	}
	if src.SortByTime {
		dst.SortByTime = src.SortByTime
	}
	if src.Reverse {
		dst.Reverse = src.Reverse
	}
	
	// LLM settings - always use file values if present
	if src.EnableLLM != dst.EnableLLM {
		dst.EnableLLM = src.EnableLLM
	}
	if src.LLMProvider != "" {
		dst.LLMProvider = src.LLMProvider
	}
	if src.LLMModel != "" {
		dst.LLMModel = src.LLMModel
	}
	if src.LLMAPIKey != "" {
		dst.LLMAPIKey = src.LLMAPIKey
	}
	if src.LLMBaseURL != "" {
		dst.LLMBaseURL = src.LLMBaseURL
	}
	if src.LLMLanguage != "" {
		dst.LLMLanguage = src.LLMLanguage
	}
	if src.LLMTimeout != 0 {
		dst.LLMTimeout = src.LLMTimeout
	}
	
	if src.SaveReport {
		dst.SaveReport = src.SaveReport
	}
	if src.ReportFile != "" {
		dst.ReportFile = src.ReportFile
	}
	if src.LogLevel != "" {
		dst.LogLevel = src.LogLevel
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