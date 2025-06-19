package scanner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reposense/pkg/cache"

	"github.com/sirupsen/logrus"
)

// CachedScanner handles repository discovery with description caching
type CachedScanner struct {
	logger       *logrus.Logger
	cacheManager *cache.Manager
}

// NewCachedScanner creates a new Scanner instance with cache support
func NewCachedScanner(cacheManager *cache.Manager) *CachedScanner {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	return &CachedScanner{
		logger:       logger,
		cacheManager: cacheManager,
	}
}

// SetLogLevel sets the logging level
func (cs *CachedScanner) SetLogLevel(level logrus.Level) {
	cs.logger.SetLevel(level)
	if cs.cacheManager != nil {
		cs.cacheManager.SetLogLevel(level)
	}
}

// ScanDirectoryWithDescription scans directory and extracts descriptions using cache
func (cs *CachedScanner) ScanDirectoryWithDescription(rootPath string, includePatterns, excludePatterns []string, llmProvider, llmModel, llmLanguage string) ([]RepositoryWithDescription, error) {
	// 首先使用普通scanner获取仓库列表
	basicScanner := NewScanner()
	repositories, err := basicScanner.ScanDirectoryWithFilter(rootPath, includePatterns, excludePatterns)
	if err != nil {
		return nil, err
	}
	
	var reposWithDesc []RepositoryWithDescription
	for _, repo := range repositories {
		repoWithDesc := RepositoryWithDescription{
			Repository: repo,
		}
		
		// 尝试从cache获取或生成描述
		if cs.cacheManager != nil {
			readmeContent := cs.readREADMEContent(repo.Path)
			description, err := cs.cacheManager.GetDescription(
				repo.Path, 
				repo.Name, 
				readmeContent, 
				llmProvider, 
				llmModel, 
				llmLanguage,
			)
			if err != nil {
				cs.logger.WithError(err).Warnf("Failed to generate LLM description for %s: %v", repo.Path, err)
				description = cs.fallbackDescription(readmeContent)
			}
			repoWithDesc.Description = description
		} else {
			// 如果没有缓存管理器，使用fallback方法
			readmeContent := cs.readREADMEContent(repo.Path)
			repoWithDesc.Description = cs.fallbackDescription(readmeContent)
		}
		
		// 获取最后提交时间（用于排序）
		if lastCommitDate := cs.getLastCommitDate(repo.Path); !lastCommitDate.IsZero() {
			repoWithDesc.LastCommitDate = lastCommitDate
		}
		
		reposWithDesc = append(reposWithDesc, repoWithDesc)
		cs.logger.Debugf("收集描述完成: %s - %s", repo.Name, repoWithDesc.Description)
	}
	
	return reposWithDesc, nil
}

// readREADMEContent reads README file content
func (cs *CachedScanner) readREADMEContent(repoPath string) string {
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
		filePath := filepath.Join(repoPath, filename)
		cs.logger.Debugf("尝试读取README: %s", filePath)
		
		content, err := ioutil.ReadFile(filePath)
		if err == nil {
			cs.logger.Debugf("Found README file: %s", filePath)
			// 限制内容长度，避免过大的文件
			contentStr := string(content)
			if len(contentStr) > 8000 {
				contentStr = contentStr[:8000] + "..."
			}
			return contentStr
		}
	}
	
	return ""
}

// fallbackDescription provides a simple fallback description when LLM is not available
func (cs *CachedScanner) fallbackDescription(readmeContent string) string {
	if readmeContent == "" {
		return "No description available"
	}
	
	// 分割成行
	lines := strings.Split(readmeContent, "\n")
	
	// 查找有意义的描述行
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 跳过空行、标题标记、图片、链接等
		if len(line) == 0 || 
		   strings.HasPrefix(line, "#") ||
		   strings.HasPrefix(line, "![") ||
		   strings.HasPrefix(line, "[!") ||
		   strings.HasPrefix(line, "[![") ||
		   strings.HasPrefix(line, "---") ||
		   strings.HasPrefix(line, "===") {
			continue
		}
		
		// 找到第一个有意义的行
		if len(line) > 10 {
			if len(line) > 80 {
				line = line[:77] + "..."
			}
			return line
		}
	}
	
	return "No description available"
}

// getLastCommitDate gets the last commit date for sorting
func (cs *CachedScanner) getLastCommitDate(repoPath string) time.Time {
	// 使用.git目录的修改时间作为近似值
	gitPath := filepath.Join(repoPath, ".git")
	if info, err := os.Stat(gitPath); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}