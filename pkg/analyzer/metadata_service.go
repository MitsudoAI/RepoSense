package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reposense/pkg/llm"

	"github.com/sirupsen/logrus"
)

// MetadataService provides comprehensive repository metadata analysis
type MetadataService struct {
	logger            *logrus.Logger
	languageDetector  *LanguageDetector
	frameworkDetector *FrameworkDetector
	licenseDetector   *LicenseDetector
	llmService        *llm.DescriptionService
}

// NewMetadataService creates a new metadata analysis service
func NewMetadataService() *MetadataService {
	return &MetadataService{
		logger:            logrus.New(),
		languageDetector:  NewLanguageDetector(),
		frameworkDetector: NewFrameworkDetector(),
		licenseDetector:   NewLicenseDetector(),
	}
}

// NewMetadataServiceWithLLM creates a new metadata service with LLM support
func NewMetadataServiceWithLLM(llmService *llm.DescriptionService) *MetadataService {
	service := NewMetadataService()
	service.llmService = llmService
	return service
}

// SetLogLevel sets the logging level for all components
func (ms *MetadataService) SetLogLevel(level logrus.Level) {
	ms.logger.SetLevel(level)
	ms.languageDetector.SetLogLevel(level)
	ms.frameworkDetector.SetLogLevel(level)
	ms.licenseDetector.SetLogLevel(level)
	if ms.llmService != nil {
		ms.llmService.SetLogLevel(level)
	}
}

// AnalyzeRepository performs comprehensive metadata analysis on a repository
func (ms *MetadataService) AnalyzeRepository(repoPath string, config *AnalysisConfig) (*ProjectMetadata, error) {
	ms.logger.Infof("开始分析项目: %s", repoPath)
	startTime := time.Now()
	
	metadata := &ProjectMetadata{
		AnalyzedAt: startTime,
	}
	
	// Basic repository information
	if err := ms.analyzeBasicInfo(repoPath, metadata, config); err != nil {
		ms.logger.WithError(err).Warn("基础信息分析失败")
	}
	
	// Language detection
	if config.IncludeLanguages {
		languages, err := ms.languageDetector.DetectLanguages(repoPath, config)
		if err != nil {
			ms.logger.WithError(err).Warn("语言检测失败")
		} else {
			metadata.Languages = languages
			metadata.MainLanguage = ms.languageDetector.GetMainLanguage(languages)
			
			// Calculate total lines of code
			for _, lang := range languages {
				metadata.TotalLinesOfCode += lang.LinesOfCode
			}
		}
	}
	
	// Framework detection
	if config.IncludeFrameworks {
		frameworks, err := ms.frameworkDetector.DetectFrameworks(repoPath, config)
		if err != nil {
			ms.logger.WithError(err).Warn("框架检测失败")
		} else {
			metadata.Frameworks = frameworks
		}
	}
	
	// License detection
	if config.IncludeLicenses {
		licenses, err := ms.licenseDetector.DetectLicenses(repoPath, config)
		if err != nil {
			ms.logger.WithError(err).Warn("许可证检测失败")
		} else {
			metadata.Licenses = licenses
			metadata.HasLicense = len(licenses) > 0
		}
	}
	
	// Dependency analysis
	if config.IncludeDependencies {
		dependencies, err := ms.analyzeDependencies(repoPath, config)
		if err != nil {
			ms.logger.WithError(err).Warn("依赖分析失败")
		} else {
			metadata.Dependencies = dependencies
		}
	}
	
	// Project type detection
	metadata.ProjectType = ms.detectProjectType(metadata)
	
	// Quality and complexity analysis
	metadata.ComplexityScore = ms.calculateComplexityScore(metadata)
	metadata.QualityScore = ms.calculateQualityScore(metadata)
	
	// Generate structure hash
	if structureHash, err := GenerateStructureHash(repoPath, config.IgnorePatterns); err == nil {
		metadata.StructureHash = structureHash
	}
	
	// Generate project description and enhanced description using LLM if available
	if ms.llmService != nil {
		// Generate basic description
		description := ms.llmService.ExtractDescription(repoPath)
		if description != "" && description != "暂无描述" {
			metadata.Description = description
		} else {
			// Fallback to metadata-based description
			metadata.Description = ms.generateDescriptionFromMetadata(metadata)
		}
		
		// Generate enhanced description with context
		enhancedDescription := ms.generateEnhancedDescriptionWithContext(repoPath, metadata)
		if enhancedDescription != "" {
			metadata.EnhancedDescription = enhancedDescription
			ms.logger.Debugf("生成增强描述: %s", enhancedDescription[:min(100, len(enhancedDescription))])
		}
	} else {
		// Without LLM, use metadata-based description
		metadata.Description = ms.generateDescriptionFromMetadata(metadata)
	}
	
	duration := time.Since(startTime)
	ms.logger.Infof("项目分析完成，耗时: %v", duration)
	
	return metadata, nil
}

