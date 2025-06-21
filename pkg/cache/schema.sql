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

-- 项目语言检测缓存
CREATE TABLE IF NOT EXISTS repository_languages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    language TEXT NOT NULL,
    percentage REAL NOT NULL DEFAULT 0.0,
    lines_of_code INTEGER DEFAULT 0,
    file_count INTEGER DEFAULT 0,
    bytes_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories (id) ON DELETE CASCADE
);

-- 项目元数据表
CREATE TABLE IF NOT EXISTS repository_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    project_type TEXT,                         -- 项目类型：library, application, cli-tool等
    main_language TEXT,                        -- 主要编程语言
    total_lines_of_code INTEGER DEFAULT 0,     -- 总代码行数
    file_count INTEGER DEFAULT 0,              -- 文件总数
    directory_count INTEGER DEFAULT 0,         -- 目录总数
    repository_size INTEGER DEFAULT 0,         -- 仓库大小（字节）
    has_readme BOOLEAN DEFAULT FALSE,          -- 是否有README
    has_license BOOLEAN DEFAULT FALSE,         -- 是否有LICENSE
    has_tests BOOLEAN DEFAULT FALSE,           -- 是否有测试
    has_ci BOOLEAN DEFAULT FALSE,              -- 是否有CI配置
    has_docs BOOLEAN DEFAULT FALSE,            -- 是否有文档
    complexity_score REAL DEFAULT 0.0,        -- 复杂度评分
    quality_score REAL DEFAULT 0.0,           -- 质量评分
    structure_hash TEXT,                       -- 项目结构哈希值
    description TEXT,                          -- 项目描述
    enhanced_description TEXT,                 -- LLM增强的项目描述
    analyzed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories (id) ON DELETE CASCADE,
    UNIQUE(repository_id)
);

-- 项目框架检测表
CREATE TABLE IF NOT EXISTS repository_frameworks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    framework TEXT NOT NULL,                   -- 框架名称：react, vue, django, spring等
    version TEXT,                              -- 版本号
    category TEXT,                             -- 分类：frontend, backend, mobile, desktop等
    confidence REAL DEFAULT 0.0,              -- 检测置信度
    detection_method TEXT,                     -- 检测方法：package.json, requirements.txt等
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories (id) ON DELETE CASCADE
);

-- 项目许可证表
CREATE TABLE IF NOT EXISTS repository_licenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    license_name TEXT NOT NULL,                -- 许可证名称：MIT, Apache-2.0等
    license_key TEXT,                          -- SPDX标识符
    license_type TEXT,                         -- 许可证类型：permissive, copyleft, proprietary等
    source_file TEXT,                          -- 检测到的文件路径
    confidence REAL DEFAULT 0.0,              -- 检测置信度
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories (id) ON DELETE CASCADE
);

-- 项目依赖表
CREATE TABLE IF NOT EXISTS repository_dependencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    dependency_name TEXT NOT NULL,             -- 依赖名称
    version TEXT,                              -- 版本号
    type TEXT,                                 -- 依赖类型：production, development, peer等
    package_manager TEXT,                      -- 包管理器：npm, pip, maven等
    source_file TEXT,                          -- 来源文件
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories (id) ON DELETE CASCADE
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_repositories_path ON repositories (path);
CREATE INDEX IF NOT EXISTS idx_repositories_readme_hash ON repositories (readme_hash);
CREATE INDEX IF NOT EXISTS idx_repositories_updated_at ON repositories (updated_at);
CREATE INDEX IF NOT EXISTS idx_repository_tags_repo_id ON repository_tags (repository_id);
CREATE INDEX IF NOT EXISTS idx_repository_languages_repo_id ON repository_languages (repository_id);
CREATE INDEX IF NOT EXISTS idx_repository_languages_language ON repository_languages (language);
CREATE INDEX IF NOT EXISTS idx_repository_metadata_repo_id ON repository_metadata (repository_id);
CREATE INDEX IF NOT EXISTS idx_repository_metadata_main_language ON repository_metadata (main_language);
CREATE INDEX IF NOT EXISTS idx_repository_metadata_project_type ON repository_metadata (project_type);
CREATE INDEX IF NOT EXISTS idx_repository_metadata_structure_hash ON repository_metadata (structure_hash);
CREATE INDEX IF NOT EXISTS idx_repository_frameworks_repo_id ON repository_frameworks (repository_id);
CREATE INDEX IF NOT EXISTS idx_repository_frameworks_framework ON repository_frameworks (framework);
CREATE INDEX IF NOT EXISTS idx_repository_frameworks_category ON repository_frameworks (category);
CREATE INDEX IF NOT EXISTS idx_repository_licenses_repo_id ON repository_licenses (repository_id);
CREATE INDEX IF NOT EXISTS idx_repository_licenses_license_key ON repository_licenses (license_key);
CREATE INDEX IF NOT EXISTS idx_repository_dependencies_repo_id ON repository_dependencies (repository_id);
CREATE INDEX IF NOT EXISTS idx_repository_dependencies_name ON repository_dependencies (dependency_name);

-- 初始化统计数据
INSERT OR IGNORE INTO cache_stats (id, total_repositories, cached_descriptions, cache_hits, cache_misses, llm_api_calls)
VALUES (1, 0, 0, 0, 0, 0);