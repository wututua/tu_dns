# TuDNS

TuDNS 是一个使用 Go 和 React 构建的 DNS 二级域名分发与解析管理系统。管理员接入已有的公网权威 DNS Zone、设置可用记录类型和积分价格，用户可在一个流程中创建子域及首条解析记录。

> 项目当前处于开发阶段。代码可构建并通过离线测试，但 DNS Provider 与支付宝功能在用于生产环境前仍需使用真实、最小权限凭据完成端到端验证。

## 功能

- 首次启动安装向导，支持 SQLite、MySQL 和 PostgreSQL
- JWT 登录认证，支持 `admin`、`user`、`premium` 角色
- 创建子域和首条 DNS 记录时只扣一次积分
- 同一子域新增记录按域名价格扣费，修改免费，删除不退款
- 管理员维护域名、用户、积分、兑换码、订单和系统设置
- DNS Provider 注册机制，支持 Zone 查询和记录增删改
- 支付宝 RSA2 接入框架和未配置密钥时的开发模拟流程
- React SPA 嵌入 Go，生产交付为单二进制

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.25、Gin、GORM |
| 数据库 | SQLite、MySQL、PostgreSQL |
| 前端 | React 19、TypeScript、Vite、HeroUI、Tailwind CSS 4 |
| 交付 | `go:embed` 单二进制、GitHub Actions |

## DNS Provider

| Key | 平台 | 实现状态 | 真实凭据联调 |
| --- | --- | --- | --- |
| `aliyun` | 阿里云 DNS | 已实现 | 待验证 |
| `baidu` | 百度智能云 DNS | 官方 SDK、离线测试 | 待验证 |
| `cloudflare` | Cloudflare | 已实现 | 待验证 |
| `dnsla` | DNS.LA | 已实现 | 待验证 |
| `dnspod` | DNSPod | 已实现 | 待验证 |
| `huaweicloud` | 华为云 DNS | 已实现 | 待验证 |
| `jdcloud` | 京东云 DNS | 已实现 | 待验证 |
| `volcengine` | 火山引擎 DNS | 官方 SDK、离线测试 | 待验证 |
| `westcn` | 西部数码 | 已实现 | 待验证 |
| `xinnet` | 新网 | 已实现 | 待验证 |

“已实现”仅表示适配器代码存在并能参与编译，不代表已通过服务商生产 API 验证。详情见 [DNS Provider 文档](docs/dns-providers.md)。

## 快速开始

要求 Go 1.25 和 Node.js 22。前端开发服务器会将 `/api` 和 `/healthz` 代理到 `127.0.0.1:8080`。

```bash
# 终端 1：后端
go run ./cmd/server

# 终端 2：前端
cd frontend
npm ci
npm run dev
```

访问 <http://127.0.0.1:5173>，按安装向导初始化数据库和管理员。

生产构建：

```bash
# Linux / macOS
./scripts/build.sh

# Windows PowerShell
./scripts/build.ps1
```

脚本先把前端构建到 `internal/webembed/dist/`，再输出 `bin/tudns` 或 `bin/tudns.exe`。完整步骤见 [安装文档](docs/install.md) 和 [部署文档](docs/deploy.md)。

## 配置

复制 `config.example.yaml` 为 `config.yaml`，生产环境必须替换 `security.secret_key`，并限制 `cors.allow_origins`。目前程序只读取一个环境变量：`TUDNS_CONFIG`，用于指定配置文件路径。

```yaml
app:
  host: 0.0.0.0
  port: 8080
  mode: release
  data_dir: data
```

完整字段见 [配置文档](docs/env-vars.md)。真实配置、运行数据和数据库文件已由 `.gitignore` 排除。

## 项目结构

```text
.
├── .github/               GitHub Actions、Issue 和 PR 模板
├── cmd/server/            服务进程入口
├── docs/                  架构与模块深度文档
├── frontend/              React SPA 源码
├── internal/              Go 私有业务包
│   ├── dns/providers/     DNS Provider 适配器
│   └── webembed/dist/     前端嵌入产物（生成目录）
├── scripts/               跨平台构建脚本
├── config.example.yaml    配置示例
├── go.mod                 Go 模块定义
└── README.md              项目入口
```

该布局遵循 Go 项目常用的 `cmd/`、`internal/` 约定，并使用 GitHub 常见的 `frontend/`、`docs/`、`scripts/`、`.github/` 分区。

## 文档

- [安装](docs/install.md)
- [使用](docs/usage.md)
- [API](docs/api.md)
- [部署与回滚](docs/deploy.md)
- [配置与环境变量](docs/env-vars.md)
- [特殊环境说明](docs/special-env.md)
- [文档索引](docs/README.md)
- [贡献指南](CONTRIBUTING.md)
- [安全策略](SECURITY.md)
- [变更记录](CHANGELOG.md)

## 验证

```bash
go test ./... -count=1
go vet ./...

cd frontend
npm ci
npm run build
```

CI 会分别验证 Go、前端以及包含嵌入资源的最终二进制构建。

## 安全

不要提交 DNS API 密钥、支付宝私钥、数据库 DSN、JWT Token、`config.yaml` 或 `data/`。发现安全问题时请按 [SECURITY.md](SECURITY.md) 私下报告，不要在公开 Issue 中披露凭据或漏洞利用细节。

## 贡献与许可

提交变更前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。当前仓库尚未声明开源许可证；在许可证文件加入前，不应推定代码可按任意开源许可证使用或再分发。
