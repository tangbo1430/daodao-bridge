# 2026-10-01 iOS 联调反馈处理

## 结论

四项仍不能统一标记通过。头像完整资料刷新补强已开发、测试、部署，并验证重复执行不重复刷新；iOS 界面待复验。图片偶发失败、被踢提示、撤回后新消息计数仍未关闭。

## 本轮证据

| 项目 | 已核实事实 | 当前状态 |
| --- | --- | --- |
| APK → iOS 图片 | 候选账号 81741 → 原生 c1fe… 的两条图片 MsgKey `1222519468_2525035365_1790833786`、`1222519469_2525035366_1790833823`：Bridge done；原生历史有 type=2、2616×1140、相对地址；原生 readed_count=1；Bridge 服务器下载最终资源 HTTP 200，191293/213020 字节，SHA256 与缓存一致 | 尚未确定是否就是用户反馈的失败图片。已读可能来自其他设备，服务器下载成功不代表 iOS 网络成功。需失败时间、账号、无气泡/加载失败区别及安装版本 |
| APK 更换头像 | 原代码昵称更新发生在头像上传前；仅换头像时不调用完整资料更新。新增上传完成后以本人身份调用原生 `/v1/user/current`，触发完整资料刷新；独立按头像内容保存完成检查点 | 已部署补强，待 iOS 确认；不能宣称已证明 iOS 缓存问题彻底消失 |
| APK 踢人 | 用户测试群 `dd_025db3aebc6347cb875cdf1284c6e3de` 已从两端移除目标成员。临时群自主复测：原生默认 join_group_remind=0，移除成功；另一临时群通过真实 `/setting` 接口设为 1 后重邀再移除，仍未在群历史获得踢人提示 | 原生私有通知是否到达被踢者尚无抓包，客户端 UI 未通过。不能把成员删除成功当成提示验收成功 |
| APK 撤回后残留 1 条新消息 | 29512 → d4e4… 的三条消息，Bridge revoked=true；原生 `/message/extra/sync` 返回 `is_mutual_deleted=1`。这套部署执行双向删除，不是普通 revoke 标记 | 原生删除已生效，客户端计数未验证。不得冒用原生收件人身份清空会话未读，也不能使用腾讯 UnreadMsgNum 覆盖含原生消息的未读数 |

临时群 `dd_ade0f048b52f4e01a745885d41b66824`、`dd_f3ce0abee077498dbf0dcd05e3b0ac46` 均已解散。首次尝试提醒字段放在群资料接口被拒绝，随后按客户端真实路由改用 `/setting` 成功；未修改用户现有群设置。

## 修改文件

- `internal/business/profiles.go`：头像上传后刷新资料；按内容哈希记录刷新完成；刷新失败沿用原队列重试，重试不重复上传；相同 URL 的新图片仍触发刷新。
- `internal/business/profile_avatar_refresh_test.go`：上传/刷新顺序、失败重试、身份、重复请求和同 URL 内容变化回归。
- `test/contract/ios-feedback-observed-20261001.json`：严格筛选脱敏真实观测，未伪造缺失的通知报文。
- `06-release-and-deploy/verify-avatar-refresh.py`：对指定已授权账号重同步其现有资料，验证完成检查点、幂等和头像字节保持。

未修改 PHP、Android、叨叨业务服务或悟空核心；APK 保持 `20261001-pretest-review` 版本，无需因这次服务端补强重新安装。

## 测试与部署

- `go test ./...` 通过；`go vet ./...` 通过。
- 按原脚本部署；备份 `/opt/hw-bridge.backup.20261001175526`，原服务器 `.env` 保留。
- 当前 Bridge 二进制 SHA256：`69cb569b61f2b37b5dea5373ed8afc24968e58b1bf634cbc9774eab8b69d5eba`。
- 对反馈账号 29512 重同步现有相同资料两次，任务 `evt_1790848611990114403`、`evt_1790848614563809631` 均 done；头像内容前后 SHA256 一致 `391fdcdfaed2dd3128800cfb403fd941dee70b6cf578d4b216a241155c437a25`；第二次未更新刷新检查点。
- 74629 为默认头像/非完整公网资料，验证脚本明确拒绝且未修改；随后使用反馈账号的现有资料验证，没有替换头像或改名。
- `/healthz`、`/readyz` 返回 200。

部署（在工作区根目录 PowerShell）：

```powershell
$env:GOCACHE = "$PWD/go-build-cache"
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-bridge-private-vpc.redacted.ps1 -GoExecutable "$PWD/toolchain/go/bin/go.exe" -SshKeyPath "$env:USERPROFILE/.ssh/hw_bridge_20260929"
```

验证（将 `<已授权账号UID>` 替换为具有现有公网头像的账号）：

```powershell
scp -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ./hw-bridge-handoff-20260929/06-release-and-deploy/verify-avatar-refresh.py ubuntu@101.32.11.27:/tmp/
ssh -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ubuntu@101.32.11.27 "sudo -n python3 /tmp/verify-avatar-refresh.py <已授权账号UID>"
```

预期：两条 `PROFILE_REFRESH_JOB_DONE`、`AVATAR_PROFILE_REFRESH_CHECKPOINT_OK`、`DUPLICATE_REFRESH_SUPPRESSED_OK`、`EXISTING_AVATAR_PRESERVED_SHA256`。该结果仅证明服务端处理，不替代 iOS UI 验收。

回滚（Bridge 服务器）：

```sh
cd /opt/hw-bridge
sudo cp /opt/hw-bridge.backup.20261001175526/bridge /opt/hw-bridge/bridge
sudo docker compose up -d --build bridge
curl -fsS http://127.0.0.1:8090/readyz
```

本轮没有表结构迁移；回滚保留 `.env` 和数据库，新增资料刷新检查点兼容旧程序。

## 后续复验

1. iOS 停留聊天页，APK 换头像，观察是否无需点头像即可刷新；再看会话列表。
2. 对应失败图片提供发送时间/双方账号和截图，区分无消息与图片无法加载，再核对同一条消息的资源和回执。
3. 被踢账号停留群内，记录操作后群提示、输入框状态以及重新进入后的状态；需要当前叨叨 iOS 版本及对应客户端日志定位私有通知处理。
4. 收件方停留会话列表，发送两条消息，仅撤回一条：保留另一条真实未读；再进入会话验证新消息条幅消除。不能通过整会话清零掩盖缺陷。

本机无对应 iOS 应用运行环境，公开上游 iOS 源码仅作协议参考，不能代替实际安装版本。参考：[原生服务](https://github.com/TangSengDaoDao/TangSengDaoDaoServer)、[iOS 客户端](https://github.com/TangSengDaoDao/TangSengDaoDaoiOS)。具体结论以本次现网报文为准。
