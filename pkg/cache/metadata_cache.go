package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"reposense/pkg/analyzer"
)

// MetadataCache handles caching of repository metadata
type MetadataCache struct {
	cache *Cache
}

// NewMetadataCache creates a new metadata cache instance
func NewMetadataCache(cache *Cache) *MetadataCache {
	return &MetadataCache{
		cache: cache,
	}
}

// GetCachedMetadata retrieves cached metadata for a repository
func (mc *MetadataCache) GetCachedMetadata(repoPath, structureHash string) (*analyzer.ProjectMetadata, bool) {
	query := `
		SELECT rm.project_type, rm.main_language, rm.total_lines_of_code, rm.file_count, rm.directory_count,
		       rm.repository_size, rm.has_readme, rm.has_license, rm.has_tests, rm.has_ci, rm.has_docs,
		       rm.complexity_score, rm.quality_score, rm.structure_hash, rm.description, rm.enhanced_description, rm.analyzed_at
		FROM repository_metadata rm
		JOIN repositories r ON r.id = rm.repository_id
		WHERE r.path = ? AND rm.structure_hash = ?
	`
	
	var metadata analyzer.ProjectMetadata
	var analyzedAtStr string
	
	err := mc.cache.db.QueryRow(query, repoPath, structureHash).Scan(
		&metadata.ProjectType, &metadata.MainLanguage, &metadata.TotalLinesOfCode,
		&metadata.FileCount, &metadata.DirectoryCount, &metadata.RepositorySize,
		&metadata.HasReadme, &metadata.HasLicense, &metadata.HasTests,
		&metadata.HasCI, &metadata.HasDocs, &metadata.ComplexityScore,
		&metadata.QualityScore, &metadata.StructureHash, &metadata.Description,
		&metadata.EnhancedDescription, &analyzedAtStr,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}
		mc.cache.logger.WithError(err).Warn("查询metadata缓存失败")
		return nil, false
	}
	
	// Parse analyzed_at time
	if analyzedAt, err := time.Parse("2006-01-02 15:04:05", analyzedAtStr); err == nil {
		metadata.AnalyzedAt = analyzedAt
	}
	
	// Load related data
	if err := mc.loadLanguages(repoPath, &metadata); err != nil {
		mc.cache.logger.WithError(err).Warn("加载语言信息失败")
	}
	
	if err := mc.loadFrameworks(repoPath, &metadata); err != nil {
		mc.cache.logger.WithError(err).Warn("加载框架信息失败")
	}
	
	if err := mc.loadLicenses(repoPath, &metadata); err != nil {
		mc.cache.logger.WithError(err).Warn("加载许可证信息失败")
	}
	
	if err := mc.loadDependencies(repoPath, &metadata); err != nil {
		mc.cache.logger.WithError(err).Warn("加载依赖信息失败")
	}
	
	mc.cache.logger.Debugf("Metadata缓存命中: %s", repoPath)
	return &metadata, true
}

