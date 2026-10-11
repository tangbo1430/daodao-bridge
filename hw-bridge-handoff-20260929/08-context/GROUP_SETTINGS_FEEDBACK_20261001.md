# 群设置测试反馈处理（2026-10-01）

## 结论与验收边界

| 反馈 | 核对结果 | 本次处理及状态 |
|---|---|---|
| 加群方式显示旧值 | 确有问题。APK 读取腾讯镜像的 SDK 值；原生 invite 为 0/1，腾讯的禁止加群被错误映射成 invite=0 | 改从现有 PHP group_sync 读取原生状态；跨系统群展示“群邀请确认”开/关，不提供原生不支持的第三档。接口已验证，手机待确认 |
| 全员禁言闪退 | 发现未移除的群监听器、异步回调直接访问视图、列表空值风险；未取得本次闪退堆栈，不能认定唯一根因 | 加入生命周期检查、主线程更新、监听器移除、结果检查；保留输入草稿。编译通过，实际闪退复验待手机 |
| 设置管理员无提示 | 腾讯已有角色系统提示；原生提示缺口的上一批修复已部署 | 本次补 APK 角色与权限刷新；新账号通过 Bridge 门面和原生接口设置/取消，角色与原生提示均通过。被任命者手机显示待确认 |
| 开启群邀请 APK 不显示 | 与第一项共用同一映射缺陷 | 统一“群邀请确认”入口，保存后重读、失败显示原因；普通腾讯群保留原流程。手机待确认 |
| 群成员页缺少华为头像 | 本次账号的头像文件和成员头像路由正常，未证明是上传失败 | 原生返回有效 PNG，与当前源头像按 Bridge 相同算法转换后 SHA256 完全一致。未修改 iOS 或原生服务；实际客户端不显示原因仍待定位 |

额外发现并修复：新群头像资源暂时 HTTP404 时，原实现中断整次群同步，导致创建请求返回 502，也可能阻塞设置同步。现在仅对**群头像 404**保留现有头像并继续同步，下次再获取；用户头像 404、403、500 等仍报错，不吞掉权限或网络故障。

不将“禁止加群”与“邀请免确认”混为一谈。腾讯镜像 ApplyJoinOption 仍为 DisableApply，避免用户绕过原生群关系单独加入腾讯镜像。

## 修改文件

完整路径及各文件 SHA256 见 `06-release-and-deploy/group-settings-20261001-MANIFEST.json`。

- Bridge：`internal/httpapi/groups.go`（返回 invite/forbidden），`internal/business/groups.go`（缺失群头像不阻塞其他同步），`internal/media/service.go`、`avatar.go`、`avatar_test.go`（结构化 HTTP 错误及限定回退测试）。
- Android TUIContact：`TUIContactService.java`、`model/BridgeContactClient.java`（保留原调用兼容，提供群设置结果）。
- Android TUIChat：`model/ProfileProvider.java`、`presenter/GroupProfilePresenter.java`（读取原生状态、错误反馈、角色刷新）。
- Android TUIChat：`classicui/page/TUIGroupChatFragment.java`（禁言回调与视图生命周期），`classicui/page/GroupInfoFragment.java`（返回页面刷新）。
- Android TUIChat：classicui / minimalistui 的 `widget/profile/GroupInfoLayout.java`、`res/values/strings.xml`（邀请确认设置）。
- 验证材料：`test/contract/group-settings-observed-20261001.json`、`06-release-and-deploy/verify-group-settings.py`。

未修改 PHP 生产代码、iOS、叨叨服务或悟空核心，无数据库迁移。

## 测试证据

- Bridge `go test ./...`、`go vet ./...` 通过，包括区分群头像404、用户头像404、403和500的测试。
- Android `assembleQaRelease --no-daemon` 成功（5分33秒）；`verify-android-release-binding.py` 检查实际 APK DEX 通过。构建需 JDK11 和 `JAVA_TOOL_OPTIONS=-Dfile.encoding=UTF-8`；否则旧工具链会将中文资源路径处理为问号。
- 测试账号：yui11111（华为 UID29512），yui22222（原生成员）。未修改测试账号资料或既有业务群。
- 最终临时群 `dd_0027a69f78d14e829aa05656fac92c74`：双入口 invite 1→0；全员禁言 1→0；双入口管理员设置/取消，共4次角色变化与4条原生提示；连续3次同步腾讯消息序号不增加。全部接口断言通过。
- 群成员头像：实际 members 列表中该华为成员默认头像路径返回 HTTP200、1864822字节 PNG；与现有源头像按 Bridge 解码/编码得到的文件哈希相同。
- 首次临时群 `dd_75491d0c0fff47b39c709c3fa6bb8039` 创建部分成功后头像404；恢复相同 request_id，未重复创建。两个临时群均已解散。
- 首轮验证脚本错误地对超级群调用 membersync，导致验证中断；已按实际 group_type 使用 members，最终整轮通过。此为测试脚本问题，业务代码原本已有分支。
- healthz=ok，readyz=ready。APK 公网 HTTP200，Content-Length=84737978，支持 Range。
- 当前 adb 无连接设备，未完成手机闪退、通知和界面验收；本次未由客户端手工点击验证原生操作。

