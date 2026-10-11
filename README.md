# daodao-bridge

华为企才汇 Android / 腾讯 IM ↔ 唐僧叨叨互通项目的源码留存仓库。

## 基线与目录

2026-10-11 从当前工作区整理。保留原目录结构，避免破坏 Android 模块引用和既有部署脚本。

| 目录（位于 hw-bridge-handoff-20260929） | 内容 |
|---|---|
| `01-bridge/tencent-tangseng-bridge` | Bridge Go 源码、单元/契约测试、数据库迁移、Docker 配置 |
| `02-authoritative-qch-source/企才汇源码交付-20260810/Android` | 当前 qch 和配套 TUIKit；构建入口为 qch |
| `02-authoritative-qch-source/企才汇源码交付-20260810/PHP-Web/qicaihui.com` | PHP 完整原始基线，含后台页面、静态资源和 ThinkPHP |
| `03-php-deployment` | 当前 PHP 改造、测试、数据库迁移、部署配置；backend 为覆盖层，不是独立完整 Web 站点 |
| `05-docs-and-contracts`、`07-test-samples` | 实施方案、API、流程和协议样例 |
| `06-release-and-deploy` | 部署/验证脚本及发布清单；不含历次 APK、压缩包 |
| `08-context` | 阶段记录和待验收项；按日期及后续说明理解 |

`github-retention` 保存盘点、归档改动和校验清单。`tools/historical-probes` 是历史联调辅助脚本，可能依赖特定工作目录/授权测试账号；不能当自动部署入口直接执行。

## 架构约束

- Bridge 独立、单实例部署；Android 继续使用腾讯 IM/TUIKit。
- PHP 处理业务接口和腾讯回调；Bridge 负责映射、转换、PC(2) 代理与同步。
- 不修改叨叨业务服务端和悟空 IM 核心，不让 APK 直连叨叨。
- 保留现有 Docker/MySQL/Redis 和部署目录；本仓库未引入主备或 Leader 改造。

## 最新留存版本

- Bridge：`hw-bridge-group-initial-avatar-20261002.tar.gz` 对应源码；原包 144 项源码 SHA256 与归档前工作区全部一致。
- Android：对应 `qch-group-notifications-20261002-r2-qa.apk` 的源码基线。
- APK 和服务包的大小、SHA256 见 [产物清单](github-retention/RELEASE_ASSETS.json)，本次仅上传代码相关内容，未上传二进制附件。
- 具体完成范围和手机待验收项见 [首次群头像报告](hw-bridge-handoff-20260929/08-context/GROUP_INITIAL_AVATAR_20261002.md) 和相邻阶段报告；源码留存不代表业务用例全部通过。

## 凭据与归档改动

本仓库不包含生产 `.env`、SSH 私钥、签名文件、数据库备份、运行日志、构建缓存或旧源码 ZIP。原工作区及生产服务未因本次归档而被修改。

归档副本移除了遗留固定凭据，并修复会漏掉 ThinkPHP/静态资源的 Git 忽略规则。完整文件列表见 [ARCHIVE_EDITS.json](github-retention/ARCHIVE_EDITS.json)。因此归档副本中的这些文件不再与原始发布包逐字节相同，恢复部署前必须配置：

| 配置 | 用途 |
|---|---|
| `TENCENT_SECRET_ID` / `TENCENT_SECRET_KEY` | PHP 遗留 UploadSql 管理工具；通过 PHP 容器环境注入，缺失时拒绝操作 |
| `QCH_SECONDARY_PASSWORD_MD5` | PHP 遗留二级密码摘要；要求 32 位十六进制值，缺失或格式错误时拒绝操作 |
| `VIVO_APPKEY` | Android Gradle 属性或构建环境变量；正式验证 Vivo 推送时必须提供 |
| `window.QCH_MAP_API_KEY` | 原有 Web 静态地图页面的 Key；页面加载模块前由部署配置注入，不在 JS 中固定保存 |

其他 Bridge、数据库、厂商推送和签名配置沿用环境模板，通过安全环境注入。第三方 SDK 示例 Token 已替换为占位符；SDK 的 PEM 拼接字符串和 Babel 方法名误报逐项记录在 [SCAN_FALSE_POSITIVES.json](github-retention/SCAN_FALSE_POSITIVES.json)，没有整目录免扫描。

## 构建与检查

- Go 1.22.12：在 Bridge 目录执行 `go test ./...`、`go vet ./...`。
- Android：JDK 11、Gradle 7.6.4、对应 Android SDK；保留 qch 与 TUIKit 的兄弟目录关系，在 qch 执行 `assembleQaRelease`。Windows 中文路径设置 `JAVA_TOOL_OPTIONS=-Dfile.encoding=UTF-8`，配置 `ANDROID_HOME` 或本机 local.properties；后者不入库。
- PHP：恢复完整原始目录后覆盖 `03-php-deployment/backend` 同路径文件，安装 composer.lock 对应依赖，配置环境并按当前部署脚本执行迁移。不要直接用 backend 覆盖成空的站点根目录而丢失 public 资源。
- 部署脚本依旧位于 `06-release-and-deploy`。先核对最新阶段报告、环境占位符和服务端现有 `.env`。旧 DEPLOY.md 含历史拓扑描述，不应据此把 Bridge 嵌回 PHP/叨叨服务。

归档时检查结果见 [VALIDATION.md](github-retention/VALIDATION.md)。库存文件清单 `FILES.tsv` 是**脱敏前来源指纹**；实际仓库文件指纹以 `REPOSITORY_SHA256SUMS.tsv` 为准。

## 只读参考与许可证

第三方源文件保留原有许可证和版权标记；本次没有为整个商业项目授予新的开源许可。

未将工具链、原生服务参考 ZIP 或另一个项目作为当前业务源码上传。已使用参考副本的提交号：

- Tang iOS 上游：`6913a3b320f6b278676ac22e4a45e416b64cb56f`，仅协议研究参考，未经确认与测试手机版本一致。
- Tang Server 上游：`7511b31c07aa9a6318252202c0dcd8759d5e1cd6`。

本次留存不包含数据库业务数据、账号凭据和生产签名；这些资源须由负责人通过独立安全备份维护。
