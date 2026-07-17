# 部署与运维

## 构建

在可信构建机运行 `scripts/build.sh` 或 `scripts/build.ps1`。产物位于 `bin/`，并已包含 React SPA。

## 生产基线

### 制品清单

| 制品 | 说明 |
| --- | --- |
| `bin/tudns*` | 不可变发布产物 |
| `config.yaml` | 部署配置与主密钥 |
| `data/database.yaml` | 安装生成的数据库参数 |
| `data/install.lock` | 安装完成标记 |
| 数据库 | 业务事实源 |

### 部署步骤

1. 创建非 root 服务账号和持久化数据目录。
2. 将二进制与独立 `config.yaml` 放入部署目录。
3. 生成高熵 `security.secret_key`，限制 CORS Origin。
4. 通过 `TUDNS_CONFIG` 指向配置文件。
5. 在反向代理或负载均衡器终止 TLS。
6. 首次安装后备份 `config.yaml`、`data/database.yaml`、`data/install.lock` 和数据库。
7. 仅向可信网络开放数据库，Provider 凭据使用最小权限。

启动示例：

```bash
TUDNS_CONFIG=/etc/tudns/config.yaml /opt/tudns/tudns
```

Windows PowerShell：

```powershell
$env:TUDNS_CONFIG = "C:\ProgramData\TuDNS\config.yaml"
& "C:\Program Files\TuDNS\tudns.exe"
```

## 进程生命周期

进程在 SIGINT/SIGTERM 后给 HTTP 15 秒关闭时间，再关闭数据库连接。

HTTP Server 超时配置：
- 读取头超时：10 秒
- 读取超时：30 秒
- 写入超时：60 秒
- 空闲超时：120 秒

启动日志包含监听地址和安装状态。

## 平台注意事项

### Windows

- 使用 `scripts/build.ps1` 构建 `bin/tudns.exe`。
- PowerShell 执行策略可能阻止本地脚本，可在组织安全策略允许的范围内调用该脚本。
- `0600` 文件权限不等同于 NTFS ACL，需显式限制 `data/` 和配置文件权限。
- Go `-race` 需要 CGO 和可用 C 编译器；缺少编译器时普通测试仍可执行，但不能视为完成竞态验证。

### Linux 与 macOS

- `scripts/build.sh` 使用 POSIX `sh`。
- 首次从不保留可执行位的介质获取脚本时，可使用 `sh scripts/build.sh`。
- 建议以非 root 专用账号运行，并让该账号只写 `data/`。

### 数据库

- SQLite 适合单实例和轻量部署，代码启用 WAL。
- MySQL/PostgreSQL 需要预先准备空数据库和网络连接。
- 应通过私网或 TLS 访问远程数据库；仓库没有提供数据库服务器或 TLS 自动配置。

### DNS 与支付外部依赖

- DNS Provider 联调需要公网访问服务商 API 和最小权限测试凭据。
- 支付宝生产联调需要公网 HTTPS 通知地址。
- 仓库未提供 Dockerfile、Kubernetes 清单、Nginx 配置或证书自动化，这些需由部署环境单独管理。

## 健康检查

- `GET /healthz`：进程存活。
- `GET /readyz`：安装和数据库可用性。

建议就绪探针期望 HTTP 200 且 `ready:true`；未安装状态虽然返回 HTTP 200，但不能接收业务流量。

## 数据备份

- SQLite：在一致性备份工具或停机窗口中备份数据库主文件及 WAL 相关状态。
- MySQL/PostgreSQL：使用平台原生逻辑或物理备份工具，并定期演练恢复。
- 同时保留 `database.yaml`、`install.lock` 和主配置；任何备份都不得公开上传。

## 升级

1. 备份数据库和配置。
2. 在预发布环境用同类数据库运行新二进制；启动时 GORM 会执行 `AutoMigrate`。
3. 验证 `/healthz`、`/readyz`、登录、Zone 查询以及一组受控 DNS CRUD。
4. 停止旧进程，替换二进制并启动。
5. 检查日志和关键用户流程。

当前没有版本化 SQL migration 和自动回滚脚本。涉及模型变更时，不能假定降级二进制能回滚数据库结构。

## 回滚

- 纯前端或无模型变更：停止新版本，恢复旧二进制并验证。
- 包含模型变更：先恢复数据库备份，再恢复匹配的二进制和配置。
- Provider 操作会改变外部 DNS 状态；回滚应用不会自动撤销已成功提交给服务商的记录变更。

## 发布前验证

1. 运行 `go test ./... -count=1` 与 `go vet ./...`。
2. 前端执行 `npm ci && npm run build`。
3. 构建后启动新二进制，检查 `/healthz`、`/readyz`、登录、数据库读写和受控 Provider CRUD。
4. 支付验证只能在隔离商户/沙箱环境进行，不能用真实用户资金做烟雾测试。

## 监控限制

当前只有 Gin/GORM/标准库日志以及两个健康端点，未内置 Prometheus 指标、分布式追踪或结构化审计导出。部署方应至少采集进程日志、HTTP 状态、数据库可用性和 DNS API 错误率。
