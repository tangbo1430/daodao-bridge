# B 端群成员列表头像刷新补偿

## 问题与源码证据

用户确认：B 端点开 A 端个人资料时头像正确，群管理/全员列表头像错误。这说明至少个人资料入口已能获得正确文件，不能继续把问题归类为头像未上传。

只读核对工程内参考源码：

- `toolchain/upstream-tang-ios/Modules/WuKongDataSource/WuKongDataSource/Classes/WKDataSourceModel.m`：成员头像为空时使用 `users/<uid>/avatar`；头像字段为空不等于丢失。
- `.../WuKongBase/Classes/Sections/Common/WKSystemMessageHandler.m:284`：`memberUpdate` 触发群成员同步；304 行 `userAvatarUpdate` 只删除头像图片缓存并广播头像通知。
- `.../Sections/ConversationSetting/WKConversationGroupSettingVC.m:97`：群成员页面监听的是 `WKNOTIFY_GROUP_MEMBERUPDATE`，不直接监听头像通知。134 行超级群分支提前返回，因此不能宣称补发成员通知必然立即重绘所有 B 端页面。
- `.../Sections/Group/WKGroupManager.m`：同步到成员增量后才发成员更新 UI 通知。
- 原生参考 `modules/user/api.go:443` 上传头像只对好友发送 `userAvatarUpdate`；`modules/group/api.go:1966` 的用户身份成员更新接口可更新自己的成员版本并发送原生 `memberUpdate`。

这些是本地参考版本的源码证据，不代表已拿到测试手机准确版本。未修改 B 端客户端或原生服务端。

## 已补齐的 Bridge 缺口

原 Bridge 上传头像后会刷新个人频道资料，但没有刷新群成员资料。现在：

1. 头像已有成功上传指纹后，按“群 + 当前用户 + 头像内容指纹”判断是否需要成员刷新。
2. 用该用户身份读取原生有效成员，再调用已有 `PUT /v1/groups/<group>/members/<自己的UID>`，请求体 `{}`。
3. 原生服务增加成员版本并发出 `memberUpdate`；不修改昵称、备注、角色或头像路径。
4. 成功才记录检查点；失败不记成功。重试与 PC(2) 收到的通知回流不会无限重复触发。
5. 个人资料同步覆盖已存在的群；群同步覆盖新群和未完成的刷新。已退群/解散群跳过。

没有伪造系统命令、改动普通腾讯会话或修改 .env。本轮无 PHP/APK 改动、无数据库迁移。

## 修改文件

Bridge 下：

- `internal/business/member_avatar_refresh.go`：成员头像刷新检查点及用户身份调用。
- `internal/business/groups.go`：原生成员核对及映射持久化后执行一次刷新。
- `internal/business/profiles.go`：个人资料同步后检查所在群的待刷新项。
- `internal/business/member_avatar_refresh_test.go`：实际契约响应、失败重试、同图去重、新图刷新、退群/解散跳过、通知回流不重复。
- `test/contract/member-avatar-refresh-observed-20261002.json`：真实临时群空更新前后协议脱敏样例。

发布目录：`verify-member-refresh-contract.py`、`verify-member-avatar-refresh.py`。脚本复用 `verify-group-settings.py` 的鉴权及清理流程，只从服务器安全配置取得凭据。

## 验证与部署

- `go test ./...` 和 `go vet ./...` 通过。
- 原生契约实测：自己的成员版本增加，UID、昵称、备注、角色、状态、头像字段不变；临时群已清理。
- 部署后真实临时群 `dd_45a4a722fa1c45a985401b70deb4a4a5`：Bridge Inbox 中捕获一次该用户的原生 `memberUpdate`，刷新检查点与成功上传指纹一致。
- 连续三次群同步后，原生成员版本未再次增加、通知仍一条、腾讯 NextMsgSeq 不变，没有新增聊天提示。临时群已清理。
- 已对测试用户 29512 所在的 6 个有效群执行补偿同步，均返回 200。
- `/healthz`、`/readyz` 均 HTTP 200。
- 现网 Bridge SHA256：`4c736bf586df349e040c597f5e97306be751d46330e345c564f01e3d46954cf7`。
- 备份：`/opt/hw-bridge.backup.20261002125600`。
- 发布包：`hw-bridge-member-avatar-refresh-20261002.tar.gz`。
- 发布包 SHA256：`3f363230e2a93a1e6b6c80aad423e3f1f487783e8e335ea0ebe01655f5ba1131`。

## 实际部署及验证命令

项目根目录 PowerShell：

```powershell
$env:GOCACHE = "$PWD/go-build-cache"
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-bridge-private-vpc.redacted.ps1 -GoExecutable "$PWD/toolchain/go/bin/go.exe" -SshKeyPath "$env:USERPROFILE/.ssh/hw_bridge_20260929"
scp -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ./hw-bridge-handoff-20260929/06-release-and-deploy/verify-member-avatar-refresh.py ./hw-bridge-handoff-20260929/06-release-and-deploy/verify-group-settings.py ubuntu@101.32.11.27:/tmp/
ssh -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ubuntu@101.32.11.27 'sudo -n python3 /tmp/verify-member-avatar-refresh.py 29512 d4e4642e820848f48ec3ed90a20b91db'
```

预期：`NATIVE_MEMBER_UPDATE_CMD_OBSERVED 1`、`MEMBER_REFRESH_NO_REPEAT_OR_CHAT_LOOP`、`TEMP_GROUP_CLEANED`。

Bridge 服务器回滚，仅替换二进制并保留环境配置：

```sh
sudo cp /opt/hw-bridge.backup.20261002125600/bridge /opt/hw-bridge/bridge
cd /opt/hw-bridge
sudo docker compose up -d --build bridge
curl -fsS http://127.0.0.1:8090/readyz
```

## 手机复验及未闭环边界

无需更换 A 端 APK。B 端退出“全员/群管理”后重新进入，核对 A 端头像；再由 A 端换一次头像，分别确认 B 端资料页和群成员页。Android、iOS 分别记录结果。

本次已证明补发通知到达 Bridge 的原生代理，不能据此声称 B 端手机已收到或 UI 已刷新。原生头像清缓存通知仍只发好友；非好友的本地旧图片缓存、超级群页面忽略成员刷新、实际群管理插件不同版本的行为，仍需真实页面验证。不能通过清空聊天数据或修改 B 端服务端绕过这些边界。
