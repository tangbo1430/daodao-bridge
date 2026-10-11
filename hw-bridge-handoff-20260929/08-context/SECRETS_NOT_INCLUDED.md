# 交接包中的敏感信息处理

为了避免把生产凭据带入新的 Agent，以下内容已从交接包中移除或替换为占位符：

- Bridge `.env`、PHP `.env`、数据库密码、Redis 密码；
- 腾讯 IM `SecretKey`、回调密钥和 UserSig 相关密钥；
- 叨叨管理账号密码、邀请码、Token；
- 历史部署目录中的运行时日志、缓存、构建缓存和旧备份。

需要部署时，由负责人在本机环境变量或服务器 `.env` 注入：

```text
BRIDGE_SHARED_SECRET
TENCENT_SECRET_KEY
TANGSENG_PASSWORD
TANGSENG_INVITE_CODE
BRIDGE_DB_PASSWORD
BRIDGE_DB_ROOT_PASSWORD
BRIDGE_ENCRYPTION_KEY
```

`06-release-and-deploy` 中的脚本是脱敏版；其中 `__SET_*__` 必须在运行前替换或通过参数/环境变量注入。不要把真实值写回交接包。

SSH 私钥没有包含。新 Agent 使用现有机器上的 SSH Agent/默认密钥，或由负责人单独配置。
