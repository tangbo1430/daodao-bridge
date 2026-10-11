# 企才汇源码交付包

交付日期：2026-08-10

## 内容

- `Android/qch`：企才汇 Android 主 App，包名 `com.qchm.xyz`。
- `Android/TUIKit`：主 App 引用的腾讯 TUIKit 组件源码。
- `PHP-Web/qicaihui.com`：企才汇 ThinkPHP 5.1 API、管理后台、页面模板和 Web 静态资源。
- `Docs/API.md`：PHP API 接口文档。
- `Docs/CODEBASE_STRUCTURE.md`：代码结构和前后端边界说明。
- `Docs/PROJECT_INVENTORY.md`：IM 根目录项目清单。

## 已排除内容

- Android：`.gradle`、`build`、IDE 配置、本机 SDK 路径、签名文件、APK/AAB、日志、厂商推送配置及测试 UserSig 文件。
- PHP/Web：`runtime`、`vendor`、上传文件、数据库备份、历史压缩包、日志。
- 敏感配置：真实 `config/database.php` 未打包，使用 `config/database.php.example` 配置。

## Android 构建

1. 使用 Android Studio 打开 `Android/qch`。
2. 确保 `Android/TUIKit` 与 `qch` 保持当前相对目录关系。
3. 创建本机 `local.properties`，配置 Android SDK。
4. Release 构建时复制并填写 `signing.properties.example`，不要提交签名密码和密钥库。
5. 按目标环境安全注入 `agconnect-services.json`、`google-services.json`、`timpush-configs.json` 等推送配置；不要使用客户端测试 UserSig 私钥。

## PHP/Web 运行

1. PHP 版本要求：`>= 5.6`，推荐使用项目现网兼容版本 PHP 8.0。
2. 在 `PHP-Web/qicaihui.com` 执行 `composer install --no-dev --optimize-autoloader`。
3. 将 `config/database.php.example` 复制为 `config/database.php` 并填写环境数据库信息。
4. 将 Web 根目录指向 `PHP-Web/qicaihui.com/public`。
5. 为 `runtime` 和业务上传目录设置 Web 服务账户写权限。
6. 生产环境关闭 `app_debug` 和数据库调试。

接口调用说明见 `Docs/API.md`。
