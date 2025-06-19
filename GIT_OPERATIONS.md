# Git操作安全性改进

## 问题描述

在批量更新大量Git仓库时，某些仓库可能会要求用户交互（如SSH密码输入），导致操作挂起数小时。

## 解决方案

### 1. 非交互模式（默认启用）

RepoSense现在默认使用非交互模式运行Git操作，防止因认证问题导致的挂起：

```bash
# 默认行为：非交互模式，快进合并
reposense update /path/to/repos

# 如果需要允许交互（不推荐）
reposense update /path/to/repos --git-allow-interactive
```

### 2. Git拉取策略选项

可以选择不同的Git拉取策略：

```bash
# 快进合并（默认，最安全）
reposense update /path/to/repos --git-pull-strategy ff-only

# 允许合并提交
reposense update /path/to/repos --git-pull-strategy merge

# 使用rebase
reposense update /path/to/repos --git-pull-strategy rebase
```

### 3. 环境变量设置

在非交互模式下，以下环境变量被自动设置：

- `GIT_TERMINAL_PROMPT=0` - 禁用终端提示
- `GIT_ASKPASS=echo` - 禁用密码提示  
- `SSH_ASKPASS=echo` - 禁用SSH密码提示
- `GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=no` - 非交互SSH

### 4. 改进的错误消息

现在提供更友好的错误消息：

- SSH认证失败
- 非快进更新警告
- 没有远程跟踪分支
- 连接超时
- 权限被拒绝

## 推荐配置

对于大规模仓库管理，推荐以下配置：

```bash
# 使用SSH密钥认证（无密码）
ssh-add ~/.ssh/id_rsa

# 配置Git使用缓存凭据
git config --global credential.helper cache

# 批量更新（推荐设置）
reposense update /path/to/repos \
    --git-pull-strategy ff-only \
    --timeout 30s \
    --workers 10
```

## SSH密钥管理

为避免SSH密码提示，建议：

1. 使用无密码的SSH密钥
2. 或使用ssh-agent加载密钥：
   ```bash
   ssh-add ~/.ssh/id_rsa
   ```
3. 或配置SSH config文件自动加载密钥

## 故障排除

如果仍有仓库更新失败：

1. 检查SSH密钥配置
2. 验证仓库访问权限
3. 确认网络连接
4. 检查仓库的远程URL配置
5. 考虑使用HTTPS而非SSH URL（对于公开仓库）

## 向后兼容性

- 默认行为已更改为更安全的非交互模式
- 如需旧的交互行为，使用 `--git-allow-interactive` 标志
- 所有现有命令行选项继续工作