// analyzeBasicInfo analyzes basic repository information
func (ms *MetadataService) analyzeBasicInfo(repoPath string, metadata *ProjectMetadata, config *AnalysisConfig) error {
	// Calculate repository size
	size, err := CalculateDirectorySize(repoPath, config.IgnorePatterns)
	if err == nil {
		metadata.RepositorySize = size
	}
	
	// Count files and directories
	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		// Get relative path
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}
		
		// Skip ignored files
		if ShouldIgnoreFile(relPath, config.IgnorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		if info.IsDir() {
			metadata.DirectoryCount++
		} else {
			metadata.FileCount++
		}
		
		return nil
	})
	
	// Check for important files
	configFiles, err := FindConfigFiles(repoPath)
	if err == nil {
		metadata.HasReadme = ms.hasAnyFile(configFiles, []string{"README"})
		metadata.HasLicense = ms.hasAnyFile(configFiles, []string{"LICENSE"})
		metadata.HasTests = ms.detectTestPresence(repoPath, configFiles)
		metadata.HasCI = ms.detectCIPresence(repoPath, configFiles)
		metadata.HasDocs = ms.detectDocsPresence(repoPath)
	}
	
	return nil
}

// analyzeDependencies analyzes project dependencies
func (ms *MetadataService) analyzeDependencies(repoPath string, config *AnalysisConfig) ([]DependencyInfo, error) {
	var dependencies []DependencyInfo
	
	configFiles, err := FindConfigFiles(repoPath)
	if err != nil {
		return dependencies, err
	}
	
	// Analyze package.json
	if packageJSON, exists := configFiles["package.json"]; exists {
		deps, err := ms.analyzePackageJSON(packageJSON)
		if err == nil {
			dependencies = append(dependencies, deps...)
		}
	}
	
	// Analyze requirements.txt
	if requirementsTxt, exists := configFiles["requirements.txt"]; exists {
		deps, err := ms.analyzeRequirementsTxt(requirementsTxt)
		if err == nil {
			dependencies = append(dependencies, deps...)
		}
	}
	
	// Analyze go.mod
	if goMod, exists := configFiles["go.mod"]; exists {
		deps, err := ms.analyzeGoMod(goMod)
		if err == nil {
			dependencies = append(dependencies, deps...)
		}
	}
	
	// Analyze Cargo.toml
	if cargoToml, exists := configFiles["Cargo.toml"]; exists {
		deps, err := ms.analyzeCargoToml(cargoToml)
		if err == nil {
			dependencies = append(dependencies, deps...)
		}
	}
	
	return dependencies, nil
}

// analyzePackageJSON analyzes package.json dependencies
func (ms *MetadataService) analyzePackageJSON(filePath string) ([]DependencyInfo, error) {
	var dependencies []DependencyInfo
	
	content, err := ReadFileContent(filePath, 1024*1024)
	if err != nil {
		return dependencies, err
	}
	
	var packageJSON struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		PeerDependencies map[string]string `json:"peerDependencies"`
	}
	
	if err := json.Unmarshal([]byte(content), &packageJSON); err != nil {
		return dependencies, err
	}
	
	// Process production dependencies
	for name, version := range packageJSON.Dependencies {
		dependencies = append(dependencies, DependencyInfo{
			Name:           name,
			Version:        ms.cleanVersion(version),
			Type:           "production",
			PackageManager: "npm",
			SourceFile:     filePath,
		})
	}
	
	// Process dev dependencies
	for name, version := range packageJSON.DevDependencies {
		dependencies = append(dependencies, DependencyInfo{
			Name:           name,
			Version:        ms.cleanVersion(version),
			Type:           "development",
			PackageManager: "npm",
			SourceFile:     filePath,
		})
	}
	
	// Process peer dependencies
	for name, version := range packageJSON.PeerDependencies {
		dependencies = append(dependencies, DependencyInfo{
			Name:           name,
			Version:        ms.cleanVersion(version),
			Type:           "peer",
			PackageManager: "npm",
			SourceFile:     filePath,
		})
	}
	
	return dependencies, nil
}

