# 注册邀请码与产品文案修正

本轮由用户人工验收发现：注册页错误地要求输入“叨叨邀请码”，产品界面泄露了内部集成术语。按用户最新要求修正，覆盖早期计划和报告中“客户端必须填写原生邀请码”的规则。旧异常引用记录按用户要求退出本轮验收。

## 规则与修改

1. APK 原有邀请码恢复选填；用户填写时由 PHP 按产品原有 invitecode 表验证有效性和使用次数。未填写不阻止注册。
2. 产品邀请码不再传递给内部注册接口。Bridge 从服务器现有 TANGSENG_INVITE_CODE 读取渠道注册凭据；不向 APK/PHP 返回该值、不写入源码或包。原生仍执行其邀请码校验。
3. 服务器缺少渠道配置时明确返回“注册服务暂不可用，请稍后重试”，不会改为管理员代注册、跳过原生规则或假报成功。
4. 保留新用户名、随机加密密码、prepare/commit/abort、持久化重试和旧账号兼容。相同请求在服务器邀请码轮换后仍复用预留，不重复注册。
5. APK 注册、好友、群、系统提示以及 Bridge/PHP 面向用户的错误文案移除“叨叨”“桥接”等内部术语，改为“联系人”“群”“账号正在开通”等产品表达。协议字段、businessID、日志技术标识保持兼容；既有聊天内容和用户自定义昵称不会被篡改。

## 修改文件

- Bridge：internal/account/registration.go 及 registration_test.go；internal/adapter/business.go、native_errors.go；internal/domain/media.go；internal/business/friends.go、groups.go、message_state.go、profiles.go；internal/httpapi/friends.go、profiles.go。
- PHP：application/api/model/User.php、controller/Login.php；application/imaccess/service/RegistrationService.php、BridgeService.php、BlocklistService.php；controller/Bridge.php。新增 registration_product_test.php，扩展 registration_transaction_test.php。
- Android：RegistActivity.java、RegistActivity2.java；TUIContactService.java、BridgeContactClient.java；ProfileProvider.java、BridgeNoticeMessageBean.java、BridgeTipMessageBean.java。
- 验证脚本：test-php-closure.ps1、test-native-registration-transaction.ps1、verify-php-registration.php、delete-disposable-registration.php。测试不再跨服务器转送内部邀请码。

## 测试与现网证据

- go test ./...、go vet ./... 通过。
- PHP 发布文件语法检查和 7 组契约脚本通过，包括空邀请码、有效/无效/用尽的产品邀请码；产品邀请码不进入内部请求。
- 真实后端注册使用专用新账号 29797，用户邀请码留空：code=200、bridge_status=confirmed、PHP_NATIVE_COMMIT_READY=true。已调用注销清理专用账号，返回200，不用于用户人工验收。
- show_reg：bridge_strict_registration=true、bridge_invite_required=false。
- Bridge 已发布，二进制 SHA256：f0461ebe3489190b9ead220043d78a8a1a8fa9a0e9e9cf488c33e948fdf1aa4b。
- Bridge 备份：/opt/hw-bridge.backup.20261001102509；PHP 备份：/root/IM/backup-php-bridge-20261001102501。现网 .env 未修改。
- Android 构建、产物哈希及下载替换结果在下方“最终产物”记录；手机注册 UI 仍需用户继续验收。后端测试调用真实注册模型，不替代图片验证码页面的人工测试。

## 部署与验证命令

在 06-release-and-deploy 执行（已经执行，无需用户重跑）：

```powershell
$key='C:/Users/Administrator/.ssh/hw_bridge_20260929'
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
./deploy-bridge-private-vpc.redacted.ps1 -GoExecutable 'C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe' -SshKeyPath $key
./deploy-php-bridge-provision.redacted.ps1 -SshKeyPath $key
./test-php-closure.ps1 -SshKeyPath $key
$r=Invoke-RestMethod -Method Post -Uri https://admin.awhaha.com/api/login/show_reg
$r.data | Select-Object bridge_strict_registration,bridge_invite_required
```

预期 HW_BRIDGE_DEPLOY_OK、PHP_BRIDGE_PROVISION_DEPLOY_OK、REGISTRATION_PRODUCT_OPTIONAL_INVITE_OK、PHP_CLOSURE_CHECK_OK，注册开关 True/False。真实注册脚本会新建专用账号，不作为常规健康探针。

无新增数据库迁移。回退到上述紧邻备份会恢复错误的客户端原生邀请码要求，需同时回退 PHP/Bridge/对应 APK，不能只回退 Bridge 造成新版 PHP 空邀请码被拒；生产优先保留本修正。所有回退继续保留 .env、加密配置、数据表和已创建账号。

## 最终产物

- assembleQaRelease：BUILD SUCCESSFUL in 5m38s，252 tasks；保留既有厂商 SDK/R8 警告。DEX 与 resources.arsc 检查未发现“叨叨”“桥接提示”“桥接确认”等旧产品文案。
- APK：qch-business-closure-20261001-product-ux-qa.apk，84,731,822 字节，SHA256 `caa17a53a2cf8bd31ac9058c6544ba276f1df8bfc6741898954943a63145bbed`。
- 服务包：hw-bridge-business-closure-20261001-product-ux.tar.gz、php-business-closure-20261001-product-ux.tar.gz。
- 累计源码：business-closure-20261001-product-ux-source.zip；源文件清单：business-closure-20261001-product-ux-SOURCE-MANIFEST.json；校验：business-closure-20261001-product-ux-SHA256SUMS.txt。
- 服务器保留旧 APK 备份后原子替换原下载地址，另保留 product-ux 命名的新版本地址。旧链接现在指向新哈希，之前报告的旧链接哈希不再代表在线内容；本地旧版本文件仍保留。

Windows 拉取命令：

```powershell
curl.exe -fL "https://admin.awhaha.com/downloads/bridge-qa-20260930/qch-business-final-20260929-qa.apk?v=caa17a53a2cf" -o "$env:USERPROFILE\Downloads\qch-latest-qa.apk"
Get-FileHash "$env:USERPROFILE\Downloads\qch-latest-qa.apk" -Algorithm SHA256
```

下一步人工验证：覆盖安装新包，在注册页将邀请码留空，使用全新用户名完成验证码和注册；确认没有内部技术术语，然后提供新 UID 继续好友验收。
