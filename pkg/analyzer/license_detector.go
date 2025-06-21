package analyzer

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// LicenseDetector handles license detection
type LicenseDetector struct {
	logger          *logrus.Logger
	licensePatterns map[string]LicensePattern
	licenseTexts    map[string][]string
}

// LicensePattern defines patterns to detect licenses
type LicensePattern struct {
	Name       string
	Key        string   // SPDX identifier
	Type       string   // permissive, copyleft, proprietary, etc.
	Keywords   []string // Key phrases to look for
	Patterns   []*regexp.Regexp
	Confidence float64
}

// NewLicenseDetector creates a new license detector
func NewLicenseDetector() *LicenseDetector {
	detector := &LicenseDetector{
		logger:          logrus.New(),
		licensePatterns: make(map[string]LicensePattern),
		licenseTexts:    make(map[string][]string),
	}
	
	detector.initLicensePatterns()
	return detector
}

// SetLogLevel sets the logging level
func (ld *LicenseDetector) SetLogLevel(level logrus.Level) {
	ld.logger.SetLevel(level)
}

// initLicensePatterns initializes license detection patterns
func (ld *LicenseDetector) initLicensePatterns() {
	licenses := map[string]LicensePattern{
		"MIT": {
			Name: "MIT License",
			Key:  "MIT",
			Type: "permissive",
			Keywords: []string{
				"MIT License",
				"Permission is hereby granted, free of charge",
				"THE SOFTWARE IS PROVIDED \"AS IS\", WITHOUT WARRANTY",
			},
			Confidence: 0.95,
		},
		"Apache-2.0": {
			Name: "Apache License 2.0",
			Key:  "Apache-2.0",
			Type: "permissive",
			Keywords: []string{
				"Apache License",
				"Version 2.0",
				"Licensed under the Apache License, Version 2.0",
				"http://www.apache.org/licenses/LICENSE-2.0",
			},
			Confidence: 0.95,
		},
		"GPL-3.0": {
			Name: "GNU General Public License v3.0",
			Key:  "GPL-3.0",
			Type: "copyleft",
			Keywords: []string{
				"GNU GENERAL PUBLIC LICENSE",
				"Version 3",
				"This program is free software: you can redistribute it",
				"GNU General Public License as published by the Free Software Foundation",
			},
			Confidence: 0.95,
		},
		"GPL-2.0": {
			Name: "GNU General Public License v2.0",
			Key:  "GPL-2.0",
			Type: "copyleft",
			Keywords: []string{
				"GNU GENERAL PUBLIC LICENSE",
				"Version 2",
				"This program is free software; you can redistribute it",
				"GNU General Public License for more details",
			},
			Confidence: 0.95,
		},
		"LGPL-3.0": {
			Name: "GNU Lesser General Public License v3.0",
			Key:  "LGPL-3.0",
			Type: "copyleft",
			Keywords: []string{
				"GNU LESSER GENERAL PUBLIC LICENSE",
				"Version 3",
				"This library is free software",
			},
			Confidence: 0.95,
		},
		"LGPL-2.1": {
			Name: "GNU Lesser General Public License v2.1",
			Key:  "LGPL-2.1",
			Type: "copyleft",
			Keywords: []string{
				"GNU LESSER GENERAL PUBLIC LICENSE",
				"Version 2.1",
				"This library is free software",
			},
			Confidence: 0.95,
		},
		"BSD-3-Clause": {
			Name: "BSD 3-Clause License",
			Key:  "BSD-3-Clause",
			Type: "permissive",
			Keywords: []string{
				"BSD 3-Clause License",
				"Redistribution and use in source and binary forms",
				"Neither the name of",
				"may be used to endorse or promote products",
			},
			Confidence: 0.9,
		},
		"BSD-2-Clause": {
			Name: "BSD 2-Clause License",
			Key:  "BSD-2-Clause",
			Type: "permissive",
			Keywords: []string{
				"BSD 2-Clause License",
				"Redistribution and use in source and binary forms",
				"THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS",
			},
			Confidence: 0.9,
		},
		"ISC": {
			Name: "ISC License",
			Key:  "ISC",
			Type: "permissive",
			Keywords: []string{
				"ISC License",
				"Permission to use, copy, modify, and/or distribute",
				"THE SOFTWARE IS PROVIDED \"AS IS\"",
			},
			Confidence: 0.9,
		},
		"MPL-2.0": {
			Name: "Mozilla Public License 2.0",
			Key:  "MPL-2.0",
			Type: "weak-copyleft",
			Keywords: []string{
				"Mozilla Public License Version 2.0",
				"This Source Code Form is subject to the terms",
				"http://mozilla.org/MPL/2.0/",
			},
			Confidence: 0.95,
		},
		"CC0-1.0": {
			Name: "Creative Commons Zero v1.0 Universal",
			Key:  "CC0-1.0",
			Type: "public-domain",
			Keywords: []string{
				"Creative Commons Legal Code",
				"CC0 1.0 Universal",
				"No Copyright",
			},
			Confidence: 0.95,
		},
		"Unlicense": {
			Name: "The Unlicense",
			Key:  "Unlicense",
			Type: "public-domain",
			Keywords: []string{
				"This is free and unencumbered software",
				"Anyone is free to copy, modify, publish, use",
				"THE SOFTWARE IS PROVIDED \"AS IS\"",
				"unlicense.org",
			},
			Confidence: 0.95,
		},
		"AGPL-3.0": {
			Name: "GNU Affero General Public License v3.0",
			Key:  "AGPL-3.0",
			Type: "copyleft",
			Keywords: []string{
				"GNU AFFERO GENERAL PUBLIC LICENSE",
				"Version 3",
				"This program is free software: you can redistribute it",
				"GNU Affero General Public License",
			},
			Confidence: 0.95,
		},
		"WTFPL": {
			Name: "Do What The F*ck You Want To Public License",
			Key:  "WTFPL",
			Type: "permissive",
			Keywords: []string{
				"DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE",
				"WTFPL",
				"You just DO WHAT THE FUCK YOU WANT TO",
			},
			Confidence: 0.95,
		},
		"Zlib": {
			Name: "zlib License",
			Key:  "Zlib",
			Type: "permissive",
			Keywords: []string{
				"zlib License",
				"This software is provided 'as-is'",
				"without any express or implied warranty",
			},
			Confidence: 0.9,
		},
		"EPL-2.0": {
			Name: "Eclipse Public License 2.0",
			Key:  "EPL-2.0",
			Type: "weak-copyleft",
			Keywords: []string{
				"Eclipse Public License - v 2.0",
				"THE ACCOMPANYING PROGRAM IS PROVIDED",
				"eclipse.org/legal/epl-2.0",
			},
			Confidence: 0.95,
		},
		"GPL-3.0-or-later": {
			Name: "GNU General Public License v3.0 or later",
			Key:  "GPL-3.0-or-later",
			Type: "copyleft",
			Keywords: []string{
				"either version 3 of the License, or (at your option) any later version",
				"GNU General Public License as published by the Free Software Foundation",
			},
			Confidence: 0.9,
		},
		"GPL-2.0-or-later": {
			Name: "GNU General Public License v2.0 or later",
			Key:  "GPL-2.0-or-later",
			Type: "copyleft",
			Keywords: []string{
				"either version 2 of the License, or (at your option) any later version",
				"GNU General Public License for more details",
			},
			Confidence: 0.9,
		},
	}
	
	// Compile regex patterns and store
	for key, license := range licenses {
		var patterns []*regexp.Regexp
		for _, keyword := range license.Keywords {
			// Create case-insensitive regex pattern
			pattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(keyword))
			patterns = append(patterns, pattern)
		}
		license.Patterns = patterns
		ld.licensePatterns[key] = license
	}
}

