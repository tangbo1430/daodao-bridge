# 群资料通知操作人修复（2026-10-02）

## 结论与证据

用户截图中的 `administrator` 是腾讯群系统通知的操作人，不是聊天群名，也不是成员昵称映射失败。上一版群名缓存修复未覆盖本问题。

只读查询截图对应群 `dd_53b345ca4e3f4bae8d449866364854ce` 的腾讯历史，确认旧通知 `Operator_Account=administrator`，变更字段分别为 `GroupIntroduction + GroupName`、`GroupFaceUrl`。

腾讯官方接口明确支持 `From_Account`；省略时将 REST 管理员作为操作人：
https://cloud.tencent.com/document/product/269/1620

## 修改文件

Bridge `internal/business/`：

- `groups.go`：将已成功的用户群资料操作、原生 1005 事件的明确 UID 传入投影；不使用 PC(2) 接收者充当操作者；停止把群名重复写入群简介。
- `group_profile_operator.go`：按实际变更字段及当前值归属操作人，使用 `From_Account`；其他补偿字段单独处理；投影前持久化精确回调记录。
- `group_events.go`：先过滤投影回调，再处理真实用户变更。原生影子操作人无需拥有华为登录态就能消化其镜像回调。
- `group_profile_operator_test.go`：覆盖真实操作者、代理接收者、影子账号、过时事件、无关字段、回调重投、窗口边界及真实脱敏回调。

其他：

- `test/contract/group-profile-operator-observed-20261002.json`：本次临时群实际腾讯回调脱敏样例。
- `06-release-and-deploy/verify-group-profile-operator.py`：真实临时群验收及清理脚本。

本次无 PHP、Android、叨叨服务端或悟空核心改动，无数据库迁移。APK 继续使用上一版 R2。

## 测试与部署结果

- `go test ./...`：通过。
- `go vet ./...`：通过。
- 已部署 Bridge；现网二进制 SHA256：`daf4ce4ed01caf0062c70d5216bdd6f39d9d841acf124a4831eb12affdca3fef`。
- `/healthz`、`/readyz` 均 HTTP 200。
- 回滚备份：`/opt/hw-bridge.backup.20261002122013`。现有 `.env` 保留。
- 真实临时群 `dd_ec6f602a71a74616aca6d4f703c65e3a`：PHP 同路径 Bridge 门面改名、原生 API 改名、门面改公告、门面上传头像，四次腾讯通知均使用授权用户 `29512`，且该用户腾讯昵称存在。
- 每次操作仅一条对应腾讯资料通知；连续三次同步无新增通知；原生历史只有两次改名及一次公告，没有反向重复写入。
- 临时群已解散并核验状态。未修改用户现有业务群及个人资料。
- 真实回调记录四次变更均为 `29512`，而初始化默认头像同步仍是后台任务。

## 发布产物

- 服务包：`06-release-and-deploy/hw-bridge-group-profile-operator-20261002.tar.gz`
- 包 SHA256：`eba54a326cd230e2bad8be0264fe6d60aa582748386133b510144fcf6f16f2d3`
- 最终打包补入真实样例和契约测试；二进制与已部署版本相同。
- 源码补丁、文件清单和 SHA256：同目录 `group-profile-operator-20261002-*`。
- APK：保持 `qch-group-notifications-20261002-r2-qa.apk`，无需为本修复换包。

## 部署、验证与回滚

在项目根目录的 PowerShell 执行（脚本已实际使用）：

```powershell
$env:GOCACHE = "$PWD/go-build-cache"
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-bridge-private-vpc.redacted.ps1 -GoExecutable "$PWD/toolchain/go/bin/go.exe" -SshKeyPath "$env:USERPROFILE/.ssh/hw_bridge_20260929"
```

预期末尾：`HW_BRIDGE_DEPLOY_OK backup=...`；启动期间短暂连接重置以脚本后续健康检查结果为准。

验收脚本依赖同目录 `verify-group-settings.py`，已放置在服务器 `/tmp`。账号通过参数传入，密码及签名只在服务器读取现有安全配置：

```powershell
ssh -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ubuntu@101.32.11.27 'sudo -n python3 /tmp/verify-group-profile-operator.py 29512 d4e4642e820848f48ec3ed90a20b91db'
ssh -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ubuntu@101.32.11.27 'curl -fsS http://127.0.0.1:8090/healthz; curl -fsS http://127.0.0.1:8090/readyz'
```

预期 `PROFILE_OPERATOR_OK` 四项、`PROFILE_NO_DUPLICATE_OR_LOOP_OK`、`TEMP_GROUP_CLEANED`。脚本会创建并清理自己的临时群。

仅恢复二进制、保留现有 `.env`，在 Bridge 服务器执行：

```bash
sudo cp /opt/hw-bridge.backup.20261002122013/bridge /opt/hw-bridge/bridge
cd /opt/hw-bridge
sudo docker compose up -d --build bridge
curl -fsS http://127.0.0.1:8090/healthz
```

## 手机复验及边界

1. 使用当前 R2 APK，在有管理权限的跨系统群修改群名，再修改群头像。
2. 查看新产生的通知，应显示实际操作账号的昵称，不再把这次用户操作标成 `administrator`；群名不应再附带自动修改群简介的提示。
3. 另一端改一次群名，检查新通知中的人名；重新打开群及等待同步，不应重复提示。

实际腾讯通知数据及回调通过，手机界面仍待用户复验。历史 `administrator` 通知没有重写或删除。原生 `groupAvatarUpdate` 命令只带群号、没有操作人；纯后台恢复/初始化也没有真实操作者，本次不将其伪装成接收用户，仍保留技术通知。这些无操作者事件尚未实现 APK 的中性文案降级，不能把本次结论扩展为所有历史和后台通知均已替换。
