package analyzer

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// LanguageDetector handles programming language detection
type LanguageDetector struct {
	logger           *logrus.Logger
	languageMap      map[string]LanguageDefinition
	extensionMap     map[string]string
}

// LanguageDefinition defines a programming language
type LanguageDefinition struct {
	Name        string
	Extensions  []string
	Category    string // programming, markup, data, config等
	Color       string // 用于可视化的颜色
}

// NewLanguageDetector creates a new language detector
func NewLanguageDetector() *LanguageDetector {
	detector := &LanguageDetector{
		logger:       logrus.New(),
		languageMap:  make(map[string]LanguageDefinition),
		extensionMap: make(map[string]string),
	}
	
	detector.initLanguageDefinitions()
	return detector
}

// SetLogLevel sets the logging level
func (ld *LanguageDetector) SetLogLevel(level logrus.Level) {
	ld.logger.SetLevel(level)
}

// initLanguageDefinitions initializes the language definitions
func (ld *LanguageDetector) initLanguageDefinitions() {
	languages := []LanguageDefinition{
		// Programming Languages
		{Name: "JavaScript", Extensions: []string{"js", "mjs", "cjs"}, Category: "programming", Color: "#f1e05a"},
		{Name: "TypeScript", Extensions: []string{"ts", "tsx"}, Category: "programming", Color: "#2b7489"},
		{Name: "Python", Extensions: []string{"py", "pyx", "pyw", "pyi"}, Category: "programming", Color: "#3572A5"},
		{Name: "Java", Extensions: []string{"java"}, Category: "programming", Color: "#b07219"},
		{Name: "Go", Extensions: []string{"go"}, Category: "programming", Color: "#00ADD8"},
		{Name: "Rust", Extensions: []string{"rs"}, Category: "programming", Color: "#dea584"},
		{Name: "C", Extensions: []string{"c", "h"}, Category: "programming", Color: "#555555"},
		{Name: "C++", Extensions: []string{"cpp", "cc", "cxx", "hpp", "hxx", "h++"}, Category: "programming", Color: "#f34b7d"},
		{Name: "C#", Extensions: []string{"cs", "csx"}, Category: "programming", Color: "#239120"},
		{Name: "PHP", Extensions: []string{"php", "phtml", "php3", "php4", "php5", "phps"}, Category: "programming", Color: "#4F5D95"},
		{Name: "Ruby", Extensions: []string{"rb", "rbw"}, Category: "programming", Color: "#701516"},
		{Name: "Swift", Extensions: []string{"swift"}, Category: "programming", Color: "#ffac45"},
		{Name: "Kotlin", Extensions: []string{"kt", "kts"}, Category: "programming", Color: "#F18E33"},
		{Name: "Scala", Extensions: []string{"scala", "sc"}, Category: "programming", Color: "#c22d40"},
		{Name: "R", Extensions: []string{"r", "R"}, Category: "programming", Color: "#198CE7"},
		{Name: "MATLAB", Extensions: []string{"m"}, Category: "programming", Color: "#e16737"},
		{Name: "Dart", Extensions: []string{"dart"}, Category: "programming", Color: "#00B4AB"},
		{Name: "Elixir", Extensions: []string{"ex", "exs"}, Category: "programming", Color: "#6e4a7e"},
		{Name: "Erlang", Extensions: []string{"erl", "hrl"}, Category: "programming", Color: "#B83998"},
		{Name: "Haskell", Extensions: []string{"hs", "lhs"}, Category: "programming", Color: "#5e5086"},
		{Name: "Clojure", Extensions: []string{"clj", "cljs", "cljc"}, Category: "programming", Color: "#db5855"},
		{Name: "F#", Extensions: []string{"fs", "fsi", "fsx"}, Category: "programming", Color: "#b845fc"},
		{Name: "OCaml", Extensions: []string{"ml", "mli"}, Category: "programming", Color: "#3be133"},
		{Name: "Lua", Extensions: []string{"lua"}, Category: "programming", Color: "#000080"},
		{Name: "Perl", Extensions: []string{"pl", "pm"}, Category: "programming", Color: "#0298c3"},
		
		// Web Technologies
		{Name: "HTML", Extensions: []string{"html", "htm", "xhtml"}, Category: "markup", Color: "#e34c26"},
		{Name: "CSS", Extensions: []string{"css"}, Category: "programming", Color: "#563d7c"},
		{Name: "SCSS", Extensions: []string{"scss"}, Category: "programming", Color: "#c6538c"},
		{Name: "Sass", Extensions: []string{"sass"}, Category: "programming", Color: "#a53b70"},
		{Name: "Less", Extensions: []string{"less"}, Category: "programming", Color: "#1d365d"},
		{Name: "Vue", Extensions: []string{"vue"}, Category: "programming", Color: "#2c3e50"},
		{Name: "Svelte", Extensions: []string{"svelte"}, Category: "programming", Color: "#ff3e00"},
		{Name: "JSX", Extensions: []string{"jsx"}, Category: "programming", Color: "#f1e05a"},
		
		// Shell Scripts
		{Name: "Shell", Extensions: []string{"sh", "bash", "zsh", "fish"}, Category: "programming", Color: "#89e051"},
		{Name: "PowerShell", Extensions: []string{"ps1", "psm1", "psd1"}, Category: "programming", Color: "#012456"},
		{Name: "Batch", Extensions: []string{"bat", "cmd"}, Category: "programming", Color: "#C1F12E"},
		
		// Data & Config
		{Name: "JSON", Extensions: []string{"json"}, Category: "data", Color: "#292929"},
		{Name: "XML", Extensions: []string{"xml", "xsd", "xsl"}, Category: "data", Color: "#0060ac"},
		{Name: "YAML", Extensions: []string{"yml", "yaml"}, Category: "data", Color: "#cb171e"},
		{Name: "TOML", Extensions: []string{"toml"}, Category: "data", Color: "#9c4221"},
		{Name: "INI", Extensions: []string{"ini", "cfg", "conf"}, Category: "data", Color: "#d1dbe0"},
		{Name: "CSV", Extensions: []string{"csv"}, Category: "data", Color: "#239120"},
		
		// Documentation
		{Name: "Markdown", Extensions: []string{"md", "markdown", "mdown", "mkd"}, Category: "markup", Color: "#083fa1"},
		{Name: "reStructuredText", Extensions: []string{"rst"}, Category: "markup", Color: "#141414"},
		{Name: "AsciiDoc", Extensions: []string{"adoc", "asciidoc"}, Category: "markup", Color: "#73a0c5"},
		{Name: "LaTeX", Extensions: []string{"tex", "latex"}, Category: "markup", Color: "#3D6117"},
		
		// Database
		{Name: "SQL", Extensions: []string{"sql"}, Category: "programming", Color: "#e38c00"},
		{Name: "PLpgSQL", Extensions: []string{"pgsql"}, Category: "programming", Color: "#336791"},
		{Name: "PLSQL", Extensions: []string{"pls", "plsql"}, Category: "programming", Color: "#dad8d8"},
		
		// Other
		{Name: "Dockerfile", Extensions: []string{"dockerfile"}, Category: "programming", Color: "#384d54"},
		{Name: "Makefile", Extensions: []string{"makefile", "mk"}, Category: "programming", Color: "#427819"},
		{Name: "CMake", Extensions: []string{"cmake"}, Category: "programming", Color: "#DA3434"},
		{Name: "Protocol Buffer", Extensions: []string{"proto"}, Category: "data", Color: "#4285f4"},
		{Name: "GraphQL", Extensions: []string{"graphql", "gql"}, Category: "data", Color: "#e10098"},
		{Name: "Assembly", Extensions: []string{"asm", "s"}, Category: "programming", Color: "#6E4C13"},
		{Name: "Vim Script", Extensions: []string{"vim"}, Category: "programming", Color: "#199f4b"},
		
		// Mobile Development
		{Name: "Objective-C", Extensions: []string{"m", "mm"}, Category: "programming", Color: "#438eff"},
		{Name: "Objective-C++", Extensions: []string{"mm"}, Category: "programming", Color: "#6866fb"},
		
		// Game Development
		{Name: "HLSL", Extensions: []string{"hlsl"}, Category: "programming", Color: "#aace60"},
		{Name: "GLSL", Extensions: []string{"glsl", "vert", "frag"}, Category: "programming", Color: "#5686a5"},
		
		// Functional Languages
		{Name: "Elm", Extensions: []string{"elm"}, Category: "programming", Color: "#60B5CC"},
		{Name: "PureScript", Extensions: []string{"purs"}, Category: "programming", Color: "#1D222D"},
		
		// Scientific Computing
		{Name: "Julia", Extensions: []string{"jl"}, Category: "programming", Color: "#a270ba"},
		{Name: "Fortran", Extensions: []string{"f", "for", "f77", "f90", "f95", "f03", "f08"}, Category: "programming", Color: "#4d41b1"},
		
		// Other Popular Languages
		{Name: "Zig", Extensions: []string{"zig"}, Category: "programming", Color: "#ec915c"},
		{Name: "Crystal", Extensions: []string{"cr"}, Category: "programming", Color: "#000100"},
		{Name: "Nim", Extensions: []string{"nim"}, Category: "programming", Color: "#ffc200"},
		{Name: "V", Extensions: []string{"v"}, Category: "programming", Color: "#4f87c4"},
	}
	
	// Build maps for fast lookup
	for _, lang := range languages {
		ld.languageMap[strings.ToLower(lang.Name)] = lang
		
		for _, ext := range lang.Extensions {
			ld.extensionMap[strings.ToLower(ext)] = strings.ToLower(lang.Name)
		}
	}
	
	// Handle special cases
	ld.handleSpecialCases()
}

