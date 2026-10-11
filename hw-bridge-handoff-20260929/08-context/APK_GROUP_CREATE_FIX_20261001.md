# APK 发起建群未同步：修复及验收

## 根因

APK 实际使用经典界面的 `CreateGroupActivity.create_group()`，以表单调用 `/api/group_v/create_group`。
此前仅在 `ContactProvider.createGroupChat()` 接入了 Bridge。经典页面绕过该 Provider，旧 PHP 模型直接创建普通腾讯群，因此叨叨没有新群。

真实故障群 `31967`，群主腾讯账号 `20261`，包含叨叨影子成员。两端关系存在，但该群没有 `dd_` 映射。这与用户报告一致。

未修改该用户群，也未迁移为另一个群；用户需重新建包含对应联系人的群。

## 修改文件

生产代码（均位于 `03-php-deployment/backend`）：

- `application/api/model/GroupV.php`：统一校验成员；选有原生联系人时，旧入口调用现有 Bridge 原生优先建群流程，返回 `dd_` 群 ID 和 Public 类型。失败不回退为普通腾讯群，不误报成功。普通腾讯成员继续原流程。
- `application/imaccess/service/BridgeService.php`：抽取共享群请求逻辑，新增仅供 PHP 模型调用的 `createGroupForUser`。操作人来自已有登录态 UID，重新核验用户状态并解析 IM 映射；不接受客户端传入的操作人。

测试及工具：

- `03-php-deployment/tests/group_creation_route_test.php`：执行真实 GroupV 模型，模拟下游，覆盖混合群路由、成员去重、业务拒绝、结果缺失和普通腾讯群流程。
- `03-php-deployment/tests/bridge_facade_test.php`：补旧入口的身份映射及不可用用户拒绝测试。
- `06-release-and-deploy/test-php-closure.ps1`：加入新契约测试及模型路径。
- `06-release-and-deploy/verify-apk-group-create.php`：以真实公网表单接口验收建群及相同 request_id 重试；凭据只在 PHP 主机读取，测试状态不含登录 token；提供临时群清理模式。

Android/Bridge 本轮无生产代码修改。已有 APK 对返回的群 ID、类型和业务错误已兼容，无需为本次修复更新 APK。

## 测试与现网证据

- PHP 部署文件语法检查、8 组契约测试通过：`PHP_CLOSURE_CHECK_OK`。
- `GROUP_CREATION_ROUTE_CONTRACT_OK`、`BRIDGE_FACADE_CONTRACT_OK`。
- Bridge `go test ./...`、`go vet ./...` 通过，包括群及双向事件契约；本轮未修改 Bridge 二进制。
- 使用授权账号 `74629`，邀请授权叨叨账号 `gBkqAmfsM` 对应成员，调用 APK 相同的 HTTPS 表单入口。
- 请求中故意附加错误 `uid`，实际群主仍是登录态对应的 `74629`。
- 两次同 request_id 请求均 HTTP 200、业务 code 200，返回同一群 `dd_a6f9e87e68f24a659f1171bd70e43b0b`；未重复建群。
- 腾讯群资料查询：群主正确，群成员为对应两人。
- **实际叨叨 EXE 界面**出现“建群接口验收-442978”，会话摘要显示邀请信息。
- 从 EXE 发送“建群修复验收：客户端反向文本”，腾讯群漫游找到一条，MsgSeq 3。
- 腾讯 REST 代表授权发送者发送“建群修复验收：腾讯方向文本”，MsgSeq 4，实际叨叨 EXE 显示该消息。
- 临时群已通过原接口解散，存储核对 `status=2, mirror_ready=false`；真实用户原群保持不变。
- Bridge `/healthz`、`/readyz` 均 200；PHP/worker 部署后运行正常；部署脚本确认 `.env` 未改变。

说明：正向消息使用真实腾讯接口，不冒充本轮已操作 APK 发送。反向群创建用户此前已确认可用，本轮未重新从 EXE 创建第二个群；现有群事件契约及真实反向群文本通过。普通腾讯群仅契约回归，未额外在生产创建普通群。

## 部署与回滚

已执行（本机项目工作目录）：

```powershell
& ./hw-bridge-handoff-20260929/06-release-and-deploy/test-php-closure.ps1
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-php-bridge-provision.redacted.ps1 -SshKeyPath C:/Users/Administrator/.ssh/hw_bridge_20260929
```

预期输出 `PHP_BRIDGE_PROVISION_DEPLOY_OK`。
此次备份：`/root/IM/backup-php-bridge-20261001112418`。无新增迁移，无新密钥配置。

PHP 主机把验证脚本放入容器 `/tmp` 后：

```sh
cd /root/IM/YCH-IM-deploy
docker compose exec -T php php /tmp/verify-apk-group-create.php <授权华为UID> <授权叨叨UID> create
docker compose exec -T php php /tmp/verify-apk-group-create.php <授权华为UID> <授权叨叨UID> cleanup
```

预期 `APK_FORM_NATIVE_GROUP_AND_RETRY_OK dd_...` 和 `TEMP_APK_GROUP_DISBANDED dd_...`。脚本保留验收状态，已清理的一轮再次创建会拒绝；复测前应审阅并归档对应状态，不重复使用已解散请求 ID。

回滚仅恢复本次两个生产文件，保留配置：

```sh
cd /root/IM/YCH-IM-deploy
cp /root/IM/backup-php-bridge-20261001112418/application/api/model/GroupV.php backend/application/api/model/GroupV.php
cp /root/IM/backup-php-bridge-20261001112418/application/imaccess/service/BridgeService.php backend/application/imaccess/service/BridgeService.php
docker compose restart php worker
```

## 发布材料

`06-release-and-deploy` 下：

- `php-business-closure-20261001-group-create-fix.tar.gz`
- `business-closure-20261001-group-create-fix-source.zip`
- `business-closure-20261001-group-create-fix-SOURCE-MANIFEST.json`
- `business-closure-20261001-group-create-fix-SHA256SUMS.txt`

累计包附带 Bridge 与 APK 已验证版本；APK 与上一版完全相同，SHA256 为 `915d2750bb289efa00023b5c246c15d94c3c393cf579e954bd0b652b8eabfb43`，下载地址无需替换。

## 用户最短复验

1. 在当前 APK 新建群，明确选中 `shybo002`，取新群名“建群复验”。
2. 创建成功后，叨叨应出现对应群。
3. 两边各发一句文本，确认互相收到。

剩余状态：服务端完整链路和叨叨界面已验证，用户当前 APK 的实际点击建群待复验。不宣称全部业务已验收。
