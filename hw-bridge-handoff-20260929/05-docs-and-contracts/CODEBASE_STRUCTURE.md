# 企才汇代码结构说明

## 结论

企才汇目前不是单一的前后端分离源码仓库。

- 老站 `wwwroot/qicaihui.com` 是 ThinkPHP 5.1 一体化项目，PHP 后端、HTML 页面模板和静态资源放在同一个站点目录中。
- `deployment/backend` 与 `deployment/frontend` 在部署包中已经物理分开。
- `deployment/frontend/public` 只有编译后的 HTML、CSS、JavaScript 和图片等产物，未发现 `package.json`、`.vue`、`.jsx` 或 `.tsx` 等完整现代前端工程源码。

因此，“企才汇前端代码是不是跟 PHP 在一起”的准确回答是：老站的模板和静态前端与 PHP 在一起；部署版本把前后端分开了，但仓库内能看到的独立前端主要是编译产物。

## Android

### 企才汇主工程

目录：`企才汇IM源代码安卓/qch`

- Gradle 入口：`settings.gradle`
- App 模块：`app`
- Android 包名：`com.qchm.xyz`
- 业务 Java/Kotlin：`app/src/main/java/com/qchm/xyz`
- 页面布局和资源：`app/src/main/res`
- 构建配置：`app/build.gradle`
- 文档：`docs`

该工程通过 `settings.gradle` 引用同级的腾讯 IM 组件：

- `TUIKit/TUICore/tuicore`
- `TUIKit/TIMCommon/timcommon`
- `TUIKit/TUIChat/tuichat`
- `TUIKit/TUIContact/tuicontact`
- `TUIKit/TUIConversation/tuiconversation`
- `TUIKit/TUISearch/tuisearch`
- `TUIKit/TUICallKit/tuicallkit-kt`

`企才汇IM源代码安卓/release` 保存 APK 和 mapping 等发布产物，不属于日常源码入口。

### 独立 Android 工程

目录：`ShangLuoSuLian`

这是另一套完整 Gradle 工程，包含 `wkbase`、`wkuikit`、`wklogin` 等多个模块。它有独立的应用配置、版本和服务地址，不应与企才汇的 `qch + TUIKit` 工程合并。

## Web 与 PHP

### PHP 业务站点

目录：`wwwroot/qicaihui.com`

- `application/api`：App/Web API 控制器和模型
- `application/admin`：后台管理控制器、模型和页面模板
- `application/index`：站点页面控制器和模板
- `application/*/view`：由 PHP 渲染的 HTML 页面
- `public`：浏览器直接访问的 HTML、JavaScript、CSS、图片和上传资源
- `config`：ThinkPHP 配置
- `route`：路由
- `vendor`、`thinkphp`：PHP 依赖和框架源码
- `runtime`：运行时缓存与日志，不应作为业务源码维护

这部分属于传统 PHP 全栈结构，所以前端页面与 PHP 在同一个项目中。

### 部署结构

目录：`deployment`

- `backend`：PHP 后端部署副本
- `frontend/public`：Web 静态前端部署产物
- `database`：数据库初始化和迁移
- `docker`：容器及 Web 服务器配置
- `scripts`：部署和校验脚本
- `docker-compose.yml`：服务编排

`deployment/backend` 和 `deployment/frontend` 由打包流程生成，并已被根 `.gitignore` 排除。业务修改应优先落在受版本控制的源目录，而不是直接修改部署副本。

## 其他目录

- `企才汇ios`：iOS 工程。
- `release`：历史部署包、APK、截图和 mapping 等发布产物。
- `.package-*`：临时打包目录。
- `wwwroot/jishitongxun.com`：另一套 PHP IM 站点，当前也包含部分正在修改的服务端代码。
- `wwwroot/www.demo.com`、`wwwroot/www.new.com`：历史或测试站点副本。
- `wwwroot/msql_back`：数据库备份数据。

## 维护约定

1. 企才汇 Android 业务开发从 `企才汇IM源代码安卓/qch` 开始；通用聊天能力再进入 `企才汇IM源代码安卓/TUIKit` 修改。
2. PHP API 和后台功能在 `wwwroot/qicaihui.com` 修改。
3. PHP 模板页面在 `application/*/view` 修改，传统静态资源在 `public` 修改。
4. 不把 `release`、`.package-*`、APK、Gradle `build`、PHP `runtime` 当作源码提交。
5. 不直接在 `deployment/backend` 或 `deployment/frontend` 做长期开发；这些目录应由源代码重新组装。
6. 若要继续开发独立 Web IM，需要补回对应的未编译前端工程源码，并单独建立例如 `web/` 的源码目录；仅靠 `deployment/frontend/public` 不适合持续开发。
