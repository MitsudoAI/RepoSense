package llm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// DescriptionService handles LLM-based description extraction
type DescriptionService struct {
	client   *Client
	language string
	logger   *logrus.Logger
	enabled  bool
}

// NewDescriptionService creates a new description service
func NewDescriptionService(provider Provider, model, apiKey, baseURL, language string, timeout time.Duration, enabled bool) *DescriptionService {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	var client *Client
	if enabled && apiKey != "" {
		client = NewClient(provider, model, apiKey, baseURL, timeout)
	}
	
	return &DescriptionService{
		client:   client,
		language: language,
		logger:   logger,
		enabled:  enabled && apiKey != "",
	}
}

// SetLogLevel sets the logging level
func (s *DescriptionService) SetLogLevel(level logrus.Level) {
	s.logger.SetLevel(level)
	if s.client != nil {
		s.client.SetLogLevel(level)
	}
}

// IsEnabled returns whether LLM description extraction is enabled
func (s *DescriptionService) IsEnabled() bool {
	return s.enabled
}

// ExtractDescription extracts project description from a repository
func (s *DescriptionService) ExtractDescription(repoPath string) string {
	if !s.enabled {
		return s.extractDescriptionFallback(repoPath)
	}
	
	// Read README content
	readmeContent := s.readREADMEContent(repoPath)
	if readmeContent == "" {
		s.logger.Debugf("No README content found for: %s", repoPath)
		return "暂无描述"
	}
	
	// Limit content length to avoid token limits
	if len(readmeContent) > 4000 {
		readmeContent = readmeContent[:4000] + "..."
	}
	
	// Generate description using LLM
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	description, err := s.client.GenerateDescription(ctx, readmeContent, s.language)
	if err != nil {
		s.logger.Warnf("Failed to generate LLM description for %s: %v", repoPath, err)
		// Fallback to simple extraction
		return s.extractDescriptionFallback(repoPath)
	}
	
	return description
}

// readREADMEContent reads and combines README file content
func (s *DescriptionService) readREADMEContent(repoPath string) string {
	readmeFiles := []string{
		"README.md",
		"README.rst",
		"README.txt",
		"README",
		"readme.md",
		"readme.rst", 
		"readme.txt",
		"readme",
		"Readme.md",
		"ReadMe.md",
	}
	
	for _, filename := range readmeFiles {
		readmePath := filepath.Join(repoPath, filename)
		if content := s.readFileContent(readmePath); content != "" {
			s.logger.Debugf("Found README file: %s", readmePath)
			return content
		}
	}
	
	return ""
}

// readFileContent reads the content of a file
func (s *DescriptionService) readFileContent(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	
	content := string(data)
	
	// Remove common markdown/HTML elements that aren't useful for description
	content = s.cleanContent(content)
	
	return content
}

// cleanContent cleans up content for better LLM processing
func (s *DescriptionService) cleanContent(content string) string {
	lines := strings.Split(content, "\n")
	var cleanedLines []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Skip lines that are mostly symbols (like horizontal rules)
		if len(line) > 0 && (strings.Count(line, "-") > len(line)/2 || 
			strings.Count(line, "=") > len(line)/2 ||
			strings.Count(line, "*") > len(line)/2) {
			continue
		}
		
		// Skip badge lines (contain ![])
		if strings.Contains(line, "![") && strings.Contains(line, "](") {
			continue
		}
		
		// Skip table of contents markers
		if strings.Contains(strings.ToLower(line), "table of contents") ||
		   strings.Contains(strings.ToLower(line), "目录") {
			continue
		}
		
		cleanedLines = append(cleanedLines, line)
		
		// Limit to first 50 lines to avoid token limits
		if len(cleanedLines) >= 50 {
			break
		}
	}
	
	return strings.Join(cleanedLines, "\n")
}

// extractDescriptionFallback provides fallback description extraction without LLM
func (s *DescriptionService) extractDescriptionFallback(repoPath string) string {
	readmeFiles := []string{
		"README.md",
		"README.rst",
		"README.txt",
		"README",
		"readme.md",
		"readme.rst",
		"readme.txt", 
		"readme",
	}
	
	for _, filename := range readmeFiles {
		readmePath := filepath.Join(repoPath, filename)
		if description := s.readFirstNonEmptyLine(readmePath); description != "" {
			return description
		}
	}
	
	return "暂无描述"
}

// readFirstNonEmptyLine reads the first meaningful line from a file
func (s *DescriptionService) readFirstNonEmptyLine(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Skip markdown headers, badges, and HTML tags
		if strings.HasPrefix(line, "#") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
		
		// Remove common markdown and HTML elements
		line = strings.TrimPrefix(line, "=")
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		
		// Skip lines that contain badges or are mostly HTML
		if strings.Contains(line, "![") || 
		   strings.Contains(line, "<img") ||
		   strings.Contains(line, "<div") ||
		   strings.Contains(line, "<p") ||
		   strings.Contains(line, "[![") {
			continue
		}
		
		if line != "" && len(line) > 3 {
			// Limit description length
			if len(line) > 100 {
				line = line[:97] + "..."
			}
			return line
		}
	}
	
	return ""
}

// ValidateConfiguration validates LLM configuration
func ValidateConfiguration(provider Provider, apiKey, baseURL string) error {
	if provider == "" {
		return fmt.Errorf("LLM provider must be specified")
	}
	
	switch provider {
	case ProviderOpenAI:
		if apiKey == "" {
			return fmt.Errorf("OpenAI API key is required")
		}
	case ProviderOpenAICompatible:
		if apiKey == "" {
			return fmt.Errorf("API key is required for OpenAI-compatible provider")
		}
		if baseURL == "" {
			return fmt.Errorf("base URL is required for OpenAI-compatible provider")
		}
	case ProviderGemini:
		if apiKey == "" {
			return fmt.Errorf("Gemini API key is required")
		}
	case ProviderClaude:
		if apiKey == "" {
			return fmt.Errorf("Claude API key is required")
		}
	case ProviderOllama:
		// Ollama doesn't require API key, but base URL is helpful
		if baseURL == "" {
			// Use default URL for Ollama
		}
	default:
		return fmt.Errorf("unsupported LLM provider: %s", provider)
	}
	
	return nil
}