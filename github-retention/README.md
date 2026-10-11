# GitHub 留存盘点（2026-10-11）

## 结论

当前工作区尚未建立 Git 仓库。当前有效基线是展开后的 `hw-bridge-handoff-20260929/`，不是根目录旧 ZIP，也不是任一增量 source.zip。建议建立一个**私有单仓库**，首轮保持现有相对路径，避免破坏 Android 模块引用和部署脚本。

本轮已生成文件分类、逐文件 SHA256、PHP 差异清单、产物去重信息和敏感内容候选位置；没有修改业务代码、部署服务、初始化 Git 或上传 GitHub。目标仓库尚待指定。

## 资源与权威入口

以下路径相对 `hw-bridge-handoff-20260929/`。

| 资源 | 权威入口 | 留存要求 |
|---|---|---|
| Bridge | `01-bridge/tencent-tangseng-bridge` | Go 源码、测试、协议样例、迁移、Docker 配置、go.mod/go.sum 一起保存；bridge.exe 属于本机构建产物 |
| Android | `02-authoritative-qch-source/企才汇源码交付-20260810/Android/qch` | 必须同时保存兄弟目录 `Android/TUIKit`；包含本地 tinypinyin JAR、Gradle wrapper JAR；排除 build/.gradle/local.properties |
| PHP 原始完整基线 | `02-authoritative-qch-source/企才汇源码交付-20260810/PHP-Web/qicaihui.com` | 含后台页面和静态资源；不能被不完整部署补丁替代 |
| PHP 当前改造 | `03-php-deployment` | backend 改造、SQL、测试、Docker 和环境模板一起保留；恢复时以改造版覆盖同路径文件，并保留原始版独有资源 |
| 部署与验证 | `06-release-and-deploy` | 部署脚本、验证脚本、版本清单进 Git；APK/服务包/增量 ZIP 单独留存，按 SHA256 识别重复产物 |
| 文档与验收证据 | `05-docs-and-contracts`、`07-test-samples`、`08-context` | 需求、方案、流程图、契约、阶段报告和待验收项保留；旧报告不代表最新完成状态 |
| 原生参考材料 | `04-tangseng-reference` 与工作区 `toolchain` 下参考副本 | 标记为只读参考，保存来源/提交号/许可；不冒充本项目开发代码或手机实际版本；原始 ZIP 上传前单独审查 |

### 盘点数量

- 普通源码、文档、配置、静态资源候选：**7,959 文件，220,705,108 字节（约 210.5 MiB）**。
- 二进制/发布归档候选：80 文件，1,706,418,835 字节。此分类包含 2 份 ThinkPHP `hiddeninput.exe` 依赖，正式整合时需作为第三方依赖单独保留，不能当无用产物丢弃。
- 原生参考 ZIP：1 文件，106,954,752 字节。
- 旧 FILE_LIST.tsv 单独标记为历史清单；以本次 FILES.tsv 为当前盘点。
- 已从源码候选中排除本机路径配置、QR 缓存，以及遍历前跳过的构建/依赖缓存目录。
- 工作区工具链约 3.44 GiB，Go 缓存约 1.00 GiB；这两项不进入源码仓库。
- `toolchain` 顶层另有 **25 个辅助脚本/参考 PHP 文件**，已记录路径和哈希，需审查后挑选有维护价值的脚本；日志与真实业务报文不直接上传。

文件统计是本地现存资源统计，不是所有功能已验收或全新机器可离线构建的证明。

## PHP 的恢复关键点

按源码候选范围比较（排除缓存和二进制）：相同文件 1,570 个；同路径不同内容 **27 个**；部署版新增 **81 个**；原始版独有 945 个。完整路径见 INVENTORY.json 的 php_comparison。

原始版独有资源大部分在 public/admin、public/assets 和 public/static。因此首轮同时保存两棵树，并写明覆盖关系；后续另做完整构建验证后再去重，不能直接删除原始树。

现有忽略规则也有缺口：

- 两个 PHP 根 `.gitignore` 都忽略 `thinkphp`。
- 原始 PHP 的 `public/static/.gitignore` 为 `*`，会漏掉静态资源。
- 首次提交需在归档副本中修正这些规则，再将 Git 实际跟踪清单与本次 manifest 逐项比较；根 `.gitignore` 模板本身不能解决子目录规则。
- Windows 存在长路径，盘点脚本已使用扩展路径；复制/首次提交须启用适当长路径支持。

## 当前版本核对

### Bridge

- 最新本地产物：`06-release-and-deploy/hw-bridge-group-initial-avatar-20261002.tar.gz`。
- 包 SHA256：`affb32ed5131aa433d06a1591358ba8835900c406c7e5085cd0e00a351b316b4`。
- 包内 bridge 二进制 SHA256：`f9aac5cc7240b76cfe82632b0316221a03f86dce720f62fb9671bca67c2b0f16`。
- 包内 SOURCE_SHA256SUMS 的 **144 项全部匹配当前源码**。
- `hw-bridge-private-vpc.tar.gz` 与该命名包内容完全相同，无需重复保存二进制。
- 上次部署/验收记录：`08-context/GROUP_INITIAL_AVATAR_20261002.md`。本轮未连接现网，不将历史记录当成今天的现网核验。

### Android

