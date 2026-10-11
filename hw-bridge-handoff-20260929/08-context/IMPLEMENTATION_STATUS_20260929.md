# 业务开发进度（2026-09-29，持续更新）

> 最新合并报告、部署哈希和验收缺口见 [BUSINESS_DELIVERY_20260929.md](BUSINESS_DELIVERY_20260929.md)；[引用测试记录问题](REPLY_CONTRACT_INCIDENT_20260929.md)尚未修复，不能认定全部完成。下方保留各阶段过程记录。

## 当前范围

保持独立 Bridge、单实例、现有 Docker/MySQL/Redis/PHP 回调和腾讯 TUIKit。没有启动主备、Leader、Redis 租约、压测、72 小时稳定性或死信专项。剩余业务尚未全部验收。

## 基线与部署证据

- 两台 SSH 已授权；私钥在交接包外，不进入产物。
- 初始 Bridge SHA256：87bf4a886517c7beb847e415ce3a88b04bdd8ef26eaef455c26ed1c1a81f6755，匹配原交接版本。
- 初始 healthz/readyz 通过；pending=0、dead=232，仅作为基线观测。
- PHP/worker/gateway/MySQL/Redis 正常；原有 cleanup 容器重启问题仍未处理。
- 最新 Bridge（媒体及资料版本）SHA256：ec9f892222b09203d5ace288707f0fa3f8bb9d28d1ec4dc894b84b5810e7f482。
- Bridge 回滚备份：/opt/hw-bridge.backup.20260930012432。
- PHP 回滚备份：/root/IM/backup-php-bridge-20260930012417。
- .env 原有密钥条目保持不变；仅新增非敏感 BRIDGE_MEDIA_PUBLIC_URL=https://admin.awhaha.com/api/bridge/media，并在服务器保存修改前备份。
- Bridge 媒体镜像含 FFmpeg 7.1.5 + libopencore_amrnb；源码包 SHA256 固定校验，实际 AMR→AAC 往返测试通过。

## 阶段 1：好友（实现与部分真实验证）

已实现搜索、申请、同意、拒绝、删除、拉黑/解除、申请状态及加密 token/vercode 存储、PHP 黑名单入口、Android 申请列表/操作/错误展示、重复问候抑制。

实测账号 Tencent AllowType 为 AllowAny，ForceAddFlags=0 会跳过待确认。新增读取本人策略：NeedConfirm 使用原生申请；AllowAny 使用 bridge_friend_apply 卡片及新的联系人列表；DenyAny 拒绝叨叨申请。卡片不带叨叨凭据。

已真实验证：搜索、已是好友的申请、拉黑/重复拉黑/解除/重复解除，均成功。待验收：新申请的双向同意/拒绝、删除后的重新申请、过期申请、APK 卡片点击。不能将本阶段标为全部完成。

## 阶段 2：群（实现与部分真实验证）

已实现创建、dd_ 映射、邀请/成员移除、退群/解散、名称/公告、全员禁言、管理员/成员角色、单成员禁言、群头像和基础邀请确认设置，腾讯群回调与叨叨结构化群事件，双向群文本；Android 混合群入口及相关操作经过 PHP。

真实协议差异已固定为契约：group_type=1 的群不支持 membersync，改用 members 分页；memberUpdate 为实际指令名；群系统消息可无 from_uid。

联调群由用户确认可见。实际群创建、改名/公告、禁言/解除、移除/重新邀请、成员同步和双向文本投递通过。待验收：真实退出/解散/所有者转移、权限拒绝、多人华为群、邀请确认、所有 Android 群界面。大群被 @ 用户镜像策略仍需补齐。

## 阶段 3：媒体与资料（当前推进）

