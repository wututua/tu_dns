# 特殊环境说明

## Windows

- 使用 `scripts/build.ps1` 构建 `bin/tudns.exe`。
- PowerShell 执行策略可能阻止本地脚本，可在组织安全策略允许的范围内调用该脚本。
- `0600` 文件权限不等同于 NTFS ACL，需显式限制 `data/` 和配置文件权限。
- Go `-race` 需要 CGO 和可用 C 编译器；缺少编译器时普通测试仍可执行，但不能视为完成竞态验证。

## Linux 与 macOS

- `scripts/build.sh` 使用 POSIX `sh`。
- 首次从不保留可执行位的介质获取脚本时，可使用 `sh scripts/build.sh`。
- 建议以非 root 专用账号运行，并让该账号只写 `data/`。

## 数据库

- SQLite 适合单实例和轻量部署，代码启用 WAL。
- MySQL/PostgreSQL 需要预先准备空数据库和网络连接。
- 应通过私网或 TLS 访问远程数据库；仓库没有提供数据库服务器或 TLS 自动配置。

## DNS 与支付外部依赖

- DNS Provider 联调需要公网访问服务商 API 和最小权限测试凭据。
- 支付宝生产联调需要公网 HTTPS 通知地址。
- 仓库未提供 Dockerfile、Kubernetes 清单、Nginx 配置或证书自动化，这些需由部署环境单独管理。
