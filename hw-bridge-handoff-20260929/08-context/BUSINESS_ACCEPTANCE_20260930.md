# 业务验收与好友资料增量修复

## 本轮实际完成

依据《详细开发实施方案》第16.3节，保持独立 Bridge、单实例和现有部署。历史账号、大视频与可靠性专项后置。本轮未修改叨叨服务端、悟空核心、PHP 或 Android。

1. 补齐上一轮逐条已读的版本、源码和发布记录，见 READ_RECEIPT_20260930.md。R3 的腾讯 IsPeerRead=1，本次部署后再次查询通过。
2. 创建临时群 dd_d314f515a07740aba753b09fc1e23365，实测名称、公告、管理员添加/移除、单人禁言/解除、全员禁言/解除、邀请开关、踢人/重新邀请、上传群头像、重复同步及解散。
3. 每步查询腾讯实际资料或成员结果；邀请开关核对 Bridge 从原生读取的群状态。群头像公网与原生内容 SHA256 均为 c9f6c674f7d5cda97a73628933584b1a26263f140184bd45880d04d6c73f7d08。连续两次同步 NextMsgSeq 不增加。
4. 临时群已解散，原生状态2、mirror_ready=false，腾讯查询返回群不存在10010。没有解散原联调群。
5. 修复好友申请/同意过程中丢头像映射及昵称写错账号，完成失败复现、全量测试、现网部署及基础回归。

## 验收状态（严格区分接口与界面）

| 用例 | 代码/接口证据 | 叨叨界面 | APK界面 | 当前结论 |
|---|---|---|---|---|
| E2 反向单聊已读 | R3 两端记录一致、腾讯 IsPeerRead=1 | 已读来自真实客户端；本机操作中断，不能归因为本机点击 | 待确认 | 已修复待手机确认 |
| E3 现有头像与昵称 | 头像 HTTP200、34082字节、两端SHA相同；前轮昵称修改/恢复通过 | 昵称前轮通过；本轮打开个人资料确认正确账号 | 待刷新确认 | 已修复待手机确认 |
| E3 主动更换个人头像 | 原头像已备份；本轮没有执行上传，不能声称1分钟刷新通过 | 桌面输入被拒绝 | 待确认 | 待外部条件 |
| D4/D6/D7/D8/D9 基础群操作 | 本轮名称/公告/角色/禁言/群头像/解散真实查询通过 | 本轮未操作 | 待确认 | 已修复待手机确认；角色转让另列 |
| D5 成员移除及重新邀请 | 原生成功后腾讯成员消失/重新出现 | 前轮部分验证，本轮仅接口 | 待确认 | 已修复待手机确认 |
| D5/D7 退群/群主转让 | 实现存在，契约覆盖；本轮未真实操作 | 待双端操作 | 待确认 | 待外部条件 |
| 邀请确认 | 开关往返通过；1009按visibles发提示，等1002实际入群同步，符合方案11.3 | 真实待审批场景未覆盖 | 待确认 | 待外部条件 |
| C1—C6 新好友申请完整闭环 | 实现/契约存在；本轮修复资料投影；不能用已是好友的接口结果代替新申请 | 拒绝、重申请、双方同意和删除需可交互客户端 | 待确认 | 待外部条件 |
| C7 黑名单 | 前轮拉黑/解除及重复操作接口通过 | 双方拒收展示待确认 | 待确认 | 待外部条件 |
| B1—B7/B11 文本、媒体、自定义 | 前轮双向文本、媒体传输、Range、语音转换、新引用等有证据 | 图片/部分消息已有用户反馈，未逐项全覆盖 | 播放、GIF、录音、名片等集中协测 | 待外部条件 |
| E1 撤回 | 前轮单聊/群本人的撤回真实通过 | 前轮已操作并核对 | 撤回展示/管理员权限待确认 | 已修复待手机确认；管理员用例另列 |
| A7 注销/停用 | 实现与契约存在 | 不注销当前联调账号 | 需专用可销毁账号 | 待外部条件 |
| B8 离线/账号会话 | 前轮持久游标、会话与漫游补回有证据 | 重新登录后显示待确认 | 厂商推送、普通腾讯回归待确认 | 待外部条件 |
| D1/D2 多华为成员、D10真实大群 | 已有实现和模拟测试 | 缺额外授权账号/实际大群 | 待确认 | 待外部条件 |
| 大视频、HA、压测、72小时 | 未开展 | — | — | 按约定后置 |

