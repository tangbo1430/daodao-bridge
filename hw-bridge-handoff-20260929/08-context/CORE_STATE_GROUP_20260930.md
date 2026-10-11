# 消息状态与群生命周期收尾

## 本次结果

已修改、测试并部署到 101.32.11.27:/opt/hw-bridge。此版本包含此前头像同步与群系统提示重复修复。保持现有单实例部署，PHP 与 APK 本轮无需更新。

### 消息状态真实协议修复

现网 syncMessageExtra 指令没有 param 对象，channel_id/channel_type 位于外层接收信封。旧实现只读 param，导致“消息状态指令缺少频道”。现已保存脱敏真实报文，读取外层频道；已持久化的旧任务可按事件 ID 和接收代理身份从原 Inbox 恢复频道。仍缺少有效频道时明确报错，不猜测目标，也不读取其他代理的事件。

对当前授权测试会话的 10 条此类失败状态任务执行定向恢复，全部由 dead 转为 done。没有重放聊天内容，也没有开展全量历史死信清理。

### 群解散业务修复

实际临时群完成创建、重复创建返回同一映射、公告更新、成员同步及解散。随后发现再次解散返回 502，现已修复：

- 已完成双端解散时直接返回持久化结果。
- 叨叨已解散但腾讯清理失败时，仅继续腾讯镜像清理。
- 腾讯仅明确的群不存在错误 10010 按已清理处理；权限和其他错误继续返回。
- 叨叨拒绝解散时不得销毁腾讯镜像。

现网再次解散返回 HTTP 200、code 200、status 2、mirror_ready false。临时验收群已解散，原联调群保留。

### 会话与重启验证

部署重启后连续两轮 session_sync、tencent_roam_sync、message_extra_sync 均返回 HTTP 200/code 200。会话同步 channels=4、accepted=0；原 started_at=1790705761 保留；腾讯漫游 accepted=0、消息扩展 processed=0。证明本次重启继续使用持久游标，没有重复接收已有消息；不据此宣称所有离线、重新登录或 Token 失效客户端场景已验收。

## 修改文件

相对于 01-bridge/tencent-tangseng-bridge：

- internal/domain/business.go：消息状态指令保留外层频道类型和 ID。
- internal/business/message_state.go：外层频道及旧 Inbox 路由恢复。
- internal/store/store.go、internal/store/inbox_read.go：按事件与接收者读取原始叨叨事件。
- internal/business/state_command_contract_test.go：真实报文、旧任务恢复、身份隔离、无频道拒绝测试。
- test/contract/message-extra-envelope.anonymized.json：脱敏真实协议样例。
- internal/adapter/groups.go：可重复调用的腾讯镜像群销毁。
- internal/business/groups.go：解散已完成及部分完成恢复。
- internal/business/groups_test.go：测试上游状态及故障注入。
- internal/adapter/group_disband_test.go、internal/business/group_disband_test.go：不存在、权限拒绝及部分失败恢复测试。

新增发布验证脚本：06-release-and-deploy/verify-core-sync.py，账号由命令参数提供，无内置测试账号。

## 测试与产物

- go test ./... 全量通过，包含新增真实协议契约和业务测试。
- go vet ./... 通过。
- 实际临时群业务操作、10 条状态任务恢复及两轮同步验证通过。
- Bridge healthz=ok、readyz=ready。
- 本轮无 PHP/Android 修改，沿用已部署 PHP 与已交付 QA APK，不重复声明本轮执行 Android 构建或 PHP 测试。
- 全量测试日志：工作区 toolchain/core-state-group-go-test.log。
- 部署日志：工作区 toolchain/deploy-core-state-group.log。
- 二进制 SHA256：3a63fa3bd23f912e56841c0e05ed4778c95d2c46c110a4990152560f93fcdb06。
- 备份：/opt/hw-bridge.backup.20260930202105。
- 服务发布包：06-release-and-deploy/hw-bridge-core-state-group-20260930.tar.gz。
- 发布包 SHA256：509e34c138eaad07377bb87a56678b3507f1c17133740a0650a81779926bdeb1。
- 累计源码补丁：06-release-and-deploy/core-state-group-20260930-source.zip，包含原业务交付之后的群提示、头像与本轮修复；校验文件 core-state-group-20260930-SHA256SUMS.txt。

原 business-20260929 业务整包保留，重新部署须使用本次最新 Bridge 包或当前源码构建，不能单独使用原整包中的旧 Bridge。

## 部署命令（已执行）

在 06-release-and-deploy 目录执行：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
```

脚本保留服务器 .env，校验占位符、备份和二进制哈希，再按原 Docker Compose 部署。预期尾标志 HW_BRIDGE_DEPLOY_OK。需要回滚时还原上述备份的二进制和 Dockerfile，再按原 Compose 重建 Bridge，保留 .env 与数据库；本次没有新增数据库迁移。该备份为本轮群解散修复前版本，仍包含消息状态修复。

## 验证命令与预期

Bridge 服务器：

```sh
cd /opt/hw-bridge
curl -fsS http://127.0.0.1:8090/healthz
curl -fsS http://127.0.0.1:8090/readyz
sha256sum bridge
sudo docker compose ps bridge mysql redis
```

预期 status=ok、status=ready、哈希与上文一致、容器 Up。

在本机 06-release-and-deploy 目录，替换已授权账号参数：

```powershell
$ownerID='<授权腾讯UID>'
$peerID='<授权叨叨UID，而非用户码>'
$script64=[Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path './verify-core-sync.py')))
ssh -T -i C:/Users/Administrator/.ssh/hw_bridge_20260929 ubuntu@101.32.11.27 "echo $script64 | base64 -d | sudo -n python3 - $ownerID $peerID"
```

预期 6 次 HTTP 200/code 200，尾标志 CORE_SESSION_ROAM_EXTRA_OK。有新消息时 accepted 可以大于 0，不应把有新业务的场景误判为重复；重复空闲验证应不重复投递。日志 account restore complete 表示账号恢复流程结束，不单独证明每条代理已连通。仍出现 business sync pending 或消息状态频道错误不能记作通过。

头像和无重复群提示可沿用 verify-avatar-sync.py、verify-group-mute-fix.py，参数和预期见前两份修复文档。

## 尚需验收和外部条件

1. 叨叨主动撤回在 APK 的展示，以及 APK 打开消息后叨叨显示已读，已向用户请求实际操作，尚未收到本轮结果。
2. 用户主动修改头像后双端界面刷新；服务器头像内容一致已验证。
3. 新好友完整申请/拒绝/重申请、群邀请确认/权限、多华为成员、普通腾讯业务回归，仍需逐项客户端结果；本轮仅补齐上述群服务器操作，不能全部勾选。
4. 停用/注销需可销毁的授权测试账号，不能注销当前用户正在使用的账号；重新登录及离线界面待验收。
5. [单条引用测试记录](REPLY_CONTRACT_INCIDENT_20260929.md)仍待已有叨叨维护入口定点处理。新代码已修复，但旧异常数据不会随部署自动消失；它属于当前测试会话，不属于已排除的历史账号迁移。
6. 大视频按用户决定后置；跨地域中转/CDN需要实际入口配置，正式发布签名尚未提供。均未宣称完成。

本次不启动主备、Leader、Redis 租约、压测或 72 小时可靠性专项。