// DetectLicenses analyzes a repository and detects licenses
func (ld *LicenseDetector) DetectLicenses(repoPath string, config *AnalysisConfig) ([]LicenseInfo, error) {
	ld.logger.Debugf("开始检测许可证: %s", repoPath)
	
	var licenses []LicenseInfo
	
	// Find potential license files
	licenseFiles := ld.findLicenseFiles(repoPath)
	
	// Analyze each license file
	for _, file := range licenseFiles {
		fileLicenses := ld.analyzeLicenseFile(file)
		for _, license := range fileLicenses {
			license.SourceFile = file
			licenses = append(licenses, license)
		}
	}
	
	// If no license files found, check source code headers
	if len(licenses) == 0 {
		headerLicenses := ld.detectLicenseFromHeaders(repoPath, config)
		licenses = append(licenses, headerLicenses...)
	}
	
	// Remove duplicates and sort by confidence
	licenses = ld.deduplicateLicenses(licenses)
	
	ld.logger.Debugf("检测到 %d 个许可证", len(licenses))
	return licenses, nil
}

// findLicenseFiles finds potential license files in the repository
func (ld *LicenseDetector) findLicenseFiles(repoPath string) []string {
	var licenseFiles []string
	
	// Common license file patterns
	licensePatterns := []string{
		"LICENSE", "LICENSE.txt", "LICENSE.md", "LICENSE.rst",
		"LICENCE", "LICENCE.txt", "LICENCE.md", "LICENCE.rst",
		"COPYING", "COPYING.txt", "COPYING.md",
		"COPYRIGHT", "COPYRIGHT.txt", "COPYRIGHT.md",
		"UNLICENSE", "UNLICENSE.txt",
		"license", "license.txt", "license.md", "license.rst",
		"licence", "licence.txt", "licence.md", "licence.rst",
		"copying", "copying.txt", "copying.md",
		"copyright", "copyright.txt", "copyright.md",
	}
	
	for _, pattern := range licensePatterns {
		filePath := filepath.Join(repoPath, pattern)
		if ld.fileExists(filePath) {
			licenseFiles = append(licenseFiles, filePath)
		}
	}
	
	return licenseFiles
}

