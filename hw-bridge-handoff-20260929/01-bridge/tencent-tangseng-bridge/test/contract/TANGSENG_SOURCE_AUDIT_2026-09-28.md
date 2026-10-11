# 唐僧叨叨源码只读核对报告

日期：2026-09-28  
源码包：`E:/沪上小登工具箱/remove-shuiyin/input/20260630聚畅信后端代码.7z`

## 结论

压缩包可正常读取，包含唐僧叨叨业务服务端和悟空 IM 源码。源码能够确认 A+ Bridge 所需的 HTTP 路由、登录字段、同步接口和 JSON-RPC 消息结构；没有修改源码、配置或数据库。

源码不能确认线上当前实际域名、端口映射、租户、邀请码、管理 Token 和账号数据。这些仍需由叨叨研发提供线上部署值。

## 已从源码确认的内容

### 目录

- 业务服务：`service/TangSengDaoDaoUltimateServer`
- 悟空 IM：`WuKongIM`
- 部署样例：`deploy`

### 业务 HTTP

部署样例将唐僧叨叨业务服务暴露在 `8090`，健康检查为：

```text
GET /v1/ping
```

用户名注册和登录路由为：

```text
POST /v1/user/usernameregister
POST /v1/user/usernamelogin
```

用户名注册字段包括 `name`、`username`、`password`、`invite_code`、`flag`、`device` 和 `manufacturer`。登录字段包括 `username`、`password`、`flag` 和 `device`；登录返回数据包括 `uid`、`token` 等字段。

源码中确认的增量同步路由包括：

```text
POST /v1/conversation/sync
POST /v1/conversation/syncack
POST /v1/message/channel/sync
POST /v1/message/sync
POST /v1/message/syncack/{last_message_seq}
POST /v1/message/extra/sync
```

这些路由使用业务认证中间件，Bridge 需要以真实登录返回的 Token 和线上要求的请求头调用，不能仅凭公网地址推断调用格式。

### 悟空 IM WebSocket

悟空进程内部 WebSocket 监听端口是 `5200`。部署样例可以将它映射为外部 `15200`：

```text
外部 15200 -> 悟空容器 5200
```

集群配置还支持通过 `WK_EXTERNAL_WSADDR` 和 `WK_EXTERNAL_WSSADDR` 配置客户端实际连接地址。因此 `15200` 是一种部署映射，不是所有环境都固定使用的端口。

JSON-RPC 源码确认了以下结构：

```text
connect：uid、token、deviceId、deviceFlag
send：channelId、channelType、payload、clientMsgNo
recv：messageId、messageSeq、fromUid、channelId、channelType、payload
recvack：messageId、messageSeq
ping/pong
```

PC 代理应使用 `deviceFlag=2`，处理 `recv` 成功或明确忽略后再发送 `recvack`。消息 `payload` 是字节数组，实际传输时需要按 JSON-RPC 编码规则处理。

## 线上联调仍需叨叨研发提供

1. 当前生产/测试环境的业务 HTTP 地址和是否必须经过 HTTPS、网关路径。
2. 当前悟空 WebSocket 对外地址：`ws://` 或 `wss://`，以及实际端口或反向代理路径。
3. 业务认证请求头名称、Token 有效期和失效码。
4. 测试租户、邀请码、测试账号和允许的 `flag=2` 注册/登录规则。
5. WebSocket 是否允许 Bridge 所在网段访问，防火墙白名单和出口 IP 要求。
6. 当前线上部署是否采用 `15200 -> 5200` 映射，不能直接套用样例值。

## 对 Bridge 的影响

- Bridge 可以继续保持独立部署，不需要与叨叨服务同机；只要能通过专线/VPN/白名单访问业务 HTTP 和悟空 WebSocket。
- 叨叨服务端和悟空 IM 核心不需要改代码。
- 在拿到上述线上值前，只能完成源码契约核对和 Mock 测试，不能声称真实 PC(2) 已联通。
- 取得线上值后，先用健康检查、登录、WebSocket `connect`、`ping`、收发一条文本和 `recvack` 完成阶段 0，再进入真实账号映射和消息桥接。

## 源码定位

- HTTP 路由：`service/TangSengDaoDaoUltimateServer/libs/TangSengDaoDaoServer/modules/user/api.go`
- 用户登录字段：`service/TangSengDaoDaoUltimateServer/libs/TangSengDaoDaoServer/modules/user/api.go`
- 用户名注册/登录实现：`service/TangSengDaoDaoUltimateServer/libs/TangSengDaoDaoServer/modules/user/api_usernamelogin.go`
- 消息同步路由：`service/TangSengDaoDaoUltimateServer/libs/TangSengDaoDaoServer/modules/message/api.go`
- 会话同步路由：`service/TangSengDaoDaoUltimateServer/libs/TangSengDaoDaoServer/modules/message/api_conversation.go`
- JSON-RPC 类型：`WuKongIM/pkg/jsonrpc/types.go`
- JSON-RPC 编解码说明：`WuKongIM/pkg/jsonrpc/README_zh.md`
- 业务部署样例：`deploy/online/docker-compose.yaml`
- 集群外部地址样例：`deploy/cluster-config/cluster.env.example`
- 网关端口映射：`WuKongIM/docker/gateway/docker-compose.yaml`
