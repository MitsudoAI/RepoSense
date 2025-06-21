package analyzer

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// FrameworkDetector handles framework detection
type FrameworkDetector struct {
	logger             *logrus.Logger
	frameworkPatterns  map[string][]FrameworkPattern
}

// FrameworkPattern defines patterns to detect frameworks
type FrameworkPattern struct {
	Name         string
	Category     string
	Files        []string        // Required files
	Patterns     []string        // Content patterns to match
	Dependencies []string        // Dependency names to look for
	Confidence   float64         // Base confidence score
	Method       string          // Detection method description
}

// PackageJSON represents package.json structure
type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
}

// RequirementsTxt represents requirements.txt structure
type RequirementsTxt struct {
	Packages []string
}

// NewFrameworkDetector creates a new framework detector
func NewFrameworkDetector() *FrameworkDetector {
	detector := &FrameworkDetector{
		logger:            logrus.New(),
		frameworkPatterns: make(map[string][]FrameworkPattern),
	}
	
	detector.initFrameworkPatterns()
	return detector
}

// SetLogLevel sets the logging level
func (fd *FrameworkDetector) SetLogLevel(level logrus.Level) {
	fd.logger.SetLevel(level)
}

// initFrameworkPatterns initializes framework detection patterns
func (fd *FrameworkDetector) initFrameworkPatterns() {
	// JavaScript/TypeScript Frameworks
	fd.frameworkPatterns["javascript"] = []FrameworkPattern{
		{
			Name: "React", Category: "frontend",
			Dependencies: []string{"react", "react-dom"},
			Confidence: 0.9, Method: "package.json dependencies",
		},
		{
			Name: "Vue.js", Category: "frontend",
			Dependencies: []string{"vue", "@vue/core"},
			Files: []string{"*.vue"},
			Confidence: 0.9, Method: "package.json dependencies or .vue files",
		},
		{
			Name: "Angular", Category: "frontend",
			Dependencies: []string{"@angular/core", "@angular/cli"},
			Files: []string{"angular.json"},
			Confidence: 0.9, Method: "package.json dependencies or angular.json",
		},
		{
			Name: "Svelte", Category: "frontend",
			Dependencies: []string{"svelte", "@sveltejs/kit"},
			Files: []string{"*.svelte"},
			Confidence: 0.9, Method: "package.json dependencies or .svelte files",
		},
		{
			Name: "Next.js", Category: "fullstack",
			Dependencies: []string{"next"},
			Confidence: 0.9, Method: "package.json dependencies",
		},
		{
			Name: "Nuxt.js", Category: "fullstack",
			Dependencies: []string{"nuxt", "@nuxt/core"},
			Files: []string{"nuxt.config.js", "nuxt.config.ts"},
			Confidence: 0.9, Method: "package.json dependencies or nuxt config",
		},
		{
			Name: "Express.js", Category: "backend",
			Dependencies: []string{"express"},
			Confidence: 0.8, Method: "package.json dependencies",
		},
		{
			Name: "Koa.js", Category: "backend",
			Dependencies: []string{"koa"},
			Confidence: 0.8, Method: "package.json dependencies",
		},
		{
			Name: "Fastify", Category: "backend",
			Dependencies: []string{"fastify"},
			Confidence: 0.8, Method: "package.json dependencies",
		},
		{
			Name: "NestJS", Category: "backend",
			Dependencies: []string{"@nestjs/core", "@nestjs/common"},
			Confidence: 0.9, Method: "package.json dependencies",
		},
		{
			Name: "Electron", Category: "desktop",
			Dependencies: []string{"electron"},
			Confidence: 0.9, Method: "package.json dependencies",
		},
		{
			Name: "React Native", Category: "mobile",
			Dependencies: []string{"react-native"},
			Confidence: 0.9, Method: "package.json dependencies",
		},
		{
			Name: "Ionic", Category: "mobile",
			Dependencies: []string{"@ionic/angular", "@ionic/react", "@ionic/vue"},
			Confidence: 0.9, Method: "package.json dependencies",
		},
		{
			Name: "Gatsby", Category: "static-site",
			Dependencies: []string{"gatsby"},
			Confidence: 0.9, Method: "package.json dependencies",
		},
		{
			Name: "Vite", Category: "build-tool",
			Dependencies: []string{"vite"},
			Files: []string{"vite.config.js", "vite.config.ts"},
			Confidence: 0.8, Method: "package.json dependencies or vite config",
		},
		{
			Name: "Webpack", Category: "build-tool",
			Dependencies: []string{"webpack"},
			Files: []string{"webpack.config.js", "webpack.config.ts"},
			Confidence: 0.8, Method: "package.json dependencies or webpack config",
		},
		{
			Name: "Parcel", Category: "build-tool",
			Dependencies: []string{"parcel"},
			Confidence: 0.8, Method: "package.json dependencies",
		},
		{
			Name: "Rollup", Category: "build-tool",
			Dependencies: []string{"rollup"},
			Files: []string{"rollup.config.js"},
			Confidence: 0.8, Method: "package.json dependencies or rollup config",
		},
	}
	
	// Python Frameworks
	fd.frameworkPatterns["python"] = []FrameworkPattern{
		{
			Name: "Django", Category: "backend",
			Dependencies: []string{"django", "Django"},
			Files: []string{"manage.py", "settings.py"},
			Confidence: 0.9, Method: "requirements.txt or Django files",
		},
		{
			Name: "Flask", Category: "backend",
			Dependencies: []string{"flask", "Flask"},
			Confidence: 0.8, Method: "requirements.txt dependencies",
		},
		{
			Name: "FastAPI", Category: "backend",
			Dependencies: []string{"fastapi"},
			Confidence: 0.9, Method: "requirements.txt dependencies",
		},
		{
			Name: "Tornado", Category: "backend",
			Dependencies: []string{"tornado"},
			Confidence: 0.8, Method: "requirements.txt dependencies",
		},
		{
			Name: "Pyramid", Category: "backend",
			Dependencies: []string{"pyramid"},
			Confidence: 0.8, Method: "requirements.txt dependencies",
		},
		{
			Name: "Streamlit", Category: "data-science",
			Dependencies: []string{"streamlit"},
			Confidence: 0.9, Method: "requirements.txt dependencies",
		},
		{
			Name: "Dash", Category: "data-science",
			Dependencies: []string{"dash"},
			Confidence: 0.9, Method: "requirements.txt dependencies",
		},
		{
			Name: "Jupyter", Category: "data-science",
			Dependencies: []string{"jupyter", "notebook"},
			Files: []string{"*.ipynb"},
			Confidence: 0.8, Method: "requirements.txt or .ipynb files",
		},
		{
			Name: "TensorFlow", Category: "machine-learning",
			Dependencies: []string{"tensorflow", "tensorflow-gpu"},
			Confidence: 0.9, Method: "requirements.txt dependencies",
		},
		{
			Name: "PyTorch", Category: "machine-learning",
			Dependencies: []string{"torch", "pytorch"},
			Confidence: 0.9, Method: "requirements.txt dependencies",
		},
		{
			Name: "Scikit-learn", Category: "machine-learning",
			Dependencies: []string{"scikit-learn", "sklearn"},
			Confidence: 0.8, Method: "requirements.txt dependencies",
		},
		{
			Name: "Pandas", Category: "data-analysis",
			Dependencies: []string{"pandas"},
			Confidence: 0.7, Method: "requirements.txt dependencies",
		},
		{
			Name: "NumPy", Category: "scientific-computing",
			Dependencies: []string{"numpy"},
			Confidence: 0.7, Method: "requirements.txt dependencies",
		},
	}
	
	// Java Frameworks
	fd.frameworkPatterns["java"] = []FrameworkPattern{
		{
			Name: "Spring Boot", Category: "backend",
			Files: []string{"pom.xml"},
			Patterns: []string{"spring-boot-starter"},
			Confidence: 0.9, Method: "pom.xml Spring Boot starter",
		},
		{
			Name: "Spring Framework", Category: "backend",
			Files: []string{"pom.xml"},
			Patterns: []string{"spring-core", "spring-context"},
			Confidence: 0.8, Method: "pom.xml Spring dependencies",
		},
		{
			Name: "Android", Category: "mobile",
			Files: []string{"AndroidManifest.xml", "build.gradle"},
			Patterns: []string{"com.android.application"},
			Confidence: 0.9, Method: "Android manifest or build.gradle",
		},
		{
			Name: "Hibernate", Category: "orm",
			Files: []string{"pom.xml"},
			Patterns: []string{"hibernate-core"},
			Confidence: 0.8, Method: "pom.xml Hibernate dependencies",
		},
		{
			Name: "Gradle", Category: "build-tool",
			Files: []string{"build.gradle", "gradle.properties"},
			Confidence: 0.9, Method: "Gradle build files",
		},
		{
			Name: "Maven", Category: "build-tool",
			Files: []string{"pom.xml"},
			Confidence: 0.9, Method: "Maven pom.xml",
		},
	}
	
	// Go Frameworks
	fd.frameworkPatterns["go"] = []FrameworkPattern{
		{
			Name: "Gin", Category: "backend",
			Files: []string{"go.mod"},
			Patterns: []string{"gin-gonic/gin"},
			Confidence: 0.9, Method: "go.mod dependencies",
		},
		{
			Name: "Echo", Category: "backend",
			Files: []string{"go.mod"},
			Patterns: []string{"labstack/echo"},
			Confidence: 0.9, Method: "go.mod dependencies",
		},
		{
			Name: "Fiber", Category: "backend",
			Files: []string{"go.mod"},
			Patterns: []string{"gofiber/fiber"},
			Confidence: 0.9, Method: "go.mod dependencies",
		},
		{
			Name: "Beego", Category: "backend",
			Files: []string{"go.mod"},
			Patterns: []string{"beego/beego"},
			Confidence: 0.9, Method: "go.mod dependencies",
		},
		{
			Name: "Cobra", Category: "cli",
			Files: []string{"go.mod"},
			Patterns: []string{"spf13/cobra"},
			Confidence: 0.8, Method: "go.mod dependencies",
		},
	}
	
	// PHP Frameworks
	fd.frameworkPatterns["php"] = []FrameworkPattern{
		{
			Name: "Laravel", Category: "backend",
			Files: []string{"composer.json", "artisan"},
			Patterns: []string{"laravel/framework"},
			Confidence: 0.9, Method: "composer.json or artisan file",
		},
		{
			Name: "Symfony", Category: "backend",
			Files: []string{"composer.json"},
			Patterns: []string{"symfony/symfony"},
			Confidence: 0.9, Method: "composer.json dependencies",
		},
		{
			Name: "CodeIgniter", Category: "backend",
			Files: []string{"system/CodeIgniter.php"},
			Confidence: 0.9, Method: "CodeIgniter system files",
		},
		{
			Name: "Zend Framework", Category: "backend",
			Files: []string{"composer.json"},
			Patterns: []string{"zendframework/"},
			Confidence: 0.8, Method: "composer.json dependencies",
		},
	}
	
	// Ruby Frameworks
	fd.frameworkPatterns["ruby"] = []FrameworkPattern{
		{
			Name: "Ruby on Rails", Category: "backend",
			Files: []string{"Gemfile", "config/application.rb"},
			Patterns: []string{"rails"},
			Confidence: 0.9, Method: "Gemfile or Rails config",
		},
		{
			Name: "Sinatra", Category: "backend",
			Files: []string{"Gemfile"},
			Patterns: []string{"sinatra"},
			Confidence: 0.8, Method: "Gemfile dependencies",
		},
		{
			Name: "Jekyll", Category: "static-site",
			Files: []string{"Gemfile", "_config.yml"},
			Patterns: []string{"jekyll"},
			Confidence: 0.9, Method: "Gemfile or Jekyll config",
		},
	}
	
	// Rust Frameworks
	fd.frameworkPatterns["rust"] = []FrameworkPattern{
		{
			Name: "Rocket", Category: "backend",
			Files: []string{"Cargo.toml"},
			Patterns: []string{"rocket ="},
			Confidence: 0.9, Method: "Cargo.toml dependencies",
		},
		{
			Name: "Actix Web", Category: "backend",
			Files: []string{"Cargo.toml"},
			Patterns: []string{"actix-web ="},
			Confidence: 0.9, Method: "Cargo.toml dependencies",
		},
		{
			Name: "Warp", Category: "backend",
			Files: []string{"Cargo.toml"},
			Patterns: []string{"warp ="},
			Confidence: 0.9, Method: "Cargo.toml dependencies",
		},
		{
			Name: "Tauri", Category: "desktop",
			Files: []string{"Cargo.toml"},
			Patterns: []string{"tauri ="},
			Confidence: 0.9, Method: "Cargo.toml dependencies",
		},
	}
	
	// C# Frameworks
	fd.frameworkPatterns["csharp"] = []FrameworkPattern{
		{
			Name: "ASP.NET Core", Category: "backend",
			Files: []string{"*.csproj"},
			Patterns: []string{"Microsoft.AspNetCore"},
			Confidence: 0.9, Method: ".csproj file dependencies",
		},
		{
			Name: "Entity Framework", Category: "orm",
			Files: []string{"*.csproj"},
			Patterns: []string{"Microsoft.EntityFrameworkCore"},
			Confidence: 0.8, Method: ".csproj file dependencies",
		},
		{
			Name: "Xamarin", Category: "mobile",
			Files: []string{"*.csproj"},
			Patterns: []string{"Xamarin."},
			Confidence: 0.9, Method: ".csproj file dependencies",
		},
	}
	
	// Testing Frameworks
	fd.frameworkPatterns["testing"] = []FrameworkPattern{
		{
			Name: "Jest", Category: "testing",
			Dependencies: []string{"jest"},
			Confidence: 0.8, Method: "package.json dependencies",
		},
		{
			Name: "Mocha", Category: "testing",
			Dependencies: []string{"mocha"},
			Confidence: 0.8, Method: "package.json dependencies",
		},
		{
			Name: "Cypress", Category: "testing",
			Dependencies: []string{"cypress"},
			Confidence: 0.9, Method: "package.json dependencies",
		},
		{
			Name: "Playwright", Category: "testing",
			Dependencies: []string{"playwright", "@playwright/test"},
			Confidence: 0.9, Method: "package.json dependencies",
		},
		{
			Name: "PyTest", Category: "testing",
			Dependencies: []string{"pytest"},
			Confidence: 0.8, Method: "requirements.txt dependencies",
		},
		{
			Name: "JUnit", Category: "testing",
			Files: []string{"pom.xml"},
			Patterns: []string{"junit"},
			Confidence: 0.8, Method: "pom.xml dependencies",
		},
	}
}

