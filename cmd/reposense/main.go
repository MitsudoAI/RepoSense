package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"reposense/internal/config"
	"reposense/pkg/reporter"
	"reposense/pkg/scanner"
	"reposense/pkg/updater"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfg *config.Config
)

func main() {
	cfg = config.DefaultConfig()
	
	var rootCmd = &cobra.Command{
		Use:   "reposense",
		Short: "Git仓库批量管理工具",
		Long: `RepoSense 是一个高效的Git仓库批量管理工具，支持：
- 自动发现指定目录下的Git仓库
- 并行执行批量git pull操作
- 收集仓库状态信息
- 提供多种输出格式`,
	}
	
	// Update command
	var updateCmd = &cobra.Command{
		Use:   "update [directory]",
		Short: "批量更新Git仓库",
		Long:  "扫描指定目录下的所有Git仓库并执行批量更新操作",
		Args:  cobra.MaximumNArgs(1),
		Run:   runUpdate,
	}
	
	// Scan command
	var scanCmd = &cobra.Command{
		Use:   "scan [directory]",
		Short: "扫描Git仓库",
		Long:  "扫描指定目录下的所有Git仓库并显示列表",
		Args:  cobra.MaximumNArgs(1),
		Run:   runScan,
	}
	
	// Status command
	var statusCmd = &cobra.Command{
		Use:   "status [directory]",
		Short: "查看仓库状态",
		Long:  "查看指定目录下所有Git仓库的详细状态信息",
		Args:  cobra.MaximumNArgs(1),
		Run:   runStatus,
	}
	
	// Global flags
	rootCmd.PersistentFlags().IntVarP(&cfg.WorkerCount, "workers", "w", cfg.WorkerCount, "并发工作协程数量 (1-50)")
	rootCmd.PersistentFlags().DurationVarP(&cfg.Timeout, "timeout", "t", cfg.Timeout, "每个操作的超时时间")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "显示详细输出")
	rootCmd.PersistentFlags().BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "模拟运行，不执行实际操作")
	rootCmd.PersistentFlags().StringVarP((*string)(&cfg.OutputFormat), "format", "f", string(cfg.OutputFormat), "输出格式 (text|table|json)")
	rootCmd.PersistentFlags().StringSliceVarP(&cfg.IncludePatterns, "include", "i", cfg.IncludePatterns, "包含模式 (可多次指定)")
	rootCmd.PersistentFlags().StringSliceVarP(&cfg.ExcludePatterns, "exclude", "e", cfg.ExcludePatterns, "排除模式 (可多次指定)")
	rootCmd.PersistentFlags().BoolVar(&cfg.SaveReport, "save-report", cfg.SaveReport, "保存报告到文件")
	rootCmd.PersistentFlags().StringVar(&cfg.ReportFile, "report-file", cfg.ReportFile, "报告文件路径")
	
	// Add commands
	rootCmd.AddCommand(updateCmd, scanCmd, statusCmd)
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func runUpdate(cmd *cobra.Command, args []string) {
	directory := getCurrentDirectory(args)
	
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		os.Exit(1)
	}
	
	// 初始化组件
	scannerInstance := scanner.NewScanner()
	reporterInstance := reporter.NewReporter(cfg.OutputFormat, cfg.Verbose)
	
	if cfg.Verbose {
		scannerInstance.SetLogLevel(logrus.DebugLevel)
	}
	
	fmt.Printf("🔍 正在扫描目录: %s\n", directory)
	
	// 扫描仓库
	repositories, err := scannerInstance.ScanDirectoryWithFilter(directory, cfg.IncludePatterns, cfg.ExcludePatterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "扫描失败: %v\n", err)
		os.Exit(1)
	}
	
	if len(repositories) == 0 {
		fmt.Println("未发现任何Git仓库")
		return
	}
	
	fmt.Printf("📦 发现 %d 个Git仓库\n", len(repositories))
	
	// 配置更新器
	updaterConfig := updater.UpdaterConfig{
		WorkerCount: cfg.WorkerCount,
		Timeout:     cfg.Timeout,
		DryRun:      cfg.DryRun,
	}
	
	updaterInstance := updater.NewUpdater(updaterConfig)
	if cfg.Verbose {
		updaterInstance.SetLogLevel(logrus.DebugLevel)
	}
	
	// 初始化进度条
	description := "更新仓库"
	if cfg.DryRun {
		description = "模拟更新"
	}
	reporterInstance.InitProgressBar(len(repositories), description)
	
	// 执行更新
	fmt.Printf("🚀 开始更新，使用 %d 个工作协程\n", cfg.WorkerCount)
	
	results, err := updaterInstance.UpdateRepositories(repositories, func(result updater.UpdateResult) {
		reporterInstance.UpdateProgress()
	})
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "更新过程出错: %v\n", err)
		os.Exit(1)
	}
	
	reporterInstance.FinishProgress()
	
	// 显示结果
	reporterInstance.ReportUpdateResults(results)
	
	// 保存报告
	if cfg.SaveReport {
		filename := cfg.ReportFile
		if filename == "" {
			filename = fmt.Sprintf("reposense-update-%s.json", time.Now().Format("20060102-150405"))
		}
		
		if err := reporterInstance.SaveReport(filename, results); err != nil {
			fmt.Fprintf(os.Stderr, "保存报告失败: %v\n", err)
		} else {
			fmt.Printf("📄 报告已保存到: %s\n", filename)
		}
	}
}

