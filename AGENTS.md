# TuDNS

DNS 二级域名分发与解析管理系统。

## Stack

- Backend: Go, Gin, GORM (SQLite / MySQL / PostgreSQL)
- Frontend: React 19, Vite, TypeScript, HeroUI v3, Tailwind CSS v4
- Delivery: single binary with embedded SPA

## Product rules

- Install wizard chooses database driver on first run
- User creates subdomain and first DNS record in one step; charge once
- Additional new records charge per domain price; update free; delete free (no refund)
- No review workflow
- Points sources: admin adjust, redeem code, Alipay official (framework)
- MVP DNS providers: Cloudflare, DNSPod, Aliyun

## Commands

```bash
# backend
go test ./
go run .

# frontend
cd frontend && npm install && npm run dev
cd frontend && npm run build
```

## Layout

后端 Go 代码位于仓库根目录，遵循标准 Go 布局，不再使用 `cmd/` 与 `internal/` 嵌套：

- `main.go` 进程入口
- 根目录下的包（`config`、`db`、`dns`、`domain`、`record`、`auth`、`server` 等）为业务模块
- `migrations` SQL migrations
- `frontend` React SPA；生产构建写入 `webembed/dist`
- `docs` 项目文档
- `scripts` 本地构建辅助脚本
- `bin/` 构建产物（已忽略）