// DetectFrameworks analyzes a repository and detects frameworks
func (fd *FrameworkDetector) DetectFrameworks(repoPath string, config *AnalysisConfig) ([]FrameworkInfo, error) {
	fd.logger.Debugf("开始检测框架: %s", repoPath)
	
	var frameworks []FrameworkInfo
	
	// Find configuration files
	configFiles, err := FindConfigFiles(repoPath)
	if err != nil {
		return nil, err
	}
	
	// Detect JavaScript/TypeScript frameworks
	if packageJSON, exists := configFiles["package.json"]; exists {
		jsFrameworks := fd.detectJavaScriptFrameworks(packageJSON)
		frameworks = append(frameworks, jsFrameworks...)
	}
	
	// Detect Python frameworks
	if requirementsTxt, exists := configFiles["requirements.txt"]; exists {
		pyFrameworks := fd.detectPythonFrameworks(requirementsTxt)
		frameworks = append(frameworks, pyFrameworks...)
	}
	
	// Detect Java frameworks
	if pomXML, exists := configFiles["pom.xml"]; exists {
		javaFrameworks := fd.detectJavaFrameworks(pomXML)
		frameworks = append(frameworks, javaFrameworks...)
	}
	
	// Detect Go frameworks
	if goMod, exists := configFiles["go.mod"]; exists {
		goFrameworks := fd.detectGoFrameworks(goMod)
		frameworks = append(frameworks, goFrameworks...)
	}
	
	// Detect other frameworks by file presence
	fileBasedFrameworks := fd.detectFrameworksByFiles(repoPath, configFiles)
	frameworks = append(frameworks, fileBasedFrameworks...)
	
	// Remove duplicates and sort by confidence
	frameworks = fd.deduplicateFrameworks(frameworks)
	
	fd.logger.Debugf("检测到 %d 个框架", len(frameworks))
	return frameworks, nil
}

