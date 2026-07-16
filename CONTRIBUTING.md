# 贡献指南

## 开始

1. 创建功能分支。
2. 安装 Go 1.25 与 Node.js 22。
3. 使用 `npm ci` 保持前端锁文件一致。
4. 不要提交真实配置、运行数据库、密钥或生成二进制。

## 目录约定

- Go 可执行入口放在 `cmd/`。
- 非公开 Go 包放在 `internal/`。
- React 源码放在 `frontend/`。
- 构建与维护脚本放在 `scripts/`。
- 深度文档放在 `docs/`，行为变化应同步更新根目录入口文档。

## 提交前验证

```bash
gofmt -w ./cmd ./internal

cd frontend
npm ci
npm run build
```

DNS Provider 变更还应使用独立测试 Zone 和最小权限凭据执行连接、Zone 列表、创建、更新、删除流程，并在 PR 中明确测试平台与未覆盖范围。不要在测试中操作生产 Zone。

## Pull Request

- 解释问题、方案和兼容性影响。
- API、配置、数据库模型或目录变化必须更新文档。
- 新依赖需说明用途和许可证。
- 不要将自动生成的 `internal/webembed/dist` 产物作为源码提交。

当前仓库尚未声明开源许可证；贡献者应在提交前确认其代码授权方式符合仓库所有者要求。
