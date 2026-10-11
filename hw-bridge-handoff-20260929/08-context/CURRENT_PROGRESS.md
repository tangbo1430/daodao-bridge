> 2026-10-01 管理员提示修复已部署：双入口角色及通知接口复验通过，重复同步无重复/回流；手机显示待确认。见 [GROUP_ROLE_NOTICE_FIX_20261001.md](GROUP_ROLE_NOTICE_FIX_20261001.md)。

> 最新 iOS 测试反馈：[IOS_FEEDBACK_20261001.md](IOS_FEEDBACK_20261001.md)。头像上传后的完整资料刷新补强已部署，重复执行及现有头像保持验证通过；iOS 界面待确认。图片偶发失败、踢人提示、撤回后新消息计数尚未关闭，不能标记本轮验收通过。

# 华为端 ↔ 唐僧叨叨互通交接包

> 送测前 review：[PRETEST_REVIEW_20261001.md](PRETEST_REVIEW_20261001.md)。修复跨系统单聊顶部通话入口漏拦截、上游无效 HTTP 200 误当成功、APK 建群返回标识缺失检查及内部异常文案。PHP 已部署，新 QA APK 已替换下载；核心接口和临时群真实回归通过。可进入测试阶段，新增 APK 入口仍需手机确认，不能据此宣称生产验收完成。

> 最新建群修复：[APK_GROUP_CREATE_FIX_20261001.md](APK_GROUP_CREATE_FIX_20261001.md)。旧 APK `/api/group_v/create_group` 漏接混合群逻辑，已修复 PHP 并部署；实际公网表单建群、重复请求、叨叨客户端群出现、双向群文本均验证通过。当前 APK 无需更新。先前数字群不自动迁移，手机重新建群仍需用户复验。

> 最新验收修复：[TIP_FILE_FIX_20261001.md](TIP_FILE_FIX_20261001.md)。修正系统提示误用普通消息容器；修复 Office MIME 超出媒体缓存字段长度导致文件发送失败。Bridge 已部署，两个真实文件补投并通过腾讯漫游、公网下载大小及 SHA256 校验。新 APK 已替换原下载地址；样式和文件打开仍待手机确认。

> APK 搜索验收发现并修复发布混淆破坏反射字段的问题，见 [SEARCH_APK_FIX_20261001.md](SEARCH_APK_FIX_20261001.md)。新 APK 需通过实际 R8 字段保留检查；Bridge/PHP 仍沿用产品文案修正版。

> 产品验收修正见 [PRODUCT_UX_FIX_20261001.md](PRODUCT_UX_FIX_20261001.md)：用户邀请码选填，内部接入凭据仅由服务器处理；清理 APK 和接口提示中的内部术语。该规则覆盖此前“叨叨邀请码必填”的说明。旧异常引用按用户要求忽略。

> 当前版本以 [BUSINESS_CLOSURE_20261001.md](BUSINESS_CLOSURE_20261001.md) 为准：五阶段业务收尾代码已部署、QA APK 已发布、严格新注册已启用；专用注册/群主注销/失败提示有真实后端证据。手机与部分原生协议场景仍待验收。集中清单见 [APK_CLOSURE_ACCEPTANCE_20261001.md](APK_CLOSURE_ACCEPTANCE_20261001.md)。以下“最新”字样均为历史记录。

> 最新业务验收及现网版本：[BUSINESS_ACCEPTANCE_20260930.md](BUSINESS_ACCEPTANCE_20260930.md)。新增群角色/禁言/头像/成员接口验收；修复好友资料丢头像和昵称串号。完整三端验收仍未结束。

> 最新现网版本及反向已读真实验证：[READ_RECEIPT_20260930.md](READ_RECEIPT_20260930.md)。R3 已确认腾讯 IsPeerRead=1；APK 显示待验收。以下旧报告保留历史记录。