// analyzeRequirementsTxt analyzes requirements.txt dependencies
func (ms *MetadataService) analyzeRequirementsTxt(filePath string) ([]DependencyInfo, error) {
	var dependencies []DependencyInfo
	
	content, err := ReadFileContent(filePath, 1024*1024)
	if err != nil {
		return dependencies, err
	}
	
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse package name and version
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '=' || r == '>' || r == '<' || r == '~' || r == '!'
		})
		
		if len(parts) > 0 {
			name := strings.TrimSpace(parts[0])
			version := ""
			if len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
			}
			
			dependencies = append(dependencies, DependencyInfo{
				Name:           name,
				Version:        version,
				Type:           "production",
				PackageManager: "pip",
				SourceFile:     filePath,
			})
		}
	}
	
	return dependencies, nil
}

// analyzeGoMod analyzes go.mod dependencies
func (ms *MetadataService) analyzeGoMod(filePath string) ([]DependencyInfo, error) {
	var dependencies []DependencyInfo
	
	content, err := ReadFileContent(filePath, 1024*1024)
	if err != nil {
		return dependencies, err
	}
	
	lines := strings.Split(content, "\n")
	inRequireBlock := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "require (" {
			inRequireBlock = true
			continue
		}
		
		if line == ")" && inRequireBlock {
			inRequireBlock = false
			continue
		}
		
		if inRequireBlock || strings.HasPrefix(line, "require ") {
			// Parse dependency line
			if strings.HasPrefix(line, "require ") {
				line = strings.TrimPrefix(line, "require ")
			}
			
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				version := parts[1]
				
				// Skip indirect dependencies for now
				if len(parts) < 3 || parts[2] != "// indirect" {
					dependencies = append(dependencies, DependencyInfo{
						Name:           name,
						Version:        version,
						Type:           "production",
						PackageManager: "go",
						SourceFile:     filePath,
					})
				}
			}
		}
	}
	
	return dependencies, nil
}

// analyzeCargoToml analyzes Cargo.toml dependencies
func (ms *MetadataService) analyzeCargoToml(filePath string) ([]DependencyInfo, error) {
	var dependencies []DependencyInfo
	
	content, err := ReadFileContent(filePath, 1024*1024)
	if err != nil {
		return dependencies, err
	}
	
	lines := strings.Split(content, "\n")
	inDependenciesSection := false
	inDevDependenciesSection := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "[dependencies]" {
			inDependenciesSection = true
			inDevDependenciesSection = false
			continue
		}
		
		if line == "[dev-dependencies]" {
			inDependenciesSection = false
			inDevDependenciesSection = true
			continue
		}
		
		if strings.HasPrefix(line, "[") && line != "[dependencies]" && line != "[dev-dependencies]" {
			inDependenciesSection = false
			inDevDependenciesSection = false
			continue
		}
		
		if (inDependenciesSection || inDevDependenciesSection) && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), "\"")
				
				depType := "production"
				if inDevDependenciesSection {
					depType = "development"
				}
				
				dependencies = append(dependencies, DependencyInfo{
					Name:           name,
					Version:        version,
					Type:           depType,
					PackageManager: "cargo",
					SourceFile:     filePath,
				})
			}
		}
	}
	
	return dependencies, nil
}