// analyzeLicenseFile analyzes a single license file
func (ld *LicenseDetector) analyzeLicenseFile(filePath string) []LicenseInfo {
	var licenses []LicenseInfo
	
	content, err := ReadFileContent(filePath, 1024*1024) // 1MB limit
	if err != nil {
		return licenses
	}
	
	// Normalize content
	content = strings.ToLower(content)
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	
	// Check against all known licenses
	for _, pattern := range ld.licensePatterns {
		score := ld.calculateLicenseScore(content, pattern)
		if score > 0.5 { // Minimum confidence threshold
			licenses = append(licenses, LicenseInfo{
				Name:       pattern.Name,
				Key:        pattern.Key,
				Type:       pattern.Type,
				Confidence: score,
			})
		}
	}
	
	return licenses
}

// calculateLicenseScore calculates a confidence score for a license match
func (ld *LicenseDetector) calculateLicenseScore(content string, pattern LicensePattern) float64 {
	matchCount := 0
	totalKeywords := len(pattern.Keywords)
	
	if totalKeywords == 0 {
		return 0
	}
	
	for _, regex := range pattern.Patterns {
		if regex.MatchString(content) {
			matchCount++
		}
	}
	
	// Calculate base score
	baseScore := float64(matchCount) / float64(totalKeywords)
	
	// Apply pattern confidence modifier
	finalScore := baseScore * pattern.Confidence
	
	// Cap at 1.0
	if finalScore > 1.0 {
		finalScore = 1.0
	}
	
	return finalScore
}

// detectLicenseFromHeaders detects licenses from source code headers
func (ld *LicenseDetector) detectLicenseFromHeaders(repoPath string, config *AnalysisConfig) []LicenseInfo {
	var licenses []LicenseInfo
	
	// Find source code files
	sourceExtensions := []string{"js", "ts", "py", "java", "go", "rs", "c", "cpp", "h", "hpp", "php", "rb", "cs"}
	files, err := FindFiles(repoPath, sourceExtensions, config.IgnorePatterns)
	if err != nil {
		return licenses
	}
	
	licenseMatches := make(map[string]int)
	totalFiles := 0
	
	// Analyze first few lines of source files (limit to avoid performance issues)
	maxFiles := 50
	for i, file := range files {
		if i >= maxFiles {
			break
		}
		
		content, err := ReadFileContent(filepath.Join(repoPath, file.Path), 4096) // Only read first 4KB
		if err != nil {
			continue
		}
		
		// Only check first 20 lines
		lines := strings.Split(content, "\n")
		if len(lines) > 20 {
			lines = lines[:20]
		}
		headerContent := strings.Join(lines, "\n")
		
		// Check for license references
		for key, pattern := range ld.licensePatterns {
			score := ld.calculateLicenseScore(strings.ToLower(headerContent), pattern)
			if score > 0.3 { // Lower threshold for headers
				licenseMatches[key]++
			}
		}
		
		totalFiles++
	}
	
	// Convert matches to license info
	for key, count := range licenseMatches {
		pattern := ld.licensePatterns[key]
		confidence := float64(count) / float64(totalFiles)
		if confidence > 0.1 { // At least 10% of files should match
			licenses = append(licenses, LicenseInfo{
				Name:       pattern.Name,
				Key:        pattern.Key,
				Type:       pattern.Type,
				Confidence: confidence * 0.7, // Reduce confidence for header detection
				SourceFile: "source code headers",
			})
		}
	}
	
	return licenses
}