> 最新客户端验收及部署：[CLIENT_ACCEPTANCE_20260930.md](CLIENT_ACCEPTANCE_20260930.md)。真实叨叨单聊/群撤回、单向已读、未读清零已核对；昵称恢复漏同步已修复并部署。当前二进制以此报告为准，历史接口异常和反向已读开启场景仍未完成。

> 最新部署：2026-09-30 [消息状态与群生命周期收尾](CORE_STATE_GROUP_20260930.md)。修复无 param 的状态指令及重复解散，已真实验证并部署；包含下述头像和群提示修复。当前版本、累计源码包及未验收项以此文为准。

> 最新：2026-09-30 头像同步修复已部署，见 [AVATAR_FIX_20260930.md](AVATAR_FIX_20260930.md)。已验证腾讯头像和叨叨原图 SHA256 一致。用户反馈多数业务可用，大视频后置；具体核心待办以该文档为准。

> 2026-09-30 增量热修复：群同步重复下发禁言、角色及资料状态，造成每分钟系统提示。已修复部署，详见 [GROUP_SYNC_FIX_20260930.md](GROUP_SYNC_FIX_20260930.md)。已纳入上述最新版本，原业务发布包需要更新 Bridge。

> 最新开发交付见 [BUSINESS_DELIVERY_20260929.md](BUSINESS_DELIVERY_20260929.md)。务必先阅读 [单条引用测试记录待处理说明](REPLY_CONTRACT_INCIDENT_20260929.md)。下文保留原交接状态，不代表最新完整验收结果。

生成日期：2026-09-29

这个目录是给新的 Agent 继续维护和部署使用的工作包。实施基线以 `05-docs-and-contracts/华为端与唐僧叨叨互通_详细开发实施方案.docx` 和同目录网络对接说明为准。叨叨业务服务端和悟空 IM 核心必须保持代码、配置零修改。

## 项目角色

- `02-authoritative-qch-source/企才汇源码交付-20260810/Android/qch`：华为端 Android App，包名 `com.qchm.xyz`，腾讯 IM/TUIKit。
- `02-authoritative-qch-source/企才汇源码交付-20260810/PHP-Web/qicaihui.com`：腾讯侧 PHP-Web 源码。
- `01-bridge/tencent-tangseng-bridge`：独立 Go Bridge，负责 PHP 回调、Inbox/Outbox、账号映射、悟空 PC(2) WebSocket、消息转换、幂等和重试。
- `04-tangseng-reference/叨叨服务端源码-轻量副本.zip`：叨叨源码参考副本，仅用于协议核对，严禁直接修改或把 Bridge 嵌入其中。

## 现网部署

### Bridge CVM

- 地址：`101.32.11.27`
- 用户：`ubuntu`
- 目录：`/opt/hw-bridge`
- 容器：`bridge`、`mysql`、`redis`
- 本机服务端口：`8090`
- 临时测试入口：`http://101.32.11.27:8443`（当前是明文 HTTP；生产必须改为 HTTPS/内网入口）
- 部署脚本：`06-release-and-deploy/deploy-bridge-private-vpc.redacted.ps1`

### 腾讯 PHP CVM

- 地址：`23.251.34.150`
- 用户：`root`
- 目录：`/root/IM/YCH-IM-deploy`
- APK 默认业务域名：`admin.awhaha.com`
- 部署脚本：`06-release-and-deploy/deploy-php-bridge-provision.redacted.ps1`

### 叨叨私网接口

- HTTP：`http://10.0.0.100:8090`
- 健康检查：`GET /v1/ping`
- WebSocket：`ws://10.0.0.100:5200/`
- WebSocket 必须在 2 秒内发送 JSON-RPC `connect`，`deviceFlag=2`，每 30 秒 `ping`。
- Bridge CVM 与叨叨机器位于同一 VPC：`vpc-5c0vuvo2`，网段 `10.0.0.0/24`。

## 当前链路

华为 APK → 腾讯 IM → 腾讯 PHP 回调 `/cb/tim` → Bridge Inbox/Outbox → 悟空 PC(2) → 叨叨桌面端。