## 发布

- Bridge 包：`06-release-and-deploy/hw-bridge-group-settings-20261001.tar.gz`。
- 已部署 Bridge 二进制 SHA256：`c9d36f841d58249c5b7a9d6a9250a413347f46bb6897a40f3167a0bcc1f1ff1b`。
- Bridge 本次备份：`/opt/hw-bridge.backup.20261001235132`。此备份包含上一轮管理员提示修复与本次群设置返回字段，回滚将恢复头像404阻塞行为。
- APK：`06-release-and-deploy/qch-group-settings-20261001-qa.apk`，84737978字节。
- APK SHA256：`85d92be06db3f27306da0d1bda7a7a7e94d4be5177e15f9276227cf8c3686a54`。
- 下载：<https://admin.awhaha.com/downloads/bridge-qa-20261001/qch-group-settings-20261001-qa.apk>。
- 旧的 20260930 final 和 20261001 closure 下载别名均已替换；旧文件备份 `/root/IM/apk-backups/20261001235341`。保持服务器 .env 与签名配置不变。
- 源码增量、版本清单与 SHA256：`group-settings-20261001-source.zip`、`group-settings-20261001-MANIFEST.json`、`group-settings-20261001-SHA256SUMS.txt`。QA 签名，不代表正式发布签名。

### 部署与验证命令（工作区根目录）

```powershell
$env:GOCACHE = "$PWD/go-build-cache"
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-bridge-private-vpc.redacted.ps1 -GoExecutable "$PWD/toolchain/go/bin/go.exe" -SshKeyPath "$env:USERPROFILE/.ssh/hw_bridge_20260929"

scp -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ./hw-bridge-handoff-20260929/06-release-and-deploy/verify-group-settings.py ubuntu@101.32.11.27:/tmp/
ssh -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ubuntu@101.32.11.27 "sudo -n python3 /tmp/verify-group-settings.py <授权华为UID> <授权原生UID>"
ssh -i "$env:USERPROFILE/.ssh/hw_bridge_20260929" ubuntu@101.32.11.27 "curl -fsS http://127.0.0.1:8090/healthz && curl -fsS http://127.0.0.1:8090/readyz"
```

脚本会使用账号安全注册凭据，创建并清理临时群；目前该脚本登录路径针对本次新注册账号。预期输出 `INVITE_SETTING_READBACK_OK`、`MUTE_SETTING_READBACK_OK`、四次 `ROLE_AND_NOTICE_OK`、`REPEATED_SYNC_NO_TENCENT_NOTICE_LOOP_OK`、`NATIVE_GROUP_MEMBER_AVATAR_OK`、`TEMP_GROUP_CLEANED`。异常时也按当前 request_id 查找并清理部分创建的群，不清理其他群。

Windows 拉取：

```powershell
curl.exe -fL "https://admin.awhaha.com/downloads/bridge-qa-20261001/qch-group-settings-20261001-qa.apk" -o "$env:USERPROFILE\Downloads\qch-latest-qa.apk"
Get-FileHash "$env:USERPROFILE\Downloads\qch-latest-qa.apk" -Algorithm SHA256
```

Bridge 回滚（Bridge 服务器执行；不覆盖 .env）：

```sh
cd /opt/hw-bridge
sudo cp /opt/hw-bridge.backup.20261001235132/bridge ./bridge
sudo docker compose up -d --build bridge
curl -fsS http://127.0.0.1:8090/readyz
```

APK 下载别名回滚（PHP服务器执行）：

```sh
cp /root/IM/apk-backups/20261001235341/qch-business-closure-20261001-qa.apk /root/IM/YCH-IM-deploy/backend/public/downloads/bridge-qa-20261001/qch-business-closure-20261001-qa.apk
cp /root/IM/apk-backups/20261001235341/qch-business-final-20260929-qa.apk /root/IM/YCH-IM-deploy/backend/public/downloads/bridge-qa-20260930/qch-business-final-20260929-qa.apk
```

## 最短手机复验

1. 覆盖安装此 APK，用 yui11111 打开既有联调群设置。另一端开/关群邀请确认；APK 返回群聊再进入设置，两端值应一致。再由 APK 改一次确认反向一致。
2. 普通成员停留群聊并输入草稿，另一端开/关全员禁言；应不闪退，禁言时输入受限，取消后恢复，草稿保留。群主/管理员权限应符合原生规则。
3. 设置/取消管理员：被任命者在群里查看提示，成员页角色更新；返回或重复打开页面不重复提示。双方向各做一次。
4. 另一端关闭群设置页再打开，查看华为成员头像。若仍为空，记录账号、群名、客户端平台/版本和截图；后端头像存在不能代替这一步。
5. 普通腾讯群简单回归：原有三档加群设置与禁言仍按腾讯原流程运行。

以上手机确认前，不将五项反馈全部标记为已验收。