不将上述“接口通过”换算成全部业务已完成百分比。本轮主攻新增服务端证据与发现的代码缺口，完整三端验收仍未结束。

## 新发现的两个代码问题

好友 friendRequest 与 friendAccept 携带部分资料；旧代码每次新建空头像 Identity 并写库，覆盖已有头像。friendAccept.from_name 属于同意人，华为用户同意申请时属于华为用户，旧代码却写给叨叨联系人。

修改文件（Bridge相对路径）：

- internal/business/friends.go：申请/同意合并联系人资料；仅当 from_uid 是叨叨联系人时采用 from_name；同步申请列表的昵称头像。
- internal/business/profiles.go：新增部分好友资料合并，保留已有映射与好友资料，媒体服务可用时获取当前头像。
- internal/business/friend_profile_test.go：覆盖申请和双向同意，修复前四个子用例失败，修复后通过。

协议判断依据为交接包只读参考 upstream_modules_user_api_friend.go 的 friendAccept 生成结构。本轮没有新采集真实 friendAccept，不能把这组回归测试称为新的真实双端验收。

新增可复用验收脚本：06-release-and-deploy/verify-group-business.py。账号由参数输入，无硬编码测试账号；临时群独立命名，finally执行解散，运行状态保存在服务器/tmp下0600文件。不包含生产密钥；执行会实际创建群和修改临时群成员/资料。

## 测试与部署

- go test ./...、go vet ./... 通过。
- 现网二进制SHA256：80fdc00078a2d7665af0822ba77ec5198cdaeaaf5daaf9251cbb1d348c7bc5a2。
- 回滚备份：/opt/hw-bridge.backup.20260930223334；日志：工作区 toolchain/deploy-friend-profile.log。
- Bridge healthz=ok、readyz=ready；PHP、worker均Up。
- 部署后头像 AVATAR_SYNC_SHA256_OK、逐条已读 TENCENT_MESSAGE_RECEIPT_OK。
- 无数据库迁移。PHP/Android无改动，未重复构建APK，沿用 qch-business-final-20260929-qa.apk。
- 服务包：hw-bridge-business-acceptance-20260930.tar.gz；累计源码：business-acceptance-20260930-source.zip；校验：business-acceptance-20260930-SHA256SUMS.txt，均位于06-release-and-deploy。

实际部署命令（发布目录）：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
```

预期 HW_BRIDGE_DEPLOY_OK。回滚从上述备份恢复二进制及Dockerfile后按现有Compose重建Bridge，保留.env和数据；会失去本轮好友资料修复。

验证命令：

```powershell
# 发布目录；此脚本会创建并清理临时群，使用获授权账号
$owner='<授权腾讯UID>'
$peer='<授权叨叨UID>'
$script64=[Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path './verify-group-business.py')))
ssh -T -i C:/Users/Administrator/.ssh/hw_bridge_20260929 ubuntu@101.32.11.27 "echo $script64 | base64 -d | sudo -n python3 - $owner $peer"
```

预期 GROUP_AVATAR_SHA256_OK、GROUP_ROLES_MUTE_INVITE_AVATAR_OK、TEMP_GROUP_DISBANDED。单次成功本轮已执行，不需要反复建群消耗原生每日额度。原始输出：工作区 toolchain/group-business-acceptance.log。

服务器健康验证：curl -fsS http://127.0.0.1:8090/healthz；curl -fsS http://127.0.0.1:8090/readyz；sha256sum /opt/hw-bridge/bridge。头像和回执验证脚本及参数见 AVATAR_FIX_20260930.md、READ_RECEIPT_20260930.md。

## 外部条件与风险

本机正确账号 shybo002/gBkqAmfsM 可读取个人资料，但连续输入失败 GetCursorPos failed: Access is denied (0x80070005)。重取窗口后仍失败，已停止桌面输入；个人头像、好友关系未改动。需恢复可交互桌面后继续原生客户端测试，不要求用户现在操作APK。

当前会话旧异常引用记录仍导致指定历史范围500，需已有叨叨数据维护入口定点处理，详见 REPLY_CONTRACT_INCIDENT_20260929.md。不能通过清空会话、改游标跳过或修改叨叨服务端冒充修复。

下一步集中手机协测清单见 APK_ACCEPTANCE_CHECKLIST_20260930.md；如发现失败，记录用例和时间后回到对应Bridge/PHP/Android修复，不标记全量验收完成。
