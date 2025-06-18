package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reposense/pkg/llm"

	"github.com/sirupsen/logrus"
)

// Repository represents a Git repository with its metadata
type Repository struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	IsGitRepo   bool   `json:"is_git_repo"`
	Error       string `json:"error,omitempty"`
}

// RepositoryWithDescription represents a Git repository with description and last commit date
type RepositoryWithDescription struct {
	Repository
	Description    string    `json:"description"`
	LastCommitDate time.Time `json:"last_commit_date"`
}

// Scanner handles repository discovery
type Scanner struct {
	logger            *logrus.Logger
	descriptionService *llm.DescriptionService
}

// NewScanner creates a new Scanner instance
func NewScanner() *Scanner {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	return &Scanner{
		logger: logger,
	}
}

// NewScannerWithLLM creates a new Scanner instance with LLM description service
func NewScannerWithLLM(descriptionService *llm.DescriptionService) *Scanner {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	return &Scanner{
		logger:            logger,
		descriptionService: descriptionService,
	}
}

// SetLogLevel sets the logging level
func (s *Scanner) SetLogLevel(level logrus.Level) {
	s.logger.SetLevel(level)
	if s.descriptionService != nil {
		s.descriptionService.SetLogLevel(level)
	}
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

// extractDescription extracts project description from README files
func (s *Scanner) extractDescription(repoPath string) string {
	// Use LLM service if available
	if s.descriptionService != nil {
		return s.descriptionService.ExtractDescription(repoPath)
	}
	
	// Fallback to simple extraction
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

// readFirstNonEmptyLine reads the first non-empty line from a file
func (s *Scanner) readFirstNonEmptyLine(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		// 跳过markdown标题标记和其他格式符号
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimPrefix(line, "=")
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		
		if line != "" && len(line) > 1 {
			// 限制描述长度
			if len(line) > 100 {
				line = line[:97] + "..."
			}
			return line
		}
	}
	
	return ""
}

// ScanDirectoryWithDescription scans directory and extracts descriptions
func (s *Scanner) ScanDirectoryWithDescription(rootPath string, includePatterns, excludePatterns []string) ([]RepositoryWithDescription, error) {
	repositories, err := s.ScanDirectoryWithFilter(rootPath, includePatterns, excludePatterns)
	if err != nil {
		return nil, err
	}
	
	var reposWithDesc []RepositoryWithDescription
	for _, repo := range repositories {
		repoWithDesc := RepositoryWithDescription{
			Repository:  repo,
			Description: s.extractDescription(repo.Path),
		}
		
		// 获取最后提交时间（用于排序）
		if lastCommitDate := s.getLastCommitDate(repo.Path); !lastCommitDate.IsZero() {
			repoWithDesc.LastCommitDate = lastCommitDate
		}
		
		reposWithDesc = append(reposWithDesc, repoWithDesc)
		s.logger.Debugf("收集描述完成: %s - %s", repo.Name, repoWithDesc.Description)
	}
	
	return reposWithDesc, nil
}

// getLastCommitDate gets the last commit date for sorting
func (s *Scanner) getLastCommitDate(repoPath string) time.Time {
	// 这里实现一个简单的git log命令来获取最后提交时间
	// 为了避免依赖复杂的git操作，我们使用.git目录的修改时间作为近似值
	gitPath := filepath.Join(repoPath, ".git")
	if info, err := os.Stat(gitPath); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}