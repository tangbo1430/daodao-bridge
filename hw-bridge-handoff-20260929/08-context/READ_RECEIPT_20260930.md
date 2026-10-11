# 逐条已读回执修复与真实验证

## 结论

已部署独立单实例 Bridge。腾讯发出、叨叨阅读的新消息 R3，Bridge 的 need_receipt、read_synced、tencent_receipt_synced 均为 true，腾讯漫游查询 IsPeerRead=1。此项服务端闭环通过，APK 已读标记仍待手机确认。

消息键：199833547_1790776675_1790776675；叨叨 ID：2105296089832755200，seq38。发送入口为腾讯 REST；不能替代 APK 发送验收。临时启用的发送方向 receipt 已恢复为 0，并经原生 GET 验证。

## 修复与测试

旧实现只调用 admin_set_msg_read 清会话未读，实际出现 read_synced=true 而腾讯 IsPeerRead=0。改为先调用 c2c_msg_read_receipt，再按原消息时间清未读；两者成功后才记录完成，失败不推进消息扩展游标。

接口依据：[逐条已读回执](https://cloud.tencent.com/document/product/269/104121)、[设置单聊消息已读](https://cloud.tencent.com/document/product/269/50349)。协议差异已保存脱敏契约。

- internal/adapter/message_state.go、read_receipt_test.go：逐条回执及未读时间边界，失败传播。
- internal/domain/message_state.go：新增兼容字段 tencent_receipt_synced，沿用 record_json，无数据库迁移。
- internal/business/message_state.go、message_records.go：成功状态与兼容旧记录，重复投递保留状态。
- internal/business/friends.go：channelUpdate 使对应联系人的回执设置缓存失效。
- internal/business/message_state_test.go、read_receipt_test.go、profile_channel_test.go：失败重试、回执关闭、重复事件、旧状态及缓存隔离测试。
- test/contract/tencent-read-receipt.anonymized.json：脱敏真实请求、响应和前后状态。
- 06-release-and-deploy/verify-message-read.py：只输出白名单字段，支持 --expect-read 严格断言。

go test ./...、go vet ./... 通过；补充缓存测试后对应 business 包测试通过。PHP、Android 无改动，沿用现有 QA APK。

## 版本与部署

- 二进制 SHA256：3481e93ed8fbeb97421563e5aa3b5588ce8a8de0eb51daa320c87a33a7d154d9。
- 备份：/opt/hw-bridge.backup.20260930215713。
- healthz=ok、readyz=ready；部署日志：工作区 toolchain/deploy-read-receipt.log。
- 发布包：06-release-and-deploy/hw-bridge-read-receipt-20260930.tar.gz。
- 累计源码：06-release-and-deploy/read-receipt-20260930-source.zip；校验文件：read-receipt-20260930-SHA256SUMS.txt。

在发布目录执行：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath 'C:/Users/Administrator/.ssh/hw_bridge_20260929'
```

预期 HW_BRIDGE_DEPLOY_OK。脚本保留 .env、Compose 和数据。回滚仅恢复上述备份的二进制及 Dockerfile 后重建 Bridge；新增 JSON 字段可保留，回滚会重新引入回执问题。

## 验证

Bridge 服务器执行 curl -fsS http://127.0.0.1:8090/healthz、curl -fsS http://127.0.0.1:8090/readyz、sha256sum /opt/hw-bridge/bridge，预期 ok、ready 和上述哈希。

在本机发布目录：

```powershell
$messageKey='199833547_1790776675_1790776675'
$script64=[Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path './verify-message-read.py')))
ssh -T -i C:/Users/Administrator/.ssh/hw_bridge_20260929 ubuntu@101.32.11.27 "echo $script64 | base64 -d | sudo -n python3 - $messageKey --expect-read"
```

预期 TENCENT_MESSAGE_RECEIPT_OK，已实际验证。脚本不打印密钥和签名 URL。

## 保留项

不批量回退旧游标，也未重放旧的回执状态。APK 展示、主动更换头像、好友完整申请链路等继续验收；当前会话旧异常引用记录仍按 REPLY_CONTRACT_INCIDENT_20260929.md 处理。未启动主备、Leader 或可靠性专项。
