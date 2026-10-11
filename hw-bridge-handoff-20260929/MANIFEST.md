# 交接包清单

## 已包含

- `01-bridge/`：当前 Bridge Go 源码、Docker 配置、测试和 README。
- `02-authoritative-qch-source/`：权威企才汇 Android/qch 和 PHP-Web/qicaihui.com 源码；已排除 Android/Gradle 构建缓存和 build 目录。
- `03-php-deployment/`：线上 PHP Bridge 相关控制器、服务、回调、配置、路由、数据库迁移、Docker 配置和部署说明。
- `04-tangseng-reference/`：叨叨轻量源码参考压缩包。
- `05-docs-and-contracts/`：详细实施方案、网络对接说明、流程图、API、代码结构、项目盘点、需求提取、工作总结和文档审阅材料。
- `06-release-and-deploy/`：最新 Bridge 部署包、脱敏部署脚本、PHP 补丁包、最新 APK、混淆 mapping、测试说明和原始企才汇交付压缩包。
- `07-test-samples/`：腾讯回调、叨叨反向事件、登录/注册和 PHP smoke 样本；密码字段已替换为 `__TEST_PASSWORD__`。
- `08-context/`：当前架构、线上地址、部署命令、已完成事项、已知问题、验收顺序和敏感信息说明。

## 有意排除

- 生产 `.env`、SSH 私钥、SecretKey、数据库密码、Token、邀请码和运行时日志；
- `.git`、Gradle/Go/npm 缓存、Android `build`、运行时 `vendor`/日志、临时目录；
- `F:\IM\20260630聚畅信后端代码.7z`（1.3GB 原始叨叨后端归档）；交接包已包含 `04-tangseng-reference/叨叨服务端源码-轻量副本.zip`；
- 历史 APK、旧 Bridge bundle、旧 PHP 备份和重复的 staging 目录；
- `LinJuGongXun-huawei` 等之前误用的工程，避免新 Agent 误改错误 APK。

## 权威源说明

本包里的 `02-authoritative-qch-source/企才汇源码交付-20260810` 是当前 APK/PHP 的权威源码。`06-release-and-deploy` 是与现网部署方式匹配的最新发布材料。叨叨服务端只作为协议参考，不能修改。
