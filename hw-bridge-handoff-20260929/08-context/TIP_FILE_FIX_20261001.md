# 验收修复：系统提示布局与 Office 文件发送

## 结果

- Android 修复已构建、发布，界面待用户覆盖安装确认。
- Bridge 修复已部署；两份实际失败文件已定点补投，腾讯漫游可查，公网下载完整。
- PHP 无代码修改。数据库只扩大字段容量，现网 `.env` 经部署脚本比较保持一致。
- 本次不代表业务总验收完成。

## 根因与修改文件

Android 根目录为 `02-authoritative-qch-source/企才汇源码交付-20260810/Android`。

1. `TUIKit/TUIChat/tuichat/src/main/java/com/tencent/qcloud/tuikit/tuichat/minimalistui/MinimalistUIService.java`
2. `TUIKit/TUIChat/tuichat/src/main/java/com/tencent/qcloud/tuikit/tuichat/classicui/ClassicUIService.java`

`BridgeTipMessageBean` 已使用系统提示 Holder，但注册时未选择空白系统消息容器，普通消息控件因此残留。两种 UI 的注册均改为第三参数 `true`。好友申请卡片继续使用原交互布局，普通腾讯消息注册不变。已有提示重新进入会话后按新布局渲染。

Bridge 根目录为 `01-bridge/tencent-tangseng-bridge`。

3. `internal/store/schema.sql`：新数据库 `media_type` 从 VARCHAR(50) 扩为 VARCHAR(255)。
4. `internal/store/schema_embed.go`：启动时检查旧表容量并增量扩容；兼容缺少 `object_json` 的旧表，先补列再检查容量。
5. `migrations/006_media_mime_length.sql`：供部署脚本使用，可重复执行。
6. `internal/media/office_test.go`：模拟 Office 文件预览响应，校验 MIME、中文文件名、公开签名网关及文件字节。
7. `06-release-and-deploy/verify-office-media-schema.py`：实际 MySQL 测试 Word/Excel/PowerPoint MIME 完整入库，事务回滚测试行。
8. `06-release-and-deploy/verify-file-delivery.py`：根据 MsgKey 查询实际腾讯漫游，下载其文件地址，校验网关、大小与内容 SHA256；不打印密钥或签名 URL。

## 脱敏真实协议证据

来自此次用户客户端发送记录及 Bridge CVM 下载，非模拟响应：

```json
{
  "payload": {"type": 8, "url": "file/preview/chat/1/<recipient>/<object>.docx", "name": "<redacted>.docx", "size": 5545274},
  "preview_response": {"http_status": 200, "content_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
  "legacy_schema": {"media_type_capacity": 50},
  "mysql_error": "ERROR 1406 (22001): Data too long for column 'media_type' at row 1"
}
```

其中 payload 是由已持久化消息字段整理的脱敏契约摘要；原始身份、文件名和对象路径均已替换。测试内容为模拟字节，未将用户文件放入测试源码。

两份源文件下载均为 HTTP 200，耗时分别 0.12 秒和 0.53 秒，大小与消息一致。数据库事务中使用相同 MIME 可复现 1406；不是根据截图推测网络或大小限制。

## 测试与实际验证

- Bridge `go test ./...`、`go vet ./...`：通过。
- Android `assembleQaRelease --no-daemon`：通过，3 分 48 秒，252 tasks。保留已有第三方 R8/Gradle 警告。
- `verify-android-release-binding.py`：实际 APK DEX 和 R8 映射检查通过，搜索接口反射字段仍保留。
- 实际 MySQL：`OFFICE_MIME_MYSQL_ROUNDTRIP_OK width=255 types=3 rolled_back=true`。
- `/healthz`：200 `{"status":"ok"}`；`/readyz`：200 `{"status":"ready"}`。
- 两个原任务通过已有管理接口定点重放，均 `done`。同一小文件更早的重复失败记录未补投，避免用户收到额外重复文件；未清理旧失败提示或批量处理历史任务。

