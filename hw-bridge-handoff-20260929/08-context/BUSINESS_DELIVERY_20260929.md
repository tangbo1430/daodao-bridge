# 业务开发交付与验收进度

> 最新业务验收及现网版本：[BUSINESS_ACCEPTANCE_20260930.md](BUSINESS_ACCEPTANCE_20260930.md)。新增群角色/禁言/头像/成员接口验收；修复好友资料丢头像和昵称串号。完整三端验收仍未结束。

> 最新现网版本及反向已读真实验证：[READ_RECEIPT_20260930.md](READ_RECEIPT_20260930.md)。R3 已确认腾讯 IsPeerRead=1；APK 显示待验收。以下旧报告保留历史记录。

> 最新：[真实客户端验收与资料回调修复](CLIENT_ACCEPTANCE_20260930.md)。已实际操作叨叨客户端并修复昵称恢复漏同步，包含下述所有补丁；未覆盖的原验收项继续保留。

> 当前版本：[消息状态与群生命周期收尾](CORE_STATE_GROUP_20260930.md)，包含头像及群提示修复。无 param 的消息状态真实协议、重复解散已修复；临时群创建/资料/同步/解散及重启同步已通过服务器验证。以下表格保留原交付时的验收记录，新增结果以最新报告为准，未覆盖的界面验收仍待完成。

> 最新部署及待办：[头像同步修复与核心收尾清单](AVATAR_FIX_20260930.md)。头像补丁包含群同步补丁。历史账号不再纳入本阶段，约三分钟视频失败按用户要求后置；原表未验收项不能因“大部分可以”的整体反馈而全部勾选。

> 后续现网热修复：[群同步系统提示重复修复](GROUP_SYNC_FIX_20260930.md)。已部署 2026-09-30 版本；本文原始二进制和业务整包不包含该后续补丁。

开发环境日期2026-09-29；服务器发布日期2026-09-30。基线：用户上传的完整交接ZIP。

## 当前结论

已交付三端业务代码、服务发布包和QA APK，**六类业务尚未全部验收完成**。保持单实例、独立Bridge及现有Docker/MySQL/Redis/PHP方式；未开展主备、Leader、Redis租约、压测或72小时专项。

先读 [引用测试记录处理说明](REPLY_CONTRACT_INCIDENT_20260929.md)：我新增的转换曾把reply.message_id写成数字，导致包含一条自动测试记录的叨叨历史范围返回500。代码已修正，新引用原生拉取200；旧记录虽已原生撤回/双向删除，仍需已有叨叨维护入口处理。未改叨叨/悟空代码，未清空用户会话。部署回滚不能消除这条数据问题。

## 分阶段状态

| 阶段 | 本轮实现与已完成验证 | 尚未验收 |
|---|---|---|
| 好友 | 搜索/申请/同意/拒绝/删除/黑名单/申请卡片及列表/资料；凭据加密。真实搜索、已是好友申请、拉黑/解除通过；重复及过期契约通过；POST搜索/申请列表HTTP200 | 新的双向待确认申请、拒绝后重申请、双方删除UI、普通腾讯好友回归 |
| 群 | dd_映射、建群/邀请/增删成员/退出/解散、资料/禁言、群消息、邀请提示；大群按需镜像和容量处理。真实建群、改名/公告、禁言/解除、移除/再邀请、成员同步、双向文本通过，用户确认群可见，群@到达原生payload | 真实退出/解散/群主转移、邀请确认、多华为成员、满员及权限UI |
| 媒体/资料 | 图片/GIF/文件/语音转码/视频封面，签名网关/Range、昵称头像、名片/@/引用、位置/富文本/合并转发降级；阅后即焚不复制内容。图片用户确认；视频完整下载SHA一致；反向REST语音送达；名片type7、@、修正后引用原生验证通过 | 手机视频播放、APK录音/短视频双向、GIF/大文件/头像刷新、普通腾讯媒体回归；旧引用测试数据处理；缺SDK ID引用按方案降级摘要 |
| 撤回/已读 | 原生ID/序号映射，双向单聊/群撤回，PHP群撤回保留真实操作人，receipt开关、已读/未读清零、消息扩展。真实腾讯单聊撤回→叨叨成功；PHP群撤回成功；extra/sync200；关闭回执原生setting=0 | 叨叨用户主动撤回、群管理员拒绝、双方已读/未读UI；接入早期缺原生映射的历史消息；群已读本期不做 |
| 账号/会话 | 资料通知、注销事务队列、按用户身份退群/解除关系/更新状态/清token，停用后注销恢复，会话/频道/指令同步、腾讯单聊/群漫游、持久游标。注销队列契约通过；真实会话同步4频道200；禁止发送后回调的两条消息由分钟漫游补回，重复同步无重复投递 | 不在用户睡眠期间注销真实测试账号；重新登录/离线双端UI；3条旧账号映射关系刷新收到腾讯30003，需核对账号有效性；问题历史范围仍待维护 |
| APK | 影子联系人/搜索/申请通知、混合建群/邀请、bridge_tip及摘要、影子会话隐藏通话、镜像群不请求群回执；查询改POST。assembleQaRelease成功，普通腾讯分支保留 | 真机总验收、普通腾讯全业务回归、正式签名 |

## 网络和视频

43,356,721字节视频此前约35秒失败：Bridge30秒/PHP35秒超时及PHP输出缓冲使响应被截断。已部署流式输出和长下载超时；PHP校验HMAC后直取已有对象源，APK仍访问业务域名。完整下载从约74.37秒降至24.53秒，SHA256相同；HEAD200、Range206、越界416。