// detectProjectType determines the type of the project
func (ms *MetadataService) detectProjectType(metadata *ProjectMetadata) string {
	// Check frameworks first
	for _, framework := range metadata.Frameworks {
		switch framework.Category {
		case "cli":
			return "cli-tool"
		case "mobile":
			return "mobile-app"
		case "desktop":
			return "desktop-app"
		case "backend":
			return "backend-service"
		case "frontend":
			return "frontend-app"
		case "fullstack":
			return "fullstack-app"
		case "static-site":
			return "static-site"
		case "machine-learning":
			return "ml-project"
		case "data-science":
			return "data-analysis"
		}
	}
	
	// Check main language
	switch strings.ToLower(metadata.MainLanguage) {
	case "javascript", "typescript":
		// Check if it's a Node.js project
		for _, dep := range metadata.Dependencies {
			if dep.Name == "express" || dep.Name == "koa" || dep.Name == "fastify" {
				return "backend-service"
			}
			if dep.Name == "react" || dep.Name == "vue" || dep.Name == "angular" {
				return "frontend-app"
			}
		}
		return "web-project"
	case "python":
		// Check for data science libraries
		for _, dep := range metadata.Dependencies {
			if dep.Name == "tensorflow" || dep.Name == "pytorch" || dep.Name == "scikit-learn" {
				return "ml-project"
			}
			if dep.Name == "pandas" || dep.Name == "numpy" || dep.Name == "jupyter" {
				return "data-analysis"
			}
			if dep.Name == "django" || dep.Name == "flask" || dep.Name == "fastapi" {
				return "backend-service"
			}
		}
		return "python-script"
	case "go":
		return "go-application"
	case "rust":
		return "rust-application"
	case "java":
		return "java-application"
	case "c", "c++":
		return "native-application"
	}
	
	// Default to library if we can't determine
	return "library"
}

// calculateComplexityScore calculates a complexity score for the project
func (ms *MetadataService) calculateComplexityScore(metadata *ProjectMetadata) float64 {
	score := 0.0
	
	// Lines of code factor (logarithmic scale)
	if metadata.TotalLinesOfCode > 0 {
		score += float64(metadata.TotalLinesOfCode) / 10000.0 // Normalize to reasonable scale
	}
	
	// Number of languages factor
	score += float64(len(metadata.Languages)) * 0.5
	
	// Number of frameworks factor
	score += float64(len(metadata.Frameworks)) * 0.3
	
	// Number of dependencies factor
	score += float64(len(metadata.Dependencies)) * 0.1
	
	// File count factor
	score += float64(metadata.FileCount) / 1000.0
	
	// Cap at 10.0
	if score > 10.0 {
		score = 10.0
	}
	
	return score
}

// calculateQualityScore calculates a quality score for the project
func (ms *MetadataService) calculateQualityScore(metadata *ProjectMetadata) float64 {
	score := 0.0
	maxScore := 10.0
	
	// Has README
	if metadata.HasReadme {
		score += 2.0
	}
	
	// Has LICENSE
	if metadata.HasLicense {
		score += 1.5
	}
	
	// Has tests
	if metadata.HasTests {
		score += 2.0
	}
	
	// Has CI
	if metadata.HasCI {
		score += 1.5
	}
	
	// Has documentation
	if metadata.HasDocs {
		score += 1.0
	}
	
	// Language diversity (but not too much)
	langCount := len(metadata.Languages)
	if langCount >= 1 && langCount <= 3 {
		score += 1.0
	} else if langCount > 3 {
		score += 0.5 // Too many languages might indicate maintenance issues
	}
	
	// Framework usage (having frameworks indicates structured development)
	if len(metadata.Frameworks) > 0 {
		score += 1.0
	}
	
	return (score / maxScore) * 10.0 // Normalize to 0-10 scale
}

// Helper methods

func (ms *MetadataService) hasAnyFile(configFiles map[string]string, keys []string) bool {
	for _, key := range keys {
		if _, exists := configFiles[key]; exists {
			return true
		}
	}
	return false
}

func (ms *MetadataService) detectTestPresence(repoPath string, configFiles map[string]string) bool {
	// Check for test directories
	testDirs := []string{"test", "tests", "__tests__", "spec", "specs"}
	for _, dir := range testDirs {
		testPath := filepath.Join(repoPath, dir)
		if _, err := os.Stat(testPath); err == nil {
			return true
		}
	}
	
	// Check for test files
	testPatterns := []string{"*test.go", "*_test.py", "*.test.js", "*.spec.js"}
	for _, pattern := range testPatterns {
		matches, _ := filepath.Glob(filepath.Join(repoPath, "**", pattern))
		if len(matches) > 0 {
			return true
		}
	}
	
	return false
}

