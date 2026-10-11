# 业务收尾发布与验收记录

发布编号：business-closure-20261001（服务器北京时间 2026-10-01，本机日期为 2026-09-30）。本报告覆盖本轮代码、部署和实际验证；较早报告保留作为历史证据。

## 1. 当前结论

五阶段收尾代码已发布到现有 Bridge/PHP，新 QA APK 已上传，严格新注册事务已启用。旧账号、用户名和映射未迁移。单实例、原 Docker/MySQL/Redis/PHP 回调、腾讯 TUIKit 和双向单聊架构保持原样。未修改叨叨业务服务端或悟空核心。

**不能标记为全部业务验收完成。** 手机界面与真实厂商推送仍待集中协测，1004/1020 的原生事件语义仍缺对应真实报文，部分封禁/邀请确认场景仅有契约测试。下表区分代码、测试、部署和实际业务证据。

## 2. 五阶段交付清单

| 阶段/用例 | 已开发与测试 | 已部署 | 真实业务证据 | 结论/未验证项 |
|---|---|---|---|---|
| 发送失败提示 | HTTP/悟空原因分类，确定拒绝与临时故障分流；独立幂等提示任务、防回环、失败提示未送达不算成功 | Bridge/PHP | 专用非好友目标：原消息拒绝，提示任务仅一条且 done；重查仍仅一条 | 服务端通过；APK 提示待确认，拉黑/禁言拒绝的提示需手机复验 |
| 媒体失败 | 下载/上传/转码错误接入同一提示路径；受控媒体地址与不支持内容降级保留 | Bridge | 前轮已有图片/短视频/录音/文件传输证据，本轮错误分支为契约覆盖 | 已修复待手机确认；大视频按约定后置 |
| 离线推送 | 单聊/群 OfflinePushInfo，沿用 TUIKit entity 扩展；媒体摘要、群 ID/头像、系统提示抑制推送 | Bridge + QA APK | 契约校验字段；未模拟成真实厂商推送成功 | 待手机确认通知到达和点击路由 |
| Token 生命周期 | 实际登录时间、超过 7 天账号每日续期、并发刷新合并、401/认证失败刷新一次、相同 token 不重建代理 | Bridge | 重启后四个原有代理恢复；专用新账号登录/代理实际成功 | 失效/相同 token/并发等单元测试通过，真实强制失效场景待补验 |
| 封禁/租户停用 | 独立桥接状态与原因、关闭代理、PHP 状态投影、显式原生登录和连接复核恢复 | Bridge/PHP | 已有四个账号及专用账号状态通知落库、幂等确认；未封禁真实联调用户 | 原生封禁/租户停用/解封仅契约通过，待外部条件 |
| 注销与群主 | 持久化检查点；退出后读取实际群主，再转移腾讯群主/移除成员；清除新注册加密凭据 | Bridge/PHP | 专用 91673 真实注册后成为临时群主；注销成功，实际群主转给 74629，凭据清除，临时群解散 | 服务端通过；未注销当前联调账号，APK 注销 UI 未验 |
| 群回调与系统通知 | 群头像、角色、禁言入口、真实操作人、防回环、拒绝后重同步；原生 content/姓名/可见范围；1009 指定用户通知 | Bridge/PHP | 前轮真实角色/头像/禁言/成员操作；本轮真实群主转让/退群/解散；1001/1002/1008/2000 报文已脱敏保存 | 已修复待手机确认；1004/1020 不按编号猜含义，真实语义待采集；1009 实际确认流程待验 |
| 每日群未读清理 | 仅当前有效桥接群、对应华为用户原生身份、每日持久日期、不产生群已读回执 | Bridge | daily checkpoint 已产生；单元测试覆盖非成员跳过及不发群回执 | 服务端任务已运行，UI 未读变化待验 |
| 好友对账 | 原生状态对照腾讯实际分页好友；仅已确认影子关系，单向增删，保留申请中/单方删除；普通腾讯关系不参与删除 | Bridge | 全量测试；现网正常好友同步和会话同步未报错 | 实际制造关系差异再修复场景待集中验收 |
| 申请到期/系统用户 | 收到 15 天、发出 30 天清理凭据；过期同意重新申请；u_10000 支持、fileHelper 排除搜索/邀请/消息/同步/注销清理 | Bridge + QA APK | 过期重申请、排除不调用上游等契约通过 | 到期提示及系统联系人 UI 待验；Android 识别限定原生 UID 格式及 u_10000，非任意非数字账号 |
| 新注册 prepare/abort | 原生用户名/邀请码验证、独立随机加密密码、稳定 request_id、丢响应仅用已存身份确认、不复用他人账号 | Bridge/PHP | 专用原生账号 prepare 重试返回同一预留/UID；abort 重复成功；失效与身份冲突有契约 | 服务端通过；重名/无效邀请码/超时部分为契约，手机提示待验 |
| 新注册 commit | PHP 本地事务与 durable commit/abort；HMAC/Nonce；加密凭据恢复；10 分钟悬挂查询；旧 provision 接口保留 | Bridge/PHP + QA APK | 专用 91673：PHP 本地创建、原生注册、commit、confirmed 和代理成功；内网签名查询 200，重放 409，未签名 401 | 后端事务真实通过；验证码/注册页面完整手机操作待验 |
| 资料与会话回归 | 保留头像同步、会话/漫游/消息扩展持久游标、已读、撤回 | Bridge + QA APK | 本轮头像原生/公网 SHA256 一致；两轮会话/漫游/扩展同步成功且 accepted=0；前轮单聊/群撤回和反向 IsPeerRead=1 有记录 | 已修复待手机确认，个人头像主动更换后 1 分钟刷新未补验 |

