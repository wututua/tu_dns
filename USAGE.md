# 使用 TuDNS

## 管理员流程

1. 完成首次安装并登录管理员账号。
2. 在域名管理中选择 DNS Provider，填写凭据并检测连接。
3. 拉取 Zone，配置根域名、允许的记录类型、积分价格和状态。
4. 在用户管理中调整用户状态、角色、密码或积分。
5. 按需创建兑换码并配置支付宝参数。

DNS Provider 凭据会使用 `security.secret_key` 加密后存入数据库。更换该密钥会导致已有 Provider 配置无法解密，因此轮换前必须规划迁移。

## 用户流程

- 注册或登录后，在域名页面选择可用根域名。
- 输入子域前缀和首条记录；提交后一次性扣除该域名价格。
- 对已有子域新增记录时再次按域名价格扣费。
- 修改记录不扣费；删除记录或子域不退积分。
- 用户只能操作自己的子域和记录，管理员可跨用户操作。

支持的业务记录类型由管理员对每个域名配置，后端校验实现覆盖 `A`、`AAAA`、`CNAME`、`TXT`、`MX`、`NS`。

## 积分来源

- 管理员手动调整
- 兑换码
- 支付宝订单

未配置支付宝 App ID/私钥时，下单返回开发模拟地址；真实生产支付必须配置 RSA2 密钥、回调地址并完成支付宝平台联调。不要在生产环境依赖模拟支付。

## 常用命令

```bash
go run ./cmd/server                  # 启动后端
go test ./... -count=1              # 后端测试
go vet ./...                        # Go 静态检查
cd frontend && npm run dev          # 前端开发
cd frontend && npm run build        # 写入嵌入资源目录
./scripts/build.sh                  # Unix 单二进制构建
./scripts/build.ps1                 # Windows 单二进制构建
```

API 调用方式见 [API.md](API.md)，Provider 验证要求见 [docs/dns-providers.md](docs/dns-providers.md)。
