# 归档检查记录 — 2026-10-11

## 已执行

- 原包 Bridge SOURCE_SHA256SUMS：144/144 项与归档前源码匹配。
- 归档副本 Bridge `go test ./...`：通过。
- 归档副本 Bridge `go vet ./...`：通过。
- 修改的 5 个 PHP 文件：PHP 8.0.30 `php -l` 全部通过。使用现有 PHP 容器通过标准输入做只读语法检查，没有安装/部署这些文件。
- `secondary_password()`：缺配置、非法配置都抛出异常；合法的测试摘要按原值返回。测试仅在单独 PHP 进程内设置环境变量，不修改服务环境。
- 6 个 Web 静态 JS 文件：Node 模块语法检查通过。
- Android：归档 qch 在 JDK 11 / Gradle 7.6.4 下执行 `help --offline --no-daemon` 通过，证明 Gradle 配置及外部 VIVO_APPKEY 表达式可以加载；本轮没有重新生成 APK。
- DOCX/PDF：抽取 3 个文档的文本/XML进行 Gitleaks 检查，通过；不等同于对嵌入图片进行 OCR 审计。
- 已知移除的 4 个不同遗留凭据/固定摘要值：在归档副本全文查找，剩余命中为 0；未输出任何原值。
- 源码中的两份 `extend/.gitignore` 还会漏掉 2,548 个第三方依赖文件，已在归档副本修正。
- 初次清单里的 2 个历史 `.log` 文件有意不提交，属于日志排除规则；保留对应阶段文档。

## 扫描范围与后续核验

使用 Gitleaks v8.30.1 做完整目录和 Git 暂存内容检查。最终退出码及 Git 文件完整性由 `REPOSITORY_CHECK.json` 记录。20 个 SDK PEM 格式拼接/Babel 方法名误报均限定到具体文件、规则和行号，说明见 SCAN_FALSE_POSITIVES.json。

文件完整性通过比较来源 manifest 与 Git 实际跟踪列表核验。`REPOSITORY_SHA256SUMS.tsv` 记录实际提交文件字节哈希（不包含清单自身），与脱敏前来源 FILES.tsv 区分。

## 范围限制

- 本轮属于源码留存，没有发布到生产、重跑真实双端业务验收或更改叨叨服务。
- QA APK、Bridge 二进制、历史 ZIP、工具链、缓存、生产环境变量与签名材料没有推入 Git；产物哈希保留供对应版本追溯。
- 归档内配置改动经过上述检查，恢复生产前仍须安全配置厂商推送、Web 地图、PHP 二级密码和遗留上传工具凭据。
- 早期盘点 README/JSON 是当时快照；当前留存入口及环境要求以仓库根 README 和本文件为准。