// deduplicateLicenses removes duplicate licenses and sorts by confidence
func (ld *LicenseDetector) deduplicateLicenses(licenses []LicenseInfo) []LicenseInfo {
	seen := make(map[string]*LicenseInfo)
	
	for _, license := range licenses {
		if existing, exists := seen[license.Key]; exists {
			// Keep the one with higher confidence
			if license.Confidence > existing.Confidence {
				seen[license.Key] = &license
			}
		} else {
			licenseCopy := license
			seen[license.Key] = &licenseCopy
		}
	}
	
	// Convert back to slice
	var unique []LicenseInfo
	for _, license := range seen {
		unique = append(unique, *license)
	}
	
	// Sort by confidence (descending)
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Confidence > unique[j].Confidence
	})
	
	return unique
}

// fileExists checks if a file exists
func (ld *LicenseDetector) fileExists(filePath string) bool {
	_, err := ReadFileContent(filePath, 1)
	return err == nil
}

// GetLicenseType categorizes license types
func (ld *LicenseDetector) GetLicenseType(licenseKey string) string {
	if pattern, exists := ld.licensePatterns[licenseKey]; exists {
		return pattern.Type
	}
	return "unknown"
}

// IsPermissiveLicense checks if a license is permissive
func (ld *LicenseDetector) IsPermissiveLicense(licenseKey string) bool {
	licenseType := ld.GetLicenseType(licenseKey)
	return licenseType == "permissive" || licenseType == "public-domain"
}

// IsCopyleftLicense checks if a license is copyleft
func (ld *LicenseDetector) IsCopyleftLicense(licenseKey string) bool {
	licenseType := ld.GetLicenseType(licenseKey)
	return licenseType == "copyleft" || licenseType == "weak-copyleft"
}

// GetLicenseCompatibility checks compatibility between licenses
func (ld *LicenseDetector) GetLicenseCompatibility(licenses []LicenseInfo) map[string]bool {
	compatibility := make(map[string]bool)
	
	if len(licenses) == 0 {
		return compatibility
	}
	
	// Check for conflicting licenses
	hasGPL := false
	hasApache := false
	hasMIT := false
	
	for _, license := range licenses {
		switch license.Key {
		case "GPL-2.0", "GPL-3.0", "AGPL-3.0":
			hasGPL = true
		case "Apache-2.0":
			hasApache = true
		case "MIT":
			hasMIT = true
		}
	}
	
	compatibility["has_copyleft"] = hasGPL
	compatibility["has_permissive"] = hasApache || hasMIT
	compatibility["potential_conflicts"] = hasGPL && (hasApache || hasMIT)
	
	return compatibility
}

// GetSupportedLicenses returns a list of all supported licenses
func (ld *LicenseDetector) GetSupportedLicenses() []string {
	var licenses []string
	
	for key := range ld.licensePatterns {
		licenses = append(licenses, key)
	}
	
	sort.Strings(licenses)
	return licenses
}

// GetLicenseInfo returns detailed information about a license
func (ld *LicenseDetector) GetLicenseInfo(licenseKey string) *LicensePattern {
	if pattern, exists := ld.licensePatterns[licenseKey]; exists {
		return &pattern
	}
	return nil
}