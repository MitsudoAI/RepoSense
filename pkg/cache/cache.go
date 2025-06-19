package cache

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

//go:embed schema.sql
var schemaFS embed.FS

// RepositoryCache represents cached repository metadata
type RepositoryCache struct {
	ID           int64     `json:"id"`
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	ReadmeHash   string    `json:"readme_hash"`
	Description  string    `json:"description"`
	LLMProvider  string    `json:"llm_provider"`
	LLMModel     string    `json:"llm_model"`
	LLMLanguage  string    `json:"llm_language"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalRepositories   int64     `json:"total_repositories"`
	CachedDescriptions  int64     `json:"cached_descriptions"`
	CacheHits          int64     `json:"cache_hits"`
	CacheMisses        int64     `json:"cache_misses"`
	LLMAPICalls        int64     `json:"llm_api_calls"`
	LastUpdated        time.Time `json:"last_updated"`
}

// Cache represents the metadata cache manager
type Cache struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewCache creates a new cache instance
func NewCache(dbPath string) (*Cache, error) {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// 确保数据库目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	cache := &Cache{
		db:     db,
		logger: logger,
	}

	// 初始化数据库
	if err := cache.initDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}

	return cache, nil
}

// Close closes the database connection
func (c *Cache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// SetLogLevel sets the logging level
func (c *Cache) SetLogLevel(level logrus.Level) {
	c.logger.SetLevel(level)
}

// initDatabase initializes the database schema
func (c *Cache) initDatabase() error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("读取schema文件失败: %w", err)
	}

	if _, err := c.db.Exec(string(schema)); err != nil {
		return fmt.Errorf("执行schema失败: %w", err)
	}

	c.logger.Debug("数据库初始化完成")
	return nil
}

// GetCachedDescription retrieves cached description for a repository
func (c *Cache) GetCachedDescription(repoPath, readmeContent string) (*RepositoryCache, bool) {
	readmeHash := c.hashContent(readmeContent)
	
	var cache RepositoryCache
	query := `
		SELECT id, path, name, readme_hash, description, llm_provider, llm_model, llm_language,
		       created_at, updated_at, last_accessed
		FROM repositories 
		WHERE path = ? AND readme_hash = ?
	`
	
	err := c.db.QueryRow(query, repoPath, readmeHash).Scan(
		&cache.ID, &cache.Path, &cache.Name, &cache.ReadmeHash, &cache.Description,
		&cache.LLMProvider, &cache.LLMModel, &cache.LLMLanguage,
		&cache.CreatedAt, &cache.UpdatedAt, &cache.LastAccessed,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.incrementCacheMisses()
			return nil, false
		}
		c.logger.WithError(err).Warn("查询缓存失败")
		return nil, false
	}
	
	// 更新最后访问时间
	c.updateLastAccessed(cache.ID)
	c.incrementCacheHits()
	
	c.logger.Debugf("缓存命中: %s", repoPath)
	return &cache, true
}

// SaveDescription saves a new description to cache
func (c *Cache) SaveDescription(repoPath, repoName, readmeContent, description, llmProvider, llmModel, llmLanguage string) error {
	readmeHash := c.hashContent(readmeContent)
	
	// 使用 UPSERT (INSERT OR REPLACE) 来处理更新情况
	query := `
		INSERT OR REPLACE INTO repositories 
		(path, name, readme_hash, description, llm_provider, llm_model, llm_language, updated_at, last_accessed)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	
	_, err := c.db.Exec(query, repoPath, repoName, readmeHash, description, llmProvider, llmModel, llmLanguage)
	if err != nil {
		return fmt.Errorf("保存描述到缓存失败: %w", err)
	}
	
	c.incrementLLMAPICalls()
	c.logger.Debugf("保存描述到缓存: %s", repoPath)
	return nil
}

// GetStats returns cache statistics
func (c *Cache) GetStats() (*CacheStats, error) {
	var stats CacheStats
	query := `
		SELECT total_repositories, cached_descriptions, cache_hits, cache_misses, llm_api_calls, last_updated
		FROM cache_stats WHERE id = 1
	`
	
	err := c.db.QueryRow(query).Scan(
		&stats.TotalRepositories, &stats.CachedDescriptions, &stats.CacheHits,
		&stats.CacheMisses, &stats.LLMAPICalls, &stats.LastUpdated,
	)
	
	if err != nil {
		return nil, fmt.Errorf("获取统计信息失败: %w", err)
	}
	
	// 更新实时统计
	stats.TotalRepositories = c.countTotalRepositories()
	stats.CachedDescriptions = c.countCachedDescriptions()
	
	return &stats, nil
}

// ClearCache clears all cached data
func (c *Cache) ClearCache() error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()
	
	// 清空所有表
	tables := []string{"repository_languages", "repository_tags", "repositories"}
	for _, table := range tables {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return fmt.Errorf("清空表 %s 失败: %w", table, err)
		}
	}
	
	// 重置统计
	if _, err := tx.Exec("UPDATE cache_stats SET total_repositories=0, cached_descriptions=0, cache_hits=0, cache_misses=0, llm_api_calls=0, last_updated=CURRENT_TIMESTAMP WHERE id=1"); err != nil {
		return fmt.Errorf("重置统计失败: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	
	c.logger.Info("缓存已清空")
	return nil
}

// RefreshRepository removes cached data for a specific repository
func (c *Cache) RefreshRepository(repoPath string) error {
	query := "DELETE FROM repositories WHERE path = ?"
	result, err := c.db.Exec(query, repoPath)
	if err != nil {
		return fmt.Errorf("刷新仓库缓存失败: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		c.logger.Debugf("已刷新仓库缓存: %s", repoPath)
	}
	
	return nil
}

// GetCacheSize returns the cache database file size
func (c *Cache) GetCacheSize(dbPath string) (int64, error) {
	info, err := os.Stat(dbPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// Helper methods

func (c *Cache) hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func (c *Cache) updateLastAccessed(id int64) {
	query := "UPDATE repositories SET last_accessed = CURRENT_TIMESTAMP WHERE id = ?"
	c.db.Exec(query, id)
}

func (c *Cache) incrementCacheHits() {
	query := "UPDATE cache_stats SET cache_hits = cache_hits + 1, last_updated = CURRENT_TIMESTAMP WHERE id = 1"
	c.db.Exec(query)
}

func (c *Cache) incrementCacheMisses() {
	query := "UPDATE cache_stats SET cache_misses = cache_misses + 1, last_updated = CURRENT_TIMESTAMP WHERE id = 1"
	c.db.Exec(query)
}

func (c *Cache) incrementLLMAPICalls() {
	query := "UPDATE cache_stats SET llm_api_calls = llm_api_calls + 1, last_updated = CURRENT_TIMESTAMP WHERE id = 1"
	c.db.Exec(query)
}

func (c *Cache) countTotalRepositories() int64 {
	var count int64
	query := "SELECT COUNT(*) FROM repositories"
	c.db.QueryRow(query).Scan(&count)
	return count
}

func (c *Cache) countCachedDescriptions() int64 {
	var count int64
	query := "SELECT COUNT(*) FROM repositories WHERE description IS NOT NULL AND description != ''"
	c.db.QueryRow(query).Scan(&count)
	return count
}