已实现：
- 结构化图片/GIF、文件、语音、视频及封面消息，兼容旧文本任务。
- 下载 30 秒/100 MB 限制、临时文件清理、SHA256 缓存、使用发送者用户 token 上传叨叨。
- 语音 AMR-NB/AAC 转换，阅后即焚只生成 bridge_tip，不复制正文或媒体。
- PHP 签名媒体网关、GET/HEAD/Range；APK 只看到业务域名签名 URL，无叨叨内网地址、token。
- 私网任意 URL、路径穿越与超大下载拒绝测试。
- 昵称/用户头像资料通知复用 Inbox/Outbox；好友和群头像公网转换；群头像修改使用真实操作者身份。
- Android bridge_tip 灰色系统提示及摘要；桥接单聊和 dd_ 群关闭不支持的音视频通话入口。

真实验证：
- 用户发来的图片 773235 字节、视频 43356721 字节及封面 198564 字节、音频文件 72148 字节全部投递腾讯 done。
- 四个公网媒体 Range 请求全部 HTTP 206，返回准确 Content-Range 和 32 字节内容。
- 新单聊文本三条 done，群文本双向 done。
- 搜索结果头像已转换到 admin.awhaha.com。
- 已发送一条腾讯 REST 语音消息验证腾讯回调→Bridge 转码→叨叨，结果仍需查询确认。
- 用户真机展示、播放和反向 APK 发送验收仍待反馈；文件音频不能代替录音语音验收。
- 资料修改、头像上传端到端，以及名片/引用/@/不支持内容全部分支仍待完善/验证。

## 测试与产物

- Bridge go test ./...、go vet ./... 均通过；含好友策略、群契约、媒体 Range/鉴权/边界、阅后即焚、资料队列测试。
- PHP 8 个部署文件语法检查通过，Bridge 门面契约脚本输出 BRIDGE_FACADE_CONTRACT_OK。
- Android assembleQaRelease 多次通过；最新群头像入口改动后的构建仍在运行，以日志为准。
- 联调 APK：06-release-and-deploy/qch-business-media-20260929-qa.apk。
  SHA256：e44c12587b1b8898718a3a88cb0fd4182553b5843fa123be3fe5b6e359e95f30。
  使用 QA 调试签名，不是正式发布签名。
- 最新服务包：06-release-and-deploy/hw-bridge-private-vpc.tar.gz、php-bridge-provision.tar.gz，包含 SHA256SUMS 和 Bridge SOURCE_SHA256SUMS。
- 先前阶段归档：friends-20260929、friends-groups-cards-20260929、media-20260929。

## 主要修改位置

Bridge：
internal/business/{friends,groups,group_events,profiles}.go；
internal/adapter/{business,groups,adapter}.go；
internal/domain/{business,groups,media,parse,domain}.go；
internal/store/{friends,groups,media,store,schema_embed}.go；
internal/httpapi/{friends,groups,profiles,server}.go；
internal/{engine/engine,proxy/wukong,session/wukong,media/service,media/avatar}.go；
internal/account/service.go、cmd/bridge/main.go；
migrations/002_friend_state.sql、003_group_state.sql、004_media_object.sql；
deploy/Dockerfile.runtime 及对应测试和脱敏契约。

PHP（03-php-deployment/backend）：
application/imaccess/service/{BridgeService,BlocklistService}.php；
application/imaccess/controller/{Bridge,BridgeMedia}.php；
application/api/model/{User,UserTencent}.php；
config/im_bridge.php、route/route.php。

Android（权威源码 Android/TUIKit）：
TUIContact 的 BridgeContactClient、ContactProvider、GroupInfoProvider、GroupMemberProvider、FriendApplicationBean、TUIContactService；
TUIChat 的 ProfileProvider、ChatInfo、BridgeNoticeMessageBean/Holder、BridgeTipMessageBean、TUIChatService、ClassicUIService、MinimalistUIService。
构建配置调整见前期版本（AGP7.4.2/Gradle7.6.4/JDK11/本地 SDK）。

## 部署命令