// handleSpecialCases handles special file detection cases
func (ld *LanguageDetector) handleSpecialCases() {
	// Special filenames that don't have extensions
	specialFiles := map[string]string{
		"dockerfile":    "dockerfile",
		"makefile":      "makefile",
		"rakefile":      "ruby",
		"gemfile":       "ruby",
		"guardfile":     "ruby",
		"podfile":       "ruby",
		"vagrantfile":   "ruby",
		"cmakelist.txt": "cmake",
		"cmakelists.txt": "cmake",
	}
	
	for filename, language := range specialFiles {
		ld.extensionMap[filename] = language
	}
}

// DetectLanguages analyzes a repository and detects programming languages
func (ld *LanguageDetector) DetectLanguages(repoPath string, config *AnalysisConfig) ([]LanguageInfo, error) {
	ld.logger.Debugf("开始检测语言: %s", repoPath)
	
	// Find all relevant files
	files, err := FindFiles(repoPath, []string{}, config.IgnorePatterns)
	if err != nil {
		return nil, err
	}
	
	// Count languages
	languageStats := make(map[string]*LanguageInfo)
	totalBytes := int64(0)
	totalLines := 0
	totalFiles := 0
	
	for _, file := range files {
		// Skip files that are too large
		if config.MaxFileSize > 0 && file.Size > config.MaxFileSize {
			continue
		}
		
		// Skip if we've reached max files
		if config.MaxFiles > 0 && totalFiles >= config.MaxFiles {
			break
		}
		
		language := ld.detectFileLanguage(file.Path)
		if language == "" {
			continue
		}
		
		if _, exists := languageStats[language]; !exists {
			languageStats[language] = &LanguageInfo{
				Name:        ld.getLanguageDisplayName(language),
				LinesOfCode: 0,
				FileCount:   0,
				BytesCount:  0,
			}
		}
		
		languageStats[language].LinesOfCode += file.Lines
		languageStats[language].FileCount++
		languageStats[language].BytesCount += int(file.Size)
		
		totalBytes += file.Size
		totalLines += file.Lines
		totalFiles++
	}
	
	// Calculate percentages and convert to slice
	var languages []LanguageInfo
	for _, stats := range languageStats {
		if totalLines > 0 {
			stats.Percentage = float64(stats.LinesOfCode) / float64(totalLines) * 100
		}
		languages = append(languages, *stats)
	}
	
	// Sort by lines of code (descending)
	sort.Slice(languages, func(i, j int) bool {
		return languages[i].LinesOfCode > languages[j].LinesOfCode
	})
	
	ld.logger.Debugf("检测到 %d 种语言，共 %d 个文件", len(languages), totalFiles)
	return languages, nil
}

