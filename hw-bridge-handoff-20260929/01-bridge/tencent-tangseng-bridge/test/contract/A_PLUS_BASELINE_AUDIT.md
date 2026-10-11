# 正确源码基线核对

本次基线来自 `企才汇安卓及PHP-Web源码交付包-20260810.zip`，而不是之前的 `LinJuGongXun` 工程或旧的 WKIM 改造目录。

## 已确认

- Android 工程：`Android/qch`。
- Android 包名和 applicationId：`com.qchm.xyz`。
- Android 当前使用腾讯 IM/TUIKit，登录凭据由 PHP 的 `/imaccess/session` 提供。
- PHP 工程：`PHP-Web/qicaihui.com`，当前默认 API 地址为 `https://admin.awhaha.com`。
- PHP 现有腾讯回调入口为 `Tencent/call_back`，并调用 `UserTencent::tencent_call_back`。
- PHP 源码包没有 Bridge 路由，也没有叨叨调用代码。
- Bridge 是独立 Go 服务；本目录中的 Bridge 代码不读取任何一侧数据库。

## 必须纠正的旧实现

- 不使用 `BridgeTencentChatManager`、WKIM 拦截或“所有会话改走腾讯”的代码。
- 不把 `tx_`/`dd_` 作为用户影子账号规则；A+ 文档要求腾讯影子账号直接使用真实叨叨 UID，`dd_<group_no>` 只用于叨叨群的腾讯镜像群 ID。
- 不把 `TANGSENG_BASE_URL`、`TANGSENG_WS_URL` 的猜测值当成真实协议。
- 不在阶段 0 之前改 PHP 或 APK 的生产行为。

## 当前阻塞

未取得叨叨/悟空真实的 V1–V12 协议样本，也未确认 Bridge 所在网络能访问 `master:8090` 和 `master:15200`。在这两项确认前，进入真实账号或生产链路会违反 A+ 方案的阶段门槛。

