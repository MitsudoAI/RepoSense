package changelog

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"reposense/pkg/reporter"
)

// ReportChangelog 报告变更日志结果
func ReportChangelog(report *ChangelogReport, format reporter.ReportFormat, verbose bool) {
	switch format {
	case reporter.FormatJSON:
		reportChangelogJSON(report)
	case reporter.FormatTable:
		reportChangelogTable(report, verbose)
	default:
		reportChangelogText(report, verbose)
	}
}

// reportChangelogText 文本格式报告
func reportChangelogText(report *ChangelogReport, verbose bool) {
	fmt.Printf("# RepoSense 代码库更新报告\n\n")
	fmt.Printf("**时间范围**: %s 至 %s\n", 
		report.TimeRange.Since.Format("2006-01-02"), 
		report.TimeRange.Until.Format("2006-01-02"))
	fmt.Printf("**扫描仓库**: %d 个\n", report.TotalRepos)
	fmt.Printf("**有更新仓库**: %d 个\n\n", report.UpdatedRepos)

	if len(report.Entries) == 0 {
		fmt.Println("📭 指定时间范围内没有仓库更新")
		return
	}

	fmt.Println("## 📊 更新概览\n")

	for i, entry := range report.Entries {
		fmt.Printf("### %d. %s\n", i+1, entry.Repository.Name)
		
		if entry.Summary.Title != "" {
			fmt.Printf("> %s\n\n", entry.Summary.Title)
		}

		// 显示要点
		if len(entry.Summary.Highlights) > 0 {
			for _, highlight := range entry.Summary.Highlights {
				fmt.Printf("- %s\n", highlight)
			}
			fmt.Println()
		}

		// 显示统计信息
		fmt.Printf("**统计**: %d commits | %d authors", 
			entry.Stats.CommitCount, entry.Stats.AuthorCount)
		
		if entry.Stats.FilesChanged > 0 {
			fmt.Printf(" | %d files | +%d -%d lines", 
				entry.Stats.FilesChanged, entry.Stats.Insertions, entry.Stats.Deletions)
		}
		fmt.Println()

		// 显示重大变更
		if len(entry.Stats.MajorChanges) > 0 {
			fmt.Printf("**⚠️  重大变更**: ")
			for j, change := range entry.Stats.MajorChanges {
				if j > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", change)
			}
			fmt.Println()
		}

		// 详细模式下显示分类信息
		if verbose && len(entry.Summary.Categories) > 0 {
			fmt.Println("\n**详细分类**:")
			for category, items := range entry.Summary.Categories {
				if len(items) > 0 {
					categoryName := getCategoryDisplayNameWithEmoji(category)
					fmt.Printf("- **%s** (%d项)\n", categoryName, len(items))
					for _, item := range items {
						if len(item) > 80 {
							item = item[:77] + "..."
						}
						fmt.Printf("  - %s\n", item)
					}
				}
			}
		}

		fmt.Println("\n---\n")
	}

	// 显示总体统计
	totalCommits := 0
	for _, entry := range report.Entries {
		totalCommits += entry.Stats.CommitCount
	}

	fmt.Printf("## 📈 总体统计\n\n")
	fmt.Printf("- **总提交数**: %d\n", totalCommits)
	fmt.Printf("- **活跃仓库**: %d / %d\n", report.UpdatedRepos, report.TotalRepos)
	fmt.Printf("- **生成时间**: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("- **分析模式**: %s\n", report.Config.Mode)
	
	if report.Config.EnableLLM {
		fmt.Printf("- **智能总结**: 启用 (%s)\n", report.Config.LLMProvider)
	} else {
		fmt.Printf("- **智能总结**: 基于规则\n")
	}
}

// reportChangelogTable 表格格式报告
func reportChangelogTable(report *ChangelogReport, verbose bool) {
	fmt.Printf("时间范围: %s 至 %s | 有更新: %d/%d 个仓库\n\n", 
		report.TimeRange.Since.Format("2006-01-02"), 
		report.TimeRange.Until.Format("2006-01-02"),
		report.UpdatedRepos, report.TotalRepos)

	if len(report.Entries) == 0 {
		fmt.Println("指定时间范围内没有仓库更新")
		return
	}

	// 表头 - 不限制宽度
	fmt.Printf("%-30s %-8s %-8s %s\n", "仓库名称", "提交数", "作者数", "主要更新")
	fmt.Println(strings.Repeat("-", 120))

	// 表格内容
	for _, entry := range report.Entries {
		name := entry.Repository.Name
		// 不截断仓库名称，但设置最小宽度
		
		// 生成详细的主要更新信息
		detailedSummary := buildDetailedSummary(entry)
		
		// 第一行：基本信息，不截断主要更新内容
		fmt.Printf("%-30s %-8d %-8d %s\n", 
			name, entry.Stats.CommitCount, entry.Stats.AuthorCount, detailedSummary)

		// 详细分类信息（多行显示）
		categoryLines := buildCategoryLines(entry)
		for _, line := range categoryLines {
			fmt.Printf("%-47s %s\n", "", line)
		}
	}

	fmt.Println()
}

// reportChangelogJSON JSON格式报告
func reportChangelogJSON(report *ChangelogReport) {
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成JSON报告失败: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

// SaveChangelogReport 保存变更日志报告
func SaveChangelogReport(report *ChangelogReport, filename string, format reporter.ReportFormat) error {
	var content []byte
	var err error

	switch format {
	case reporter.FormatJSON:
		content, err = json.MarshalIndent(report, "", "  ")
	default:
		// 默认保存为Markdown格式
		content = []byte(generateMarkdownReport(report))
	}

	if err != nil {
		return fmt.Errorf("生成报告内容失败: %w", err)
	}

	return os.WriteFile(filename, content, 0644)
}

// generateMarkdownReport 生成Markdown格式报告
func generateMarkdownReport(report *ChangelogReport) string {
	var md strings.Builder

	md.WriteString("# RepoSense 代码库更新报告\n\n")
	md.WriteString(fmt.Sprintf("**时间范围**: %s 至 %s  \n", 
		report.TimeRange.Since.Format("2006-01-02"), 
		report.TimeRange.Until.Format("2006-01-02")))
	md.WriteString(fmt.Sprintf("**扫描仓库**: %d 个  \n", report.TotalRepos))
	md.WriteString(fmt.Sprintf("**有更新仓库**: %d 个\n\n", report.UpdatedRepos))

	if len(report.Entries) == 0 {
		md.WriteString("📭 指定时间范围内没有仓库更新\n")
		return md.String()
	}

	md.WriteString("## 更新概览\n\n")

	// 按提交数排序显示
	for i, entry := range report.Entries {
		md.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, entry.Repository.Name))
		
		if entry.Summary.Title != "" {
			md.WriteString(fmt.Sprintf("> %s\n\n", entry.Summary.Title))
		}

		// 要点
		if len(entry.Summary.Highlights) > 0 {
			for _, highlight := range entry.Summary.Highlights {
				md.WriteString(fmt.Sprintf("- %s\n", highlight))
			}
			md.WriteString("\n")
		}

		// 统计信息
		md.WriteString(fmt.Sprintf("**统计**: %d commits | %d authors", 
			entry.Stats.CommitCount, entry.Stats.AuthorCount))
		
		if entry.Stats.FilesChanged > 0 {
			md.WriteString(fmt.Sprintf(" | %d files | +%d -%d lines", 
				entry.Stats.FilesChanged, entry.Stats.Insertions, entry.Stats.Deletions))
		}
		md.WriteString("\n\n")

		// 重大变更
		if len(entry.Stats.MajorChanges) > 0 {
			md.WriteString("**⚠️  重大变更**:\n")
			for _, change := range entry.Stats.MajorChanges {
				md.WriteString(fmt.Sprintf("- %s\n", change))
			}
			md.WriteString("\n")
		}

		// 分类详情
		if len(entry.Summary.Categories) > 0 {
			md.WriteString("**详细分类**:\n\n")
			for category, items := range entry.Summary.Categories {
				if len(items) > 0 {
					categoryName := getCategoryDisplayNameWithEmoji(category)
					md.WriteString(fmt.Sprintf("#### %s (%d项)\n\n", categoryName, len(items)))
					for _, item := range items {
						md.WriteString(fmt.Sprintf("- %s\n", item))
					}
					md.WriteString("\n")
				}
			}
		}

		md.WriteString("---\n\n")
	}

	// 总体统计
	totalCommits := 0
	for _, entry := range report.Entries {
		totalCommits += entry.Stats.CommitCount
	}

	md.WriteString("## 总体统计\n\n")
	md.WriteString(fmt.Sprintf("| 指标 | 数值 |\n"))
	md.WriteString(fmt.Sprintf("|------|------|\n"))
	md.WriteString(fmt.Sprintf("| 总提交数 | %d |\n", totalCommits))
	md.WriteString(fmt.Sprintf("| 活跃仓库 | %d / %d |\n", report.UpdatedRepos, report.TotalRepos))
	md.WriteString(fmt.Sprintf("| 生成时间 | %s |\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("| 分析模式 | %s |\n", report.Config.Mode))
	
	if report.Config.EnableLLM {
		md.WriteString(fmt.Sprintf("| 智能总结 | 启用 (%s) |\n", report.Config.LLMProvider))
	} else {
		md.WriteString(fmt.Sprintf("| 智能总结 | 基于规则 |\n"))
	}

	md.WriteString("\n---\n\n")
	md.WriteString(fmt.Sprintf("*报告由 RepoSense 于 %s 生成*\n", 
		time.Now().Format("2006-01-02 15:04:05")))

	return md.String()
}

// getCategoryDisplayNameWithEmoji 获取分类的显示名称（带emoji）
func getCategoryDisplayNameWithEmoji(category string) string {
	switch category {
	case "features":
		return "🆕 新功能"
	case "fixes":
		return "🐛 Bug修复"
	case "docs":
		return "📚 文档更新"
	case "refactoring":
		return "🔧 代码重构"
	case "tests":
		return "🧪 测试改进"
	case "performance":
		return "⚡ 性能优化"
	case "dependencies":
		return "📦 依赖更新"
	case "ci":
		return "🔄 CI/CD"
	case "security":
		return "🔒 安全修复"
	default:
		return "📝 其他变更"
	}
}

// buildDetailedSummary 构建详细的摘要信息的第一行
func buildDetailedSummary(entry ChangelogEntry) string {
	// 直接返回标题，它已经包含了主要信息
	if entry.Summary.Title != "" {
		return entry.Summary.Title
	}
	return fmt.Sprintf("%d个提交的更新", entry.Stats.CommitCount)
}

// buildCategoryLines 构建分类详情的多行显示
func buildCategoryLines(entry ChangelogEntry) []string {
	var lines []string
	
	// 按优先级顺序显示分类
	categoryOrder := []string{"features", "fixes", "performance", "docs", "refactoring", "tests", "dependencies", "ci", "security", "other"}
	
	// 统计总的条目数，如果太多则只显示概要
	totalItems := 0
	for _, items := range entry.Summary.Categories {
		totalItems += len(items)
	}
	
	// 如果条目过多（>10个），只显示分类汇总
	if totalItems > 10 {
		var summaryParts []string
		for _, category := range categoryOrder {
			if items, exists := entry.Summary.Categories[category]; exists && len(items) > 0 {
				categoryName := getCategoryName(category)
				summaryParts = append(summaryParts, fmt.Sprintf("%s×%d", categoryName, len(items)))
			}
		}
		if len(summaryParts) > 0 {
			lines = append(lines, fmt.Sprintf("包含: %s", strings.Join(summaryParts, ", ")))
		}
	} else {
		// 条目较少，显示详细内容
		for _, category := range categoryOrder {
			if items, exists := entry.Summary.Categories[category]; exists && len(items) > 0 {
				categoryName := getCategoryName(category)
				
				// 显示每个分类的详细内容
				for _, item := range items {
					// 生成行内容，不限制长度
					line := fmt.Sprintf("• %s: %s", categoryName, item)
					lines = append(lines, line)
				}
			}
		}
	}
	
	return lines
}

// getCategoryName 获取简短的分类名称（不带emoji）
func getCategoryName(category string) string {
	switch category {
	case "features":
		return "新功能"
	case "fixes":
		return "修复"
	case "docs":
		return "文档"
	case "refactoring":
		return "重构"
	case "tests":
		return "测试"
	case "performance":
		return "性能"
	case "dependencies":
		return "依赖"
	case "ci":
		return "CI"
	case "security":
		return "安全"
	default:
		return "其他"
	}
}