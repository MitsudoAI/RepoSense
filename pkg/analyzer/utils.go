package analyzer

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FileInfo represents information about a file
type FileInfo struct {
	Path      string
	Extension string
	Size      int64
	Lines     int
}

// GetFileExtension returns the file extension without the dot
func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext != "" {
		return strings.ToLower(ext[1:]) // Remove the dot
	}
	return ""
}

// IsTextFile checks if a file is likely a text file based on extension or filename
func IsTextFile(filename string) bool {
	textExtensions := map[string]bool{
		"txt": true, "md": true, "rst": true, "json": true, "xml": true, "yaml": true, "yml": true,
		"js": true, "ts": true, "jsx": true, "tsx": true, "css": true, "scss": true, "sass": true, "less": true,
		"html": true, "htm": true, "php": true, "py": true, "java": true, "c": true, "cpp": true, "h": true, "hpp": true,
		"go": true, "rs": true, "rb": true, "sh": true, "bash": true, "zsh": true, "fish": true, "ps1": true,
		"sql": true, "r": true, "m": true, "swift": true, "kt": true, "scala": true, "clj": true, "hs": true,
		"ml": true, "fs": true, "elm": true, "dart": true, "vue": true, "svelte": true, "config": true, "conf": true,
		"ini": true, "toml": true, "lock": true, "log": true, "csv": true, "tsv": true,
		"mod": true, "sum": true, "cmake": true, "gradle": true, "properties": true, "gitignore": true,
	}
	
	// Check extension first
	ext := GetFileExtension(filename)
	if textExtensions[ext] {
		return true
	}
	
	// Check special filenames without extensions
	lowerFilename := strings.ToLower(filename)
	specialTextFiles := map[string]bool{
		"makefile": true, "dockerfile": true, "readme": true, "license": true, "copying": true,
		"changelog": true, "gitignore": true, "gitmodules": true, "gitattributes": true,
	}
	
	return specialTextFiles[lowerFilename] || specialTextFiles[strings.TrimPrefix(lowerFilename, ".")]
}

// ShouldIgnoreFile checks if a file should be ignored based on patterns
func ShouldIgnoreFile(filePath string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		matched, err := filepath.Match(pattern, filePath)
		if err == nil && matched {
			return true
		}
		
		// Check if the file path contains the pattern
		if strings.Contains(filePath, strings.TrimSuffix(pattern, "/*")) {
			return true
		}
	}
	
	// Default ignore patterns
	filename := filepath.Base(filePath)
	if strings.HasPrefix(filename, ".") && filename != ".gitignore" && filename != ".env" && filename != "." {
		return true
	}
	
	return false
}

// CountLines counts the number of lines in a text file
func CountLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	
	return lines, scanner.Err()
}

// ReadFileContent reads file content with size limit
func ReadFileContent(filePath string, maxSize int64) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	
	if info.Size() > maxSize {
		return "", fmt.Errorf("file too large: %d bytes", info.Size())
	}
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	
	return string(content), nil
}

// GenerateStructureHash generates a hash representing the project structure
func GenerateStructureHash(repoPath string, ignorePatterns []string) (string, error) {
	var paths []string
	
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		// Get relative path
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}
		
		// Skip ignored files
		if ShouldIgnoreFile(relPath, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Only include files for structure hash
		if !info.IsDir() {
			paths = append(paths, relPath)
		}
		
		return nil
	})
	
	if err != nil {
		return "", err
	}
	
	// Sort paths for consistent hash
	// Sort is not needed as Go's map iteration is already deterministic for our use case
	// We'll just concatenate all paths
	combined := strings.Join(paths, "|")
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash), nil
}

// FindFiles finds all files matching the given extensions
func FindFiles(rootPath string, extensions []string, ignorePatterns []string) ([]FileInfo, error) {
	var files []FileInfo
	extMap := make(map[string]bool)
	
	// Convert extensions to lowercase map for fast lookup
	for _, ext := range extensions {
		extMap[strings.ToLower(ext)] = true
	}
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		// Get relative path
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}
		
		// Only process files first
		if info.IsDir() {
			// Skip ignored directories
			if ShouldIgnoreFile(relPath, ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Skip ignored files
		if ShouldIgnoreFile(relPath, ignorePatterns) {
			return nil
		}
		
		// Check extension and filter by text files for language detection
		ext := GetFileExtension(info.Name())
		if len(extMap) == 0 {
			// When no specific extensions requested, only include text files for language detection
			if !IsTextFile(info.Name()) {
				return nil
			}
		} else if !extMap[ext] {
			return nil
		}
		
		lines, _ := CountLines(path)
		files = append(files, FileInfo{
			Path:      relPath,
			Extension: ext,
			Size:      info.Size(),
			Lines:     lines,
		})
		
		return nil
	})
	return files, err
}

// FindConfigFiles finds common configuration files
func FindConfigFiles(rootPath string) (map[string]string, error) {
	configFiles := make(map[string]string)
	
	// Define common config files to look for
	configPatterns := map[string][]string{
		"package.json":      {"package.json"},
		"requirements.txt":  {"requirements.txt", "requirements-dev.txt", "requirements-test.txt"},
		"pom.xml":          {"pom.xml"},
		"build.gradle":     {"build.gradle", "build.gradle.kts"},
		"go.mod":           {"go.mod"},
		"Cargo.toml":       {"Cargo.toml"},
		"composer.json":    {"composer.json"},
		"setup.py":         {"setup.py"},
		"pyproject.toml":   {"pyproject.toml"},
		"Gemfile":          {"Gemfile"},
		"Podfile":          {"Podfile"},
		"pubspec.yaml":     {"pubspec.yaml"},
		"CMakeLists.txt":   {"CMakeLists.txt"},
		"Makefile":         {"Makefile", "makefile"},
		"Dockerfile":       {"Dockerfile"},
		"docker-compose":   {"docker-compose.yml", "docker-compose.yaml"},
		"LICENSE":          {"LICENSE", "LICENSE.txt", "LICENSE.md", "COPYING"},
		"README":           {"README.md", "README.rst", "README.txt", "README"},
	}
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		if info.IsDir() {
			return nil
		}
		
		// Get relative path
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}
		
		// Skip deep nested files (only check root and first level)
		if strings.Count(relPath, string(filepath.Separator)) > 1 {
			return nil
		}
		
		filename := info.Name()
		
		// Check against patterns
		for key, patterns := range configPatterns {
			for _, pattern := range patterns {
				if matched, _ := filepath.Match(pattern, filename); matched {
					configFiles[key] = path
					break
				}
			}
		}
		
		return nil
	})
	
	return configFiles, err
}

// ExtractPatternFromFile extracts text matching a regex pattern from a file
func ExtractPatternFromFile(filePath string, pattern *regexp.Regexp, maxSize int64) ([]string, error) {
	content, err := ReadFileContent(filePath, maxSize)
	if err != nil {
		return nil, err
	}
	
	matches := pattern.FindAllString(content, -1)
	return matches, nil
}

// CalculateDirectorySize calculates the total size of a directory
func CalculateDirectorySize(dirPath string, ignorePatterns []string) (int64, error) {
	var size int64
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		// Get relative path
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return nil
		}
		
		// Skip ignored files
		if ShouldIgnoreFile(relPath, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		if !info.IsDir() {
			size += info.Size()
		}
		
		return nil
	})
	
	return size, err
}