在 06-release-and-deploy 目录执行，密钥路径仅指向本机现有文件：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
./deploy-php-bridge-provision.redacted.ps1 -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
```

脚本先校验 __SET_*__、SHA256 和 PHP 语法；备份后应用增量迁移，仅更新相应服务；保留现有 .env 和 compose。预期 HW_BRIDGE_DEPLOY_OK / PHP_BRIDGE_PROVISION_DEPLOY_OK。

## 服务器验证命令

```sh
cd /opt/hw-bridge
curl -fsS http://127.0.0.1:8090/healthz
curl -fsS http://127.0.0.1:8090/readyz
sudo docker compose ps bridge
sudo docker compose exec -T bridge ffmpeg -hide_banner -encoders
sudo docker compose logs --since 5m bridge
```

预期 health/ready 返回成功，bridge Up，编码器列表含 libopencore_amrnb 与 aac。
PC 接收日志含 tangseng recv / tangseng event accepted。投递依据 message_mapping 的 done 与目标 message ID，不能只凭收包日志认定送达。

```sh
cd /root/IM/YCH-IM-deploy
docker compose ps php worker
php_container=$(docker compose ps -q php)
docker exec "$php_container" php -l /var/www/app/application/imaccess/controller/BridgeMedia.php
curl -sS -o /dev/null -w '%{http_code}\n' 'https://admin.awhaha.com/api/bridge/media?id=invalid&sig=invalid'
```

预期 php/worker Up、语法检查无错误、伪造媒体签名 HTTP403。真实签名 URL 由服务器读取消息后验证，不写入交接文档；Range 应为206、HEAD为200。

回滚使用上述对应备份的 bridge/Dockerfile 或 PHP 文件，并重新构建/启动相应服务；保留当前密钥。增量字段可留存，不删除已写业务数据。

## 后续顺序和风险

1. 继续实际媒体双向/真机验证，资料与好友/群剩余验收，不停留在方案。
2. 撤回、单聊已读/未读、消息 ID/序号映射。
3. 账号停用注销、影子状态、会话频道和漫游同步游标。
4. APK 总验收、普通腾讯流程回归、最终版本清单。
5. 六类业务验收以后才另行讨论可靠性。

当前不能称为全部功能完成。回调权限、真机行为、正式签名仍以实测/安全配置为准。媒体签名采用持久文件凭证以支持历史查看；shared secret 轮换会使旧签名失效。资料队列失败仍使用现有基本重试。原媒体解析失败记录没有批量重放，仅检查无成功映射后重新处理本次用户提供的三个样例。

原始叨叨参考 ZIP 不完整；protocol-reference/upstream_* 只是只读上游参考，未作为部署基线。所有真实差异以 test/contract 的脱敏样例为据。

## 视频 60009 修复及消息状态开发增量

- 用户已确认图片可见；43 MB 视频公网下载超时已修复并部署。旧链路 35 秒失败，流式修复后完整下载 74.37 秒，受控对象源直取后 24.53 秒；均使用同一个原视频复测，后两次 SHA256 一致。详见 NETWORK_MEDIA_20260929.md。
- 用户已休息，真机视频播放、录音语音及 APK 交互不能代为认定通过。继续服务端开发和验证。
- 反向语音 REST 联调已确认：腾讯测试消息经 MP3→AMR 和叨叨上传/PC 投递成功，目标消息 ID 已记录在安全联调记录；不等同于 APK 录音发送验收。
- 资料 APK 编译 BUILD SUCCESSFUL，后续发布包需使用最新构建产物。
- 已增加 bridge_message_record 与 bridge_business_cursor、原生消息序号和 client_msg_no；群发送现在记录腾讯真实 MsgSeq，不再以本地哈希充当可撤回序号。
- 真实单聊撤回测试已通过：新发测试消息经腾讯回调投递叨叨，再调用腾讯撤回，叨叨对应消息映射 revoked=1。群管理员真实测试、叨叨用户发起撤回、双端已读 UI 仍待验证。
- 消息扩展接口使用 H 用户 token；单聊及测试群真实 extra/sync 请求均成功。新代码仅对已建立原生映射的消息处理撤回/已读；早期消息映射不全仍需补齐，不能报告历史撤回全部可用。
- 本阶段继续业务开发，不启动主备、Leader、死信专项或长时间压测。
