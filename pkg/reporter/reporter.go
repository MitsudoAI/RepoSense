package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"reposense/pkg/scanner"
	"reposense/pkg/updater"

	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
)

// ReportFormat represents different output formats
type ReportFormat string

const (
	FormatTable ReportFormat = "table"
	FormatJSON  ReportFormat = "json"
	FormatText  ReportFormat = "text"
)

// Reporter handles progress display and result reporting
type Reporter struct {
	logger     *logrus.Logger
	progressBar *progressbar.ProgressBar
	format     ReportFormat
	verbose    bool
}

// NewReporter creates a new Reporter instance
func NewReporter(format ReportFormat, verbose bool) *Reporter {
	logger := logrus.New()
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
	
	return &Reporter{
		logger:  logger,
		format:  format,
		verbose: verbose,
	}
}

// InitProgressBar initializes progress bar for updates
func (r *Reporter) InitProgressBar(total int, description string) {
	r.progressBar = progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(50),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
}

// UpdateProgress updates the progress bar
func (r *Reporter) UpdateProgress() {
	if r.progressBar != nil {
		r.progressBar.Add(1)
	}
}

// FinishProgress finishes the progress bar
func (r *Reporter) FinishProgress() {
	if r.progressBar != nil {
		r.progressBar.Finish()
		fmt.Println() // 添加换行
	}
}

// ReportScanResults reports repository scan results
func (r *Reporter) ReportScanResults(repositories []scanner.Repository) {
	total := len(repositories)
	
	switch r.format {
	case FormatJSON:
		r.reportScanResultsJSON(repositories)
	case FormatTable:
		r.reportScanResultsTable(repositories)
	default:
		r.reportScanResultsText(repositories)
	}
	
	if r.verbose {
		r.logger.Infof("扫描完成，共发现 %d 个Git仓库", total)
	}
}

// ReportUpdateResults reports batch update results
func (r *Reporter) ReportUpdateResults(results []updater.UpdateResult) {
	switch r.format {
	case FormatJSON:
		r.reportUpdateResultsJSON(results)
	case FormatTable:
		r.reportUpdateResultsTable(results)
	default:
		r.reportUpdateResultsText(results)
	}
	
	// 显示统计信息
	r.reportStatistics(results)
}

// ReportStatusResults reports repository status results
func (r *Reporter) ReportStatusResults(statuses []scanner.RepositoryStatus) {
	switch r.format {
	case FormatJSON:
		r.reportStatusResultsJSON(statuses)
	case FormatTable:
		r.reportStatusResultsTable(statuses)
	default:
		r.reportStatusResultsText(statuses)
	}
}

// ReportListResults reports repository list results with descriptions
func (r *Reporter) ReportListResults(repositories []scanner.RepositoryWithDescription, sortByTime, reverse bool) {
	// 排序
	sortedRepos := make([]scanner.RepositoryWithDescription, len(repositories))
	copy(sortedRepos, repositories)
	
	if sortByTime {
		sort.Slice(sortedRepos, func(i, j int) bool {
			if reverse {
				return sortedRepos[i].LastCommitDate.After(sortedRepos[j].LastCommitDate)
			}
			return sortedRepos[i].LastCommitDate.Before(sortedRepos[j].LastCommitDate)
		})
	} else {
		sort.Slice(sortedRepos, func(i, j int) bool {
			if reverse {
				return sortedRepos[i].Name > sortedRepos[j].Name
			}
			return sortedRepos[i].Name < sortedRepos[j].Name
		})
	}
	
	switch r.format {
	case FormatJSON:
		r.reportListResultsJSON(sortedRepos)
	case FormatTable:
		r.reportListResultsTable(sortedRepos)
	default:
		r.reportListResultsText(sortedRepos)
	}
}

// reportScanResultsText reports scan results in text format
func (r *Reporter) reportScanResultsText(repositories []scanner.Repository) {
	fmt.Printf("扫描结果 (%d个仓库):\n", len(repositories))
	fmt.Println(strings.Repeat("-", 60))
	
	for i, repo := range repositories {
		fmt.Printf("%d. %s\n", i+1, repo.Name)
		if r.verbose {
			fmt.Printf("   路径: %s\n", repo.Path)
		}
	}
	fmt.Println()
}

// reportScanResultsTable reports scan results in table format
func (r *Reporter) reportScanResultsTable(repositories []scanner.Repository) {
	fmt.Printf("%-4s %-30s %s\n", "序号", "仓库名称", "路径")
	fmt.Println(strings.Repeat("-", 80))
	
	for i, repo := range repositories {
		name := repo.Name
		if len(name) > 28 {
			name = name[:25] + "..."
		}
		
		path := repo.Path
		if len(path) > 45 {
			path = "..." + path[len(path)-42:]
		}
		
		fmt.Printf("%-4d %-30s %s\n", i+1, name, path)
	}
	fmt.Println()
}