叨叨桌面端 → 悟空 PC(2) `recv` → Bridge 根据 `friend_link` 路由 → 腾讯 REST `sendmsg` → 华为 APK。

测试已验证过搜索叨叨用户、好友建立和文本双向消息。普通腾讯用户聊天链路保持原样。

## 已完成

- 独立 Bridge 服务和 MySQL/Redis 容器。
- 叨叨私网 HTTP/5200 WebSocket 连接和 PC(2) 代理。
- PHP 注册/账号 provision、Bridge HMAC 回调入口、搜索和好友申请相关接口。
- 单聊文本双向转发、账号映射、源消息幂等、来源标记防回环、会话串行、重试/死信。
- 文本中的腾讯表情 `TIMFaceElem`、`TIMCustomElem` 基础归一化；不支持的媒体转成占位文本，避免无声丢弃。
- Bridge 重启从 MySQL 恢复账号和任务；过期 processing 任务回收；快速连续消息保持会话顺序。
- SSH 公钥认证已配置：`ssh -o BatchMode=yes ubuntu@101.32.11.27` 和 `ssh -o BatchMode=yes root@23.251.34.150` 可用。

## 最近稳定性修复

源码中已加入：

1. 解析腾讯表情和自定义消息，避免 `[TUIEmoji_Expect]` 等内容被误判为空。
2. 下行发送使用稳定的 `MsgRandom`/`Random`，减少腾讯 REST 拒绝。
3. Inbox/Outbox 按会话串行，旧任务未完成时不抢发后续任务。
4. processing 超时自动回收；重试延时为 1、3、10、30、120 秒。
5. 失败记录保留真实错误和消息 ID；日志包含腾讯 UID、叨叨 UID、device_id、message_id、trace_id。

最新 Bridge 部署包：`06-release-and-deploy/hw-bridge-private-vpc.tar.gz`。如果线上未部署该包，使用下面的部署命令重新发布。

## 当前观测

曾观察到：

```text
bridge_events_received 18
bridge_events_accepted 18
bridge_events_failed 0
bridge_inbox_received 1214
bridge_outbox_pending 1
bridge_outbox_dead 228
bridge_messages_done 845
```

`dead=228` 是历史失败任务累计值，不能直接清零；新 Agent 应先按错误原因审计，重复回调和已经成功的任务不得重复投递。最新代码主要解决后续稳定性问题，老死信需要按需要重放或归档。

## 重新部署

先设置本机环境变量（值不要提交到仓库或交接包）：

```powershell
$env:BRIDGE_SHARED_SECRET = '<与 PHP 相同的 HMAC 密钥>'
$env:TENCENT_SECRET_KEY = '<腾讯 IM SecretKey>'
$env:TANGSENG_PASSWORD = '<叨叨管理账号密码>'
$env:TANGSENG_INVITE_CODE = '<叨叨邀请码>'
```

然后：

```powershell
powershell -ExecutionPolicy Bypass -File `
  "F:\IM\_handoff_build\hw-bridge-handoff-20260929\06-release-and-deploy\deploy-bridge-private-vpc.redacted.ps1" `
  -ServerHost 101.32.11.27
```

PHP 补丁部署：

```powershell
powershell -ExecutionPolicy Bypass -File `
  "F:\IM\_handoff_build\hw-bridge-handoff-20260929\06-release-and-deploy\deploy-php-bridge-provision.redacted.ps1" `
  -ServerHost 23.251.34.150 `
  -BridgeURL "http://101.32.11.27:8443" `
  -BridgeSecret $env:BRIDGE_SHARED_SECRET
```

看到 `HW_BRIDGE_DEPLOY_OK` 和 `PHP_BRIDGE_PROVISION_DEPLOY_OK` 才算部署成功。部署脚本会保留服务器备份。

## 验证顺序

