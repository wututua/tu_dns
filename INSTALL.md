# 安装 TuDNS

## 环境要求

- Go 1.25（由 `go.mod` 声明）
- Node.js 22 和 npm（CI 使用 Node.js 22）
- 可选：MySQL 或 PostgreSQL；本地试用可直接使用 SQLite

## 源码开发

```bash
go mod download
go run ./cmd/server
```

另开终端：

```bash
cd frontend
npm ci
npm run dev
```

浏览器访问 <http://127.0.0.1:5173>。后端默认监听 `0.0.0.0:8080`。

## 单二进制构建

```bash
# Linux / macOS
./scripts/build.sh

# Windows PowerShell
./scripts/build.ps1
```

也可手动执行：

```bash
cd frontend
npm ci
npm run build
cd ..
go build -trimpath -o bin/tudns ./cmd/server
```

Vite 直接清空并写入 `internal/webembed/dist/`，因此必须先构建前端，再构建 Go 二进制。

## 首次安装

1. 从 `config.example.yaml` 创建 `config.yaml`。
2. 更换 `security.secret_key`。
3. 启动服务并访问根路径。
4. 选择 SQLite、MySQL 或 PostgreSQL，并测试连接。
5. 填写管理员用户名、密码、邮箱和站点名。
6. 安装成功后，程序在 `data/` 写入 `database.yaml` 与 `install.lock`。

SQLite 默认文件为 `data/tudns.db`，启用 WAL 和 5 秒 busy timeout。MySQL/PostgreSQL 应先创建空数据库并授予建表权限；安装流程拒绝包含现有用户的数据库。

## 验证

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/readyz
```

`/healthz` 表示 HTTP 进程存活；未安装时 `/readyz` 返回 HTTP 200 和 `ready:false`，数据库异常时返回 HTTP 503。

更多信息见 [部署](DEPLOY.md)、[配置](ENV_VARS.md) 和 [故障排查](docs/troubleshooting.md)。
