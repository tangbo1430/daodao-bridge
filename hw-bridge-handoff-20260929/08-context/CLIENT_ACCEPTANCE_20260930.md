# 真实叨叨客户端验收与资料回调修复

## 环境和范围

用户授权操作本机已登录的邻距共讯（叨叨）Windows 客户端。先在个人信息核对 shybo002 / gBkqAmfsM，再与原授权企才汇 74629（ceshi101）及桥接联调群测试。本轮未操作 APK 界面，不能代替手机显示验收。未修改叨叨业务服务端或悟空核心。

## 真实验收结果

| 项目 | 实际证据 | 结论 |
|---|---|---|
| 叨叨单聊发送 | 客户端发送“桥接客户端验收-撤回-0930-A1”，腾讯消息键 658300657_290803994_1790773963，叨叨 ID 2105284709025058816、seq 35 | 到达腾讯，已有双端映射 |
| 华为阅读 → 叨叨已读 | 叨叨上述消息气泡显示“已读”；Bridge read_synced=true，腾讯 IsPeerRead=1 | 本方向通过，未由代理操作手机 |
| 叨叨主动单聊撤回 | 客户端右键撤回刚创建的 A1，气泡消失；Bridge revoked=true，腾讯漫游 MsgFlagBits=8 | 服务处理与叨叨界面通过；APK 撤回提示待确认 |
| 叨叨群文本及撤回 | 客户端在桥接联调群发送并撤回“桥接客户端验收-群撤回-0930-G1”；叨叨 ID 2105285136944730112，seq14；腾讯群消息 seq3142 | 腾讯群历史 IsPlaceMsg=2，Bridge revoked=true；本人的群撤回通过，不能替代管理员权限用例 |
| 腾讯 → 叨叨接收 | 用腾讯 REST 发送“桥接客户端验收-腾讯到叨叨-0930-R1”，消息键 1649510043_923093267_1790774170；客户端显示 ceshi101 消息正文 | 真实接收显示通过；发送入口为 REST，非 APK |
| 单聊未读数 | R1 到达时会话角标为1、总数为4；打开后该会话角标消失、总数恢复3 | 叨叨本会话未读清零通过，未一键清除其他会话 |
| 叨叨阅读 → 腾讯已读 | R1 的原生 setting=0、Bridge need_receipt=false；腾讯 IsPeerRead=0 | 本轮不能算通过。华为影子身份 GET /v1/users/对端 返回 receipt=0；叨叨自己的会话开关开启，不等于相反方向开启。未更改回执或隐私开关 |
| 昵称修改与恢复 | 客户端将昵称临时修改，腾讯更新；恢复原名时发现漏同步，修复后再次修改为 shybo002-验收2 再恢复 shybo002，两次腾讯查询均匹配 | 修复后真实往返通过，最终已恢复原名，腾讯头像仍非空 |
| 头像 | 客户端个人头像可见，已有头像内容校验通过；昵称往返后腾讯头像仍保留 | 本轮没有更换真实头像，不能标记“更换新头像后刷新”已通过 |

两条撤回专用测试消息已撤回；R1 普通文本保留用于核对。未删除好友、退出原联调群或注销账号。

## 发现并修复的问题

真实昵称修改发送的是 channelUpdate(type99)，param 中含 channel_id 和 channel_type=1。原处理只调用 SyncFriends。好友快照值没有变化时，未必调用腾讯 portrait_set；实际出现任务已 done、叨叨已恢复 shybo002，而腾讯仍为临时昵称的情况。

修复后 channelUpdate 先保留关系同步，再按指定用户直接获取当前资料并更新腾讯。关系已删除的联系人不恢复资料投影。两段错误均保留，避免关系错误掩盖或阻断其他资料刷新。群频道仍走原群处理路径。

本轮修改文件（相对于 Bridge 根目录）：

- internal/business/friends.go：channelUpdate 增加指定用户资料刷新。
- internal/business/profile_channel_test.go：真实命令解析及昵称恢复到旧快照值时的回归测试，检查用户身份与明确的 portrait_set。
- test/contract/profile-channel-update.anonymized.json：保存脱敏真实回调结构。

新增发布脚本：06-release-and-deploy/verify-profile-name.py，账号和期望昵称均由参数传入，读取服务器安全配置且不打印凭据或头像签名地址。PHP、Android 本轮无改动，现有 QA APK 可继续使用。

## 测试、发布和回滚

- go test ./... 全部通过；go vet ./... 通过。
- 已部署到 ubuntu@101.32.11.27:/opt/hw-bridge，healthz=ok，readyz=ready。
- 二进制 SHA256：abb5cc447f7227d71891e521414aab00d47c3d2ee77bb8586b6145deb132eefd。
- 备份目录：/opt/hw-bridge.backup.20260930212552。
- 服务包：06-release-and-deploy/hw-bridge-client-profile-20260930.tar.gz。
- 服务包 SHA256：431cb6a409698c0377a5bcf7c562b413c5380df5ce36f5fffc566a0a7c68355b。
- 累计源码包：06-release-and-deploy/client-profile-20260930-source.zip，包含此前群提示、头像、状态指令及群解散补丁。
- 校验文件：06-release-and-deploy/client-profile-20260930-SHA256SUMS.txt。
- 部署日志：工作区 toolchain/deploy-client-profile.log。

在 06-release-and-deploy 执行的部署命令：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
```

预期 HW_BRIDGE_DEPLOY_OK；保留 .env、现有 Docker Compose 和数据库。本次无迁移。回滚时从上述备份恢复二进制和 Dockerfile，按现有 Compose 重建 Bridge，保留 .env 和数据；回滚将重新引入本次资料回调遗漏。

## 验证命令和预期

Bridge 服务器：

```sh
cd /opt/hw-bridge
curl -fsS http://127.0.0.1:8090/healthz
curl -fsS http://127.0.0.1:8090/readyz
sha256sum bridge
```

预期 ok、ready 和上述二进制哈希。在本机发布目录验证授权测试账号的当前昵称：

```powershell
$peerID='<授权叨叨UID>'
$expectedName='<当前期望昵称>'
$script64=[Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path './verify-profile-name.py')))
ssh -T -i C:/Users/Administrator/.ssh/hw_bridge_20260929 ubuntu@101.32.11.27 "echo $script64 | base64 -d | sudo -n python3 - $peerID '$expectedName'"
```

参数使用不含 shell 特殊字符的验收昵称。预期 PROFILE_NICKNAME_SYNC_OK 和头像存在；脚本最多做8次查询，未匹配会明确报错，不能当作同步成功。实际已分别验证第二轮临时昵称和恢复后的原昵称。

## 仍未完成的关键项

1. **当前测试会话的旧异常引用记录**：本轮重新验证 seq20–22 范围仍 HTTP500，seq35–37 新消息范围 HTTP200；客户端首次打开该单聊历史区为空。问题记录为我此前测试创建，详情见 [引用数据事故说明](REPLY_CONTRACT_INCIDENT_20260929.md)。需已有叨叨数据维护入口定点修正，不能靠重新部署 Bridge 或清空整个会话解决。
2. **反向已读开启场景**：当前发送方向 receipt=0，只完成关闭状态与未读角标验证；开启后腾讯已读的真实闭环仍待验收，不伪造回执。
3. APK 撤回/头像更新显示、管理员撤回权限、好友新申请完整闭环及账号停用/注销等其余未覆盖用例保留待验收。
4. 大视频、CDN配置及正式签名延续此前范围决定；本轮没有开展主备或可靠性专项。
