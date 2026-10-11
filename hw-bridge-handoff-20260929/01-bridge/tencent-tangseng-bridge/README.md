# 腾讯 IM 与唐僧叨叨互通 Bridge

这是独立的消息互通服务。它只通过腾讯 IM REST/回调、唐僧叨叨业务 API、悟空 IM Webhook/PC 代理接口工作，不直接读写任何一侧的数据库。

## 本地启动

```powershell
$env:BRIDGE_SHARED_SECRET='replace-me'
go run ./cmd/bridge
```

默认使用内存存储，适合协议和转换联调。生产环境使用 `BRIDGE_STORE=mysql`、`BRIDGE_AUTO_MIGRATE=true`，并配置独立 MySQL、Redis、腾讯 SDKAppID/密钥及唐僧叨叨测试环境。Redis 不可用时会自动退回 MySQL outbox 扫描。

## 内部接口

- `POST /internal/v1/events/tencent`
- `POST /cb/tim`（PHP 原始腾讯回调入口，`X-HWB-Timestamp`/`X-HWB-Nonce`/`X-HWB-Sign`）
- `POST /internal/v1/events/tangseng`
- `POST /internal/v1/events/tangseng/webhook`（可直接接收悟空 IM `msg.notify` 数组）
- `GET /healthz`
- `GET /readyz`
- `GET /metrics`
- `POST /admin/dead-letter/{job_id}/retry`
- `POST /api/v1/account/reserve|confirm|cancel|profile`

迁移兼容接口使用 `X-Bridge-Signature: sha256=<HMAC-SHA256(body)>`；正式 `/cb/tim` 和账号接口使用
`HMAC-SHA256(timestamp + "\\n" + nonce + "\\n" + body)`。

## 当前首期范围

- 单聊和群聊文本消息。
- 自动影子账号映射。
- 事件幂等、outbox 重试、死信、来源标记和防回环。
- 已根据 `TangSengDaoDaoUltimateServer` 源码对齐首期接口：用户注册/登录 `/v1/user/usernameregister`、`/v1/user/usernamelogin`，建群 `/v1/group/create`，历史同步 `/v1/message/sync`，Webhook 消息为 `POST /v1/webhook/message/notify` 或带 `event=msg.notify` 的 `/v1/webhook`。
- 配置 `TANGSENG_WS_URL` 后，消息投递使用每个影子账号独立的悟空 JSON-RPC PC(2) 连接（connect/send/recv/recvack/ping）；未配置时保留 HTTP 代发作为联调后备路径。
- 唐僧叨叨代发要求服务账号 token；单聊还要求双方互为好友，群聊要求发送者已在群内。请配置 `TANGSENG_TOKEN`、`TANGSENG_SHADOW_OWNER_UID`，并按实际租户注册策略配置影子账号密码/邀请码。
- 注册接口会生成随机 UID，适配器会从注册响应读取真实 UID 并回写映射；如果服务端关闭用户名注册或要求邀请码，任务会进入重试/死信。

媒体转换、完整关系/群同步、撤回已读和 Tencent 漫游补偿仍是后续阶段；A+ 的同步客户端、主租约和契约测试骨架已经提供。

账号 provision 完成后会启动对应 TangSeng PC(2) 接收循环；APK 通过 PHP
`/v1/imaccess/bridge/config` 获取短期 UserSig，腾讯密钥只保留在服务端。
