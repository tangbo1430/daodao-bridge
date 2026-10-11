# 送测前代码 review 与自测

## 结论

**可交测试团队执行功能测试，使用本报告标明的新 QA APK。**

本轮未发现需要阻止进入功能测试的核心链路问题。用户已反馈当前人工测试未见大问题；这不是全部业务用例或生产稳定性验收完成的证明。

## 发现并修复

| 问题 | 原因与修复 | 验证情况 |
|---|---|---|
| 跨系统单聊顶部仍出现通话菜单 | 经典聊天页自定义按钮直接加载通话扩展，绕过 ChatInfo 限制。跨系统会话现在改成资料入口；普通会话仍沿用音视频菜单并传递功能开关。 | 编译通过，入口代码检查通过；实际新 APK 点击待手机确认 |
| HTTP 200 空内容或非法 JSON 可能误当成功 | PHP 以前 `json_decode(...) ?: []` 隐藏了响应错误。现在拒绝空对象、数组、null、标量、HTML 和非法 JSON，异常走现有失败路径。合法空搜索结果继续有效。 | GET/POST 异常契约、注册和业务门面回归通过，真实接口复验通过 |
| APK 部分建群入口可接受空 group_id | 创建成功前核验 `dd_` 标识；缺失时不清理幂等请求标识，明确提示重试。 | QA 构建通过；原公网建群及同 request_id 重试通过 |
| 好友/群网络异常直接显示内部异常文字 | 统一筛选反射、网络和 JSON 异常的用户文案，保留明确的业务拒绝提示。 | 编译及调用路径检查通过 |

## 修改文件

Android 根目录：`02-authoritative-qch-source/企才汇源码交付-20260810/Android`。

- `TUIKit/TUIChat/tuichat/src/main/java/com/tencent/qcloud/tuikit/tuichat/classicui/page/TUIC2CChatFragment.java`
- `TUIKit/TUIContact/tuicontact/src/main/java/com/tencent/qcloud/tuikit/tuicontact/model/BridgeContactClient.java`

PHP：

- `03-php-deployment/backend/application/imaccess/service/BridgeService.php`
- `03-php-deployment/tests/bridge_facade_test.php`

本轮无 Bridge 生产代码、数据库迁移或架构变更。保留上一轮文件与建群修复。

## 测试证据

1. Bridge `go test ./...`、`go vet ./...` 通过。
2. PHP 全部发布文件语法检查、8 组契约测试通过，含 `BRIDGE_INVALID_200_FAILS_CLOSED_OK`。
3. Android `assembleQaRelease` 通过，3 分 45 秒，252 tasks；保留既有第三方 R8 警告。
4. 实际 APK DEX/R8 搜索反射校验通过：`APK_PUBLIC_STATIC_ENDPOINT_FIELD_OK`、`ANDROID_RELEASE_ENDPOINT_REFLECTION_OK`。
5. 公网搜索：账号名 `shybo002`、用户码 `gBkqAmfsM` 均返回同一用户，HTTP 200。
6. 公网旧建群入口：使用授权 `74629` 登录态，创建 `dd_76eb2f5c41eb479481ff668817dd34bf`；相同请求重试返回同一群。已解散清理。
7. 实际临时群 `dd_06ee22faa0f0470ab321ad632b27f59c`：群名、公告、管理员/普通成员、禁言/解除、全员禁言/解除、邀请开关、踢人/重新邀请全部通过。头像公网内容与原生头像 SHA256 一致；连续两次同步未增加腾讯消息序号。已解散清理。
8. 真实两份 Word 文件再次从腾讯漫游获取其公网地址，下载 61,237 / 5,545,274 字节，SHA256 仍与源文件一致。
9. 未登录调用实际注册的 `/api/v1/im/bridge/config`、`/api/bridge/group_create`、`/api/bridge/friend_apply` 均返回 HTTP 401。
10. Bridge 健康接口 200，PHP/worker 重启后正常；服务器 `.env` 经部署脚本比较未修改。

