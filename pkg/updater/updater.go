package updater

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"reposense/pkg/scanner"

	"github.com/sirupsen/logrus"
)

// UpdateResult represents the result of updating a repository
type UpdateResult struct {
	Repository scanner.Repository `json:"repository"`
	Success    bool               `json:"success"`
	Message    string             `json:"message"`
	Error      string             `json:"error,omitempty"`
	Duration   time.Duration      `json:"duration"`
	StartTime  time.Time          `json:"start_time"`
	EndTime    time.Time          `json:"end_time"`
}

// UpdaterConfig holds configuration for the updater
type UpdaterConfig struct {
	WorkerCount int           `json:"worker_count"`
	Timeout     time.Duration `json:"timeout"`
	DryRun      bool          `json:"dry_run"`
}

// Updater handles batch Git operations
type Updater struct {
	config UpdaterConfig
	logger *logrus.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

// NewUpdater creates a new Updater instance
func NewUpdater(config UpdaterConfig) *Updater {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Updater{
		config: config,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetLogLevel sets the logging level
func (u *Updater) SetLogLevel(level logrus.Level) {
	u.logger.SetLevel(level)
}

// UpdateRepositories performs batch git pull operations
func (u *Updater) UpdateRepositories(repositories []scanner.Repository, progressCallback func(UpdateResult)) ([]UpdateResult, error) {
	if len(repositories) == 0 {
		return []UpdateResult{}, nil
	}
	
	u.logger.Infof("开始更新 %d 个仓库，使用 %d 个工作协程", len(repositories), u.config.WorkerCount)
	
	// 创建任务通道和结果通道
	jobs := make(chan scanner.Repository, len(repositories))
	results := make(chan UpdateResult, len(repositories))
	
	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < u.config.WorkerCount; i++ {
		wg.Add(1)
		go u.worker(i, jobs, results, &wg)
	}
	
	// 发送任务
	go func() {
		defer close(jobs)
		for _, repo := range repositories {
			select {
			case jobs <- repo:
			case <-u.ctx.Done():
				return
			}
		}
	}()
	
	// 收集结果
	var updateResults []UpdateResult
	go func() {
		wg.Wait()
		close(results)
	}()
	
	for result := range results {
		updateResults = append(updateResults, result)
		if progressCallback != nil {
			progressCallback(result)
		}
	}
	
	u.logger.Infof("更新完成，共处理 %d 个仓库", len(updateResults))
	return updateResults, nil
}

// worker is a worker goroutine that processes repository updates
func (u *Updater) worker(id int, jobs <-chan scanner.Repository, results chan<- UpdateResult, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for repo := range jobs {
		select {
		case <-u.ctx.Done():
			return
		default:
			result := u.updateRepository(repo)
			u.logger.Debugf("工作协程 %d 完成仓库 %s: %s", id, repo.Name, result.Message)
			results <- result
		}
	}
}

// updateRepository updates a single repository
func (u *Updater) updateRepository(repo scanner.Repository) UpdateResult {
	startTime := time.Now()
	
	result := UpdateResult{
		Repository: repo,
		StartTime:  startTime,
	}
	
	if u.config.DryRun {
		result.Success = true
		result.Message = "DRY RUN: 模拟更新成功"
	} else {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(u.ctx, u.config.Timeout)
		defer cancel()
		
		// 执行 git pull
		cmd := exec.CommandContext(ctx, "git", "pull")
		cmd.Dir = repo.Path
		
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = fmt.Sprintf("更新失败: %s", string(output))
		} else {
			result.Success = true
			result.Message = u.parseGitPullOutput(string(output))
		}
	}
	
	// 计算时间（在函数结束前）
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	return result
}

// parseGitPullOutput parses git pull output to provide meaningful messages
func (u *Updater) parseGitPullOutput(output string) string {
	if output == "" {
		return "无输出"
	}
	
	// 简化常见的 git pull 输出消息
	switch {
	case contains(output, "Already up to date"):
		return "已是最新版本"
	case contains(output, "Already up-to-date"):
		return "已是最新版本"
	case contains(output, "Fast-forward"):
		return "快进更新成功"
	case contains(output, "Merge made by"):
		return "合并更新成功"
	case contains(output, "files changed"):
		return "更新成功"
	default:
		// 截取输出的前100个字符
		if len(output) > 100 {
			return output[:100] + "..."
		}
		return output
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Stop stops the updater
func (u *Updater) Stop() {
	u.cancel()
}

// GetStatistics returns update statistics
func (u *Updater) GetStatistics(results []UpdateResult) map[string]interface{} {
	total := len(results)
	successful := 0
	failed := 0
	var totalDuration time.Duration
	
	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
		totalDuration += result.Duration
	}
	
	var avgDuration time.Duration
	if total > 0 {
		avgDuration = totalDuration / time.Duration(total)
	}
	
	return map[string]interface{}{
		"total":              total,
		"successful":         successful,
		"failed":             failed,
		"success_rate":       float64(successful) / float64(total) * 100,
		"total_duration":     totalDuration,
		"average_duration":   avgDuration,
	}
}