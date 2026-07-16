# TuDNS HTTP API

## 约定

默认地址为 `http://127.0.0.1:8080`。除健康检查和支付宝通知外，JSON 接口统一返回：

```json
{"code":0,"message":"ok","data":{}}
```

失败响应使用非零 `code` 和 `message`。受保护接口要求 `Authorization: Bearer <token>`；`/api/admin/*` 还要求 `admin` 角色。当前 API 无版本前缀、速率限制和幂等键协议。

## 健康与安装

| 方法 | 路径 | 权限 | 请求 |
| --- | --- | --- | --- |
| GET | `/healthz` | 公开 | 无 |
| GET | `/readyz` | 公开 | 无 |
| GET | `/api/install/status` | 公开 | 无 |
| POST | `/api/install/test-db` | 未安装 | `driver`, `dsn`, `sqlite_path` |
| POST | `/api/install` | 未安装 | `driver`, `dsn`, `sqlite_path`, `admin_user`, `admin_pass`, `admin_email`, `site_name` |

安装请求示例：

```json
{
  "driver": "sqlite",
  "sqlite_path": "tudns.db",
  "admin_user": "admin",
  "admin_pass": "replace-with-strong-password",
  "admin_email": "admin@example.com",
  "site_name": "TuDNS"
}
```

## 认证与公开资源

| 方法 | 路径 | 权限 | 请求/结果 |
| --- | --- | --- | --- |
| POST | `/api/auth/register` | 公开、已安装 | `username`, `password`, `email`；返回 token 与 user |
| POST | `/api/auth/login` | 公开、已安装 | `username`, `password`；返回 token 与 user |
| GET | `/api/public/domains` | 公开、已安装 | 已上架域名 |
| GET | `/api/dns/providers` | 公开、已安装 | Provider 及配置字段 |
| GET | `/api/auth/me` | Bearer | 当前用户 |
| PUT | `/api/auth/password` | Bearer | `old_password`, `new_password` |

注册要求用户名至少 3 位、密码至少 6 位。客户端示例：

```bash
curl -X POST http://127.0.0.1:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"your-password"}'
```

## 子域与解析记录

| 方法 | 路径 | 权限 | 请求/说明 |
| --- | --- | --- | --- |
| GET | `/api/subdomains` | Bearer | 当前用户子域 |
| POST | `/api/subdomains/bundle` | Bearer | `domain_id`, `subdomain_name`, `type`, `value`, `ttl`, `line` |
| DELETE | `/api/subdomains/:id` | Bearer | 删除子域及上游记录，不退款 |
| GET | `/api/records` | Bearer | 当前用户记录 |
| POST | `/api/records` | Bearer | `subdomain_id`, `type`, `value`, `ttl`, `line`；扣费 |
| PUT | `/api/records/:id` | Bearer | `type`, `value`, `ttl`, `line`；不扣费 |
| DELETE | `/api/records/:id` | Bearer | 删除记录，不退款 |

创建子域与首条记录：

```json
{
  "domain_id": 1,
  "subdomain_name": "demo",
  "type": "A",
  "value": "192.0.2.10",
  "ttl": 600,
  "line": ""
}
```

后端支持校验 `A`、`AAAA`、`CNAME`、`TXT`、`MX`、`NS`。实际可选类型还受域名的 `record_types` 限制。

## 积分、兑换码与支付

| 方法 | 路径 | 权限 | 请求/说明 |
| --- | --- | --- | --- |
| GET | `/api/points?page=1` | Bearer | 每页固定 20 条 |
| POST | `/api/redeem` | Bearer | `code` |
| POST | `/api/pay/alipay/create` | Bearer | `amount`，最低 0.01 元 |
| GET | `/api/pay/orders` | Bearer | 当前用户订单，最多 100 条 |
| GET | `/api/pay/orders/:out_trade_no` | Bearer | 当前用户指定订单 |
| POST | `/api/pay/alipay/notify` | 公开、已安装 | 支付宝表单通知，成功返回文本 `success` |
| POST | `/api/pay/alipay/mock` | 公开、已安装 | `out_trade_no`，仅开发模拟 |

模拟支付端点不受 Bearer 保护，但生产配置启用私钥后会拒绝模拟。部署方必须避免在生产中暴露未验证的模拟支付路径。

## 管理接口

以下接口均要求 Bearer Token 和 `admin` 角色。

| 方法 | 路径 | 请求/说明 |
| --- | --- | --- |
| GET | `/api/admin/users?page=1` | 用户列表，每页 20 条 |
| PUT | `/api/admin/users/:id` | 更新用户字段 |
| POST | `/api/admin/users/:id/points` | `delta`, `remark` |
| POST | `/api/admin/users/:id/password` | `password` |
| GET | `/api/admin/domains` | 全部域名 |
| POST | `/api/admin/domains` | 新建域名 |
| PUT | `/api/admin/domains/:id` | 更新域名 |
| DELETE | `/api/admin/domains/:id` | 删除域名 |
| POST | `/api/admin/dns/check` | `provider_key`, `config` |
| POST | `/api/admin/dns/zones` | `provider_key`, `config` |
| GET | `/api/admin/subdomains` | 全部子域 |
| GET | `/api/admin/records` | 全部记录 |
| GET | `/api/admin/points?page=1` | 全部积分流水 |
| GET | `/api/admin/logs?page=1` | 操作日志 |
| GET | `/api/admin/redeem` | 兑换码列表 |
| POST | `/api/admin/redeem` | `points`, `max_uses`, `expires_at` (RFC3339) |
| GET | `/api/admin/settings` | 系统设置 |
| PUT | `/api/admin/settings` | 任意 string-to-string JSON map |
| GET | `/api/admin/pay/alipay/config` | 支付配置，密钥字段返回 `***` |
| PUT | `/api/admin/pay/alipay/config` | 支付宝配置对象 |
| GET | `/api/admin/pay/orders` | 全部订单，最多 100 条 |

域名保存请求：

```json
{
  "name": "example.com",
  "provider_key": "cloudflare",
  "remote_zone_id": "provider-zone-id",
  "config": {"api_token": "redacted"},
  "record_types": "A,AAAA,CNAME,TXT",
  "points_cost": 10,
  "description": "Example zone",
  "status": 1
}
```

Provider 的配置字段由 `GET /api/dns/providers` 动态返回，应优先按该结果构造表单。

## 状态码

| HTTP | 语义 |
| --- | --- |
| 200 | 成功；部分业务错误以其他状态返回 |
| 204 | CORS 预检 |
| 400 | 参数或业务规则错误 |
| 401 | 未登录或 Token 无效 |
| 403 | 用户禁用或权限不足 |
| 404 | 资源/API 不存在 |
| 500 | 服务端错误 |
| 503 | 未安装或数据库不可用 |

路由和请求结构的权威来源是 `internal/server/router.go` 及对应 service 输入类型；此文档不替代生成式 OpenAPI 合约，当前仓库尚未提供 OpenAPI 文件。
