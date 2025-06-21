package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"reposense/pkg/llm"

	"github.com/sirupsen/logrus"
)

// Manager manages the cache and LLM interactions
type Manager struct {
	cache           *Cache
	llmClient       *llm.Client
	config          *CacheConfig
	logger          *logrus.Logger
	enableCache     bool
	forceRefresh    bool
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	DatabasePath    string `json:"database_path"`
	EnableCache     bool   `json:"enable_cache"`
	CacheDirectory  string `json:"cache_directory"`
}

// NewManager creates a new cache manager
func NewManager(enableLLM bool, llmProvider, llmModel, llmAPIKey, llmBaseURL, llmLanguage string, llmTimeout time.Duration, enableCache, forceRefresh bool) (*Manager, error) {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// 设置缓存配置
	cacheConfig := &CacheConfig{
		EnableCache:    enableCache,
		CacheDirectory: getCacheDirectory(),
	}
	cacheConfig.DatabasePath = filepath.Join(cacheConfig.CacheDirectory, "reposense.db")

	var cache *Cache
	if enableCache {
		var err error
		cache, err = NewCache(cacheConfig.DatabasePath)
		if err != nil {
			return nil, fmt.Errorf("初始化缓存失败: %w", err)
		}
	}

	// 初始化LLM客户端
	var llmClient *llm.Client
	if enableLLM {
		provider := llm.Provider(llmProvider)
		llmClient = llm.NewClient(provider, llmModel, llmAPIKey, llmBaseURL, llmTimeout)
	}

	return &Manager{
		cache:        cache,
		llmClient:    llmClient,
		config:       cacheConfig,
		logger:       logger,
		enableCache:  enableCache,
		forceRefresh: forceRefresh,
	}, nil
}

// Close closes the cache manager
func (m *Manager) Close() error {
	if m.cache != nil {
		return m.cache.Close()
	}
	return nil
}

// SetLogLevel sets the logging level
func (m *Manager) SetLogLevel(level logrus.Level) {
	m.logger.SetLevel(level)
	if m.cache != nil {
		m.cache.SetLogLevel(level)
	}
}

// GetDescription gets description from cache or generates it using LLM
func (m *Manager) GetDescription(repoPath, repoName, readmeContent, llmProvider, llmModel, llmLanguage string) (string, error) {
	// 如果没有README内容，返回空描述
	if readmeContent == "" {
		return "", nil
	}

	// 如果启用缓存且不强制刷新，先尝试从缓存获取
	if m.enableCache && !m.forceRefresh && m.cache != nil {
		if cached, found := m.cache.GetCachedDescription(repoPath, readmeContent); found {
			m.logger.Debugf("使用缓存描述: %s", repoPath)
			return cached.Description, nil
		}
	}

	// 如果没有LLM客户端，返回空描述
	if m.llmClient == nil {
		return "", nil
	}

	// 调用LLM生成描述
	m.logger.Debugf("调用LLM生成描述: %s", repoPath)
	description, err := m.llmClient.GenerateDescription(context.Background(), readmeContent, llmLanguage)
	if err != nil {
		return "", fmt.Errorf("LLM生成描述失败: %w", err)
	}

	// 如果启用缓存，保存到缓存
	if m.enableCache && m.cache != nil {
		if err := m.cache.SaveDescription(repoPath, repoName, readmeContent, description, llmProvider, llmModel, llmLanguage); err != nil {
			m.logger.WithError(err).Warn("保存描述到缓存失败")
		}
	}

	return description, nil
}

// GetCacheStats returns cache statistics
func (m *Manager) GetCacheStats() (*CacheStats, error) {
	if !m.enableCache || m.cache == nil {
		return &CacheStats{}, nil
	}
	return m.cache.GetStats()
}

// ClearCache clears all cached data
func (m *Manager) ClearCache() error {
	if !m.enableCache || m.cache == nil {
		return fmt.Errorf("缓存未启用")
	}
	return m.cache.ClearCache()
}

// RefreshRepository removes cached data for a specific repository
func (m *Manager) RefreshRepository(repoPath string) error {
	if !m.enableCache || m.cache == nil {
		return fmt.Errorf("缓存未启用")
	}
	return m.cache.RefreshRepository(repoPath)
}

// GetCacheSize returns cache database file size
func (m *Manager) GetCacheSize() (int64, error) {
	if !m.enableCache || m.cache == nil {
		return 0, nil
	}
	return m.cache.GetCacheSize(m.config.DatabasePath)
}

// GetDatabasePath returns the database file path
func (m *Manager) GetDatabasePath() string {
	return m.config.DatabasePath
}

// GetCache returns the cache instance
func (m *Manager) GetCache() *Cache {
	return m.cache
}

// getCacheDirectory returns the cache directory path
func getCacheDirectory() string {
	// 尝试使用 XDG_CACHE_HOME
	if cacheDir := os.Getenv("XDG_CACHE_HOME"); cacheDir != "" {
		return filepath.Join(cacheDir, "reposense")
	}

	// 使用用户主目录下的 .cache
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// 如果无法获取主目录，使用当前目录下的 .cache
		return ".cache/reposense"
	}

	return filepath.Join(homeDir, ".cache", "reposense")
}