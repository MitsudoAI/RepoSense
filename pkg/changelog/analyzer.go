package changelog

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"reposense/pkg/llm"
	"reposense/pkg/reporter"
	"reposense/pkg/scanner"

	"github.com/sirupsen/logrus"
)

// ChangelogAnalyzer 变更日志分析器
type ChangelogAnalyzer struct {
	scanner    *scanner.Scanner
	llmService *llm.DescriptionService
	reporter   *reporter.Reporter
	workers    int
	timeout    time.Duration
	logger     *logrus.Logger
}

// NewChangelogAnalyzer 创建新的分析器实例
func NewChangelogAnalyzer(opts ChangelogOptions) *ChangelogAnalyzer {
	// 初始化扫描器
	scannerInstance := scanner.NewScanner()

	// 初始化LLM服务（如果启用）
	var llmService *llm.DescriptionService
	if opts.EnableLLM {
		llmService = llm.NewDescriptionService(
			llm.Provider(opts.LLMProvider),
			opts.LLMModel,
			opts.LLMAPIKey,
			opts.LLMBaseURL,
			opts.Language,
			opts.LLMTimeout,
			true,
		)
	}

	// 初始化报告器
	reporterInstance := reporter.NewReporter(opts.OutputFormat, opts.Verbose)

	logger := logrus.New()
	if opts.Verbose {
		logger.SetLevel(logrus.DebugLevel)
		scannerInstance.SetLogLevel(logrus.DebugLevel)
	}

	return &ChangelogAnalyzer{
		scanner:    scannerInstance,
		llmService: llmService,
		reporter:   reporterInstance,
		workers:    opts.WorkerCount,
		timeout:    opts.Timeout,
		logger:     logger,
	}
}

// Analyze 执行变更日志分析
func (a *ChangelogAnalyzer) Analyze(opts ChangelogOptions) (*ChangelogReport, error) {
	a.logger.Infof("开始changelog分析，目录: %s", opts.Directory)

	// 1. 扫描仓库（复用现有逻辑）
	repos, err := a.scanner.ScanDirectoryWithFilter(opts.Directory, opts.IncludePatterns, opts.ExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("扫描仓库失败: %w", err)
	}

	a.logger.Infof("发现 %d 个Git仓库", len(repos))

	// 2. 筛选有更新的仓库
	updatedRepos := a.filterUpdatedRepos(repos, opts.TimeRange)
	a.logger.Infof("其中 %d 个仓库在指定时间范围内有更新", len(updatedRepos))

	if len(updatedRepos) == 0 {
		a.logger.Info("没有仓库在指定时间范围内有更新")
		return &ChangelogReport{
			TimeRange:    opts.TimeRange,
			TotalRepos:   len(repos),
			UpdatedRepos: 0,
			Entries:      []ChangelogEntry{},
			GeneratedAt:  time.Now(),
			Config: ChangelogConfig{
				Mode:        opts.Mode,
				Language:    opts.Language,
				EnableLLM:   opts.EnableLLM,
				LLMProvider: opts.LLMProvider,
				LLMModel:    opts.LLMModel,
			},
		}, nil
	}

	// 3. 并发分析每个仓库
	entries := a.analyzeReposParallel(updatedRepos, opts)

	// 4. 生成完整报告
	report := &ChangelogReport{
		TimeRange:    opts.TimeRange,
		TotalRepos:   len(repos),
		UpdatedRepos: len(updatedRepos),
		Entries:      entries,
		GeneratedAt:  time.Now(),
		Config: ChangelogConfig{
			Mode:        opts.Mode,
			Language:    opts.Language,
			EnableLLM:   opts.EnableLLM,
			LLMProvider: opts.LLMProvider,
			LLMModel:    opts.LLMModel,
		},
	}

	a.logger.Infof("分析完成，生成了 %d 个仓库的变更记录", len(entries))
	return report, nil
}

