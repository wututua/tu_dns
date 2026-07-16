# 安全策略

## 报告漏洞

请通过仓库维护者提供的私密安全报告渠道提交漏洞。若 GitHub Security Advisories 已启用，优先使用私密报告；不要公开创建包含利用步骤、密钥、Token、DSN 或用户数据的 Issue。

报告应包含受影响版本、影响、最小复现、建议修复和已采取的临时缓解措施。请对凭据和个人数据做脱敏。

## 部署责任

- 替换默认 `security.secret_key`。
- 限制 `cors.allow_origins`，在可信反向代理后启用 HTTPS。
- 使用最小权限 DNS Provider 凭据和数据库账号。
- 保护 `config.yaml`、`data/`、数据库备份和支付宝私钥。
- 禁止把开发模拟支付暴露为生产充值路径。
- 在真实服务商测试 Zone 上完成 Provider CRUD 联调后再上线。

## 当前安全边界

项目实现了 bcrypt 密码哈希、JWT Bearer 认证、管理员中间件、Provider 配置加密、基础安全响应头和参数化 GORM 查询。当前尚未内置速率限制、CSRF 专项防护、Prometheus 指标、集中式密钥管理或完整安全审计流水线，生产部署需补充外围控制。

详见 [docs/security.md](docs/security.md)。