// detectJavaScriptFrameworks detects JavaScript/TypeScript frameworks from package.json
func (fd *FrameworkDetector) detectJavaScriptFrameworks(packageJSONPath string) []FrameworkInfo {
	var frameworks []FrameworkInfo
	
	content, err := ReadFileContent(packageJSONPath, 1024*1024) // 1MB limit
	if err != nil {
		return frameworks
	}
	
	var packageJSON PackageJSON
	if err := json.Unmarshal([]byte(content), &packageJSON); err != nil {
		return frameworks
	}
	
	// Combine dependencies
	allDeps := make(map[string]string)
	for k, v := range packageJSON.Dependencies {
		allDeps[k] = v
	}
	for k, v := range packageJSON.DevDependencies {
		allDeps[k] = v
	}
	
	// Check against JavaScript patterns
	for _, pattern := range fd.frameworkPatterns["javascript"] {
		for _, dep := range pattern.Dependencies {
			if version, exists := allDeps[dep]; exists {
				frameworks = append(frameworks, FrameworkInfo{
					Name:            pattern.Name,
					Version:         fd.cleanVersion(version),
					Category:        pattern.Category,
					Confidence:      pattern.Confidence,
					DetectionMethod: pattern.Method,
				})
				break // Only add once per framework
			}
		}
	}
	
	// Check testing frameworks
	for _, pattern := range fd.frameworkPatterns["testing"] {
		for _, dep := range pattern.Dependencies {
			if version, exists := allDeps[dep]; exists {
				frameworks = append(frameworks, FrameworkInfo{
					Name:            pattern.Name,
					Version:         fd.cleanVersion(version),
					Category:        pattern.Category,
					Confidence:      pattern.Confidence,
					DetectionMethod: pattern.Method,
				})
				break
			}
		}
	}
	
	return frameworks
}