- 最终 QA APK：`06-release-and-deploy/qch-group-notifications-20261002-r2-qa.apk`。
- 大小：84,737,990 字节。
- SHA256：`2e97a3e97a043d88508d73fed8bce76b52b18ae58dbd79818a1ff710d7132afb`。
- 同日期不带 r2 的 APK 是中间版本，不能按日期或大小判断相同。
- 上次构建/测试结论沿用阶段报告，本轮仅校验文件；未重新编译 APK 或进行手机验收。

### 构建环境应记录，不提交工具链目录

- Go 1.22.12；模块要求 Go 1.22.0。
- Gradle 7.6.4，JDK 11.0.32.1+1，Android SDK 由工程 Gradle 配置指定。
- Android 构建入口：qch 目录内执行 `assembleQaRelease`；中文路径需 UTF-8 Java 编码。
- 保留 composer.lock、go.sum、Gradle wrapper 和必要本地依赖。
- PHP DEPLOY.md 存在旧部署描述（例如 start.sh 与实际 starh.sh 命名、旧内嵌 Bridge 描述），正式仓库首页应链接当前独立 Bridge 的部署脚本和最新阶段报告；不能将旧描述直接当当前生产拓扑。

## 上传前已发现的具体风险

扫描只输出路径、行号、规则，不输出凭据内容：

1. `03-php-deployment/backend/application/admin/model/UploadSql.php:30,32` 存在腾讯云 SecretId/SecretKey 硬编码字面量。有效性未知；必须在归档副本中改为安全配置读取，并核查是否需要由负责人轮换。不能原样提交，即使私有仓库也不应存放凭据。
2. 原始/部署 PHP 的 application/common.php 包含固定二级密码摘要；UserTencent.php 注释中包含固定 token 字面量。需要随归档脱敏处理，不因为是注释或旧代码而忽略。
3. 共 63 个启发式匹配点，很多来自模拟测试、SDK 拼接 PEM 标记、语言包等，不等同于 63 个真实泄密。逐项位置见 CREDENTIAL_REVIEW.json。
4. ZIP、APK、压缩服务包、DOCX/PDF 内嵌内容以及大体积/非 UTF-8 文本尚未完成全面内容审查。外层源码脱敏不会清除旧压缩包里的内容，历史源码 ZIP 不可直接整体上传。
5. 生产 `.env`、SSH 私钥、签名材料不属于归档输入；本次没有读取项目外 SSH 私钥。

现阶段标记为**已盘点、待完成归档脱敏与目标仓库确认**，不是“已安全上传”或“已完成 GitHub 备份”。

## GitHub 整合方式

建议私有单仓库，首轮结构：

```text
repo/
  README.md                         # 本项目入口、构建、部署、状态
  hw-bridge-handoff-20260929/       # 原相对路径保留；仅审核通过的源码/文档
    01-bridge/
    02-authoritative-qch-source/
    03-php-deployment/
    05-docs-and-contracts/
    06-release-and-deploy/          # 脚本、清单，不包含历次二进制
    07-test-samples/
    08-context/
  github-retention/                # 清单、哈希、留存说明
  references/                      # 只读参考来源、提交号、许可说明
```

最新 APK、Bridge 服务包及匹配的混淆 mapping 作为经审查的 Release 附件保存；历史版本按哈希去重并保留别名表。源码中的两个约 39.3 MiB 的 ip.merge.txt 先保持可恢复性，不直接删除；未来确认运行用途后再考虑去重。

GitHub 普通 Git 阻止超过 100 MiB 的文件，发布二进制可使用 Releases；当前约 102 MiB 的原生参考 ZIP 不适合直接推普通 Git。[GitHub 官方大文件说明](https://docs.github.com/en/repositories/working-with-files/managing-large-files/about-large-files-on-github)

实施次序：

1. 确定目标私有仓库（现有仓库 URL，或账号/组织及新仓库名）。
2. 在独立归档副本中处理上述固定凭据及文档/归档内容；保留脱敏变更记录，运行必要语法/契约检查，原工作区和现网保持不动。
3. 修正副本内忽略规则；按 manifest 核对实际提交文件，重点检查 TUIKit、ThinkPHP、public/static、wrapper/JAR/必要 EXE 依赖。
4. 创建首次基线提交并记录 commit SHA；给已核对发布产物建对应版本标签和 Release，上传经审核的附件。
5. 从远端重新克隆到新目录，核对文件哈希、相对路径和构建入口；成功后才标记 GitHub 留存完成。

## 本次生成文件

- `FILES.tsv`：逐文件分类、大小、SHA256、内容扫描范围。
- `INVENTORY.json`：分类统计、PHP 差异和完全相同的产物组。
- `RELEASE_ASSETS.json`：二进制/归档候选路径、大小和哈希，不代表已批准上传。
- `BASELINE_VERIFICATION.json`：Bridge 包源码一致性、参考提交号、工作区辅助脚本清单。
- `CREDENTIAL_REVIEW.json`：敏感内容候选位置；没有匹配值。
- `inventory.py`：可重复执行的本地盘点脚本，不上传、不部署、不修改源文件。
- `gitignore.template`、`gitattributes.template`：未来归档副本的根规则参考；仍需处理上文子目录规则。

复跑：在工作区根目录用 Python 3 执行 `python github-retention/inventory.py`。脚本会更新 FILES/INVENTORY/RELEASE_ASSETS/CREDENTIAL_REVIEW；BASELINE_VERIFICATION 是本轮额外检查记录，发布版本变化后须重新核对。