// reportScanResultsJSON reports scan results in JSON format
func (r *Reporter) reportScanResultsJSON(repositories []scanner.Repository) {
	output := map[string]interface{}{
		"scan_results": repositories,
		"total":        len(repositories),
		"timestamp":    time.Now(),
	}
	
	jsonData, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonData))
}

// reportUpdateResultsText reports update results in text format
func (r *Reporter) reportUpdateResultsText(results []updater.UpdateResult) {
	fmt.Printf("更新结果 (%d个仓库):\n", len(results))
	fmt.Println(strings.Repeat("-", 80))
	
	successful := 0
	failed := 0
	
	for _, result := range results {
		status := "✓"
		if !result.Success {
			status = "✗"
			failed++
		} else {
			successful++
		}
		
		fmt.Printf("%s %s: %s", status, result.Repository.Name, result.Message)
		if r.verbose {
			fmt.Printf(" (耗时: %s)", formatDuration(result.Duration))
		}
		fmt.Println()
		
		if !result.Success && result.Error != "" {
			fmt.Printf("   错误: %s\n", result.Error)
		}
	}
	
	fmt.Printf("\n成功: %d, 失败: %d\n", successful, failed)
}

// reportUpdateResultsTable reports update results in table format
func (r *Reporter) reportUpdateResultsTable(results []updater.UpdateResult) {
	fmt.Printf("%-4s %-30s %-8s %-10s %s\n", "序号", "仓库名称", "状态", "耗时", "消息")
	fmt.Println(strings.Repeat("-", 90))
	
	for i, result := range results {
		status := "成功"
		if !result.Success {
			status = "失败"
		}
		
		name := result.Repository.Name
		if len(name) > 28 {
			name = name[:25] + "..."
		}
		
		message := result.Message
		if len(message) > 35 {
			message = message[:32] + "..."
		}
		
		duration := formatDuration(result.Duration)
		
		fmt.Printf("%-4d %-30s %-8s %-10s %s\n", i+1, name, status, duration, message)
	}
	fmt.Println()
}

// reportUpdateResultsJSON reports update results in JSON format
func (r *Reporter) reportUpdateResultsJSON(results []updater.UpdateResult) {
	output := map[string]interface{}{
		"update_results": results,
		"total":          len(results),
		"timestamp":      time.Now(),
	}
	
	jsonData, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonData))
}

// reportStatusResultsText reports status results in text format
func (r *Reporter) reportStatusResultsText(statuses []scanner.RepositoryStatus) {
	fmt.Printf("仓库状态 (%d个仓库):\n", len(statuses))
	fmt.Println(strings.Repeat("-", 80))
	
	for _, status := range statuses {
		fmt.Printf("📁 %s (%s)\n", status.Repository.Name, status.Branch)
		
		if status.Error != "" {
			fmt.Printf("   ❌ 错误: %s\n", status.Error)
			continue
		}
		
		if status.LastCommitMsg != "" {
			msg := status.LastCommitMsg
			if len(msg) > 50 {
				msg = msg[:47] + "..."
			}
			fmt.Printf("   📝 最后提交: %s\n", msg)
		}
		
		if !status.LastCommitDate.IsZero() {
			fmt.Printf("   🕐 提交时间: %s\n", status.LastCommitDate.Format("2006-01-02 15:04"))
		}
		
		if status.HasChanges {
			fmt.Printf("   🔄 工作区: %s\n", status.Status)
		} else {
			fmt.Printf("   ✅ 工作区: 干净\n")
		}
		
		if status.Behind > 0 || status.Ahead > 0 {
			fmt.Printf("   🔀 远程差异: 领先%d个提交, 落后%d个提交\n", status.Ahead, status.Behind)
		}
		
		fmt.Println()
	}
}

// reportStatusResultsTable reports status results in table format
func (r *Reporter) reportStatusResultsTable(statuses []scanner.RepositoryStatus) {
	fmt.Printf("%-25s %-15s %-15s %-10s %-20s\n", "仓库名称", "分支", "工作区状态", "远程差异", "最后提交")
	fmt.Println(strings.Repeat("-", 100))
	
	for _, status := range statuses {
		name := status.Repository.Name
		if len(name) > 23 {
			name = name[:20] + "..."
		}
		
		branch := status.Branch
		if len(branch) > 13 {
			branch = branch[:10] + "..."
		}
		
		workStatus := "干净"
		if status.HasChanges {
			workStatus = "有变更"
		}
		if status.Error != "" {
			workStatus = "错误"
		}
		
		remoteDiff := fmt.Sprintf("+%d/-%d", status.Ahead, status.Behind)
		
		lastCommit := ""
		if !status.LastCommitDate.IsZero() {
			lastCommit = status.LastCommitDate.Format("01-02 15:04")
		}
		
		fmt.Printf("%-25s %-15s %-15s %-10s %-20s\n", name, branch, workStatus, remoteDiff, lastCommit)
	}
	fmt.Println()
}

