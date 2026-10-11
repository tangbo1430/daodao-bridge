# 管理员提示修复（2026-10-01）

## 结论

管理员身份同步已生效，叨叨原生接口仅发送 memberUpdate，缺少可见提示。腾讯镜像已有原生管理员通知：本次真实报文设置为 OpType=4 / Role=300，取消为 OpType=5 / Role=200；APK TipsMessageBean 已支持两种提示。因此没有重复向腾讯补发桥接提示。

本次只修改 Bridge。同步确认普通成员与管理员之间的真实变化后，通过现有 Inbox/Outbox 向原生群发 type=2000 提示，使用在群内的华为用户原生身份（优先已映射的群主）。不使用管理员服务身份绕过权限。首次导入群、无变化同步、失败且未生效的操作不产生成功提示。

提示键包含群、成员、角色变化和原生成员版本；入队先于推进群快照，投递成功后保存独立检查点。发送失败保留重试；已被后续角色变化取代的待发提示不再发送。PC(2) 回来的同一提示通过标记作为 Inbox 消费，不转回腾讯，不触发新角色修改。

## 修改文件

- Bridge `internal/business/group_role_notice.go`：角色变化通知入队、恢复、发送和完成检查点。
- `internal/business/group_role_notice_test.go`：首次同步、重复同步、身份缺失/重试、快照未提交、取消/过时提示、多代理回声防回环测试。
- `internal/business/groups.go`：同步实际角色后入队。
- `internal/business/friends.go`：内部通知任务分发。
- `internal/domain/message_metadata.go`：PC(2) 消息保留明确的桥接提示标记。
- `internal/domain/business.go`：角色提示回声交给既有消息防回环分支。
- `test/contract/group-role-notice-observed-20261001.json`：脱敏真实观测，与模拟单元测试区分。
- `06-release-and-deploy/verify-group-role-notice.py`：创建临时群，双入口设置/取消、重复同步、消息数量与清理断言。依赖同目录 verify-group-business.py；目前验证登录路径适用于既有 tx_ 用户，参数传入账号，不硬编码账号。

## 测试与真实验证

- `go test ./...` 通过；`go vet ./...` 通过。
- 临时群 `dd_bbccea759958409e86d81b93457c0957`：Bridge 门面设置/取消、原生管理员接口设置/取消，共四次；原生角色 2/0、腾讯 Admin/Member 一致，原生可见提示共四条，腾讯角色提示共四条。
- 原生接口验证使用群主对应的华为账号原生身份，绕过 Bridge 操作门面验证 memberUpdate 的处理；不是由原生客户端手工操作，也不是第二个华为账号的被设置者 UI 验收。
- 连续三次 group_sync 后，原生提示仍四条，腾讯 NextMsgSeq 不变，没有提示回流。
- 三个本轮临时群 `dd_620e0c85fd444e1081574904b38670ae`、`dd_51f51f3f4c0e44cdb92d2adb97ec1914`、最终验证群均已解散。
- 健康检查：healthz 返回 ok，readyz 返回 ready。

## 发布与回滚

已部署 `101.32.11.27:/opt/hw-bridge`，保留服务器 .env；备份 `/opt/hw-bridge.backup.20261001193215`。

二进制 SHA256：`92434f69b7721bb2c254574f78fccc08fcc0a4d8976232d3006ad31d002fa410`。

发布包：`06-release-and-deploy/hw-bridge-group-role-notice-20261001.tar.gz`。对应增量源码与清单在相同目录的 `group-role-notice-20261001-source.zip`、`group-role-notice-20261001-SHA256SUMS.txt`。无需数据库迁移；APK、PHP、iOS、叨叨服务与悟空核心未修改。

工作区根目录部署命令（已执行）：

```powershell
$env:GOCACHE = "$PWD/go-build-cache"
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-bridge-private-vpc.redacted.ps1 -GoExecutable "$PWD/toolchain/go/bin/go.exe" -SshKeyPath "$env:USERPROFILE/.ssh/hw_bridge_20260929"
```

接口复验命令（会创建并最终解散临时群）：

```powershell
scp -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ./hw-bridge-handoff-20260929/06-release-and-deploy/verify-group-role-notice.py ./hw-bridge-handoff-20260929/06-release-and-deploy/verify-group-business.py ubuntu@101.32.11.27:/tmp/
ssh -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ubuntu@101.32.11.27 "sudo -n python3 /tmp/verify-group-role-notice.py <已授权华为UID> <已授权原生UID>"
```

预期输出：FACADE_PROMOTE_OK、FACADE_DEMOTE_OK、NATIVE_PROMOTE_OK、NATIVE_DEMOTE_OK、REPEATED_SYNC_NO_DUPLICATE_OR_LOOP_OK、TEMP_GROUP_CLEANED。

服务器回滚命令：

```sh
cd /opt/hw-bridge
sudo cp /opt/hw-bridge.backup.20261001193215/bridge /opt/hw-bridge/bridge
sudo docker compose up -d --build bridge
curl -fsS http://127.0.0.1:8090/readyz
```

回滚不会撤销已有管理员身份或删除历史提示；旧版本不处理新增的内部通知任务，回滚前应确认该类任务已完成并保存未完成任务记录。

## 手机最后确认与限制

1. APK 设置一名叨叨成员为管理员：被设置者停留群内，检查出现“某某已被设置为群管理员”，成员页显示管理员。
2. 取消管理员：出现“某某的群管理员身份已取消”，成员页恢复普通成员。
3. 原生群主设置/取消一个华为成员：APK 检查系统提示与角色，普通腾讯群保持原行为。
4. 等待一分钟或重新进入群，提示不重复。

以上手机显示仍待确认，不能将群历史与接口通过当作被设置手机已显示。若原生群全员禁言且可用的华为发送者无发送权限，提示会遵守原生权限失败并保留重试记录，不能以服务管理员身份绕过。未在本轮验证第二个华为账号的受任 UI 或原生群主手机操作。
