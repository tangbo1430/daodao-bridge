# A+ 阶段 0：真实协议核对清单

此清单是进入账号、代理和生产代码联调的门槛。`UNKNOWN` 表示没有来自真实环境的可复核样本，不能用猜测值代替。样本必须脱敏后放在本目录，并在 `evidence` 列填写来源和采集时间。

设计文档已经给出了待验证的候选路径（例如 `master:8090`、`master:15200`
以及若干 `/v1/...` 接口），但候选路径不等于真实环境证据；成功码、反代
Host、租户和限频仍必须用叨叨测试环境确认。

| 编号 | 需要确认的事实 | 状态 | 证据 |
| --- | --- | --- | --- |
| V1 | 叨叨用户名注册 `flag=2` 请求字段、成功码、UID 返回位置 | UNKNOWN | |
| V2 | 叨叨用户名登录、Token 续期和失效错误码 | UNKNOWN | |
| V3 | 悟空 WebSocket 地址、TLS/反代路径、`connect` 认证字段 | UNKNOWN | |
| V4 | 悟空 `recv`、`recvack`、`send`、`ping` 的完整 JSON-RPC 帧和回执 | UNKNOWN | |
| V5 | 叨叨会话/频道/消息增量同步请求、响应和游标字段 | UNKNOWN | |
| V6 | 腾讯 13 类回调的实际请求体、查询参数和签名校验 | UNKNOWN | |
| V7 | 腾讯单聊/群聊/好友/撤回/已读/漫游 REST 的成功码和限频 | UNKNOWN | |
| V8 | 腾讯回调 `ForbidCallbackControl`、来源标记和回环处理方式 | UNKNOWN | |
| V9 | 腾讯好友申请 `ForceAddFlags` 以及影子账号黑名单行为 | UNKNOWN | |
| V10 | 语音/图片/文件上传格式、大小限制和 URL 有效期 | UNKNOWN | |
| V11 | PHP-FPM 是否支持 `fastcgi_finish_request`，以及 Redis 重试执行方式 | UNKNOWN | |
| V12 | 叨叨租户、邀请码、限频、Bridge 到 `master:8090/15200` 的内网/VPN 路由 | UNKNOWN | |

## 阶段出口

只有 V1–V12 全部变为 `CONFIRMED`，并且完成以下两项，才允许修改 PHP/APK 的生产行为：

1. `master:8090` 和 `master:15200` 从 Bridge 所在网络可访问；
2. 叨叨服务端和悟空 IM 上线前后的代码、配置校验和一致。

公网 HTTP 只能用于阶段 0。没有可用的悟空 WebSocket 时，不能把 HTTP 轮询冒充 A+ PC(2) 实时链路。

