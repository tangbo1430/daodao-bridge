# 业务联调发布包

先阅读 BUSINESS_DELIVERY_20260929.md 和 REPLY_CONTRACT_INCIDENT_20260929.md。本包不是全部业务已验收的声明。

1. 以原交接包解压目录为基线，把 source-patch.zip 中的相对路径覆盖到该目录；不包含生产.env、私钥、签名材料或本机SDK路径。
2. 服务包与当前已部署文件一致；两个原部署脚本在补丁的06-release-and-deploy下。使用业务报告中的已验证部署命令。脚本保留服务器.env和compose，拒绝未填配置占位符。
3. Android本机需要JDK11、Gradle7.6.4、Android SDK。在qch/local.properties写本机sdk.dir，再构建assembleQaRelease。APK仅QA签名。
4. CHANGED_FILES.json为源码变化及原始/当前SHA；SOURCE_SHA256SUMS在Bridge tar.gz内。SHA256SUMS为本目录全部交付文件的哈希。
5. 迁移为增量；不回删业务数据。回滚版本、命令和风险见业务报告；回滚不能修复已存在的单条引用异常数据。

服务验证：healthz/readyz应200，PHP及worker Up，伪造媒体签名403，真实Range206、完整文件校验一致。真机验收和旧测试记录维护仍待完成。