// detectPythonFrameworks detects Python frameworks from requirements.txt
func (fd *FrameworkDetector) detectPythonFrameworks(requirementsPath string) []FrameworkInfo {
	var frameworks []FrameworkInfo
	
	content, err := ReadFileContent(requirementsPath, 1024*1024)
	if err != nil {
		return frameworks
	}
	
	lines := strings.Split(content, "\n")
	dependencies := make(map[string]string)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse dependency name and version
		parts := regexp.MustCompile(`[>=<~!]`).Split(line, 2)
		if len(parts) > 0 {
			depName := strings.ToLower(strings.TrimSpace(parts[0]))
			version := ""
			if len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
			}
			dependencies[depName] = version
		}
	}
	
	// Check against Python patterns
	for _, pattern := range fd.frameworkPatterns["python"] {
		for _, dep := range pattern.Dependencies {
			if version, exists := dependencies[strings.ToLower(dep)]; exists {
				frameworks = append(frameworks, FrameworkInfo{
					Name:            pattern.Name,
					Version:         version,
					Category:        pattern.Category,
					Confidence:      pattern.Confidence,
					DetectionMethod: pattern.Method,
				})
				break
			}
		}
	}
	
	return frameworks
}

// detectJavaFrameworks detects Java frameworks from pom.xml
func (fd *FrameworkDetector) detectJavaFrameworks(pomXMLPath string) []FrameworkInfo {
	var frameworks []FrameworkInfo
	
	content, err := ReadFileContent(pomXMLPath, 1024*1024)
	if err != nil {
		return frameworks
	}
	
	// Check against Java patterns
	for _, pattern := range fd.frameworkPatterns["java"] {
		for _, patternStr := range pattern.Patterns {
			if strings.Contains(content, patternStr) {
				frameworks = append(frameworks, FrameworkInfo{
					Name:            pattern.Name,
					Category:        pattern.Category,
					Confidence:      pattern.Confidence,
					DetectionMethod: pattern.Method,
				})
				break
			}
		}
	}
	
	return frameworks
}

