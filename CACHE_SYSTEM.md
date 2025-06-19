# LLM描述缓存系统

## 概述

RepoSense 现在集成了智能缓存系统，大幅提升了 `reposense list` 命令的性能。该系统基于SQLite，可以缓存LLM生成的项目描述，避免重复的API调用。

## 主要特性

### 🚀 **智能缓存**
- 基于README内容的SHA256哈希值进行缓存
- 只有当README内容发生变化时才重新调用LLM
- 支持多种LLM提供商和模型的缓存

### 📊 **缓存统计**
- 实时统计缓存命中率、API调用次数
- 数据库大小监控
- 详细的使用统计信息

### 🔧 **灵活管理**
- 支持选择性刷新特定仓库缓存
- 一键清空所有缓存数据
- 可禁用缓存回退到直接API调用

## 使用方法

### 基本使用

```bash
# 默认启用缓存
reposense list /path/to/repos

# 强制刷新所有缓存
reposense list /path/to/repos --force-refresh

# 禁用缓存
reposense list /path/to/repos --enable-cache=false
```

### 缓存管理

```bash
# 查看缓存统计
reposense cache stats

# 查看缓存文件路径
reposense cache path

# 清空所有缓存
reposense cache clear

# 刷新特定仓库缓存
reposense cache refresh /path/to/specific/repo

# 刷新所有缓存
reposense cache refresh
```

## 缓存策略

### 缓存键设计
缓存使用以下组合作为唯一标识：
- 仓库绝对路径
- README内容的SHA256哈希值

### 缓存失效机制
缓存在以下情况下会失效：
1. README文件内容发生变化
2. 手动刷新缓存
3. 清空缓存数据库

### 智能更新
- 系统自动检测README文件变化
- 仅在内容变化时重新生成描述
- 保持缓存数据的新鲜度

## 性能提升

### 速度对比
- **首次扫描**：需要调用LLM API（正常速度）
- **后续扫描**：直接从缓存读取（几乎瞬时）
- **大型项目集合**：速度提升可达 **10-100倍**

### 成本节约
- 减少重复的LLM API调用
- 降低API使用费用
- 减少网络请求次数

## 数据存储

### 存储位置
- **Linux/macOS**: `~/.cache/reposense/reposense.db`
- **Windows**: `%LOCALAPPDATA%\reposense\reposense.db`

### 数据库结构
```sql
-- 项目元数据
repositories (
    path,           -- 仓库路径
    name,           -- 仓库名称
    readme_hash,    -- README内容哈希
    description,    -- LLM生成的描述
    llm_provider,   -- LLM提供商
    llm_model,      -- LLM模型
    llm_language,   -- 描述语言
    created_at,     -- 创建时间
    updated_at,     -- 更新时间
    last_accessed   -- 最后访问时间
)

-- 缓存统计
cache_stats (
    total_repositories,  -- 总仓库数
    cached_descriptions, -- 已缓存描述数
    cache_hits,         -- 缓存命中次数
    cache_misses,       -- 缓存未命中次数
    llm_api_calls       -- LLM API调用次数
)
```

### 扩展性设计
数据库schema为将来扩展预留了空间：
- 项目标签系统
- 编程语言检测缓存
- 其他元数据缓存

## 配置选项

### 命令行标志
```bash
--enable-cache         # 启用缓存（默认true）
--force-refresh        # 强制刷新缓存
--verbose             # 显示缓存统计信息
```

### 环境变量
```bash
XDG_CACHE_HOME        # 自定义缓存目录（Linux/macOS）
```

## 故障排除

### 常见问题

**1. 缓存文件过大**
```bash
# 查看缓存大小
reposense cache stats

# 清空缓存
reposense cache clear
```

**2. 缓存不生效**
```bash
# 检查缓存是否启用
reposense list --help | grep cache

# 强制刷新测试
reposense list /path --force-refresh --verbose
```

**3. 权限问题**
```bash
# 检查缓存目录权限
ls -la ~/.cache/reposense/

# 手动创建目录
mkdir -p ~/.cache/reposense
```

### 调试模式
```bash
# 显示详细缓存操作
reposense list /path --verbose

# 查看具体的缓存命中情况
reposense cache stats
```

## 最佳实践

### 推荐工作流
1. **首次使用**：让缓存自然建立
2. **定期维护**：定期查看缓存统计
3. **项目更新后**：使用 `--force-refresh` 更新描述
4. **存储清理**：定期清理不再需要的缓存

### 性能优化
- 对于大型项目集合，首次扫描建议在网络良好时进行
- 使用 `--verbose` 模式监控缓存效果
- 定期查看缓存命中率，优化使用模式

### 多环境使用
- 不同机器间的缓存不共享（基于绝对路径）
- 可通过配置文件统一LLM设置
- 支持团队共享的配置模板

## 技术细节

### 缓存一致性
- 使用SHA256确保内容一致性
- 原子操作保证数据完整性
- 事务支持确保并发安全

### 性能优化
- 索引优化查询性能
- 批量操作减少I/O
- 内存缓存热点数据

### 安全考虑
- 缓存数据本地存储
- 不缓存敏感信息
- 支持完全禁用缓存模式

这个缓存系统将显著提升 RepoSense 在处理大量仓库时的性能，同时保持了灵活性和可控性。