func (ms *MetadataService) detectCIPresence(repoPath string, configFiles map[string]string) bool {
	// Check for CI configuration files
	ciFiles := []string{
		".github/workflows",
		".gitlab-ci.yml",
		".travis.yml",
		"circle.yml",
		".circleci/config.yml",
		"jenkins.yml",
		"Jenkinsfile",
		".drone.yml",
		"azure-pipelines.yml",
	}
	
	for _, file := range ciFiles {
		ciPath := filepath.Join(repoPath, file)
		if _, err := os.Stat(ciPath); err == nil {
			return true
		}
	}
	
	return false
}

func (ms *MetadataService) detectDocsPresence(repoPath string) bool {
	// Check for documentation directories
	docDirs := []string{"docs", "doc", "documentation", "wiki"}
	for _, dir := range docDirs {
		docPath := filepath.Join(repoPath, dir)
		if _, err := os.Stat(docPath); err == nil {
			return true
		}
	}
	
	// Check for documentation files
	docFiles := []string{"CHANGELOG.md", "CONTRIBUTING.md", "API.md", "docs.md"}
	for _, file := range docFiles {
		docPath := filepath.Join(repoPath, file)
		if _, err := os.Stat(docPath); err == nil {
			return true
		}
	}
	
	return false
}

func (ms *MetadataService) cleanVersion(version string) string {
	// Remove common prefixes
	version = strings.TrimPrefix(version, "^")
	version = strings.TrimPrefix(version, "~")
	version = strings.TrimPrefix(version, ">=")
	version = strings.TrimPrefix(version, "<=")
	version = strings.TrimPrefix(version, ">")
	version = strings.TrimPrefix(version, "<")
	version = strings.TrimPrefix(version, "=")
	
	return strings.TrimSpace(version)
}

// GetAnalysisReport generates a comprehensive analysis report
func (ms *MetadataService) GetAnalysisReport(metadata *ProjectMetadata) map[string]interface{} {
	report := make(map[string]interface{})
	
	// Basic information
	report["project_type"] = metadata.ProjectType
	report["main_language"] = metadata.MainLanguage
	report["total_lines_of_code"] = metadata.TotalLinesOfCode
	report["file_count"] = metadata.FileCount
	report["directory_count"] = metadata.DirectoryCount
	report["repository_size"] = metadata.RepositorySize
	
	// Quality indicators
	report["has_readme"] = metadata.HasReadme
	report["has_license"] = metadata.HasLicense
	report["has_tests"] = metadata.HasTests
	report["has_ci"] = metadata.HasCI
	report["has_docs"] = metadata.HasDocs
	
	// Scores
	report["complexity_score"] = fmt.Sprintf("%.1f/10.0", metadata.ComplexityScore)
	report["quality_score"] = fmt.Sprintf("%.1f/10.0", metadata.QualityScore)
	
	// Languages
	report["languages"] = metadata.Languages
	
	// Frameworks
	if len(metadata.Frameworks) > 0 {
		report["frameworks"] = metadata.Frameworks
	}
	
	// Licenses
	if len(metadata.Licenses) > 0 {
		report["licenses"] = metadata.Licenses
	}
	
	// Top dependencies
	if len(metadata.Dependencies) > 0 {
		var topDeps []DependencyInfo
		limit := 10
		if len(metadata.Dependencies) < limit {
			limit = len(metadata.Dependencies)
		}
		topDeps = metadata.Dependencies[:limit]
		report["top_dependencies"] = topDeps
	}
	
	report["analyzed_at"] = metadata.AnalyzedAt
	
	return report
}

// generateEnhancedDescriptionWithContext generates an enhanced project description with full context
func (ms *MetadataService) generateEnhancedDescriptionWithContext(repoPath string, metadata *ProjectMetadata) string {
	if ms.llmService == nil {
		return ""
	}
	
	// Create a comprehensive project summary for LLM context
	summary := ms.createProjectSummary(repoPath, metadata)
	
	// For now, return the summary as enhanced description
	// In the future, this could be sent to LLM for further enhancement
	return ms.createEnhancedDescriptionFromSummary(summary, metadata)
}