// SaveMetadata saves metadata to cache
func (mc *MetadataCache) SaveMetadata(repoPath, repoName string, metadata *analyzer.ProjectMetadata) error {
	tx, err := mc.cache.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()
	
	// Get or create repository record
	repoID, err := mc.getOrCreateRepository(tx, repoPath, repoName)
	if err != nil {
		return fmt.Errorf("获取仓库ID失败: %w", err)
	}
	
	// Save metadata
	if err := mc.saveRepositoryMetadata(tx, repoID, metadata); err != nil {
		return fmt.Errorf("保存metadata失败: %w", err)
	}
	
	// Save languages
	if err := mc.saveLanguages(tx, repoID, metadata.Languages); err != nil {
		return fmt.Errorf("保存语言信息失败: %w", err)
	}
	
	// Save frameworks
	if err := mc.saveFrameworks(tx, repoID, metadata.Frameworks); err != nil {
		return fmt.Errorf("保存框架信息失败: %w", err)
	}
	
	// Save licenses
	if err := mc.saveLicenses(tx, repoID, metadata.Licenses); err != nil {
		return fmt.Errorf("保存许可证信息失败: %w", err)
	}
	
	// Save dependencies
	if err := mc.saveDependencies(tx, repoID, metadata.Dependencies); err != nil {
		return fmt.Errorf("保存依赖信息失败: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	
	mc.cache.logger.Debugf("保存metadata到缓存: %s", repoPath)
	return nil
}

// getOrCreateRepository gets repository ID or creates new record
func (mc *MetadataCache) getOrCreateRepository(tx *sql.Tx, repoPath, repoName string) (int64, error) {
	// Try to get existing repository
	var repoID int64
	err := tx.QueryRow("SELECT id FROM repositories WHERE path = ?", repoPath).Scan(&repoID)
	if err == nil {
		return repoID, nil
	}
	
	if err != sql.ErrNoRows {
		return 0, err
	}
	
	// Create new repository record
	result, err := tx.Exec(
		"INSERT INTO repositories (path, name, created_at, updated_at, last_accessed) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		repoPath, repoName,
	)
	if err != nil {
		return 0, err
	}
	
	return result.LastInsertId()
}

// saveRepositoryMetadata saves main metadata
func (mc *MetadataCache) saveRepositoryMetadata(tx *sql.Tx, repoID int64, metadata *analyzer.ProjectMetadata) error {
	query := `
		INSERT OR REPLACE INTO repository_metadata 
		(repository_id, project_type, main_language, total_lines_of_code, file_count, 
		 directory_count, repository_size, has_readme, has_license, has_tests, has_ci, 
		 has_docs, complexity_score, quality_score, structure_hash, description, 
		 enhanced_description, analyzed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	
	_, err := tx.Exec(query,
		repoID, metadata.ProjectType, metadata.MainLanguage, metadata.TotalLinesOfCode,
		metadata.FileCount, metadata.DirectoryCount, metadata.RepositorySize,
		metadata.HasReadme, metadata.HasLicense, metadata.HasTests, metadata.HasCI,
		metadata.HasDocs, metadata.ComplexityScore, metadata.QualityScore,
		metadata.StructureHash, metadata.Description, metadata.EnhancedDescription,
		metadata.AnalyzedAt.Format("2006-01-02 15:04:05"),
	)
	
	return err
}

// saveLanguages saves language information
func (mc *MetadataCache) saveLanguages(tx *sql.Tx, repoID int64, languages []analyzer.LanguageInfo) error {
	// Delete existing languages
	if _, err := tx.Exec("DELETE FROM repository_languages WHERE repository_id = ?", repoID); err != nil {
		return err
	}
	
	// Insert new languages
	for _, lang := range languages {
		query := `
			INSERT INTO repository_languages 
			(repository_id, language, percentage, lines_of_code, file_count, bytes_count, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		`
		
		_, err := tx.Exec(query, repoID, lang.Name, lang.Percentage, lang.LinesOfCode, lang.FileCount, lang.BytesCount)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// saveFrameworks saves framework information
func (mc *MetadataCache) saveFrameworks(tx *sql.Tx, repoID int64, frameworks []analyzer.FrameworkInfo) error {
	// Delete existing frameworks
	if _, err := tx.Exec("DELETE FROM repository_frameworks WHERE repository_id = ?", repoID); err != nil {
		return err
	}
	
	// Insert new frameworks
	for _, framework := range frameworks {
		query := `
			INSERT INTO repository_frameworks 
			(repository_id, framework, version, category, confidence, detection_method, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		`
		
		_, err := tx.Exec(query, repoID, framework.Name, framework.Version, framework.Category, framework.Confidence, framework.DetectionMethod)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// saveLicenses saves license information
func (mc *MetadataCache) saveLicenses(tx *sql.Tx, repoID int64, licenses []analyzer.LicenseInfo) error {
	// Delete existing licenses
	if _, err := tx.Exec("DELETE FROM repository_licenses WHERE repository_id = ?", repoID); err != nil {
		return err
	}
	
	// Insert new licenses
	for _, license := range licenses {
		query := `
			INSERT INTO repository_licenses 
			(repository_id, license_name, license_key, license_type, source_file, confidence, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		`
		
		_, err := tx.Exec(query, repoID, license.Name, license.Key, license.Type, license.SourceFile, license.Confidence)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// saveDependencies saves dependency information
func (mc *MetadataCache) saveDependencies(tx *sql.Tx, repoID int64, dependencies []analyzer.DependencyInfo) error {
	// Delete existing dependencies
	if _, err := tx.Exec("DELETE FROM repository_dependencies WHERE repository_id = ?", repoID); err != nil {
		return err
	}
	
	// Insert new dependencies
	for _, dep := range dependencies {
		query := `
			INSERT INTO repository_dependencies 
			(repository_id, dependency_name, version, type, package_manager, source_file)
			VALUES (?, ?, ?, ?, ?, ?)
		`
		
		_, err := tx.Exec(query, repoID, dep.Name, dep.Version, dep.Type, dep.PackageManager, dep.SourceFile)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// loadLanguages loads language information
func (mc *MetadataCache) loadLanguages(repoPath string, metadata *analyzer.ProjectMetadata) error {
	query := `
		SELECT rl.language, rl.percentage, rl.lines_of_code, rl.file_count, rl.bytes_count
		FROM repository_languages rl
		JOIN repositories r ON r.id = rl.repository_id
		WHERE r.path = ?
		ORDER BY rl.lines_of_code DESC
	`
	
	rows, err := mc.cache.db.Query(query, repoPath)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var languages []analyzer.LanguageInfo
	for rows.Next() {
		var lang analyzer.LanguageInfo
		if err := rows.Scan(&lang.Name, &lang.Percentage, &lang.LinesOfCode, &lang.FileCount, &lang.BytesCount); err != nil {
			return err
		}
		languages = append(languages, lang)
	}
	
	metadata.Languages = languages
	return rows.Err()
}

// loadFrameworks loads framework information
func (mc *MetadataCache) loadFrameworks(repoPath string, metadata *analyzer.ProjectMetadata) error {
	query := `
		SELECT rf.framework, rf.version, rf.category, rf.confidence, rf.detection_method
		FROM repository_frameworks rf
		JOIN repositories r ON r.id = rf.repository_id
		WHERE r.path = ?
		ORDER BY rf.confidence DESC
	`
	
	rows, err := mc.cache.db.Query(query, repoPath)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var frameworks []analyzer.FrameworkInfo
	for rows.Next() {
		var framework analyzer.FrameworkInfo
		if err := rows.Scan(&framework.Name, &framework.Version, &framework.Category, &framework.Confidence, &framework.DetectionMethod); err != nil {
			return err
		}
		frameworks = append(frameworks, framework)
	}
	
	metadata.Frameworks = frameworks
	return rows.Err()
}

// loadLicenses loads license information
func (mc *MetadataCache) loadLicenses(repoPath string, metadata *analyzer.ProjectMetadata) error {
	query := `
		SELECT rl.license_name, rl.license_key, rl.license_type, rl.source_file, rl.confidence
		FROM repository_licenses rl
		JOIN repositories r ON r.id = rl.repository_id
		WHERE r.path = ?
		ORDER BY rl.confidence DESC
	`
	
	rows, err := mc.cache.db.Query(query, repoPath)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var licenses []analyzer.LicenseInfo
	for rows.Next() {
		var license analyzer.LicenseInfo
		if err := rows.Scan(&license.Name, &license.Key, &license.Type, &license.SourceFile, &license.Confidence); err != nil {
			return err
		}
		licenses = append(licenses, license)
	}
	
	metadata.Licenses = licenses
	return rows.Err()
}

// loadDependencies loads dependency information
func (mc *MetadataCache) loadDependencies(repoPath string, metadata *analyzer.ProjectMetadata) error {
	query := `
		SELECT rd.dependency_name, rd.version, rd.type, rd.package_manager, rd.source_file
		FROM repository_dependencies rd
		JOIN repositories r ON r.id = rd.repository_id
		WHERE r.path = ?
		ORDER BY rd.dependency_name
	`
	
	rows, err := mc.cache.db.Query(query, repoPath)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var dependencies []analyzer.DependencyInfo
	for rows.Next() {
		var dep analyzer.DependencyInfo
		if err := rows.Scan(&dep.Name, &dep.Version, &dep.Type, &dep.PackageManager, &dep.SourceFile); err != nil {
			return err
		}
		dependencies = append(dependencies, dep)
	}
	
	metadata.Dependencies = dependencies
	return rows.Err()
}

// RefreshMetadata removes cached metadata for a specific repository
func (mc *MetadataCache) RefreshMetadata(repoPath string) error {
	// The foreign key constraints will automatically clean up related data
	return mc.cache.RefreshRepository(repoPath)
}

// GetMetadataStats returns metadata cache statistics
func (mc *MetadataCache) GetMetadataStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Total repositories with metadata
	var repoCount int64
	err := mc.cache.db.QueryRow("SELECT COUNT(*) FROM repository_metadata").Scan(&repoCount)
	if err != nil {
		return nil, err
	}
	stats["repositories_with_metadata"] = repoCount
	
	// Language distribution
	languageStats := make(map[string]int)
	rows, err := mc.cache.db.Query(`
		SELECT rm.main_language, COUNT(*) as count 
		FROM repository_metadata rm 
		WHERE rm.main_language IS NOT NULL AND rm.main_language != ''
		GROUP BY rm.main_language 
		ORDER BY count DESC 
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var lang string
			var count int
			if err := rows.Scan(&lang, &count); err == nil {
				languageStats[lang] = count
			}
		}
	}
	stats["top_languages"] = languageStats
	
	// Framework distribution
	frameworkStats := make(map[string]int)
	rows, err = mc.cache.db.Query(`
		SELECT rf.framework, COUNT(*) as count 
		FROM repository_frameworks rf 
		GROUP BY rf.framework 
		ORDER BY count DESC 
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var framework string
			var count int
			if err := rows.Scan(&framework, &count); err == nil {
				frameworkStats[framework] = count
			}
		}
	}
	stats["top_frameworks"] = frameworkStats
	
	// License distribution
	licenseStats := make(map[string]int)
	rows, err = mc.cache.db.Query(`
		SELECT rl.license_key, COUNT(*) as count 
		FROM repository_licenses rl 
		GROUP BY rl.license_key 
		ORDER BY count DESC 
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var license string
			var count int
			if err := rows.Scan(&license, &count); err == nil {
				licenseStats[license] = count
			}
		}
	}
	stats["top_licenses"] = licenseStats
	
	// Average scores
	var avgComplexity, avgQuality float64
	err = mc.cache.db.QueryRow(`
		SELECT AVG(complexity_score), AVG(quality_score) 
		FROM repository_metadata 
		WHERE complexity_score > 0 AND quality_score > 0
	`).Scan(&avgComplexity, &avgQuality)
	if err == nil {
		stats["average_complexity_score"] = avgComplexity
		stats["average_quality_score"] = avgQuality
	}
	
	return stats, nil
}

// SearchRepositories searches repositories by various criteria
func (mc *MetadataCache) SearchRepositories(criteria map[string]interface{}) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	// Build dynamic query based on criteria
	query := `
		SELECT r.path, r.name, rm.project_type, rm.main_language, 
		       rm.total_lines_of_code, rm.complexity_score, rm.quality_score
		FROM repositories r
		JOIN repository_metadata rm ON r.id = rm.repository_id
		WHERE 1=1
	`
	args := []interface{}{}
	
	if lang, ok := criteria["language"]; ok {
		query += " AND rm.main_language = ?"
		args = append(args, lang)
	}
	
	if projectType, ok := criteria["project_type"]; ok {
		query += " AND rm.project_type = ?"
		args = append(args, projectType)
	}
	
	if minLOC, ok := criteria["min_lines_of_code"]; ok {
		query += " AND rm.total_lines_of_code >= ?"
		args = append(args, minLOC)
	}
	
	if maxLOC, ok := criteria["max_lines_of_code"]; ok {
		query += " AND rm.total_lines_of_code <= ?"
		args = append(args, maxLOC)
	}
	
	if minQuality, ok := criteria["min_quality_score"]; ok {
		query += " AND rm.quality_score >= ?"
		args = append(args, minQuality)
	}
	
	query += " ORDER BY rm.quality_score DESC, rm.total_lines_of_code DESC LIMIT 100"
	
	rows, err := mc.cache.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var path, name, projectType, mainLanguage string
		var totalLOC int
		var complexityScore, qualityScore float64
		
		if err := rows.Scan(&path, &name, &projectType, &mainLanguage, &totalLOC, &complexityScore, &qualityScore); err != nil {
			continue
		}
		
		result := map[string]interface{}{
			"path":              path,
			"name":              name,
			"project_type":      projectType,
			"main_language":     mainLanguage,
			"total_lines_of_code": totalLOC,
			"complexity_score":  complexityScore,
			"quality_score":     qualityScore,
		}
		
		results = append(results, result)
	}
	
	return results, rows.Err()
}

// ExportMetadata exports metadata to JSON format
func (mc *MetadataCache) ExportMetadata(repoPath string) (string, error) {
	metadata, found := mc.GetCachedMetadata(repoPath, "")
	if !found {
		return "", fmt.Errorf("未找到仓库的metadata: %s", repoPath)
	}
	
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON序列化失败: %w", err)
	}
	
	return string(jsonData), nil
}