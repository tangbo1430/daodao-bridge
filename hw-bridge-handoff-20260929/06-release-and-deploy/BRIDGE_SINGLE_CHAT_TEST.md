# 华为 APK → 叨叨单聊联调

## 部署

1. 先部署 Bridge（独立 CVM，能访问 `10.0.0.100:8090` 和 `10.0.0.100:5200`）：

```powershell
powershell -ExecutionPolicy Bypass -File F:\IM\release\deploy-bridge-private-vpc.ps1 -ServerHost 101.32.11.27
```

2. 再部署 PHP 侧接口。脚本会先备份现有文件：

```powershell
powershell -ExecutionPolicy Bypass -File F:\IM\release\deploy-php-bridge-provision.ps1 -ServerHost 23.251.34.150
```

## 测试顺序

1. 用 APK 登录腾讯账号。
2. 在“添加好友”中输入叨叨用户的 UID 或用户名。APK 会优先调用 PHP `/api/bridge/search_user`，Bridge 查询叨叨并导入腾讯影子联系人。
3. 点击添加好友。32 位叨叨 UID 会调用 `/api/bridge/friend_apply`，不会直接调用叨叨接口。
4. 叨叨桌面端同意申请。
5. Bridge 收到 PC(2) 的好友确认后，调用腾讯 `sns/friend_add` 建立影子好友关系。
6. 在 APK 发送文本；腾讯回调经 PHP 进入 Bridge，再由对应华为影子账号的 PC(2) 发到叨叨。
7. 叨叨回复；Bridge 收到 `recv` 后调用腾讯 REST，消息回到 APK。

## 观察日志

```text
account restore complete / account provisioned
tangseng proxy connecting
tangseng recv
tangseng event accepted
friend link activated
```

APK 的完整 Gradle 构建还需要项目原有的 Bugly 依赖缓存；`tuicontact` Java 编译已通过，Bridge Go 全量测试已通过。
