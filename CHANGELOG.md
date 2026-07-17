# Changelog

本文件记录面向使用者的重要变化。项目尚未发布稳定版本。

## Unreleased

### Added

- SQLite、MySQL、PostgreSQL 首次安装向导。
- 用户认证、角色、积分、兑换码与支付宝接入框架。
- 子域与首条解析一体化创建及积分计费。
- 10 个 DNS Provider 注册适配器。
- React 管理与用户界面，嵌入 Go 单二进制。
- GitHub Actions、Issue/PR 模板、跨平台构建脚本和完整项目文档。

### Changed

- 前端源码从 `web/` 迁移到常见的 `frontend/` 目录。
- Vite 构建直接输出到 `webembed/dist/`，移除手工复制步骤。

- 仓库采用 flat Go 布局，配合 `frontend/`、`docs/`、`scripts/`、`.github/` 分区。
### Known limitations

- Provider 仍需真实凭据端到端联调。
- 支付宝生产流程仍需真实商户环境验证。
- 尚未提供版本化数据库 migration、容器镜像或正式发布包。
- 采用 MIT 开源协议。
