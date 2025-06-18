# RepoSense - Git仓库批量管理工具

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

RepoSense 是一个高效的 Git 仓库批量管理工具，专为需要管理大量本地 Git 仓库的开发者设计。

## ✨ 特性

- 🔍 **智能扫描**: 自动发现指定目录下的所有 Git 仓库
- 🚀 **并行更新**: 使用工作池模式并行执行批量 `git pull` 操作
- 📊 **状态收集**: 获取仓库详细状态信息（分支、提交、工作区状态等）
- 📋 **仓库列表**: 列出仓库及其描述，自动从README提取项目描述
- 🤖 **LLM智能描述**: 支持OpenAI、Gemini、Claude、Ollama等LLM API智能生成项目描述
- 🌍 **多语言支持**: 支持中文、英文、日文的项目描述生成
- 🔤 **智能排序**: 支持按时间或字母排序，可正序/倒序显示
- 📈 **进度显示**: 实时显示更新进度和统计信息
- 🎯 **智能过滤**: 支持包含/排除模式过滤仓库
- 📄 **多种输出**: 支持文本、表格、JSON 三种输出格式
- 💾 **报告保存**: 可将结果保存为 JSON 报告文件
- 🧪 **模拟运行**: 支持 dry-run 模式预览操作

## 📦 安装

### 从源码构建

```bash
git clone <repository-url>
cd RepoSense
go mod tidy
go build -o reposense ./cmd/reposense
```

### 使用

```bash
# 将构建的二进制文件移动到 PATH 目录
sudo mv reposense /usr/local/bin/
```

## 🚀 快速开始

### 基本用法

```bash
# 扫描当前目录下的所有 Git 仓库
reposense scan

# 扫描指定目录
reposense scan /path/to/repositories

# 列出仓库及其描述（按字母排序）
reposense list

# 按更新时间倒序列出仓库
reposense list --sort-by-time --reverse

# 批量更新当前目录下的所有 Git 仓库
reposense update

# 查看仓库状态
reposense status

# 使用表格格式显示
reposense list --format table

# 使用 JSON 格式输出
reposense update --format json
```

### 高级用法

```bash
# 使用 20 个并发工作协程进行更新
reposense update --workers 20

# 设置超时时间为 60 秒
reposense update --timeout 60s

# 只显示包含 "golang" 的仓库
reposense list --include golang

# 排除包含 "test" 的仓库列表
reposense list --exclude test --sort-by-time

# 按时间排序并保存为表格格式
reposense list --sort-by-time --format table --save-report

# 使用LLM智能生成中文描述
export OPENAI_API_KEY=your_api_key  
reposense list --enable-llm --llm-language zh

# 使用Gemini生成英文描述
reposense list --enable-llm --llm-provider gemini --llm-api-key your_key --llm-language en

# 使用本地Ollama模型
reposense list --enable-llm --llm-provider ollama --llm-model llama3 --llm-base-url http://localhost:11434

# 模拟运行，不执行实际操作
reposense update --dry-run

# 保存报告到文件
reposense update --save-report --report-file update-report.json

# 显示详细输出（包含更新时间）
reposense list --verbose --sort-by-time
```

## 📋 命令参考

### 全局选项

| 选项 | 简写 | 默认值 | 描述 |
|------|------|--------|------|
| `--workers` | `-w` | 10 | 并发工作协程数量 (1-50) |
| `--timeout` | `-t` | 30s | 每个操作的超时时间 |
| `--format` | `-f` | text | 输出格式 (text/table/json) |
| `--verbose` | `-v` | false | 显示详细输出 |
| `--dry-run` | | false | 模拟运行，不执行实际操作 |
| `--include` | `-i` | | 包含模式 (可多次指定) |
| `--exclude` | `-e` | | 排除模式 (可多次指定) |
| `--save-report` | | false | 保存报告到文件 |
| `--report-file` | | | 报告文件路径 |

### LLM选项

