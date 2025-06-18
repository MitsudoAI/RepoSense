package scanner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RepositoryStatus represents detailed status of a Git repository
type RepositoryStatus struct {
	Repository     Repository      `json:"repository"`
	Branch         string          `json:"branch"`
	LastCommitHash string          `json:"last_commit_hash"`
	LastCommitMsg  string          `json:"last_commit_message"`
	LastCommitDate time.Time       `json:"last_commit_date"`
	HasChanges     bool            `json:"has_changes"`
	Status         string          `json:"status"`
	RemoteURL      string          `json:"remote_url"`
	Ahead          int             `json:"ahead"`
	Behind         int             `json:"behind"`
	Error          string          `json:"error,omitempty"`
}

// StatusCollector collects detailed status information from repositories
type StatusCollector struct {
	logger  *logrus.Logger
	timeout time.Duration
}

// NewStatusCollector creates a new StatusCollector
func NewStatusCollector(timeout time.Duration) *StatusCollector {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	return &StatusCollector{
		logger:  logger,
		timeout: timeout,
	}
}

// SetLogLevel sets the logging level
func (sc *StatusCollector) SetLogLevel(level logrus.Level) {
	sc.logger.SetLevel(level)
}

// CollectStatus collects status for a single repository
func (sc *StatusCollector) CollectStatus(repo Repository) RepositoryStatus {
	status := RepositoryStatus{
		Repository: repo,
	}
	
	if !repo.IsGitRepo {
		status.Error = "不是有效的Git仓库"
		return status
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), sc.timeout)
	defer cancel()
	
	// 获取当前分支
	if branch, err := sc.getCurrentBranch(ctx, repo.Path); err != nil {
		status.Error = "获取分支信息失败: " + err.Error()
		return status
	} else {
		status.Branch = branch
	}
	
	// 获取最后一次提交信息
	if err := sc.getLastCommitInfo(ctx, repo.Path, &status); err != nil {
		sc.logger.Warnf("获取提交信息失败 %s: %v", repo.Path, err)
	}
	
	// 获取工作区状态
	if hasChanges, statusStr, err := sc.getWorkingStatus(ctx, repo.Path); err != nil {
		sc.logger.Warnf("获取工作区状态失败 %s: %v", repo.Path, err)
	} else {
		status.HasChanges = hasChanges
		status.Status = statusStr
	}
	
	// 获取远程仓库URL
	if remoteURL, err := sc.getRemoteURL(ctx, repo.Path); err != nil {
		sc.logger.Debugf("获取远程URL失败 %s: %v", repo.Path, err)
	} else {
		status.RemoteURL = remoteURL
	}
	
	// 获取与远程分支的差异
	if ahead, behind, err := sc.getRemoteDiff(ctx, repo.Path); err != nil {
		sc.logger.Debugf("获取远程差异失败 %s: %v", repo.Path, err)
	} else {
		status.Ahead = ahead
		status.Behind = behind
	}
	
	return status
}

// CollectBatchStatus collects status for multiple repositories
func (sc *StatusCollector) CollectBatchStatus(repositories []Repository) []RepositoryStatus {
	var results []RepositoryStatus
	
	sc.logger.Infof("开始收集 %d 个仓库的状态信息", len(repositories))
	
	for _, repo := range repositories {
		status := sc.CollectStatus(repo)
		results = append(results, status)
		sc.logger.Debugf("收集状态完成: %s", repo.Name)
	}
	
	sc.logger.Infof("状态收集完成")
	return results
}

// getCurrentBranch gets the current branch name
func (sc *StatusCollector) getCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

// getLastCommitInfo gets information about the last commit
func (sc *StatusCollector) getLastCommitInfo(ctx context.Context, repoPath string, status *RepositoryStatus) error {
	// 获取提交哈希
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	
	if output, err := cmd.Output(); err != nil {
		return err
	} else {
		status.LastCommitHash = strings.TrimSpace(string(output))
	}
	
	// 获取提交消息
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%s")
	cmd.Dir = repoPath
	
	if output, err := cmd.Output(); err != nil {
		return err
	} else {
		status.LastCommitMsg = strings.TrimSpace(string(output))
	}
	
	// 获取提交日期
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%ci")
	cmd.Dir = repoPath
	
	if output, err := cmd.Output(); err != nil {
		return err
	} else {
		if date, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(string(output))); err == nil {
			status.LastCommitDate = date
		}
	}
	
	return nil
}

// getWorkingStatus gets the working directory status
func (sc *StatusCollector) getWorkingStatus(ctx context.Context, repoPath string) (bool, string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return false, "", err
	}
	
	statusOutput := strings.TrimSpace(string(output))
	hasChanges := len(statusOutput) > 0
	
	if !hasChanges {
		return false, "干净", nil
	}
	
	// 简化状态描述
	lines := strings.Split(statusOutput, "\n")
	modified := 0
	added := 0
	deleted := 0
	untracked := 0
	
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		
		switch {
		case strings.HasPrefix(line, "??"):
			untracked++
		case strings.HasPrefix(line, "A "):
			added++
		case strings.HasPrefix(line, "D "):
			deleted++
		case strings.HasPrefix(line, "M ") || strings.HasPrefix(line, " M"):
			modified++
		}
	}
	
	var statusParts []string
	if modified > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d个修改", modified))
	}
	if added > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d个新增", added))
	}
	if deleted > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d个删除", deleted))
	}
	if untracked > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d个未跟踪", untracked))
	}
	
	return true, strings.Join(statusParts, ", "), nil
}

// getRemoteURL gets the remote repository URL
func (sc *StatusCollector) getRemoteURL(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

// getRemoteDiff gets the difference with remote branch
func (sc *StatusCollector) getRemoteDiff(ctx context.Context, repoPath string) (int, int, error) {
	// 首先尝试获取远程信息（不拉取）
	cmd := exec.CommandContext(ctx, "git", "remote", "show", "origin")
	cmd.Dir = repoPath
	
	if _, err := cmd.Output(); err != nil {
		// 如果无法连接远程，返回0
		return 0, 0, nil
	}
	
	// 获取当前分支
	branch, err := sc.getCurrentBranch(ctx, repoPath)
	if err != nil {
		return 0, 0, err
	}
	
	remoteBranch := "origin/" + branch
	
	// 检查远程分支是否存在
	cmd = exec.CommandContext(ctx, "git", "rev-parse", "--verify", remoteBranch)
	cmd.Dir = repoPath
	
	if err := cmd.Run(); err != nil {
		// 远程分支不存在
		return 0, 0, nil
	}
	
	// 获取ahead数量
	cmd = exec.CommandContext(ctx, "git", "rev-list", "--count", remoteBranch+"..HEAD")
	cmd.Dir = repoPath
	
	var ahead int
	if output, err := cmd.Output(); err == nil {
		if n, err := parseInt(strings.TrimSpace(string(output))); err == nil {
			ahead = n
		}
	}
	
	// 获取behind数量
	cmd = exec.CommandContext(ctx, "git", "rev-list", "--count", "HEAD.."+remoteBranch)
	cmd.Dir = repoPath
	
	var behind int
	if output, err := cmd.Output(); err == nil {
		if n, err := parseInt(strings.TrimSpace(string(output))); err == nil {
			behind = n
		}
	}
	
	return ahead, behind, nil
}

// parseInt converts string to int
func parseInt(s string) (int, error) {
	var result int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid number: %s", s)
		}
		result = result*10 + int(ch-'0')
	}
	return result, nil
}

