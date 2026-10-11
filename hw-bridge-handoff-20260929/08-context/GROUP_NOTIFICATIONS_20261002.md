# 群通知、管理入口、名称与头像复查（2026-10-02）

## 本次反馈结论

| 项目 | 根因与处理 | 验收状态 |
|---|---|---|
| 开启群邀请 A 端不显示 | 真实原生历史包含 type=1005、data.invite；Bridge 同步了群状态，却没有转发该类可见消息。现在补发邀请确认/群状态提示；名称、公告、禁言继续使用腾讯已有通知，避免重复 | 双入口四次开关收到四条腾讯提示，重复同步不增、不回环；APK显示待手机确认 |
| 两个“群管理”，下方不能点击 | 下方实际是旧的成员资料保护开关，被 XML 标成导航“群管理”；与后台管理员人数无关。跨系统群隐藏该无原生对应能力的控件；普通腾讯群恢复其真实标签与开关。保留上方真正的群管理入口 | 最终 APK 编译资源检查；手机点击待确认 |
| B 端群成员页缺 A 端头像 | 公网与内网读取同一头像均成功；另发现原生参考源码的首次上传通知早于 is_upload_avatar 标记提交，存在默认头像缓存竞态。Bridge 对首次上传补一次提交后的原生刷新，成功后才记完成；失败保留重试 | 模拟时序/失败重试测试通过；当前账号同内容原生刷新已执行。尚不能认定 Android/iOS 群成员页均恢复 |
| A 端群名称不正确 | 测试群“10.02”“10.02(2)”在原生与腾讯后台名称一致。APK 收到改名仅更新标题、未更新完整会话缓存；设置页还可能使用同步前的 SDK 名称。现在更新 groupName/chatName 两字段，设置页采用 Bridge 返回的原生名称 | 原生改名事件自动更新腾讯验证通过；具体错误显示位置尚待截图，手机待确认 |

Bridge、APK 已修改。PHP 生产代码、原生服务、悟空核心、B端客户端未修改；维持单实例、原目录、原 Docker 与 .env。无数据库迁移。

## 头像证据与限制

- 本次华为测试账号 UID29512，对应原生 UID d47a8ce270434980ade277377082409a；原生 is_upload_avatar=1。
- “10.02”“10.02(2)”成员接口 avatar 字段为空；交接的 iOS 参考代码在字段为空时生成 `users/<uid>/avatar`，因此空字段本身不能等同于头像丢失。
- 从本机运行的桌面客户端配置找到公网入口 `api.efgst.com`。通过该入口读取该用户头像：302→dl.lxode.com HTTPS→200，与内网获取文件一致；带缓存规避查询参数结果也一致。未输出带签名的真实下载 URL。
- PNG 1864822字节，SHA256 `e732ffc337a6c2b922dc29cec5e148678974be08cc2e179c8fe92ef5173a7329`。2026-10-02 已通过当事账号正常上传接口重新提交相同文件，前后字节哈希不变，以再次触发原生头像通知。
- 原生参考源码 `modules/user/api.go` 的上传处理先向好友发送 userAvatarUpdate，再提交 is_upload_avatar=1。此顺序是代码证据；本次用户的实际故障是否正由该竞态引起，未获得设备端记录证明。
- Bridge 新处理仅对实际首次上传（查询 flag=0）执行第二次正常上传；flag=1 正常上传一次；相同图片完成后不重复执行。第二次失败不提前保存完成指纹，后续重试可恢复。没有伪造系统 CMD 或以管理员身份替代用户。
- 原生通知范围仍仅为好友，不能保证刷新非好友群成员的本地缓存；B端手机的实际版本与入口尚未提供。该边界不以“HTTP200”代替 UI 验收，也不通过修改服务器解决。

## 修改文件

完整路径和哈希见发布目录的 `group-notifications-20261002-MANIFEST.json`。

- Bridge `internal/business/groups.go`：补齐 type1005 邀请/群状态通知，保留消息级去重和防回环。
- Bridge `internal/business/group_notice_test.go` 与 `test/contract/group-update-observed-20261002.json`：真实报文脱敏契约，重复投递、姓名替换、禁言/改名不重复测试。
- Bridge `internal/media/avatar.go`、`avatar_test.go`：首次头像提交后刷新、用户身份检查、失败恢复与指纹去重。
- Bridge `internal/business/profile_avatar_refresh_test.go`：更新真实用户资料协议的测试样例，保留“资料通知失败时不重复上传”的回归。
- Android `TUIChat/.../classicui/widget/profile/GroupInfoLayout.java` 和 `res/layout/group_info_layout.xml`：修复假群管理入口。
- Android `TUIContact/.../TUIContactService.java`、`TUIChat/.../model/ProfileProvider.java`：返回并使用原生群名。
- Android `TUIChat/.../presenter/GroupProfilePresenter.java`：群设置提示到达时重读状态、避免重复注册监听器、丢弃已失效的资料请求结果。
- Android `TUIChat/.../presenter/GroupChatPresenter.java`：改名事件和改名消息统一更新 groupName/chatName，避免返回后使用旧会话缓存。
- 脚本 `verify-group-notifications.py`、`refresh-own-avatar.py`（依赖同目录 `verify-group-settings.py`，参数传入授权账号，不硬编码测试账号）。

## 测试

