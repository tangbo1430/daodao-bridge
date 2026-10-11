# 企才汇 IM

本仓库包含企才汇 IM 的 Android、iOS、PHP 服务端、Web 静态前端及部署资源。

## 开发入口

| 模块 | 目录 | 说明 |
| --- | --- | --- |
| 企才汇 Android | `企才汇IM源代码安卓/qch` | 主 App 工程，包名 `com.qchm.xyz` |
| Android IM 组件 | `企才汇IM源代码安卓/TUIKit` | 主 App 依赖的腾讯 TUIKit 源码 |
| PHP 服务端 | `wwwroot/qicaihui.com` | ThinkPHP 5.1 业务站点及 API |
| Web 部署前端 | `deployment/frontend/public` | 已编译的浏览器静态文件，不是完整前端工程源码 |
| Web 部署后端 | `deployment/backend` | 从 PHP 站点组装出的部署副本 |
| iOS | `企才汇ios` | iOS 工程 |

`ShangLuoSuLian` 是另一套独立 Android IM 工程，不是 `com.qchm.xyz` 这套企才汇 Android App 的子模块。

项目清单见 [IM 目录项目清单](docs/PROJECT_INVENTORY.md)；前后端边界见 [代码结构说明](docs/CODEBASE_STRUCTURE.md)。
