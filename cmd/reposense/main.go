package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"reposense/internal/config"
	"reposense/pkg/llm"
	"reposense/pkg/reporter"
	"reposense/pkg/scanner"
	"reposense/pkg/updater"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfg                 *config.Config
	disableLLM          bool
	gitPullStrategy     string
	gitAllowInteractive bool
)

func main() {
	cfg = config.LoadConfig()
	
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
	
	// List command
	var listCmd = &cobra.Command{
		Use:   "list [directory]",
		Short: "列出仓库及描述",
		Long:  "列出指定目录下的所有Git仓库，并显示项目描述。默认启用LLM智能描述",
		Args:  cobra.MaximumNArgs(1),
		Run:   runList,
	}
	
	// Config command
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "配置管理",
		Long:  "管理RepoSense的配置设置",
	}
	
	var configShowCmd = &cobra.Command{
		Use:   "show",
		Short: "显示当前配置",
		Long:  "显示当前的配置设置",
		Run:   runConfigShow,
	}
	
	var configSetCmd = &cobra.Command{
		Use:   "set",
		Short: "保存当前配置",
		Long:  "将当前的命令行参数保存为默认配置",
		Run:   runConfigSet,
	}
	
	var configPathCmd = &cobra.Command{
		Use:   "path",
		Short: "显示配置文件路径",
		Long:  "显示配置文件的完整路径",
		Run:   runConfigPath,
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
	
	// LLM flags
	rootCmd.PersistentFlags().BoolVar(&cfg.EnableLLM, "enable-llm", cfg.EnableLLM, "启用LLM智能描述提取 (默认启用)")
	rootCmd.PersistentFlags().BoolVar(&disableLLM, "disable-llm", false, "禁用LLM智能描述提取")
	rootCmd.PersistentFlags().StringVar(&cfg.LLMProvider, "llm-provider", cfg.LLMProvider, "LLM提供商 (openai|openai-compatible|gemini|claude|ollama)")
	rootCmd.PersistentFlags().StringVar(&cfg.LLMModel, "llm-model", cfg.LLMModel, "LLM模型名称")
	rootCmd.PersistentFlags().StringVar(&cfg.LLMAPIKey, "llm-api-key", cfg.LLMAPIKey, "LLM API密钥")
	rootCmd.PersistentFlags().StringVar(&cfg.LLMBaseURL, "llm-base-url", cfg.LLMBaseURL, "LLM API基础URL")
	rootCmd.PersistentFlags().StringVar(&cfg.LLMLanguage, "llm-language", cfg.LLMLanguage, "描述语言 (zh|en|ja)")
	rootCmd.PersistentFlags().DurationVar(&cfg.LLMTimeout, "llm-timeout", cfg.LLMTimeout, "LLM请求超时时间")
	
	// Git operation flags
	rootCmd.PersistentFlags().StringVar(&gitPullStrategy, "git-pull-strategy", "ff-only", "Git拉取策略 (ff-only|merge|rebase)")
	rootCmd.PersistentFlags().BoolVar(&gitAllowInteractive, "git-allow-interactive", false, "允许Git交互操作 (可能导致挂起)")
	
	// List command specific flags
	listCmd.Flags().BoolVar(&cfg.SortByTime, "sort-by-time", cfg.SortByTime, "按更新时间排序")
	listCmd.Flags().BoolVarP(&cfg.Reverse, "reverse", "r", cfg.Reverse, "倒序显示")
	
	// Add sub-commands to config
	configCmd.AddCommand(configShowCmd, configSetCmd, configPathCmd)
	
	// Add commands
	rootCmd.AddCommand(updateCmd, scanCmd, statusCmd, listCmd, configCmd)
	
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
		WorkerCount:       cfg.WorkerCount,
		Timeout:           cfg.Timeout,
		DryRun:            cfg.DryRun,
		GitPullStrategy:   gitPullStrategy,
		GitNonInteractive: !gitAllowInteractive, // 反转：不允许交互 = 启用非交互模式
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

func runList(cmd *cobra.Command, args []string) {
	directory := getCurrentDirectory(args)
	
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		os.Exit(1)
	}
	
	// 检查是否从环境变量读取API密钥
	if cfg.LLMAPIKey == "" && cfg.EnableLLM {
		if key := os.Getenv("OPENAI_API_KEY"); key != "" && cfg.LLMProvider == "openai" {
			cfg.LLMAPIKey = key
		} else if key := os.Getenv("GEMINI_API_KEY"); key != "" && cfg.LLMProvider == "gemini" {
			cfg.LLMAPIKey = key
		} else if key := os.Getenv("CLAUDE_API_KEY"); key != "" && cfg.LLMProvider == "claude" {
			cfg.LLMAPIKey = key
		} else if key := os.Getenv("LLM_API_KEY"); key != "" {
			cfg.LLMAPIKey = key
		}
	}
	
	// 初始化LLM描述服务
	var descriptionService *llm.DescriptionService
	if cfg.EnableLLM {
		provider := llm.Provider(cfg.LLMProvider)
		if err := llm.ValidateConfiguration(provider, cfg.LLMAPIKey, cfg.LLMBaseURL); err != nil {
			fmt.Fprintf(os.Stderr, "LLM配置错误: %v\n", err)
			fmt.Println("提示: 使用 --llm-api-key 设置API密钥，或设置环境变量")
			os.Exit(1)
		}
		
		descriptionService = llm.NewDescriptionService(
			provider,
			cfg.LLMModel,
			cfg.LLMAPIKey,
			cfg.LLMBaseURL,
			cfg.LLMLanguage,
			cfg.LLMTimeout,
			true,
		)
		
		fmt.Printf("🤖 已启用LLM智能描述 (提供商: %s, 模型: %s, 语言: %s)\n", 
			cfg.LLMProvider, cfg.LLMModel, cfg.LLMLanguage)
	}
	
	// 初始化组件
	var scannerInstance *scanner.Scanner
	if descriptionService != nil {
		scannerInstance = scanner.NewScannerWithLLM(descriptionService)
	} else {
		scannerInstance = scanner.NewScanner()
	}
	
	reporterInstance := reporter.NewReporter(cfg.OutputFormat, cfg.Verbose)
	
	if cfg.Verbose {
		scannerInstance.SetLogLevel(logrus.DebugLevel)
	}
	
	fmt.Printf("🔍 正在扫描目录: %s\n", directory)
	
	// 扫描仓库并获取描述
	repositories, err := scannerInstance.ScanDirectoryWithDescription(directory, cfg.IncludePatterns, cfg.ExcludePatterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "扫描失败: %v\n", err)
		os.Exit(1)
	}
	
	if len(repositories) == 0 {
		fmt.Println("未发现任何Git仓库")
		return
	}
	
	fmt.Printf("📦 发现 %d 个Git仓库\n", len(repositories))
	
	// 显示结果
	reporterInstance.ReportListResults(repositories, cfg.SortByTime, cfg.Reverse)
	
	// 保存报告
	if cfg.SaveReport {
		filename := cfg.ReportFile
		if filename == "" {
			filename = fmt.Sprintf("reposense-list-%s.json", time.Now().Format("20060102-150405"))
		}
		
		if err := reporterInstance.SaveReport(filename, repositories); err != nil {
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

func runConfigShow(cmd *cobra.Command, args []string) {
	fmt.Printf("配置文件路径: %s\n", config.GetConfigPath())
	fmt.Println("\n当前配置:")
	fmt.Printf("  工作协程数: %d\n", cfg.WorkerCount)
	fmt.Printf("  超时时间: %v\n", cfg.Timeout)
	fmt.Printf("  输出格式: %s\n", cfg.OutputFormat)
	fmt.Printf("  启用LLM: %v\n", cfg.EnableLLM)
	if cfg.EnableLLM {
		fmt.Printf("  LLM提供商: %s\n", cfg.LLMProvider)
		fmt.Printf("  LLM模型: %s\n", cfg.LLMModel)
		fmt.Printf("  LLM基础URL: %s\n", cfg.LLMBaseURL)
		fmt.Printf("  LLM语言: %s\n", cfg.LLMLanguage)
		fmt.Printf("  LLM超时: %v\n", cfg.LLMTimeout)
		if cfg.LLMAPIKey != "" {
			fmt.Printf("  LLM API密钥: %s...%s\n", cfg.LLMAPIKey[:min(8, len(cfg.LLMAPIKey))], cfg.LLMAPIKey[max(0, len(cfg.LLMAPIKey)-4):])
		} else {
			fmt.Printf("  LLM API密钥: (未设置)\n")
		}
	}
}

func runConfigSet(cmd *cobra.Command, args []string) {
	if err := cfg.SaveConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 配置已保存到: %s\n", config.GetConfigPath())
}

func runConfigPath(cmd *cobra.Command, args []string) {
	fmt.Println(config.GetConfigPath())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	// 设置字符串到ReportFormat的转换
	cobra.OnInitialize(func() {
		// 处理 disable-llm 标志
		if disableLLM {
			cfg.EnableLLM = false
		}
		
		// 设置git策略默认值
		if gitPullStrategy == "" {
			gitPullStrategy = "ff-only"
		}
		
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