补充排查：初次近期任务统计误用秒级时间与纳秒级 `created_at` 比较；已按存储实际单位纠正。抽查留下两条状态类失败：未映射旧账号的消息扩展事件、当前及已存群成员列表均无该账号的成员更新事件。不把这些记录描述为新消息投递失败；未批量重放或清理，仍保留给约定后置的状态事件/可靠性审计。不能宣称全库零失败。

## 发布版本

- APK：`qch-business-closure-20261001-pretest-review-qa.apk`
- 大小：84,731,822 字节。
- SHA256：`514392530be913e344bf22f1496432f0d897f12dbb2ac398394b06c3a2c3186c`
- Bridge 二进制仍为 `db8f7d68fd18222ae85630b42d090813244e1560c7f07887b1133b3012a538c9`。
- PHP 已部署，本次备份 `/root/IM/backup-php-bridge-20261001114723`。
- APK 下载旧别名已原子替换，本次旧 APK 备份 `/root/IM/apk-backups/20261001115024`。

不可变送测下载地址：
`https://admin.awhaha.com/downloads/bridge-qa-20261001/qch-business-closure-20261001-pretest-review-qa.apk`

累计发布物位于 `06-release-and-deploy`：

- `business-closure-20261001-pretest-review-source.zip`
- `business-closure-20261001-pretest-review-SOURCE-MANIFEST.json`
- `business-closure-20261001-pretest-review-SHA256SUMS.txt`
- `php-business-closure-20261001-pretest-review.tar.gz`
- `hw-bridge-business-closure-20261001-pretest-review.tar.gz`（Bridge 与上一版一致）

## 部署、验证和回滚

已执行：

```powershell
& ./hw-bridge-handoff-20260929/06-release-and-deploy/test-php-closure.ps1
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-php-bridge-provision.redacted.ps1 -SshKeyPath C:/Users/Administrator/.ssh/hw_bridge_20260929
```

预期输出 `PHP_CLOSURE_CHECK_OK`、`PHP_BRIDGE_PROVISION_DEPLOY_OK`。
Android 使用已有 JDK11/Gradle/SDK 构建 `assembleQaRelease --no-daemon`，预期 `BUILD SUCCESSFUL`。

下载并核验送测包：

```powershell
curl.exe -fL "https://admin.awhaha.com/downloads/bridge-qa-20261001/qch-business-closure-20261001-pretest-review-qa.apk" -o "$env:USERPROFILE\Downloads\qch-latest-qa.apk"
Get-FileHash "$env:USERPROFILE\Downloads\qch-latest-qa.apk" -Algorithm SHA256
```

PHP 本次回滚（保留上轮群修复）：

```sh
cd /root/IM/YCH-IM-deploy
cp /root/IM/backup-php-bridge-20261001114723/application/imaccess/service/BridgeService.php backend/application/imaccess/service/BridgeService.php
docker compose restart php worker
```

APK 回滚使用上一版 `qch-business-closure-20261001-tip-file-fix-qa.apk`，SHA256 `915d2750bb289efa00023b5c246c15d94c3c393cf579e954bd0b652b8eabfb43`；服务器旧别名备份见上。不涉及服务器配置或数据库回滚。

## 交给测试团队的重点

1. 新账号注册：邀请码留空、合法邀请码、用户名冲突，提示中无内部开发术语。
2. 搜索：用户名、用户码；好友申请/拒绝/重申请/同意。
3. 双向单聊、图片、文件、录音及短视频，失败提示不重复。
4. APK 建包含跨系统联系人的群；叨叨建群；群名、成员、禁言及群消息。
5. 双向撤回、单聊已读及未读清零。
6. 新 APK 跨系统单聊顶部应是资料入口，无音视频通话菜单；普通腾讯会话仍保留原功能。
7. 覆盖安装后，好友成功提示居中，不出现红点/横线等残留控件。
8. 网络中断/服务拒绝时不提示成功，重试建群不重复创建。

仍待条件：真实厂商离线推送点击路由、正式签名及跨地域 CDN 实际配置不因本轮通过而视为完成。大视频、主备、Leader、压测、72 小时稳定性、历史账号及旧异常引用按约定后置或排除。新 APK 本轮未在用户手机上安装点击验收。