// reportStatusResultsJSON reports status results in JSON format
func (r *Reporter) reportStatusResultsJSON(statuses []scanner.RepositoryStatus) {
	output := map[string]interface{}{
		"status_results": statuses,
		"total":          len(statuses),
		"timestamp":      time.Now(),
	}
	
	jsonData, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonData))
}

// reportStatistics reports update statistics
func (r *Reporter) reportStatistics(results []updater.UpdateResult) {
	if len(results) == 0 {
		return
	}
	
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
	
	avgDuration := totalDuration / time.Duration(len(results))
	successRate := float64(successful) / float64(len(results)) * 100
	
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("📊 统计信息:")
	fmt.Printf("   总计: %d 个仓库\n", len(results))
	fmt.Printf("   成功: %d 个 (%.1f%%)\n", successful, successRate)
	fmt.Printf("   失败: %d 个 (%.1f%%)\n", failed, 100-successRate)
	fmt.Printf("   总耗时: %s\n", formatDuration(totalDuration))
	fmt.Printf("   平均耗时: %s\n", formatDuration(avgDuration))
	fmt.Println(strings.Repeat("=", 60))
}

// formatDuration formats duration to a readable string
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	
	// 如果时间很短，显示毫秒
	if d < time.Second {
		ms := d.Nanoseconds() / 1000000
		if ms == 0 {
			// 如果毫秒也是0，显示微秒
			us := d.Nanoseconds() / 1000
			if us == 0 {
				return fmt.Sprintf("%dns", d.Nanoseconds())
			}
			return fmt.Sprintf("%dμs", us)
		}
		return fmt.Sprintf("%dms", ms)
	}
	
	// 如果时间较长，使用标准格式但更精确
	seconds := d.Seconds()
	if seconds < 60 {
		return fmt.Sprintf("%.2fs", seconds)
	}
	
	minutes := int(seconds / 60)
	remainingSeconds := seconds - float64(minutes*60)
	
	if minutes < 60 {
		if remainingSeconds < 1 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm%.1fs", minutes, remainingSeconds)
	}
	
	hours := minutes / 60
	remainingMinutes := minutes % 60
	
	if remainingMinutes == 0 && remainingSeconds < 1 {
		return fmt.Sprintf("%dh", hours)
	} else if remainingSeconds < 1 {
		return fmt.Sprintf("%dh%dm", hours, remainingMinutes)
	}
	
	return fmt.Sprintf("%dh%dm%.1fs", hours, remainingMinutes, remainingSeconds)
}

// reportListResultsText reports list results in text format
func (r *Reporter) reportListResultsText(repositories []scanner.RepositoryWithDescription) {
	fmt.Printf("仓库列表 (%d个仓库):\n", len(repositories))
	fmt.Println(strings.Repeat("-", 80))
	
	for _, repo := range repositories {
		fmt.Printf("%s: %s\n", repo.Name, repo.Description)
		if r.verbose && !repo.LastCommitDate.IsZero() {
			fmt.Printf("   最后更新: %s\n", repo.LastCommitDate.Format("2006-01-02 15:04"))
		}
	}
	fmt.Println()
}

// reportListResultsTable reports list results in table format
func (r *Reporter) reportListResultsTable(repositories []scanner.RepositoryWithDescription) {
	fmt.Printf("%-30s %-50s %-20s\n", "仓库名称", "描述", "最后更新")
	fmt.Println(strings.Repeat("-", 105))
	
	for _, repo := range repositories {
		name := repo.Name
		if len(name) > 28 {
			name = name[:25] + "..."
		}
		
		description := repo.Description
		if len(description) > 48 {
			description = description[:45] + "..."
		}
		
		lastUpdate := ""
		if !repo.LastCommitDate.IsZero() {
			lastUpdate = repo.LastCommitDate.Format("2006-01-02 15:04")
		}
		
		fmt.Printf("%-30s %-50s %-20s\n", name, description, lastUpdate)
	}
	fmt.Println()
}

// reportListResultsJSON reports list results in JSON format
func (r *Reporter) reportListResultsJSON(repositories []scanner.RepositoryWithDescription) {
	output := map[string]interface{}{
		"list_results": repositories,
		"total":        len(repositories),
		"timestamp":    time.Now(),
	}
	
	jsonData, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonData))
}

// SaveReport saves report to file
func (r *Reporter) SaveReport(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	
	_, err = file.Write(jsonData)
	return err
}