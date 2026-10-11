# 部署步骤

服务器：`23.251.34.150`，系统：Ubuntu 24.04。

## 部署前

1. Cloudflare 添加 `im.awhaha.com`、`admin.awhaha.com` 的 A 记录，均指向 `23.251.34.150`。
2. 服务器开放 TCP `80`、`443`，Cloudflare SSL/TLS 设置为 `Full (strict)`。
3. 安装 Docker：`curl -fsSL https://get.docker.com | sh`。
4. 将本目录的 `start.sh`、`.env` 和 `YCH-IM-deploy.zip` 上传到服务器同一目录。
5. 编辑 `.env`，填写数据库密码、腾讯 IM 密钥、回调 Token、管理员 ID、短信宝账号密码和腾讯 COS 配置。
   同时填写 `IM_BRIDGE_SHARED_SECRET`、`BRIDGE_DB_PASSWORD`、`BRIDGE_DB_ROOT_PASSWORD`，并核对唐僧叨叨的 `TANGSENG_BASE_URL`、`TANGSENG_LOGIN_ENDPOINT`、`TANGSENG_WS_URL` 和注册策略。生产环境再注入 32 字节 `BRIDGE_ENCRYPTION_KEY`。
6. 业务图片默认使用 `UPLOAD_STORAGE_DRIVER=cos`；回退本地存储时改为 `UPLOAD_STORAGE_DRIVER=local`。

升级时 `start.sh` 会自动执行 `database/migrations`，包括默认头像修正。历史本地头像迁移前先预览：

```bash
docker compose exec php php application/common/scripts/migrate_legacy_user_avatars.php
# 确认清单后执行：在命令末尾增加 --execute
```

## 一键部署

```bash
chmod +x start.sh
./start.sh
```

访问地址：

- Web IM：`https://im.awhaha.com`
- 后台：`https://admin.awhaha.com/admin/`
- 腾讯回调：`https://admin.awhaha.com/imcallback/tencent/receive?token=你的IM_CALLBACK_TOKEN`

查看状态：`docker compose ps`；查看日志：`docker compose logs -f --tail=200`。

Bridge 服务使用独立的 `bridge_mysql` 和 `bridge_redis` 容器，首次启动会在 Bridge 数据库中自动创建映射、A+ 账号/关系/群/游标、inbox、outbox、媒体和审计表。PHP 将腾讯原始回调通过内部网络签名转交到 `http://bridge:8090/cb/tim`，不需要额外暴露公网端口。配置 `TANGSENG_WS_URL` 后才启用每用户悟空 PC(2) 代理。