// detectFileLanguage detects the language of a single file
func (ld *LanguageDetector) detectFileLanguage(filePath string) string {
	filename := strings.ToLower(filepath.Base(filePath))
	ext := GetFileExtension(filePath)
	
	// Check special filenames first
	if language, exists := ld.extensionMap[filename]; exists {
		return language
	}
	
	// Check extension
	if language, exists := ld.extensionMap[ext]; exists {
		return language
	}
	
	// Handle ambiguous extensions
	return ld.handleAmbiguousExtensions(filePath, ext)
}

// handleAmbiguousExtensions handles cases where extensions could mean multiple languages
func (ld *LanguageDetector) handleAmbiguousExtensions(filePath, ext string) string {
	filename := strings.ToLower(filepath.Base(filePath))
	
	switch ext {
	case "m":
		// Could be MATLAB or Objective-C
		if strings.Contains(filename, "objc") || strings.Contains(filePath, "ios") || 
		   strings.Contains(filePath, "macos") || strings.Contains(filePath, "cocoa") {
			return "objective-c"
		}
		return "matlab"
		
	case "h":
		// Could be C or C++
		if ld.hasNearbyFile(filePath, []string{"cpp", "cc", "cxx"}) {
			return "c++"
		}
		return "c"
		
	case "r":
		// Usually R, but check context
		return "r"
		
	case "pl":
		// Usually Perl
		return "perl"
		
	case "v":
		// Usually V language, but could be Verilog
		if strings.Contains(filePath, "verilog") || strings.Contains(filePath, "hdl") {
			return "verilog"
		}
		return "v"
	}
	
	return ""
}

