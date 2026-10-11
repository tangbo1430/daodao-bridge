# 头像同步修复与核心收尾清单

## 用户最新反馈

用户在另一台机器测试后反馈“大部分都可以”，约 3 分钟的大视频仍可能未送达。本阶段优先核心功能，大视频暂后置。该反馈未逐项列出用例，不能据此把所有好友、群、已读、注销操作标记为通过。历史账号不迁移、不补齐。

## 已确认的头像问题

修复前：叨叨原头像 HTTP 200、34082 字节；腾讯影子账号 Image 为空；Bridge 好友状态仍有头像，但账号映射已被群成员同步覆盖为空。

根因与修改：

1. account_import 每次携带空 FaceUrl。改为缺少字段时省略，避免抹去已同步资料。
2. 群同步使用成员昵称和空头像覆盖账号映射。改为保留账号原有昵称、头像；缺头像时通过现有媒体网关补齐。
3. 好友同步遇到一条错误立即停止，后面的正常好友无法刷新资料。改为处理其余好友后汇总错误；未知历史删除记录不导入、不操作腾讯账号。
4. 删除目标不存在的 30003，仅在查询确认发起人存在、目标未导入时按已删除处理；其他错误继续报告，不直接吞掉 30003。
5. userUpdate/userAvatarUpdate 有明确 UID 时直接同步该影子资料，支持仅在群内、尚非好友的影子成员；仍使用对应华为用户的叨叨 token，不覆盖华为原生账号资料。

涉及 Bridge 文件：

- internal/adapter/adapter.go
- internal/adapter/business.go
- internal/adapter/profile_preservation_test.go
- internal/business/groups.go
- internal/business/friends.go
- internal/business/profiles.go
- internal/business/avatar_preservation_test.go
- test/contract/avatar-profile.anonymized.json

PHP、Android 本轮无修改，当前 APK 可以继续使用。

## 测试和部署结果

- go test ./...、go vet ./... 通过。
- 回归覆盖空字段导入、连续群同步保留头像、旧关系错误不阻断正常资料、未知历史记录不操作、群内非好友资料事件、删除不存在账号时的严格条件。
- 已部署 Bridge 二进制 SHA256：742239964418cd247b699bca16d7fa04fdfe2fae6d052ae1b9c3f6af7bf2fa28。
- 最终部署备份：/opt/hw-bridge.backup.20260930194506。该备份包含前一次头像补丁；需要回到本轮之前的群同步热修复版本时使用 /opt/hw-bridge.backup.20260930194313。
- 发布包：06-release-and-deploy/hw-bridge-avatar-fix-20260930.tar.gz，包含先前群同步热修复。
- 发布包 SHA256：f681d4e92c789eb5e31c35901b1f2cabe3e569591d2b6a96678d2b0b9850ac66。
- 日志：工作区 toolchain/avatar-fix-go-test.log、toolchain/deploy-avatar-fix.log。
- healthz=ok、readyz=ready。

现网验证：腾讯头像已指向 PHP 业务域名，HTTP 200；叨叨原头像与腾讯公开头像均为 34082 字节，SHA256 均为 f10e312708e644ab5b4d4214abbe305c230e7fc9a91434e7a7b9c0f7bcb3f1d9。连续群同步后头像仍一致，群 NextMsgSeq 保持 3142。

客户端旧头像缓存的刷新，以及用户在叨叨主动换一张新头像后的实时刷新，仍需界面验收；本次未擅自修改用户实际头像。

## 已执行部署命令

从 06-release-and-deploy 运行：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
```

脚本保留服务器 .env，备份并重建现有 Bridge 容器。成功标志 HW_BRIDGE_DEPLOY_OK。

## 验证命令

脚本从服务器安全读取 .env，不输出凭据或签名 URL。替换真实授权叨叨 UID（不是用户码）：

```powershell
$peerID='<授权叨叨UID>'
$script64=[Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path './verify-avatar-sync.py')))
ssh -T -i C:/Users/Administrator/.ssh/hw_bridge_20260929 ubuntu@101.32.11.27 "echo $script64 | base64 -d | sudo -n python3 - $peerID"
```

预期两个 http 200、相同 SHA256、AVATAR_SYNC_SHA256_OK。再执行 verify-group-mute-fix.py，应为 GROUP_SYNC_NO_NEW_TIPS_OK。

## 核心剩余

- **头像/昵称**：本次修复服务器侧空头像问题；双端主动修改资料与客户端刷新待逐项确认。
- **已读/未读、会话状态**：已有实现，曾观察到“消息状态指令缺少频道”的实际任务错误，需要按真实报文补齐并完成双端验证。
- **撤回**：单聊/群撤回已有实现及部分真实验证，叨叨主动撤回、管理员权限和手机显示仍未逐项验收。
- **账号与会话恢复**：重新登录、离线后恢复、停用/注销流程已有代码，完整真实闭环仍未验收。
- **好友、群**：基础流程已有实现；拒绝后重申请、退群/解散、邀请确认及权限边界未收到逐项验证结果。
- **当前会话异常引用记录**：新消息格式已修复；之前那一条异常测试数据仍待已有维护入口处理，不属于历史账号迁移。
- **后置**：约 3 分钟视频失败、跨地域中转/CDN落地、正式发布签名；不启动主备、Leader 或可靠性专项。

协议参考：[腾讯账号导入](https://cloud.tencent.com/document/product/269/1608)、[账号存在状态查询](https://cloud.tencent.com/document/product/269/38417)。