// createEnhancedDescriptionFromSummary creates enhanced description from project summary
func (ms *MetadataService) createEnhancedDescriptionFromSummary(summary string, metadata *ProjectMetadata) string {
	var desc strings.Builder
	
	// Start with project type and main language
	desc.WriteString(ms.generateDescriptionFromMetadata(metadata))
	
	// Add technical details
	if len(metadata.Languages) > 1 {
		desc.WriteString(fmt.Sprintf(" 该项目使用多种编程语言，其中%s占主导地位", metadata.MainLanguage))
		if len(metadata.Languages) > 2 {
			desc.WriteString(fmt.Sprintf("，还包含%s等", metadata.Languages[1].Name))
		}
		desc.WriteString("。")
	}
	
	// Add framework information
	if len(metadata.Frameworks) > 0 {
		frameworksByCategory := make(map[string][]string)
		for _, framework := range metadata.Frameworks {
			frameworksByCategory[framework.Category] = append(frameworksByCategory[framework.Category], framework.Name)
		}
		
		for category, frameworks := range frameworksByCategory {
			if len(frameworks) > 0 {
				desc.WriteString(fmt.Sprintf(" 在%s方面使用了%s", 
					ms.translateCategory(category), strings.Join(frameworks[:min(2, len(frameworks))], "和")))
				if len(frameworks) > 2 {
					desc.WriteString("等框架")
				} else {
					desc.WriteString("框架")
				}
				desc.WriteString("。")
			}
		}
	}
	
	// Add project scale and quality indicators
	if metadata.TotalLinesOfCode > 0 {
		var scale string
		if metadata.TotalLinesOfCode > 50000 {
			scale = "大型"
		} else if metadata.TotalLinesOfCode > 10000 {
			scale = "中大型"
		} else if metadata.TotalLinesOfCode > 1000 {
			scale = "中型"
		} else {
			scale = "小型"
		}
		desc.WriteString(fmt.Sprintf(" 这是一个%s项目，包含约%d行代码", scale, metadata.TotalLinesOfCode))
		
		if metadata.FileCount > 0 {
			desc.WriteString(fmt.Sprintf("，分布在%d个文件中", metadata.FileCount))
		}
		desc.WriteString("。")
	}
	
	// Add quality indicators
	qualityFeatures := []string{}
	if metadata.HasTests {
		qualityFeatures = append(qualityFeatures, "完善的测试")
	}
	if metadata.HasCI {
		qualityFeatures = append(qualityFeatures, "持续集成")
	}
	if metadata.HasDocs {
		qualityFeatures = append(qualityFeatures, "详细文档")
	}
	if metadata.HasLicense {
		qualityFeatures = append(qualityFeatures, "开源许可")
	}
	
	if len(qualityFeatures) > 0 {
		desc.WriteString(fmt.Sprintf(" 项目具备%s", strings.Join(qualityFeatures, "、")))
		if metadata.QualityScore >= 8.0 {
			desc.WriteString("，整体质量优秀")
		} else if metadata.QualityScore >= 6.0 {
			desc.WriteString("，整体质量良好")
		}
		desc.WriteString("。")
	}
	
	// Add license information
	if len(metadata.Licenses) > 0 {
		license := metadata.Licenses[0]
		desc.WriteString(fmt.Sprintf(" 采用%s许可证", license.Name))
		switch license.Type {
		case "permissive":
			desc.WriteString("，允许商业使用和修改")
		case "copyleft":
			desc.WriteString("，要求衍生作品保持开源")
		case "public-domain":
			desc.WriteString("，完全开放使用")
		}
		desc.WriteString("。")
	}
	
	return desc.String()
}

// translateCategory translates framework category to Chinese
func (ms *MetadataService) translateCategory(category string) string {
	switch category {
	case "frontend":
		return "前端开发"
	case "backend":
		return "后端开发"
	case "fullstack":
		return "全栈开发"
	case "mobile":
		return "移动开发"
	case "desktop":
		return "桌面开发"
	case "testing":
		return "测试"
	case "build-tool":
		return "构建工具"
	case "data-science":
		return "数据科学"
	case "machine-learning":
		return "机器学习"
	case "cli":
		return "命令行"
	case "orm":
		return "数据库"
	default:
		return category
	}
}