// hasNearbyFile checks if there are files with given extensions in the same directory
func (ld *LanguageDetector) hasNearbyFile(filePath string, extensions []string) bool {
	dir := filepath.Dir(filePath)
	
	for _, ext := range extensions {
		pattern := filepath.Join(dir, "*."+ext)
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			return true
		}
	}
	
	return false
}

// getLanguageDisplayName returns the proper display name for a language
func (ld *LanguageDetector) getLanguageDisplayName(languageKey string) string {
	if lang, exists := ld.languageMap[strings.ToLower(languageKey)]; exists {
		return lang.Name
	}
	
	// Capitalize first letter if not found
	if len(languageKey) > 0 {
		return strings.ToUpper(languageKey[:1]) + languageKey[1:]
	}
	
	return languageKey
}

// GetLanguageColor returns the color associated with a language
func (ld *LanguageDetector) GetLanguageColor(languageName string) string {
	if lang, exists := ld.languageMap[strings.ToLower(languageName)]; exists {
		return lang.Color
	}
	return "#cccccc" // Default gray
}

// GetMainLanguage returns the primary language based on lines of code
func (ld *LanguageDetector) GetMainLanguage(languages []LanguageInfo) string {
	if len(languages) == 0 {
		return ""
	}
	
	// Filter out non-programming languages for main language detection
	programmingLanguages := []LanguageInfo{}
	for _, lang := range languages {
		if langDef, exists := ld.languageMap[strings.ToLower(lang.Name)]; exists {
			if langDef.Category == "programming" {
				programmingLanguages = append(programmingLanguages, lang)
			}
		}
	}
	
	if len(programmingLanguages) > 0 {
		return programmingLanguages[0].Name
	}
	
	// Fallback to first language
	return languages[0].Name
}

// GetSupportedLanguages returns a list of all supported languages
func (ld *LanguageDetector) GetSupportedLanguages() []string {
	var languages []string
	for _, lang := range ld.languageMap {
		languages = append(languages, lang.Name)
	}
	
	sort.Strings(languages)
	return languages
}