| 文件 | 腾讯 MsgKey | 字节数 | 原文件与公网下载 SHA256 |
|---|---|---:|---|
| 小 Word 文件 | `1128739637_1072385597_1790823263` | 61237 | `4d4becd5c0cad1b2981ef6e679bf22eebf90cea5369f12e81ffd34bcd44abe4d` |
| 大 Word 文件 | `943151268_1625892402_1790823264` | 5545274 | `e8ad5e1f5d71b347bce6fccfbf097c8bc75dc7ba44df96203ecb55fe3442564b` |

二者验证输出均为 `TENCENT_FILE_PUBLIC_SHA256_OK`。这证明真实腾讯消息及其公开文件下载，不替代 APK 点击打开验收。

## 发布产物

位于 `06-release-and-deploy`：

- `qch-business-closure-20261001-tip-file-fix-qa.apk`
- `hw-bridge-business-closure-20261001-tip-file-fix.tar.gz`
- `business-closure-20261001-tip-file-fix-source.zip`
- `business-closure-20261001-tip-file-fix-SOURCE-MANIFEST.json`
- `business-closure-20261001-tip-file-fix-SHA256SUMS.txt`

Bridge 二进制 SHA256：`db8f7d68fd18222ae85630b42d090813244e1560c7f07887b1133b3012a538c9`。

APK：84,731,822 字节，SHA256 `915d2750bb289efa00023b5c246c15d94c3c393cf579e954bd0b652b8eabfb43`。

服务器旧下载别名均已原子替换，另保留新版本不可变文件名，HTTPS 返回 200。

## 部署与验证命令

本机项目工作目录：

```powershell
$env:GOCACHE='C:/Users/Administrator/Desktop/tx2daodao/go-build-cache'
& ./hw-bridge-handoff-20260929/06-release-and-deploy/deploy-bridge-private-vpc.redacted.ps1 -GoExecutable C:/Users/Administrator/Desktop/tx2daodao/toolchain/go/bin/go.exe -SshKeyPath C:/Users/Administrator/.ssh/hw_bridge_20260929
```

预期部署输出 `HW_BRIDGE_DEPLOY_OK`；脚本检查占位符、校验包、备份并保留 `.env`，执行增量迁移，再启动 Bridge。此次备份 `/opt/hw-bridge.backup.20261001105329`。

将两个 `verify-*.py` 文件复制到 Bridge CVM 的 `/tmp` 后执行：

```sh
sudo python3 /tmp/verify-office-media-schema.py
sudo python3 /tmp/verify-file-delivery.py 1128739637_1072385597_1790823263 943151268_1625892402_1790823264
curl -fsS http://127.0.0.1:8090/healthz
curl -fsS http://127.0.0.1:8090/readyz
```

Windows 下载并覆盖安装 APK：

```powershell
curl.exe -fL "https://admin.awhaha.com/downloads/bridge-qa-20260930/qch-business-final-20260929-qa.apk?v=915d2750" -o "$env:USERPROFILE\Downloads\qch-latest-qa.apk"
Get-FileHash "$env:USERPROFILE\Downloads\qch-latest-qa.apk" -Algorithm SHA256
```

## 回滚

Bridge CVM 上：

```sh
cd /opt/hw-bridge
sudo cp /opt/hw-bridge.backup.20261001105329/bridge ./bridge
sudo cp /opt/hw-bridge.backup.20261001105329/Dockerfile ./Dockerfile
sudo docker compose up -d --build --no-deps bridge
curl -fsS http://127.0.0.1:8090/readyz
```

保留 VARCHAR(255)，不得缩回 50，否则已存在的 Office MIME 会被截断；旧程序兼容宽列。

APK 旧别名备份目录为 PHP CVM `/root/IM/apk-backups/20261001105443`。恢复文件同名旧版本即可回滚下载；已有安装需覆盖安装对应旧包。不恢复或覆盖任何 `.env`。

## 待人工确认

1. 当前 APK 进入原单聊，打开补投的两份 Word 文件，检查下载及外部文档查看器能否打开。
2. 覆盖安装新 APK 后重新进入会话，好友成功提示应居中显示，不带普通气泡的状态红点、分隔线或多余控件。
3. 再从客户端新发送一份 Word 文件，确认无需人工补投即可收到。
4. 本轮未验证顶部通话入口，不能依据样式修复宣称该入口已通过；大视频及先前待验收项继续保留。