新注册及注销专用账号由服务器安全配置中的邀请码自动创建并完成清理，现已不需要用户额外提供邀请码或可销毁账号才能证明这条后端事务。专用原生 abort 账号按方案保留为未完成注册，不调用不存在的管理员注销接口。多华为成员的接口路径已覆盖；两台实际手机界面仍未覆盖。

## 3. 修改文件与实现要点

完整可复现文件列表及 SHA256 见发布目录 `business-closure-20261001-SOURCE-MANIFEST.json`。这是相对交接压缩包的累计源码差异，包含前轮已经部署的改动。本轮主要文件如下。

- Bridge `internal/domain/failure.go`、`internal/engine/failure.go`、`internal/engine/engine.go`、`internal/adapter/native_errors.go`、`offline_push.go`、`internal/session/wukong.go`：错误分类、失败提示和推送。
- Bridge `internal/account/{registration,php_lookup,status,token_lifecycle}.go`、`service.go`，`internal/adapter/registration.go`、`internal/httpapi/{registration,account_status}.go`、`cmd/bridge/main.go`：注册预留/加密身份、状态通知和刷新生命周期、鉴权与重放防护。
- Bridge `internal/business/{friends,friend_reconcile,groups,account_lifecycle,daily_tasks}.go` 及对应测试，`internal/httpapi/server.go`：好友关系、申请有效期、系统用户排除、群通知、未读任务和注销群主处理。
- PHP `application/imaccess/service/{RegistrationService,BridgeService}.php`、`controller/BridgeInternal.php`、`application/api/{model/User,controller/Login}.php`、`application/imcallback/service/WorkerService.php`、`route/route.php`、`config/im_bridge.php`：事务、可靠通知、内部接口与入口开关；`UserTencent.php` 保留系统联系人问候抑制。
- Android `RegistActivity.java`、`RegistActivity2.java`：读取注册规则、邀请码要求、待确认状态和安全错误展示；TUIKit 的 `BridgeContactClient.java`、`ChatInfo.java`、`ProfileProvider.java`：受支持影子/系统会话识别。注册/登录响应中的凭据日志已移除。
- 发布脚本：沿用 `deploy-bridge-private-vpc.redacted.ps1`、`deploy-php-bridge-provision.redacted.ps1`；新增配置/验收脚本。PHP 脚本修复通过 SSH stdin 执行时 PHP 子命令吞掉后续脚本的问题，语法检查和实际重启均已核对。

## 4. 协议差异与测试

真实新注册第一次调用已返回原生 UID，后续登录因 41 字符设备标识返回 HTTP400“更新用户登录设备失败”。保留已生成身份，改为最长 40 字符 HTTP 登录设备标识后登录成功，没有重复注册。分别保存 `registration-captured-20261001.json`、`registration-short-device-captured-20261001.json`，并补契约。此处是实际请求结果差异，不推断未读取的数据库字段定义，也未改 PC(2) 的部署架构。

`group-system-captured-20261001.json` 是真实 Inbox 脱敏报文；其余命名 fixture 的测试为模拟契约。没有采到 1004/1020 就不为编号编造“踢人/退出”等含义，现使用真实 content、人员信息及可见范围。

测试记录：