// detectGoFrameworks detects Go frameworks from go.mod
func (fd *FrameworkDetector) detectGoFrameworks(goModPath string) []FrameworkInfo {
	var frameworks []FrameworkInfo
	
	content, err := ReadFileContent(goModPath, 1024*1024)
	if err != nil {
		return frameworks
	}
	
	// Check against Go patterns
	for _, pattern := range fd.frameworkPatterns["go"] {
		for _, patternStr := range pattern.Patterns {
			if strings.Contains(content, patternStr) {
				frameworks = append(frameworks, FrameworkInfo{
					Name:            pattern.Name,
					Category:        pattern.Category,
					Confidence:      pattern.Confidence,
					DetectionMethod: pattern.Method,
				})
				break
			}
		}
	}
	
	return frameworks
}

// detectFrameworksByFiles detects frameworks based on file presence
func (fd *FrameworkDetector) detectFrameworksByFiles(repoPath string, configFiles map[string]string) []FrameworkInfo {
	var frameworks []FrameworkInfo
	
	// Check for Vue.js files
	vueFiles, _ := filepath.Glob(filepath.Join(repoPath, "**/*.vue"))
	if len(vueFiles) > 0 {
		frameworks = append(frameworks, FrameworkInfo{
			Name:            "Vue.js",
			Category:        "frontend",
			Confidence:      0.9,
			DetectionMethod: ".vue files found",
		})
	}
	
	// Check for Svelte files
	svelteFiles, _ := filepath.Glob(filepath.Join(repoPath, "**/*.svelte"))
	if len(svelteFiles) > 0 {
		frameworks = append(frameworks, FrameworkInfo{
			Name:            "Svelte",
			Category:        "frontend",
			Confidence:      0.9,
			DetectionMethod: ".svelte files found",
		})
	}
	
	// Check for Django
	if _, exists := configFiles["manage.py"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:            "Django",
			Category:        "backend",
			Confidence:      0.9,
			DetectionMethod: "manage.py file found",
		})
	}
	
	// Check for Jekyll
	if _, exists := configFiles["_config.yml"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:            "Jekyll",
			Category:        "static-site",
			Confidence:      0.8,
			DetectionMethod: "_config.yml file found",
		})
	}
	
	return frameworks
}

