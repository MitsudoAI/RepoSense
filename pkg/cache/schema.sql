-- RepoSense 元数据缓存数据库 Schema

-- 项目元数据表
CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,                 -- 仓库绝对路径
    name TEXT NOT NULL,                        -- 仓库名称
    readme_hash TEXT,                          -- README内容的SHA256哈希
    description TEXT,                          -- LLM生成的描述
    llm_provider TEXT,                         -- 使用的LLM提供商
    llm_model TEXT,                           -- 使用的LLM模型
    llm_language TEXT,                        -- 描述语言
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_accessed DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 缓存统计表
CREATE TABLE IF NOT EXISTS cache_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    total_repositories INTEGER DEFAULT 0,     -- 总仓库数
    cached_descriptions INTEGER DEFAULT 0,     -- 已缓存描述数
    cache_hits INTEGER DEFAULT 0,             -- 缓存命中次数
    cache_misses INTEGER DEFAULT 0,           -- 缓存未命中次数
    llm_api_calls INTEGER DEFAULT 0,          -- LLM API调用次数
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 项目标签表（为将来扩展准备）
CREATE TABLE IF NOT EXISTS repository_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    tag TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories (id) ON DELETE CASCADE,
    UNIQUE(repository_id, tag)
);

-- 项目语言检测缓存（为将来扩展准备）
CREATE TABLE IF NOT EXISTS repository_languages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    language TEXT NOT NULL,
    percentage REAL NOT NULL DEFAULT 0.0,
    lines_of_code INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories (id) ON DELETE CASCADE
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_repositories_path ON repositories (path);
CREATE INDEX IF NOT EXISTS idx_repositories_readme_hash ON repositories (readme_hash);
CREATE INDEX IF NOT EXISTS idx_repositories_updated_at ON repositories (updated_at);
CREATE INDEX IF NOT EXISTS idx_repository_tags_repo_id ON repository_tags (repository_id);
CREATE INDEX IF NOT EXISTS idx_repository_languages_repo_id ON repository_languages (repository_id);

-- 初始化统计数据
INSERT OR IGNORE INTO cache_stats (id, total_repositories, cached_descriptions, cache_hits, cache_misses, llm_api_calls)
VALUES (1, 0, 0, 0, 0, 0);