// createProjectSummary creates a comprehensive project summary
func (ms *MetadataService) createProjectSummary(repoPath string, metadata *ProjectMetadata) string {
	var summary strings.Builder
	
	summary.WriteString(fmt.Sprintf("项目类型: %s\n", metadata.ProjectType))
	summary.WriteString(fmt.Sprintf("主要语言: %s\n", metadata.MainLanguage))
	summary.WriteString(fmt.Sprintf("代码行数: %d\n", metadata.TotalLinesOfCode))
	
	if len(metadata.Languages) > 0 {
		summary.WriteString("编程语言: ")
		for i, lang := range metadata.Languages {
			if i > 0 {
				summary.WriteString(", ")
			}
			summary.WriteString(fmt.Sprintf("%s(%.1f%%)", lang.Name, lang.Percentage))
			if i >= 2 { // Only show top 3 languages
				break
			}
		}
		summary.WriteString("\n")
	}
	
	if len(metadata.Frameworks) > 0 {
		summary.WriteString("主要框架: ")
		for i, framework := range metadata.Frameworks {
			if i > 0 {
				summary.WriteString(", ")
			}
			summary.WriteString(framework.Name)
			if i >= 2 { // Only show top 3 frameworks
				break
			}
		}
		summary.WriteString("\n")
	}
	
	if len(metadata.Licenses) > 0 {
		summary.WriteString(fmt.Sprintf("许可证: %s\n", metadata.Licenses[0].Name))
	}
	
	// Project characteristics
	characteristics := []string{}
	if metadata.HasTests {
		characteristics = append(characteristics, "有测试")
	}
	if metadata.HasCI {
		characteristics = append(characteristics, "有CI")
	}
	if metadata.HasDocs {
		characteristics = append(characteristics, "有文档")
	}
	if len(characteristics) > 0 {
		summary.WriteString(fmt.Sprintf("项目特征: %s\n", strings.Join(characteristics, ", ")))
	}
	
	summary.WriteString(fmt.Sprintf("质量评分: %.1f/10.0\n", metadata.QualityScore))
	
	return summary.String()
}

// generateDescriptionFromMetadata generates a description based on metadata analysis
func (ms *MetadataService) generateDescriptionFromMetadata(metadata *ProjectMetadata) string {
	var desc strings.Builder
	
	// Start with project type
	switch metadata.ProjectType {
	case "cli-tool":
		desc.WriteString("一个命令行工具")
	case "backend-service":
		desc.WriteString("一个后端服务")
	case "frontend-app":
		desc.WriteString("一个前端应用")
	case "fullstack-app":
		desc.WriteString("一个全栈应用")
	case "mobile-app":
		desc.WriteString("一个移动应用")
	case "desktop-app":
		desc.WriteString("一个桌面应用")
	case "library":
		desc.WriteString("一个软件库")
	case "ml-project":
		desc.WriteString("一个机器学习项目")
	case "data-analysis":
		desc.WriteString("一个数据分析项目")
	case "static-site":
		desc.WriteString("一个静态网站")
	default:
		desc.WriteString("一个软件项目")
	}
	
	// Add main language
	if metadata.MainLanguage != "" {
		desc.WriteString(fmt.Sprintf("，主要使用%s开发", metadata.MainLanguage))
	}
	
	// Add frameworks
	if len(metadata.Frameworks) > 0 {
		frameworkNames := []string{}
		for i, framework := range metadata.Frameworks {
			frameworkNames = append(frameworkNames, framework.Name)
			if i >= 1 { // Only mention top 2 frameworks
				break
			}
		}
		desc.WriteString(fmt.Sprintf("，采用%s框架", strings.Join(frameworkNames, "和")))
	}
	
	// Add scale information
	if metadata.TotalLinesOfCode > 0 {
		if metadata.TotalLinesOfCode > 10000 {
			desc.WriteString("，规模较大")
		} else if metadata.TotalLinesOfCode > 1000 {
			desc.WriteString("，规模适中")
		} else {
			desc.WriteString("，规模较小")
		}
	}
	
	// Add quality indication
	if metadata.QualityScore >= 8.0 {
		desc.WriteString("，项目质量较高")
	} else if metadata.QualityScore >= 6.0 {
		desc.WriteString("，项目质量良好")
	}
	
	desc.WriteString("。")
	
	return desc.String()
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}