// filterUpdatedRepos 筛选在指定时间范围内有更新的仓库
func (a *ChangelogAnalyzer) filterUpdatedRepos(repos []scanner.Repository, timeRange TimeRange) []scanner.Repository {
	var updatedRepos []scanner.Repository

	for _, repo := range repos {
		if a.hasUpdatesInRange(repo.Path, timeRange) {
			updatedRepos = append(updatedRepos, repo)
			a.logger.Debugf("仓库 %s 在指定时间范围内有更新", repo.Name)
		}
	}

	return updatedRepos
}

// hasUpdatesInRange 检查仓库在指定时间范围内是否有更新
func (a *ChangelogAnalyzer) hasUpdatesInRange(repoPath string, timeRange TimeRange) bool {
	// 使用git log检查指定时间范围内的提交
	cmd := exec.Command("git", "log", "--oneline", "--since="+timeRange.Since.Format("2006-01-02T15:04:05"), "--until="+timeRange.Until.Format("2006-01-02T15:04:05"))
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		a.logger.Debugf("检查仓库更新失败 %s: %v", repoPath, err)
		return false
	}

	// 如果有输出，说明有提交
	return len(strings.TrimSpace(string(output))) > 0
}

// analyzeReposParallel 并发分析多个仓库
func (a *ChangelogAnalyzer) analyzeReposParallel(repos []scanner.Repository, opts ChangelogOptions) []ChangelogEntry {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var entries []ChangelogEntry

	// 使用semaphore限制并发数
	semaphore := make(chan struct{}, a.workers)

	for _, repo := range repos {
		wg.Add(1)
		go func(r scanner.Repository) {
			defer wg.Done()

			// 获取semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			entry := a.analyzeRepository(r, opts)
			if entry != nil {
				mu.Lock()
				entries = append(entries, *entry)
				mu.Unlock()
			}
		}(repo)
	}

	wg.Wait()
	return entries
}

// analyzeRepository 分析单个仓库
func (a *ChangelogAnalyzer) analyzeRepository(repo scanner.Repository, opts ChangelogOptions) *ChangelogEntry {
	a.logger.Debugf("开始分析仓库: %s", repo.Name)

	// 获取提交记录
	commits, err := a.getCommitsInRange(repo.Path, opts.TimeRange)
	if err != nil {
		a.logger.Errorf("获取仓库 %s 提交记录失败: %v", repo.Name, err)
		return nil
	}

	if len(commits) == 0 {
		a.logger.Debugf("仓库 %s 在指定时间范围内没有提交", repo.Name)
		return nil
	}

	// 生成统计信息
	stats := a.generateStats(repo.Path, commits, opts.TimeRange)

	// 生成摘要
	summary := a.generateSummary(repo, commits, opts)

	entry := &ChangelogEntry{
		Repository: repo,
		TimeRange:  opts.TimeRange,
		Commits:    commits,
		Summary:    summary,
		Stats:      stats,
		UpdatedAt:  time.Now(),
	}

	a.logger.Debugf("完成分析仓库: %s，共 %d 个提交", repo.Name, len(commits))
	return entry
}

// getCommitsInRange 获取指定时间范围内的提交记录
func (a *ChangelogAnalyzer) getCommitsInRange(repoPath string, timeRange TimeRange) ([]Commit, error) {
	// 构建git log命令
	args := []string{
		"log",
		"--pretty=format:%H|%an|%ai|%s|%B",
		"--since=" + timeRange.Since.Format("2006-01-02T15:04:05"),
		"--until=" + timeRange.Until.Format("2006-01-02T15:04:05"),
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行git log失败: %w", err)
	}

	return a.parseGitLog(string(output))
}

