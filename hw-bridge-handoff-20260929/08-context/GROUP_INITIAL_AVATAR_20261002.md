# 首次建群头像通知归属修复

客户端日期：2026-10-01（America/Los_Angeles）。产物及服务器备份保留服务器的 20261002 标识。

## 原因与修改

截图第一条 `administrator 修改了群头像` 来自镜像初始化。原逻辑先 `create_group`，随后单独写 `FaceUrl`；后续用户主动编辑已有操作人信息，但初始化没有。

本次 Bridge 修改：

- `internal/business/groups.go`：创建镜像前读取头像，可用则随 `create_group` 的 `FaceUrl` 一起传入；不存在时允许建群，后续第一次填充空头像使用持久化的建群身份。主动编辑继续保留实际操作者；不把代理接收账号当操作者。
- `internal/business/group_profile_operator.go`：创建前保存初始群主的原生 UID；重试不覆盖。首次头像补齐复用既有精准回调去重。
- `internal/business/groups_test.go`：测试服务模拟腾讯保存创建时的 Name、Notification、FaceUrl。
- `internal/business/group_initial_avatar_test.go`：覆盖创建直接带头像、初始 404 后补齐、原生创建者与代理接收者不同、重复同步、非 404 错误。
- `06-release-and-deploy/verify-group-initial-avatar.py`：两种入口创建真实临时群，验证首次头像通知、重复同步及后续编辑归属，结束自动清理。

API 依据：[腾讯创建群组接口的 FaceUrl 字段](https://cloud.tencent.cn/document/product/269/1615)。本次未修改 APK、PHP、叨叨服务或悟空核心，无数据库迁移，保留服务器 .env。

## 验证证据

- `go test ./...`、`go vet ./...` 均通过。
- 真实 A 端同路径门面创建群：`dd_5a6c45f1e5a34683bf0c8d35fe2513b1`。
- 真实原生接口创建群：`dd_556f7042e1fe4a11a487f0957d6af83d`，由原生事件自动建立镜像。
- 两次真实测试均走延迟头像补齐：各一条 FaceUrl 通知，操作人均为授权建群账号 29512；之后 Name 通知也为 29512。腾讯 Inbox 回调再次核验通过。
- 每群连续三次同步没有新增资料通知；两个临时群已解散。
- 创建时头像已可用路径有单元测试覆盖，本次两次真实测试没有命中该路径，不伪称已真实验收。
- `/healthz`、`/readyz` 均 HTTP 200。手机新建群界面仍待用户复验。

## 发布与回滚

现网二进制 SHA256：`f9aac5cc7240b76cfe82632b0316221a03f86dce720f62fb9671bca67c2b0f16`。

包：`06-release-and-deploy/hw-bridge-group-initial-avatar-20261002.tar.gz`。

包 SHA256：`affb32ed5131aa433d06a1591358ba8835900c406c7e5085cd0e00a351b316b4`。

源码补丁与清单：`group-initial-avatar-20261002-source.zip`、`group-initial-avatar-20261002-MANIFEST.json`、`group-initial-avatar-20261002-SHA256SUMS.txt`。

实际部署命令（工作区根目录 PowerShell）：

```powershell
$env:GOCACHE = "$PWD/go-build-cache"
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-bridge-private-vpc.redacted.ps1 -GoExecutable "$PWD/toolchain/go/bin/go.exe" -SshKeyPath "$env:USERPROFILE/.ssh/hw_bridge_20260929"
```

预期 `HW_BRIDGE_DEPLOY_OK`。本次备份 `/opt/hw-bridge.backup.20261002132041`。

复验（脚本及 verify-group-settings.py 已在服务器 /tmp；测试账号由参数提供）：

```powershell
ssh -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ubuntu@101.32.11.27 'sudo -n python3 /tmp/verify-group-initial-avatar.py 29512 d4e4642e820848f48ec3ed90a20b91db'
```

预期各入口均输出 `INITIAL_AVATAR_NO_ADMIN_NOTICE`、`REPEAT_SYNC_AND_LATER_OPERATOR_OK`、`TEMP_GROUP_CLEANED`。

Bridge 服务器回滚：

```sh
sudo cp /opt/hw-bridge.backup.20261002132041/bridge /opt/hw-bridge/bridge
cd /opt/hw-bridge
sudo docker compose up -d --build bridge
curl -fsS http://127.0.0.1:8090/readyz
```

## 最短手机复验及范围

无需更新 APK。新建一个跨系统群，确认首次头像同步不再显示 administrator；再手动改头像一次，确认操作人仍是自己。既有历史通知未重写。

本次覆盖首次镜像创建与第一次空头像补齐，不将没有操作者信息的任意后台更新都冒认为用户操作。以前已存在的头像缓存、B 端群成员列表展示验收仍按 MEMBER_AVATAR_REFRESH_20261002.md 单列，不借本次结果宣称全部通过。