Bridge元数据实测地域ap-hongkong，IM REST使用adminapisgp.im.qcloud.com；不能认定二者自动走VPC内网。PHP签发UserSig、腾讯验证，发送前业务回调是另一条路径。PHP发送前回调服务内耗时24小时样本平均18.93ms、最大51ms，不等于公网RTT/丢包率。当前没有用户所述大陆中转/CDN配置，未声称跨境丢包和CDN路径已全部解决。详见 [网络与媒体记录](NETWORK_MEDIA_20260929.md)。

## 修改文件

与原交接ZIP逐文件SHA256比较：Bridge99个、Android17个、PHP及测试16个，合计132个新增/修改文件；完整路径、基线哈希和新哈希见发布目录CHANGED_FILES.json。比较时恢复了原ZIP中文文件名编码，未将乱码差异误报为修改。

- Bridge：internal/business的好友、群、资料、消息状态/元数据、账号清理、会话/漫游及同步任务；internal/adapter、domain、store、httpapi、account、media、engine、proxy、session、cmd/bridge及对应测试。
- 数据迁移：002_friend_state.sql、003_group_state.sql、004_media_object.sql、005_message_state.sql，均增量，不删除原数据。
- PHP部署11文件：User.php、GroupV.php、UserTencent.php、Bridge.php、BridgeMedia.php、BridgeService.php、MediaSourceService.php、BlocklistService.php、HandlerRegistry.php、config/im_bridge.php、route/route.php；另有5份契约测试。
- Android：qch构建配置3份；TUIChat的ChatInfo、ProfileProvider、消息注册、bridge_tip/申请卡片Bean及Holder；TUIContact的BridgeContactClient、ContactProvider、GroupInfoProvider、GroupMemberProvider、FriendApplicationBean、TUIContactService。
- 发布脚本扩展PHP清单、FFmpeg运行时、迁移及SHA校验；继续保留服务器.env。

## 测试结果

1. Bridge：go test ./...、go vet ./...全部通过；新增媒体上传身份隔离、容量、名片加密、引用类型、命令队列序号重置、已读开关、注销恢复等测试。
2. PHP：11个部署文件语法通过；BRIDGE_FACADE_CONTRACT_OK、MEDIA_SOURCE_CONTRACT_OK、MESSAGE_STATE_CALLBACKS_OK、ACCOUNT_DELETION_QUEUE_OK；缓冲测试仅输出fixture!，框架缓冲未泄漏。
3. Android：assembleQaRelease，BUILD SUCCESSFUL，3分47秒，252任务；QA签名，非正式签名。
4. 现网：Bridge healthz/readyz200，PHP及worker运行；session_sync200（4频道、accepted=0）、tencent_roam_sync200；PHP POST搜索返回users，申请列表code200。
5. 原生契约差异存入脱敏样例：大群members分页、成员指令、more=0不代表无后续、64位ID/版本、命令队列清空后序号8→2、reply.message_id必须字符串。

## 已部署版本和产物

- Bridge二进制SHA256：66ca99e7e176968f07a00f28922464d3dba684e92203115a03550d50b4804fb5。
- Bridge本次备份：/opt/hw-bridge.backup.20260930025326。
- PHP本次备份：/root/IM/backup-php-bridge-20260930023259。
- qch-business-final-20260929-qa.apk：SHA256 3f80b1c6530e691a97417fd935ccc84fc8864b89160d4b4e8b2361fb6df50a27。
- 汇总：06-release-and-deploy/business-20260929，含两个服务包、APK、源码补丁、迁移、版本/文件清单、SHA256SUMS及验证说明。
- 包中不含SSH私钥和生产.env；.env.template仅占位符。

## 部署命令（已经执行）

在06-release-and-deploy目录：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
./deploy-php-bridge-provision.redacted.ps1 -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
```

脚本检查服务器现有配置中的__SET_*__、哈希、备份和迁移，保留.env和现有compose。成功尾标志HW_BRIDGE_DEPLOY_OK / PHP_BRIDGE_PROVISION_DEPLOY_OK。重建瞬间connection reset不能作为最终结果，须等健康门槛通过。

## 验证命令与预期

```sh
# Bridge CVM
cd /opt/hw-bridge
curl -fsS http://127.0.0.1:8090/healthz
curl -fsS http://127.0.0.1:8090/readyz
sha256sum bridge
sudo docker compose ps bridge mysql redis
sudo docker compose logs --since 5m bridge
# PHP CVM
cd /root/IM/YCH-IM-deploy
docker compose ps php worker
docker compose exec -T php php -l /var/www/app/application/imaccess/service/BridgeService.php
curl -sS -o /dev/null -w '%{http_code}\n' 'https://admin.awhaha.com/api/bridge/media?id=invalid&sig=invalid'
```

预期健康200、容器Up、哈希如上、PHP无语法错误、伪造媒体签名403。同步有新消息时日志business sync complete owner=… channels=… accepted=…；无变化不要求打印成功日志。business sync pending是待处理错误，不计通过。PC收包日志不能代替最终投递映射及原生消息验证。

真实签名媒体URL只在服务器安全读取；完整GET字节数及SHA必须与对象一致。业务HMAC在服务器使用现有配置计算，不把凭据放进命令参数、日志或文档。

## 回滚与下一步

回滚仅还原对应备份的Bridge二进制/Dockerfile或PHP改动文件，再按原compose重建/重启；保留.env、数据库及新增表。若需回退到未增加引用转换的业务版本，使用Bridge备份/opt/hw-bridge.backup.20260930024228，不选择曾含数字型reply实现的版本。PHP回退到POST查询接口之前时，APK须配套回退。

优先处理单条引用测试数据；取得现有中转/CDN配置后核对实际路由；再完成表中双端UI、旧账号映射、历史映射缺失及正式签名验收。不得将代码交付等同于全部业务跑通。可靠性专项另行安排。