- Bridge 最终 `go test ./...`、`go vet ./...` 通过，含发送失败、注册丢响应/幂等/冲突、加密缺配置、状态刷新、防重放、群主注销、单向好友、群未读与 fileHelper 回归。
- PHP 15 个发布 PHP 文件语法通过，6 组脚本：注册事务、门面、注销队列、媒体流、媒体源、消息状态回调全部成功。补测 reserve/confirm 失败不能报告桥接成功。
- Android `assembleQaRelease` 成功，252 tasks。存在已有厂商 SDK/R8 可选类警告；不以编译通过代替实际厂商推送。
- 后端真实验收：`PHP_NATIVE_REGISTRATION_TRANSACTION_OK`、`DISPOSABLE_OWNER_TRANSFER_AND_CREDENTIAL_ERASE_OK`、`DISPOSABLE_TEMP_GROUP_DISBANDED`、`NONFRIEND_FAILURE_ONCE_VERIFIED`、`AVATAR_SYNC_SHA256_OK`、`CORE_SESSION_ROAM_EXTRA_OK`。
- 最新健康：healthz=200/ok、readyz=200/ready；PHP/worker Up。状态通知、注册协调、业务同步 pending 和 panic 日志在最后 5 分钟均为 0。
- Outbox 快照：done=18513、dead=256，未出现 pending/processing 行。dead 含原有 255 条和本轮真实非好友确定拒绝 1 条；未回放历史死信，也未开展死信专项。该快照不构成长期稳定性结论。

## 5. 发布物与版本

- Bridge：`hw-bridge-business-closure-20261001.tar.gz`。
- PHP：`php-business-closure-20261001.tar.gz`（15 个 PHP 文件及 SHA256SUMS）。
- 累计源码/测试/部署与记录：`business-closure-20261001-source.zip`，覆盖到原交接包相同相对路径。
- 校验：`business-closure-20261001-SHA256SUMS.txt`、`business-closure-20261001-SOURCE-MANIFEST.json`。
- QA APK：`qch-business-closure-20261001-qa.apk`，84,731,822 字节，SHA256 `ec919eb24b6cd199a76584dea338bea17170220b26b5979ca308f8509260485f`。
- 下载：https://admin.awhaha.com/downloads/bridge-qa-20261001/qch-business-closure-20261001-qa.apk 。公网 HTTP200，Range 支持，服务器文件哈希与本机相同。此为联调签名，正式签名未配置。
- 最终 Bridge 二进制 SHA256：`6d625bd6226a6bd65a96f549c05a3d9c71b0a2d78c40516722a4d6359e37e6d6`。
- Bridge 紧邻备份 `/opt/hw-bridge.backup.20261001012115`；PHP 备份 `/root/IM/backup-php-bridge-20261001005716`。
- 新注册开关当前 true；开启前环境备份 `/root/IM/YCH-IM-deploy/.env.before-strict-registration-20261001011550`。只修改开关，真实密钥未覆盖、未输出、未入包。

## 6. 部署、验证与迁移

