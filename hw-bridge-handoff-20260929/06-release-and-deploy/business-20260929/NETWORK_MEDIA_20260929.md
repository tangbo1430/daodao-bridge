# 媒体下载与跨地域路径实测

## 2026-09-29 开发环境日期 / 服务器日志 2026-09-30

用户反馈：图片可见，视频报 60009。使用用户已经发送的视频复现，文件大小 43,356,721 字节。未更改叨叨服务、悟空核心、主备或 Docker 网络拓扑。

| 路径 | 实际结果 |
| --- | --- |
| Bridge → 叨叨媒体预览 → 对象源 | 完整下载约 3.18 秒，SHA256 一致 |
| Bridge 本机媒体接口 | 完整下载约 2.81 秒，SHA256 一致 |
| 旧 PHP 公网媒体入口 | 约 35 秒失败；PHP 从 Bridge 仅收到约 19,976,545 字节即中断 |
| 延长媒体流超时并清除 PHP 缓冲后 | HTTP 200，完整下载 74.37 秒，SHA256 一致 |
| PHP 直接读取 Bridge 签发的现有 COS/CDN 对象源后 | HTTP 200，完整下载 24.53 秒，SHA256 一致 |

这些是单次功能复测，不能据此推算整体丢包率或作带宽承诺。手机端视频播放仍待用户复验。

## 根因及已部署修复

Bridge 媒体转发继承了 30 秒对象下载总超时，PHP 另有 35 秒 cURL 超时，ThinkPHP 缓冲还会延迟返回首字节。公网跨服务器传输对上游形成反压，导致视频未传完就被截断。

1. 媒体流独立使用 10 分钟上限，覆盖本路由的 Go HTTP 写时限；短业务请求时限不变。PHP 设置低速超时、释放输出缓冲并 flush。
2. Bridge 新增内部 HMAC 保护的 `/api/bridge/media-source`。只解析数据库中已有媒体对象，返回有效期 60 秒且带响应 HMAC 的对象源信息。
3. PHP 验证签名、对象 ID、时效、HTTPS 和公网 IP，并将认证过的 IP 固定给 cURL。外部源不接收 Bridge HMAC 或叨叨 token；禁止自动重定向。
4. APK 继续请求 `https://admin.awhaha.com/api/bridge/media`。PHP 直接从现有对象源取媒体，避免大文件再次经过 Bridge CVM 公网出口。
5. 实际对象源 `dl.lxode.com`。当前从 Bridge 解析到 `43.159.99.107`、`43.159.98.112`。未擅自替换域名、配置 CDN 或购买线路。
6. 实测发现预览生成的 COS URL 签名只授权 GET，直接 HEAD 会返回 403。HEAD 已改为由 Bridge 持久化媒体元数据应答，Range GET 继续读取对象源。

## 鉴权与回调结论

- PHP 为 APK 提供 UserSig，腾讯 IM 验证该签名。不能把所有消息/媒体请求都理解成腾讯先到 PHP 做一次权限握手。[腾讯登录鉴权](https://www.tencentcloud.com/document/product/1047/33517)
- 本项目开启了发送前回调，PHP 使用本地用户及黑名单状态判断；发送后事件持久化后由后台任务转发 Bridge。发送前回调的默认时限为 2 秒。[发送前回调](https://cloud.tencent.com/document/product/269/1632)
- 现网过去 24 小时单聊发送前回调 478 条，PHP 记录的处理均值 18.93 ms、最大 51 ms；这不包含腾讯到 PHP 的网络往返，不能用它证明网络没有丢包。
- `admin.awhaha.com` 当前直接解析到 `23.251.34.150`，返回 Caddy。没有证据说明这个 APK 业务域名已使用用户描述的国内中转/CDN。
- 当前 Tencent REST 配置使用 `adminapisgp.im.qcloud.com`。不能仅因 Bridge 是腾讯 CVM 就假设 IM REST 使用 VPC 内网。
- CVM元数据placement/region实测为ap-hongkong，非仅按IP推断地域。
- COS 同地域 CVM 的源站访问可走内网，跨地域默认公网；应依据实际存储桶地域及 DNS 核实。[腾讯 CVM 访问 COS](https://cloud.tencent.com/document/product/213/57441)

## 文件与测试

Bridge：`internal/media/service.go`、`source.go` 及测试，`internal/httpapi/server.go`、`cmd/bridge/main.go`。
PHP：`BridgeMedia.php`、`MediaSourceService.php`、`tests/bridge_media_stream_test.php`、`tests/media_source_test.php`；发布脚本增加 MediaSourceService。

`go test ./...`、`go vet ./...` 通过。PHP 语法检查、媒体签名及公网地址固定契约、输出缓冲测试通过。公网完整视频 SHA256 一致，起始及末尾 Range 均为 206，越界为 416。

## 部署及验证

在 `06-release-and-deploy` 执行现有两个 `deploy-*.redacted.ps1`，通过 `-SshKeyPath` 提供本机私钥路径；Go 通过 `-GoExecutable` 指定。脚本备份现有版本、校验文件并保留服务器 `.env`。生产密钥未写入源码。

```sh
# Bridge CVM
cd /opt/hw-bridge
curl -fsS http://127.0.0.1:8090/healthz
curl -fsS http://127.0.0.1:8090/readyz
sudo docker compose ps bridge
# PHP CVM
cd /root/IM/YCH-IM-deploy
docker compose exec -T php php -l /var/www/app/application/imaccess/controller/BridgeMedia.php
docker compose exec -T php php -l /var/www/app/application/imaccess/service/MediaSourceService.php
```

健康接口预期 200，服务 Up，PHP 无语法错误。真实媒体 URL 从服务器安全配置签发，勿写入工单或日志；GET 预期完整字节数、SHA256 与媒体记录一致，HEAD 200，Range 206。失败日志只包含 `bridge_media_download_failed id=... curl_errno=... upstream_status=... bytes=...`，不包含 URL 签名或凭据。

尚待核验：国内中转/CDN 的实际域名及配置、客户端到 PHP 的网络路径、回调公网往返、用户真机播放。未获得的基础设施配置不作已优化声明。
