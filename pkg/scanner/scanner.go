package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Repository represents a Git repository with its metadata
type Repository struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	IsGitRepo   bool   `json:"is_git_repo"`
	Error       string `json:"error,omitempty"`
}

// Scanner handles repository discovery
type Scanner struct {
	logger *logrus.Logger
}

// NewScanner creates a new Scanner instance
func NewScanner() *Scanner {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	return &Scanner{
		logger: logger,
	}
}

// SetLogLevel sets the logging level
func (s *Scanner) SetLogLevel(level logrus.Level) {
	s.logger.SetLevel(level)
}

// ScanDirectory scans a directory for Git repositories
func (s *Scanner) ScanDirectory(rootPath string) ([]Repository, error) {
	var repositories []Repository
	
	s.logger.Infof("开始扫描目录: %s", rootPath)
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.logger.Warnf("访问路径失败 %s: %v", path, err)
			return nil
		}
		
		// 跳过非目录
		if !info.IsDir() {
			return nil
		}
		
		// 检查是否为 Git 仓库
		if s.isGitRepository(path) {
			repo := Repository{
				Path:      path,
				Name:      filepath.Base(path),
				IsGitRepo: true,
			}
			repositories = append(repositories, repo)
			s.logger.Debugf("发现 Git 仓库: %s", path)
			
			// 跳过 .git 子目录
			return filepath.SkipDir
		}
		
		// 跳过 .git 目录
		if info.Name() == ".git" {
			return filepath.SkipDir
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	s.logger.Infof("扫描完成，共发现 %d 个 Git 仓库", len(repositories))
	return repositories, nil
}

// ScanDirectoryWithFilter scans directory with filtering options
func (s *Scanner) ScanDirectoryWithFilter(rootPath string, includePatterns, excludePatterns []string) ([]Repository, error) {
	repositories, err := s.ScanDirectory(rootPath)
	if err != nil {
		return nil, err
	}
	
	if len(includePatterns) == 0 && len(excludePatterns) == 0 {
		return repositories, nil
	}
	
	var filtered []Repository
	for _, repo := range repositories {
		if s.shouldIncludeRepository(repo, includePatterns, excludePatterns) {
			filtered = append(filtered, repo)
		} else {
			s.logger.Debugf("过滤掉仓库: %s", repo.Path)
		}
	}
	
	s.logger.Infof("过滤后剩余 %d 个 Git 仓库", len(filtered))
	return filtered, nil
}

// isGitRepository checks if a directory is a Git repository
func (s *Scanner) isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	
	// 检查 .git 是否存在（可能是文件或目录）
	if _, err := os.Stat(gitPath); err != nil {
		return false
	}
	
	return true
}

// shouldIncludeRepository checks if repository matches filter criteria
func (s *Scanner) shouldIncludeRepository(repo Repository, includePatterns, excludePatterns []string) bool {
	repoName := strings.ToLower(repo.Name)
	repoPath := strings.ToLower(repo.Path)
	
	// 检查排除模式
	for _, pattern := range excludePatterns {
		pattern = strings.ToLower(pattern)
		if strings.Contains(repoName, pattern) || strings.Contains(repoPath, pattern) {
			return false
		}
	}
	
	// 如果没有包含模式，则通过排除检查的都包含
	if len(includePatterns) == 0 {
		return true
	}
	
	// 检查包含模式
	for _, pattern := range includePatterns {
		pattern = strings.ToLower(pattern)
		if strings.Contains(repoName, pattern) || strings.Contains(repoPath, pattern) {
			return true
		}
	}
	
	return false
}

// GetRepositoryCount returns the count of repositories that would be found
func (s *Scanner) GetRepositoryCount(rootPath string) (int, error) {
	repositories, err := s.ScanDirectory(rootPath)
	if err != nil {
		return 0, err
	}
	return len(repositories), nil
}