func runScan(cmd *cobra.Command, args []string) {
	directory := getCurrentDirectory(args)
	
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		os.Exit(1)
	}
	
	// 初始化组件
	scannerInstance := scanner.NewScanner()
	reporterInstance := reporter.NewReporter(cfg.OutputFormat, cfg.Verbose)
	
	if cfg.Verbose {
		scannerInstance.SetLogLevel(logrus.DebugLevel)
	}
	
	fmt.Printf("🔍 正在扫描目录: %s\n", directory)
	
	// 扫描仓库
	repositories, err := scannerInstance.ScanDirectoryWithFilter(directory, cfg.IncludePatterns, cfg.ExcludePatterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "扫描失败: %v\n", err)
		os.Exit(1)
	}
	
	// 显示结果
	reporterInstance.ReportScanResults(repositories)
	
	// 保存报告
	if cfg.SaveReport {
		filename := cfg.ReportFile
		if filename == "" {
			filename = fmt.Sprintf("reposense-scan-%s.json", time.Now().Format("20060102-150405"))
		}
		
		if err := reporterInstance.SaveReport(filename, repositories); err != nil {
			fmt.Fprintf(os.Stderr, "保存报告失败: %v\n", err)
		} else {
			fmt.Printf("📄 报告已保存到: %s\n", filename)
		}
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	directory := getCurrentDirectory(args)
	
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		os.Exit(1)
	}
	
	// 初始化组件
	scannerInstance := scanner.NewScanner()
	reporterInstance := reporter.NewReporter(cfg.OutputFormat, cfg.Verbose)
	statusCollector := scanner.NewStatusCollector(cfg.Timeout)
	
	if cfg.Verbose {
		scannerInstance.SetLogLevel(logrus.DebugLevel)
		statusCollector.SetLogLevel(logrus.DebugLevel)
	}
	
	fmt.Printf("🔍 正在扫描目录: %s\n", directory)
	
	// 扫描仓库
	repositories, err := scannerInstance.ScanDirectoryWithFilter(directory, cfg.IncludePatterns, cfg.ExcludePatterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "扫描失败: %v\n", err)
		os.Exit(1)
	}
	
	if len(repositories) == 0 {
		fmt.Println("未发现任何Git仓库")
		return
	}
	
	fmt.Printf("📦 发现 %d 个Git仓库，正在收集状态信息...\n", len(repositories))
	
	// 收集状态
	statuses := statusCollector.CollectBatchStatus(repositories)
	
	// 显示结果
	reporterInstance.ReportStatusResults(statuses)
	
	// 保存报告
	if cfg.SaveReport {
		filename := cfg.ReportFile
		if filename == "" {
			filename = fmt.Sprintf("reposense-status-%s.json", time.Now().Format("20060102-150405"))
		}
		
		if err := reporterInstance.SaveReport(filename, statuses); err != nil {
			fmt.Fprintf(os.Stderr, "保存报告失败: %v\n", err)
		} else {
			fmt.Printf("📄 报告已保存到: %s\n", filename)
		}
	}
}

func getCurrentDirectory(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	
	// 使用当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法获取当前目录: %v\n", err)
		os.Exit(1)
	}
	
	return wd
}

func init() {
	// 设置字符串到ReportFormat的转换
	cobra.OnInitialize(func() {
		// 验证输出格式
		switch strings.ToLower(string(cfg.OutputFormat)) {
		case "text":
			cfg.OutputFormat = reporter.FormatText
		case "table":
			cfg.OutputFormat = reporter.FormatTable
		case "json":
			cfg.OutputFormat = reporter.FormatJSON
		default:
			cfg.OutputFormat = reporter.FormatText
		}
	})
}