- Bridge `go test ./...`、`go vet ./...` 通过。首次头像竞态测试是模拟时序，不能伪称真机复现。
- 真实临时群 `dd_b325700c9ac8419fa874be144a95e775`：原生 API、Bridge 门面各开启/关闭邀请确认，共四条可见腾讯提示；占位姓名已替换。随后原生改名，无主动 group_sync 干预，腾讯名称与持久状态自动更新。
- 连续3次额外同步后，腾讯邀请提示仍四条，原生设置事件仍四条，没有提示回流。临时群已解散。
- 最终服务二进制部署后 healthz=ok、readyz=ready。邀请真实验证先于最后的头像补丁；头像新增代码运行全量回归，未用现有账号重置原生首次上传标记制造测试。
- Android 构建使用 JDK11、UTF-8。第一次构建遇旧 AGP 资源增量索引问题，执行 app:clean 后重建成功；随后继续补齐 chatName 缓存字段并重新构建最终包。
- `verify-android-release-binding.py` 校验实际 R8 DEX 的公共业务地址字段。
- 当前 `adb devices` 无可用设备，因此不宣称真机 UI 或普通腾讯群完整回归已通过。

## 部署和复验

工作区根目录：

```powershell
$env:GOCACHE = "$PWD/go-build-cache"
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-bridge-private-vpc.redacted.ps1 -GoExecutable "$PWD/toolchain/go/bin/go.exe" -SshKeyPath "$env:USERPROFILE/.ssh/hw_bridge_20260929"
scp -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ./hw-bridge-handoff-20260929/06-release-and-deploy/verify-group-notifications.py ./hw-bridge-handoff-20260929/06-release-and-deploy/verify-group-settings.py ubuntu@101.32.11.27:/tmp/
ssh -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ubuntu@101.32.11.27 "sudo -n python3 /tmp/verify-group-notifications.py <授权华为UID> <授权原生UID>"
```

预期输出：四次 `INVITATION_TIP_OK`、`NATIVE_NAME_EVENT_SYNC_OK`、`INVITE_TIPS_NO_DUPLICATE_OR_LOOP_OK`、`TEMP_GROUP_CLEANED`。验证会创建并清理自身临时群。

Bridge 二进制 SHA256：`561cdb4e5af586236f4fe73d16b6db0efc4419148cc1572e297d38b593fe2f8b`。
最后一次备份：`/opt/hw-bridge.backup.20261002115927`；该备份已有本次群提示补丁，没有首次头像刷新补丁。
上一完整版本备份：`/opt/hw-bridge.backup.20261002115232`（上轮群设置修复）。

回滚整个本轮 Bridge 修改（Bridge服务器执行，保留 .env）：

```sh
cd /opt/hw-bridge
sudo cp /opt/hw-bridge.backup.20261002115232/bridge ./bridge
sudo docker compose up -d --build bridge
curl -fsS http://127.0.0.1:8090/readyz
```

已发送提示不删除，回滚不倒退群状态或头像完成记录。

## 最短手机复验

1. A端覆盖安装最终 APK；进入群设置，确认只有一个“群管理”导航，且能打开。
2. B端开/关邀请确认：A端聊天各出现一条对应提示；停留设置页时状态更新，返回重进后值仍正确。
3. B端修改群名：A端当前标题、返回后重新打开的标题、群设置名称一致；A端再改一次验证反向。
4. B端 Android/iOS 分别退出群设置页再打开，核对华为成员头像。若仍为空，保留截图、客户端版本与实际入口，不能以清空聊天数据作为修复。
5. 普通腾讯群检查原有加群方式、群管理和成员资料保护开关正常。

发布产物与最终 APK 校验信息见下方发布记录。

## 最终发布记录

- 最终 APK：`qch-group-notifications-20261002-r2-qa.apk`，84737990字节。最终 R2 构建成功（3分56秒），DEX 业务地址反射检查通过。
- APK SHA256：`2e97a3e97a043d88508d73fed8bce76b52b18ae58dbd79818a1ff710d7132afb`。
- Bridge 包：`hw-bridge-group-notifications-20261002.tar.gz`；源码与清单：`group-notifications-20261002-source.zip`、`group-notifications-20261002-MANIFEST.json`、`group-notifications-20261002-SHA256SUMS.txt`。
- 本轮发布前 APK 下载别名备份：`/root/IM/apk-backups/20261002120116`。该目录保存上一轮群设置版；R2为最终交付，未含r2的同日期APK是本轮中间构建。

```powershell
curl.exe -fL "https://admin.awhaha.com/downloads/bridge-qa-20261002/qch-group-notifications-20261002-r2-qa.apk" -o "$env:USERPROFILE\Downloads\qch-latest-qa.apk"
Get-FileHash "$env:USERPROFILE\Downloads\qch-latest-qa.apk" -Algorithm SHA256
```

APK 下载别名回滚（PHP服务器执行，不涉及PHP业务代码）：

```sh
cp /root/IM/apk-backups/20261002120116/qch-business-closure-20261001-qa.apk /root/IM/YCH-IM-deploy/backend/public/downloads/bridge-qa-20261001/qch-business-closure-20261001-qa.apk
cp /root/IM/apk-backups/20261002120116/qch-business-final-20260929-qa.apk /root/IM/YCH-IM-deploy/backend/public/downloads/bridge-qa-20260930/qch-business-final-20260929-qa.apk
```
