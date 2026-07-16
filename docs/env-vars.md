# 配置与环境变量

## 环境变量

| 名称 | 默认值 | 必需 | 敏感 | 说明 |
| --- | --- | --- | --- | --- |
| `TUDNS_CONFIG` | `config.yaml` | 否 | 否 | 指定主配置文件路径 |

当前代码没有把单个 YAML 字段绑定到环境变量。容器或进程管理器应挂载配置文件，再通过 `TUDNS_CONFIG` 指向它。

## YAML 配置

| Key | 类型 | 默认值 | 生产要求 |
| --- | --- | --- | --- |
| `app.name` | string | `tudns` | 可选 |
| `app.host` | string | `0.0.0.0` | 按网络边界设置 |
| `app.port` | integer | `8080` | 非零端口 |
| `app.mode` | string | `release` | `dev` 会增加 Gin/GORM 日志 |
| `app.data_dir` | string | `data` | 需持久化并限制访问 |
| `security.secret_key` | string | 开发占位值 | 必须更换并安全保管 |
| `security.token_ttl_hours` | integer | `72` | 按会话策略调整 |
| `security.install_token` | string | 空 | 当前配置结构保留，安装路由未使用 |
| `cors.allow_origins` | string[] | `['*']` | 生产应限制可信 Origin |

数据库选择由安装向导写入 `data/database.yaml`：

| Key | 说明 |
| --- | --- |
| `database.driver` | `sqlite`、`mysql`、`postgres`/`postgresql` |
| `database.dsn` | MySQL/PostgreSQL 连接字符串，包含敏感信息 |
| `database.path` | SQLite 文件名或路径 |

`database.yaml` 权限按 `0600` 写入。Windows 不完全遵循 POSIX 权限位，仍需使用 NTFS ACL 和专用服务账号保护目录。

## 密钥管理

- 不提交 `config.yaml`、`data/database.yaml` 或 Provider 凭据。
- `security.secret_key` 同时用于 JWT 和 DNS Provider 配置加密，丢失后无法恢复已有密文。
- 支付宝密钥保存在数据库设置表中；管理 API 返回时以 `***` 遮蔽，但数据库本身仍必须限制访问。
- 示例文件中的值仅为占位符，不能用于生产。