| 选项 | 默认值 | 描述 |
|------|--------|------|
| `--enable-llm` | false | 启用LLM智能描述提取 |
| `--llm-provider` | openai | LLM提供商 (openai/openai-compatible/gemini/claude/ollama) |
| `--llm-model` | gpt-4o-mini | LLM模型名称 |
| `--llm-api-key` | | LLM API密钥 (可通过环境变量设置) |
| `--llm-base-url` | | LLM API基础URL |
| `--llm-language` | zh | 描述语言 (zh/en/ja) |
| `--llm-timeout` | 10s | LLM请求超时时间 |

#### 环境变量支持

- `OPENAI_API_KEY`: OpenAI API密钥
- `GEMINI_API_KEY`: Gemini API密钥  
- `CLAUDE_API_KEY`: Claude API密钥
- `LLM_API_KEY`: 通用LLM API密钥

### 子命令

#### `scan [directory]`
扫描指定目录下的所有 Git 仓库并显示列表。

```bash
reposense scan /home/user/projects --format table
```

#### `update [directory]`
批量更新指定目录下的所有 Git 仓库。

```bash
reposense update /home/user/projects --workers 15 --timeout 45s
```

#### `status [directory]`
查看指定目录下所有 Git 仓库的详细状态信息。

```bash
reposense status /home/user/projects --format json
```

#### `list [directory]`
列出指定目录下的所有 Git 仓库及其描述信息。

```bash
reposense list /home/user/projects --sort-by-time --reverse
reposense list --format table --include golang
```

## 🏗️ 架构设计

RepoSense 采用模块化设计，主要包含以下组件：

- **Scanner**: 仓库发现和扫描
- **Updater**: 批量 Git 操作管理
- **Reporter**: 进度显示和结果报告
- **StatusCollector**: 仓库状态收集

### 核心特性

- **工作池模式**: 使用 goroutine 池并行处理多个仓库
- **超时控制**: 每个 Git 操作都有独立的超时设置
- **错误处理**: 单个仓库失败不影响其他仓库的处理
- **进度追踪**: 实时显示处理进度和统计信息

## 🔧 配置

RepoSense 支持通过命令行参数进行配置，未来计划支持配置文件。

### 性能调优

- **并发数**: 根据机器性能和网络状况调整 `--workers` 参数
- **超时时间**: 根据网络环境调整 `--timeout` 参数
- **过滤模式**: 使用 `--include` 和 `--exclude` 减少处理的仓库数量

## 📊 输出格式

### 文本格式 (默认)
```
更新结果 (3个仓库):
--------------------------------------------------------------------------------
✓ project1: 已是最新版本 (耗时: 1.2s)
✓ project2: 快进更新成功 (耗时: 2.1s)
✗ project3: 更新失败: network timeout
```

### 表格格式
```
序号   仓库名称           状态     耗时      消息
----------------------------------------
1    project1         成功     1.20s    已是最新版本
2    project2         成功     2.10s    快进更新成功
3    project3         失败     30.00s   network timeout
```

### JSON 格式
```json
{
  "update_results": [
    {
      "repository": {
        "path": "/path/to/project1",
        "name": "project1",
        "is_git_repo": true
      },
      "success": true,
      "message": "已是最新版本",
      "duration": 1200000000,
      "start_time": "2023-12-01T10:00:00Z",
      "end_time": "2023-12-01T10:00:01Z"
    }
  ],
  "total": 3,
  "timestamp": "2023-12-01T10:00:01Z"
}
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 🎯 使用场景

RepoSense 特别适合以下场景：

- 🎓 **学习研究**: 管理大量克隆的开源项目
- 💼 **企业开发**: 维护多个项目仓库
- 🔧 **DevOps**: 批量更新部署相关仓库
- 🏗️ **代码审查**: 快速同步多个待审查项目

## 🛣️ 路线图

- [ ] 配置文件支持
- [ ] GUI 界面
- [ ] 更多 Git 操作支持 (fetch, status, branch)
- [ ] 插件系统
- [ ] 性能监控和分析
- [ ] AI 增强功能 (代码搜索、项目分析)

---

如果 RepoSense 对你有帮助，请给个 ⭐️ 支持一下！