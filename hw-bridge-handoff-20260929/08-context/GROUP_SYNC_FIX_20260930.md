# 群禁言系统提示重复修复

## 定位

用户反馈聊天中不断出现禁言、解除禁言。现网联调群历史记录显示，约每分钟新增三条管理员系统消息：群资料更新（OpType 6、ShutupAll 0）、成员角色更新（OpType 5）、成员解除禁言（OpType 7、ShutupTime 0）。

根因是 RunSync 每分钟通过 SyncAccount 同步群，而 syncGroup 无条件调用 modify_group_base_info、modify_group_member_info、forbid_send_msg。相同状态的写操作仍会产生腾讯系统提示。已确认这条重复写入路径；现网查询时全员禁言为 Off，并未发现这些记录代表实际反复开关禁言。管理员回调原有防回环分支保留。

腾讯解除禁言后，ShutUpUntil 可返回已经过去的取消时间而非 0，必须按当前时间判断。真实返回已脱敏保存为 test/contract/group-repeated-tips.anonymized.json。

## 修改文件

均相对于 01-bridge/tencent-tangseng-bridge：

- internal/adapter/groups.go：读取群资料、成员角色和禁言截止时间；兼容 ShutUpUntil/MuteUntil。
- internal/business/groups.go：按腾讯实际状态比较，只更新变化字段；只加入缺少成员；过期禁言视为未禁言；禁言时间允许 5 秒传输和取整偏差。
- internal/business/groups_test.go：测试上游保存真实投影状态。
- internal/business/group_projection_test.go：无变化重复同步、真正禁言/解除/角色变化、远端状态偏离修正测试。
- internal/adapter/group_projection_test.go：脱敏真实返回契约测试。
- test/contract/group-repeated-tips.anonymized.json：现网系统消息与资料契约。

PHP、Android 无需修改或重新安装，数据库无需迁移。生产 .env 保留。

## 测试与部署

- go test ./... 通过。
- go vet ./... 通过。
- 当前部署二进制 SHA256：61f5760d395235c651e360e2ba27eb9907b227a10a70ae6a760980f1ce0cee54。
- 旧版本备份：/opt/hw-bridge.backup.20260930192814。
- 发布包：06-release-and-deploy/hw-bridge-group-sync-fix-20260930.tar.gz。
- 发布包 SHA256：fe7ad594ad539c128be5a71ee8ceb1eac3b3a2563cade5d5778427854297ebce。
- 增量源码包：06-release-and-deploy/group-sync-fix-20260930-source.zip，校验文件 group-sync-fix-20260930-SHA256SUMS.txt。
- 部署日志：工作区 toolchain/deploy-group-mute-fix.log。
- 全量测试日志：工作区 toolchain/group-mute-fix-go-test.log。

从 06-release-and-deploy 运行的实际部署命令：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
```

预期部署日志：HW_BRIDGE_DEPLOY_OK。容器重建时可能短暂连接重置，随后 healthz/readyz 必须成功。

## 验证

现网主动连续三次 group_sync：HTTP 200、业务 code 200。NextMsgSeq 前后均为 3142，MuteAllMember 均为 Off；没有新增系统消息。healthz 为 ok，readyz 为 ready。

随后观察至服务器 UTC 2026-09-30 11:30:21，覆盖启动同步及两个自动定时周期；累计三轮、每轮三次主动同步均成功，群 NextMsgSeq 始终为 3142，未出现新的刷屏提示。

验证脚本使用服务器 .env，不输出签名或密钥。Windows PowerShell 在 06-release-and-deploy 运行，替换已授权的群和操作人：

```powershell
$groupID = 'dd_<待验证群号>'
$ownerID = '<已授权华为账号>'
$script64 = [Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path './verify-group-mute-fix.py')))
ssh -T -i C:/Users/Administrator/.ssh/hw_bridge_20260929 ubuntu@101.32.11.27 "echo $script64 | base64 -d | sudo -n python3 - $groupID $ownerID"
ssh -T -i C:/Users/Administrator/.ssh/hw_bridge_20260929 ubuntu@101.32.11.27 "curl -fsS http://127.0.0.1:8090/healthz && curl -fsS http://127.0.0.1:8090/readyz"
```

预期输出：三次 sync http 200 code 200，前后 next_seq 一致，GROUP_SYNC_NO_NEW_TIPS_OK。验证期间若有人发消息，序号会正常变化，需要检查新消息后再判断。

已写入腾讯历史的系统提示保留，客户端可能继续加载旧记录；验收应观察修复之后的新消息时间。此补丁不代表媒体、历史引用消息和其他未验收功能已完成。

另见后台既有问题：本次观察到 owner=74629 的好友同步也出现 friend_delete 30003，不能全部归类为旧账号问题。本补丁只修复群重复写入，好友同步错误尚未修复。

协议参考：[腾讯群详细资料](https://cloud.tencent.com/document/product/269/1616)、[腾讯群成员详细资料](https://cloud.tencent.com/document/product/269/1617)。