// parseGitLog 解析git log输出
func (a *ChangelogAnalyzer) parseGitLog(logOutput string) ([]Commit, error) {
	if strings.TrimSpace(logOutput) == "" {
		return []Commit{}, nil
	}

	var commits []Commit
	commitBlocks := strings.Split(strings.TrimSpace(logOutput), "\n\n")

	for _, block := range commitBlocks {
		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue
		}

		// 解析第一行的格式化信息
		parts := strings.Split(lines[0], "|")
		if len(parts) < 4 {
			continue
		}

		// 解析日期
		date, err := time.Parse("2006-01-02 15:04:05 -0700", parts[2])
		if err != nil {
			a.logger.Debugf("解析提交时间失败: %v", err)
			continue
		}

		// 收集完整消息（包括多行）
		fullMessage := parts[3] // 短消息
		if len(lines) > 1 {
			// 如果有多行，收集完整消息
			messageLines := lines[1:]
			fullMessage = strings.Join(messageLines, "\n")
		}

		commit := Commit{
			Hash:     parts[0],
			Author:   parts[1],
			Date:     date,
			ShortMsg: parts[3],
			Message:  strings.TrimSpace(fullMessage),
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// generateStats 生成变更统计
func (a *ChangelogAnalyzer) generateStats(repoPath string, commits []Commit, timeRange TimeRange) ChangeStats {
	// 统计基本信息
	authorSet := make(map[string]bool)
	for _, commit := range commits {
		authorSet[commit.Author] = true
	}

	stats := ChangeStats{
		CommitCount: len(commits),
		AuthorCount: len(authorSet),
	}

	// 获取文件变更统计
	cmd := exec.Command("git", "diff", "--stat",
		"--since="+timeRange.Since.Format("2006-01-02T15:04:05"),
		"--until="+timeRange.Until.Format("2006-01-02T15:04:05"))
	cmd.Dir = repoPath

	if output, err := cmd.Output(); err == nil {
		stats.FilesChanged, stats.Insertions, stats.Deletions = a.parseGitDiffStat(string(output))
	}

	// 识别重大变更
	stats.MajorChanges = a.identifyMajorChanges(commits)

	return stats
}

// parseGitDiffStat 解析git diff --stat输出
func (a *ChangelogAnalyzer) parseGitDiffStat(diffStat string) (files, insertions, deletions int) {
	lines := strings.Split(strings.TrimSpace(diffStat), "\n")
	if len(lines) == 0 {
		return 0, 0, 0
	}

	// 最后一行通常包含总计信息
	lastLine := lines[len(lines)-1]
	if strings.Contains(lastLine, "files changed") || strings.Contains(lastLine, "file changed") {
		// 解析类似 "3 files changed, 120 insertions(+), 45 deletions(-)" 的格式
		parts := strings.Fields(lastLine)
		for i, part := range parts {
			if num, err := strconv.Atoi(part); err == nil {
				if i+1 < len(parts) {
					switch parts[i+1] {
					case "files", "file":
						files = num
					case "insertions(+)", "insertion(+)":
						insertions = num
					case "deletions(-)", "deletion(-)":
						deletions = num
					}
				}
			}
		}
	} else {
		// 如果没有总计行，计算文件数
		files = len(lines)
	}

	return files, insertions, deletions
}

// identifyMajorChanges 识别重大变更
func (a *ChangelogAnalyzer) identifyMajorChanges(commits []Commit) []string {
	var majorChanges []string
	majorKeywords := []string{
		"BREAKING CHANGE", "breaking change", "major", "重大", "不兼容",
		"v2.", "v3.", "v4.", "v5.", // 版本号变更
		"feat!", "fix!", "refactor!", // conventional commits的重大变更标记
	}

	for _, commit := range commits {
		message := strings.ToLower(commit.Message)
		for _, keyword := range majorKeywords {
			if strings.Contains(message, strings.ToLower(keyword)) {
				majorChanges = append(majorChanges, commit.ShortMsg)
				break
			}
		}
	}

	return majorChanges
}

// generateSummary 生成变更摘要
func (a *ChangelogAnalyzer) generateSummary(repo scanner.Repository, commits []Commit, opts ChangelogOptions) Summary {
	if opts.EnableLLM && a.llmService != nil {
		return a.generateLLMSummary(repo, commits, opts)
	}
	return a.generateRuleBasedSummary(repo, commits, opts)
}

// generateRuleBasedSummary 基于规则的摘要生成
func (a *ChangelogAnalyzer) generateRuleBasedSummary(repo scanner.Repository, commits []Commit, opts ChangelogOptions) Summary {
	categories := make(map[string][]string)
	var highlights []string

	// 分析提交消息，分类
	for _, commit := range commits {
		message := strings.ToLower(commit.Message)
		shortMsg := commit.ShortMsg
		
		// 增强的关键词匹配
		classified := false
		
		// 新功能
		featureKeywords := []string{"feat", "feature", "add", "新增", "增加", "实现", "implement", "支持", "support"}
		for _, keyword := range featureKeywords {
			if strings.Contains(message, keyword) {
				categories["features"] = append(categories["features"], shortMsg)
				classified = true
				break
			}
		}
		
		if !classified {
			// Bug修复
			fixKeywords := []string{"fix", "bug", "修复", "解决", "repair", "solve", "修正", "correct"}
			for _, keyword := range fixKeywords {
				if strings.Contains(message, keyword) {
					categories["fixes"] = append(categories["fixes"], shortMsg)
					classified = true
					break
				}
			}
		}
		
		if !classified {
			// 文档更新
			docKeywords := []string{"doc", "readme", "文档", "documentation", "guide", "说明"}
			for _, keyword := range docKeywords {
				if strings.Contains(message, keyword) {
					categories["docs"] = append(categories["docs"], shortMsg)
					classified = true
					break
				}
			}
		}
		
		if !classified {
			// 性能优化
			perfKeywords := []string{"perf", "performance", "optimize", "优化", "提升", "improve", "speed", "faster"}
			for _, keyword := range perfKeywords {
				if strings.Contains(message, keyword) {
					categories["performance"] = append(categories["performance"], shortMsg)
					classified = true
					break
				}
			}
		}
		
		if !classified {
			// 代码重构
			refactorKeywords := []string{"refactor", "重构", "重写", "restructure", "rewrite", "cleanup", "clean"}
			for _, keyword := range refactorKeywords {
				if strings.Contains(message, keyword) {
					categories["refactoring"] = append(categories["refactoring"], shortMsg)
					classified = true
					break
				}
			}
		}
		
		if !classified {
			// 测试相关
			testKeywords := []string{"test", "testing", "测试", "单测", "集成测试", "spec"}
			for _, keyword := range testKeywords {
				if strings.Contains(message, keyword) {
					categories["tests"] = append(categories["tests"], shortMsg)
					classified = true
					break
				}
			}
		}
		
		if !classified {
			// 依赖更新
			depKeywords := []string{"dependency", "dependencies", "依赖", "update", "upgrade", "bump", "更新"}
			for _, keyword := range depKeywords {
				if strings.Contains(message, keyword) {
					categories["dependencies"] = append(categories["dependencies"], shortMsg)
					classified = true
					break
				}
			}
		}
		
		if !classified {
			// CI/CD相关
			ciKeywords := []string{"ci", "cd", "pipeline", "workflow", "action", "构建", "build", "deploy", "部署"}
			for _, keyword := range ciKeywords {
				if strings.Contains(message, keyword) {
					categories["ci"] = append(categories["ci"], shortMsg)
					classified = true
					break
				}
			}
		}
		
		if !classified {
			// 安全相关
			securityKeywords := []string{"security", "安全", "vulnerability", "漏洞", "auth", "permission", "权限"}
			for _, keyword := range securityKeywords {
				if strings.Contains(message, keyword) {
					categories["security"] = append(categories["security"], shortMsg)
					classified = true
					break
				}
			}
		}
		
		// 如果没有匹配到任何分类，归为其他
		if !classified {
			categories["other"] = append(categories["other"], shortMsg)
		}
	}

	// 生成要点
	for category, items := range categories {
		if len(items) > 0 {
			var categoryName string
			switch category {
			case "features":
				categoryName = "新功能"
			case "fixes":
				categoryName = "Bug修复"
			case "docs":
				categoryName = "文档更新"
			case "refactoring":
				categoryName = "代码重构"
			case "tests":
				categoryName = "测试改进"
			default:
				categoryName = "其他变更"
			}
			highlights = append(highlights, fmt.Sprintf("%s: %d项", categoryName, len(items)))
		}
	}

	// 生成标题 - 不包含仓库名称，避免重复
	title := fmt.Sprintf("%d个提交的更新", len(commits))
	
	// 找到最主要的分类来生成简洁标题
	var maxCategory string
	var maxCount int
	for category, items := range categories {
		if len(items) > maxCount {
			maxCount = len(items)
			maxCategory = category
		}
	}
	
	if maxCategory != "" && maxCount > 0 {
		categoryName := getCategoryDisplayName(maxCategory)
		if maxCount == len(commits) {
			// 如果所有提交都是同一类型
			title = fmt.Sprintf("%s (%d项)", categoryName, maxCount)
		} else {
			// 如果是混合类型，显示主要类型
			title = fmt.Sprintf("主要是%s (%d项)", categoryName, maxCount)
		}
	}

	return Summary{
		Title:       title,
		Highlights:  highlights,
		Categories:  categories,
		Language:    opts.Language,
		GeneratedBy: "rule-based",
	}
}

// generateLLMSummary 基于LLM的摘要生成
func (a *ChangelogAnalyzer) generateLLMSummary(repo scanner.Repository, commits []Commit, opts ChangelogOptions) Summary {
	// 暂时禁用LLM，因为当前的ExtractDescription方法不适合changelog场景
	// TODO: 实现专门的changelog LLM服务
	a.logger.Debugf("LLM changelog功能暂未实现，使用规则引擎")
	return a.generateRuleBasedSummary(repo, commits, opts)
}

// buildChangelogPrompt 构建LLM提示词
func (a *ChangelogAnalyzer) buildChangelogPrompt(repo scanner.Repository, commits []Commit, language string) string {
	var commitMsgs []string
	for _, commit := range commits {
		commitMsgs = append(commitMsgs, "- "+commit.ShortMsg)
	}

	timeRange := fmt.Sprintf("%s 至 %s", commits[len(commits)-1].Date.Format("2006-01-02"), commits[0].Date.Format("2006-01-02"))

	langPrompt := "中文"
	if language == "en" {
		langPrompt = "English"
	} else if language == "ja" {
		langPrompt = "日本語"
	}

	prompt := fmt.Sprintf(`请分析以下代码库的更新内容：

仓库名称: %s
时间范围: %s
提交数量: %d

主要提交信息:
%s

请用%s生成一份简洁的更新总结，包括：
1. 一句话概述（不超过50字）
2. 3-5个要点
3. 按类别分组的详细变更（新功能/修复/性能/文档等）

请以自然语言形式回复，不需要JSON格式。`,
		repo.Name,
		timeRange,
		len(commits),
		strings.Join(commitMsgs, "\n"),
		langPrompt)

	return prompt
}

// parseLLMResponse 解析LLM响应
func (a *ChangelogAnalyzer) parseLLMResponse(response, language string) Summary {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	if len(lines) == 0 {
		return Summary{
			Title:      "无法生成摘要",
			Language:   language,
			Highlights: []string{},
			Categories: make(map[string][]string),
		}
	}

	// 简单解析：第一行作为标题，其余作为亮点
	title := strings.TrimSpace(lines[0])
	var highlights []string

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.HasPrefix(line, "#") {
			highlights = append(highlights, line)
		}
	}

	return Summary{
		Title:      title,
		Highlights: highlights,
		Categories: make(map[string][]string), // LLM响应的分类解析可以后续完善
		Language:   language,
	}
}

// getCategoryDisplayName 获取分类的显示名称
func getCategoryDisplayName(category string) string {
	switch category {
	case "features":
		return "新功能"
	case "fixes":
		return "Bug修复"
	case "docs":
		return "文档更新"
	case "refactoring":
		return "代码重构"
	case "tests":
		return "测试改进"
	case "performance":
		return "性能优化"
	case "dependencies":
		return "依赖更新"
	case "ci":
		return "CI/CD"
	case "security":
		return "安全修复"
	default:
		return "其他变更"
	}
}

// GetReporter 获取报告器实例
func (a *ChangelogAnalyzer) GetReporter() *reporter.Reporter {
	return a.reporter
}