1. `curl http://127.0.0.1:8090/healthz` 应返回 `{"status":"ok"}`。
2. Bridge 日志应出现 `account restore complete`、`proxy connecting`、`proxy connected`。
3. APK 登录腾讯账号，搜索叨叨用户名，发起好友申请；在叨叨桌面同意。
4. APK 发送连续文本、中文、Emoji；确认叨叨收到。
5. 叨叨回复连续文本和 Emoji；确认 APK 收到且只出现一次。
6. 观察 `curl http://127.0.0.1:8090/metrics`，重点看 `outbox_pending`、`outbox_dead`、`messages_done`。
7. 重启 Bridge 后再次发消息，确认账号和 PC(2) 自动恢复。

## 构建测试

在 `01-bridge/tencent-tangseng-bridge` 执行：

```powershell
$env:GOCACHE = 'F:\IM\go-build-cache'
go test ./...
```

最近一次全量 Go 测试通过。Android APK 联调包位于 `06-release-and-deploy/企才汇-叨叨桥接联调基线-20260928.apk`。

## 新 Agent 继续工作的优先级

> 2026-10-02 更新：以下为原始交接时的历史顺序，已被用户后续指令覆盖。当前优先处理业务验收反馈；不启动主备、Leader、压测、死信专项或72小时稳定性。最新群通知/管理入口/群名/头像处理见 `GROUP_NOTIFICATIONS_20261002.md`。Bridge 已部署，最终 QA APK 为 `qch-group-notifications-20261002-r2-qa.apk`；头像两端实际显示与其他 UI 项继续待手机确认，不可宣称全部通过。不要使用顶部所列旧基线 APK 代替最新发布包。

1. 先部署并验证最新 Bridge 包，确认不再出现新的 dead/pending 增长。
2. 对 `bridge_outbox_dead` 按错误码分类，补做可安全重放的任务。
3. 继续按详细方案推进好友、群、媒体、撤回/已读、补偿、主备和压测；每阶段必须保留契约样本和验收记录。
4. 任何涉及叨叨服务端或悟空 IM 核心的修改都停止并先向负责人确认；方案要求零修改。

## 2026-10-02 群系统通知操作人补充修复

截图中的 administrator 为 REST 群资料通知操作人；上一版群名缓存修复不覆盖此项。现已部署实际操作人 From_Account 及精确回调防回环，取消群名自动写群简介。全量 Go 测试/vet 与临时群四项真实接口验证通过；手机新通知待复验，历史通知及无操作者的后台同步保留原状。详情见 GROUP_PROFILE_OPERATOR_20261002.md。现网 bridge SHA256 daf4ce4ed01caf0062c70d5216bdd6f39d9d841acf124a4831eb12affdca3fef，APK 继续 R2，无需换包。

## 2026-10-02 B 端群成员头像刷新补偿

用户确认个人资料页头像正确、群成员列表错误。已只读核对参考 iOS 与原生源码，并在 Bridge 补齐自己的群成员版本刷新通知；每群/用户/头像指纹去重。全量测试和真实 CMD/防回环验证通过；6 个现有测试群已执行补偿，手机页面待复验。超级群页面监听及非好友缓存限制仍需核对，不能标为全端已验收。报告 MEMBER_AVATAR_REFRESH_20261002.md；现网 binary SHA256 4c736bf586df349e040c597f5e97306be751d46330e345c564f01e3d46954cf7，APK 保持 R2。

## 首次建群头像通知修复（客户端 2026-10-01；产物标识 20261002）

已补齐初始化路径：建群时直接带头像，初始404则记录建群身份用于首次补齐，避免 administrator 通知。Go 全量测试/vet通过；门面和原生接口各创建临时群，实际首次 FaceUrl 与后续 Name 通知均为真实建群账号，重复同步无新增通知，测试群已清理。已部署 binary f9aac5cc7240b76cfe82632b0316221a03f86dce720f62fb9671bca67c2b0f16；APK仍为R2。详情 GROUP_INITIAL_AVATAR_20261002.md，手机新建群待复验。
