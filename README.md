<p align="center">
  <h1 align="center">TuDNS</h1>
  <p align="center">DNS 二级域名分发与解析管理系统</p>
  <p align="center">
    <a href="#快速开始"><img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go" alt="Go"></a>
    <a href="#快速开始"><img src="https://img.shields.io/badge/React-19-61DAFB?logo=react" alt="React"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="MIT License"></a>
    <a href="docs/install.md"><img src="https://img.shields.io/badge/status-active-brightgreen" alt="Status"></a>
  </p>
</p>

**TuDNS** 面向需要分发自有根域名下二级域名的运营者。管理员接入已有的公网权威 DNS Zone、设置可用记录类型和积分价格，用户可在一个流程中创建子域及首条解析记录。系统把用户、积分、域名商品、上游 DNS 和解析记录放在同一管理面中，不提供权威 DNS 服务本身，而是调用第三方 DNS API。

## 功能特性

- **开箱即用** — 首次启动安装向导，支持 SQLite / MySQL / PostgreSQL
- **JWT 认证** — 支持 `admin` / `user` / `premium` 角色
- **一次扣费** — 创建子域和首条 DNS 记录时只扣一次积分
- **按需付费** — 新增记录再扣费，修改免费，删除不退
- **完整管理后台** — 域名、用户、积分、兑换码、订单、系统设置
- **多 Provider** — 支持 10 个 DNS 服务商，Provider 注册机制可扩展
- **支付集成** — 支付宝 RSA2 接入框架，含开发模拟流程
- **单二进制交付** — React SPA 嵌入 Go，一个文件搞定部署

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.25, Gin, GORM |
| 数据库 | SQLite, MySQL, PostgreSQL |
| 前端 | React 19, TypeScript, Vite 6, HeroUI v3, Tailwind CSS 4 |
| 交付 | `go:embed` 单二进制, GitHub Actions |

## DNS Provider

| Key | 平台 | 状态 |
| --- | --- | --- |
| `aliyun` | 阿里云 DNS | ✅ 已实现 |
| `baidu` | 百度智能云 DNS | ✅ 已实现 |
| `cloudflare` | Cloudflare | ✅ 已实现 |
| `dnsla` | DNS.LA | ✅ 已实现 |
| `dnspod` | DNSPod | ✅ 已实现 |
| `huaweicloud` | 华为云 DNS | ✅ 已实现 |
| `jdcloud` | 京东云 DNS | ✅ 已实现 |
| `volcengine` | 火山引擎 DNS | ✅ 已实现 |
| `westcn` | 西部数码 | ✅ 已实现 |
| `xinnet` | 新网 | ✅ 已实现 |

详情见 [DNS Provider 文档](docs/dns-providers.md)。

## 快速开始

要求 Go 1.25 和 Node.js 22。

```bash
# 终端 1：后端
go run .

# 终端 2：前端
cd frontend
npm ci
npm run dev
```

访问 <http://127.0.0.1:5173>，按安装向导初始化数据库和管理员。

**生产构建：**

```bash
# Linux / macOS
./scripts/build.sh

# Windows PowerShell
.\scripts\build.ps1
```

完整步骤见 [安装](docs/install.md) 和 [部署](docs/deploy.md)。

## 项目结构

```
.
├── .github/              GitHub Actions、Issue 和 PR 模板
├── docs/                 架构与模块深度文档
├── frontend/             React SPA 源码
├── scripts/              跨平台构建脚本
│
├── config/               配置管理与 SettingsStore
├── crypto/               AES-GCM 加解密工具
├── db/                   数据库连接与迁移
├── dns/                  DNS Provider 接口与适配器
│   └── providers/        10 个服务商实现
├── models/               数据模型
├── server/               路由、中间件、HTTP 处理器
├── auth/                 JWT 签发与密码认证
├── domain/               域名管理
├── record/               解析记录管理
├── points/               积分账本
├── redeem/               兑换码
├── payment/alipay/       支付宝支付
├── admin/                管理后台
├── install/              安装向导
└── webembed/             go:embed 前端构建产物
```

## 配置

复制 `config.example.yaml` 为 `config.yaml`，生产环境必须替换 `security.secret_key` 并限制 `cors.allow_origins`。

```yaml
app:
  host: 0.0.0.0
  port: 8080
  mode: release
  data_dir: data
```

完整字段见 [配置文档](docs/env-vars.md)。

## 文档

| 类别 | 链接 |
| --- | --- |
| 入门 | [安装](docs/install.md) · [使用](docs/usage.md) · [API](docs/api.md) |
| 运维 | [部署与回滚](docs/deploy.md) · [配置与环境变量](docs/env-vars.md) |
| 参考 | [文档索引](docs/README.md) · [贡献指南](CONTRIBUTING.md) · [安全策略](SECURITY.md) |
| 其他 | [特殊环境说明](docs/special-env.md) · [变更记录](CHANGELOG.md) |

## 验证

```bash
go test ./... -count=1
go vet ./...

cd frontend
npm ci
npm run build
```

## 安全

不要提交 DNS API 密钥、支付宝私钥、数据库 DSN、JWT Token、`config.yaml` 或 `data/`。发现安全问题请按 [SECURITY.md](SECURITY.md) 私下报告。

## 许可

本项目采用 [MIT](LICENSE) 协议。提交变更前请阅读 [贡献指南](CONTRIBUTING.md)。