在 `06-release-and-deploy` 执行，以下 key 为本机私钥路径，私钥内容不得复制进包：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
$key='C:/Users/Administrator/.ssh/hw_bridge_20260929'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath $key
./deploy-php-bridge-provision.redacted.ps1 -SshKeyPath $key
./test-php-closure.ps1 -SshKeyPath $key
```

预期 `HW_BRIDGE_DEPLOY_OK`、`REGISTRATION_SCHEMA_OK`、`PHP_BRIDGE_PROVISION_DEPLOY_OK`、`PHP_CLOSURE_CHECK_OK`。已在本轮实际执行；不必为查看结果重复部署。脚本拒绝现网关键配置占位符，保留 `.env` 和 Docker 数据卷。

Android 复现构建（qch 目录，SDK 路径通过本机 `local.properties` 或环境提供，不入发布包）：

```powershell
$env:JAVA_HOME='C:/Users/Administrator/Desktop/tx2daodao/toolchain/jdk11/jdk-11.0.32.1+1'
$env:ANDROID_HOME='C:/Users/Administrator/Desktop/tx2daodao/toolchain/android-sdk'
$env:GRADLE_USER_HOME='C:/Users/Administrator/Desktop/tx2daodao/toolchain/gradle-cache'
& 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/gradle-7.6.4/bin/gradle.bat' assembleQaRelease --no-daemon
```

预期 `BUILD SUCCESSFUL`；输出 `app/build/outputs/apk/qaRelease`。正式签名从安全配置加载，本包不含签名文件或生产口令。

数据库为增量：Bridge 沿用既有 002—005 迁移及 business cursor 持久化注册/任务，不重置游标；PHP 安装器幂等创建按现有前缀命名的 `im_bridge_registration`、`im_bridge_account_status`，不保存原生明文密码/token。新注册凭据加密使用服务器现有 `BRIDGE_ENCRYPTION_KEY`；旧账号恢复路径未替换。既有 Inbox/Outbox 保留。

安全验证命令（只打印筛选后的结果）：

```powershell
$b64=[Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path './verify-closure-state.py')))
ssh -T -i $key ubuntu@101.32.11.27 "echo $b64 | base64 -d | sudo -n python3 -"
$b64=[Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path './verify-closure-internal.py')))
ssh -T -i $key ubuntu@101.32.11.27 "echo $b64 | base64 -d | sudo -n python3 - --bridge"
$r=Invoke-RestMethod -Method Post -Uri https://admin.awhaha.com/api/login/show_reg
$r.data | Select-Object bridge_strict_registration,bridge_invite_required
```

预期 ready、PHP_LOOKUP=200、PHP_NONCE_REPLAY=409、PHP_STATUS_UNSIGNED=401、BRIDGE_NONCE_REPLAY=409、两个注册开关 True。真实注册/注销验收脚本会产生业务数据，已验证后不要作为日常健康检查重复运行。

Windows 下载：

```powershell
curl.exe -fL 'https://admin.awhaha.com/downloads/bridge-qa-20261001/qch-business-closure-20261001-qa.apk' -o qch-business-closure-20261001-qa.apk
Get-FileHash ./qch-business-closure-20261001-qa.apk -Algorithm SHA256
```

## 7. 回滚边界与命令

新注册入口需要暂停严格流程时，只切换非敏感开关，原有 commit/abort 重试和已建号保护继续运行：

```powershell
$b64=[Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path './enable-strict-registration.py')))
ssh -T -i $key root@23.251.34.150 "echo $b64 | base64 -d | python3 - --disable"
```

注意 `--disable` 是恢复旧注册入口规则，不是关闭整个注册页面；仅在决定恢复旧入口时使用。预期 `STRICT_REGISTRATION_ENABLED False`，环境新备份路径随输出返回。不恢复整个旧 `.env`，避免覆盖后续安全配置。

Bridge 最终批次可回退紧邻备份（仍包含命名注册和设备长度修复，只失去最后的 fileHelper 入口补齐）：

```bash
cd /opt/hw-bridge
sudo cp /opt/hw-bridge.backup.20261001012115/bridge ./bridge
sudo cp /opt/hw-bridge.backup.20261001012115/Dockerfile ./Dockerfile
sudo docker compose up -d --build --no-deps bridge
curl -fsS http://127.0.0.1:8090/readyz
```

该备份二进制 SHA256 为 `b8ab88ea7536c8e25155f79abc8406c17dd0c704d574ced15df643cab2f269bc`。**新注册账号上线后禁止直接回滚到不支持加密命名账号的早期版本**；保留注册表、cursor、加密配置和数据卷。PHP 当前兼容代码需继续保留，以免已提交事务失去确认路径；紧邻备份用于逐文件恢复，不能把回滚理解为删除新表或清空待确认记录。

## 8. 尚未完成及后置

集中手机操作见 `APK_CLOSURE_ACCEPTANCE_20261001.md`；逐项要求实际结果，不能把接口成功改记为 UI 通过。

- 1004/1020 原生报文语义、1009 实际邀请确认、封禁/租户停用与恢复、原生好友拒绝重申请等仍有真实场景证据缺口。采到后先补脱敏契约，再修映射。
- 腾讯成员变更回调未必提供单人禁言字段，当前通过 PHP 业务入口和原生同步覆盖禁言。不能宣称控制台不存在的回调字段已实测；需按实际回调继续核验。
- 主动更换头像 1 分钟刷新、双端 UI、离线推送点击、普通腾讯会话回归待手机协测。本机叨叨桌面输入此前受权限错误阻断，没有伪造本次 UI 点击记录。
- 旧异常引用记录仍见 `REPLY_CONTRACT_INCIDENT_20260929.md`，需已有维护入口定点处理；本轮未清空会话或修改叨叨服务。
- 跨地域中转/CDN 实际线路和正式签名缺部署配置，单列待办；未宣称已改善跨境丢包。
- 历史账号迁移、大视频、主备/Leader/租约、压测、死信专项与 72 小时稳定性按约定后置，本轮不会自动启动。