// deduplicateFrameworks removes duplicate frameworks and sorts by confidence
func (fd *FrameworkDetector) deduplicateFrameworks(frameworks []FrameworkInfo) []FrameworkInfo {
	seen := make(map[string]bool)
	var unique []FrameworkInfo
	
	for _, framework := range frameworks {
		key := framework.Name + "|" + framework.Category
		if !seen[key] {
			seen[key] = true
			unique = append(unique, framework)
		}
	}
	
	// Sort by confidence (descending)
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Confidence > unique[j].Confidence
	})
	
	return unique
}

// cleanVersion removes version prefixes and suffixes
func (fd *FrameworkDetector) cleanVersion(version string) string {
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

// GetFrameworksByCategory returns frameworks grouped by category
func (fd *FrameworkDetector) GetFrameworksByCategory(frameworks []FrameworkInfo) map[string][]FrameworkInfo {
	categories := make(map[string][]FrameworkInfo)
	
	for _, framework := range frameworks {
		categories[framework.Category] = append(categories[framework.Category], framework)
	}
	
	return categories
}

// GetSupportedFrameworks returns a list of all supported frameworks
func (fd *FrameworkDetector) GetSupportedFrameworks() []string {
	var frameworks []string
	
	for _, patterns := range fd.frameworkPatterns {
		for _, pattern := range patterns {
			frameworks = append(frameworks, pattern.Name)
		}
	}
	
	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, framework := range frameworks {
		if !seen[framework] {
			seen[framework] = true
			unique = append(unique, framework)
		}
	}
	
	sort.Strings(unique)
	return unique
}