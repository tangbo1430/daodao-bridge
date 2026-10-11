# APK 添加好友搜索修复

## 现象与已定位原因

用户反馈按原生账号名或用户码都搜索不到。已发布 QA APK 的 R8 mapping 显示：DemoApplication 类名保留，但 `java.lang.String NewAddress -> b`。TUIKit 的业务客户端使用 `loadClass("com.qchm.xyz.DemoApplication").getField("NewAddress")` 读取 PHP 地址，因此抛出 NoSuchFieldException，HTTP 搜索尚未开始。

AddMorePresenter 之前把任何内部搜索异常都忽略并回退腾讯用户查询，腾讯不认识原生用户名/用户码，最终掩盖为“找不到用户”。同一字段反射也被好友申请、申请列表、群业务入口复用，所以修复适用于这些入口。

## 修改

- Android qch/app/proguard-rules.pro：精准保留 DemoApplication.NewAddress 的公开静态字段名，继续保留现有混淆和签名方式。
- TUIKit TUIContact/.../presenter/AddMorePresenter.java：保留普通腾讯查询成功路径；两边均失败时不把网络/登录/初始化失败伪装为用户不存在。修复腾讯结果空列表访问和重复成功回调。
- TUIKit TUIContact/.../model/BridgeContactClient.java：搜索异常使用产品文案，反射/网络/JSON 错误不显示内部字段或服务器地址。
- 新增 verify-android-release-binding.py：检查实际 R8 产物中的类名与字段名；修改前检查失败，不能只以编译成功代替运行所需的绑定检查。
- 新增 verify-search-facade.php：通过正式 PHP HTTPS 搜索入口验证账号名/用户码；登录 token 仅在服务器内存使用，不输出。

## 实测与范围

- Bridge API：使用获授权联调账号 74629，按 shybo002 与 gBkqAmfsM 均返回 HTTP200、1 个结果、同一个预期 UID。
- 正式 PHP HTTPS 门面：相同两组输入均 HTTP200、1 个结果，PHP_SEARCH_NAME_AND_CODE_SAME_USER_OK。
- SharedPreferences 文件名 userInfo、字段 userToken 与实际 UserInfo 持久化一致，已有 Gson 字段保留规则存在。
- 修改前 verify-android-release-binding.py 输出 field_name_preserved=false、mapped_field_name=b 并失败。
- 本轮没有修改或部署 Bridge/PHP 业务源码；邀请码选填与产品文案修正继续保留，现网服务器版本仍以 PRODUCT_UX_FIX_20261001.md 为准。
- 手机点击搜索的最终展示由用户复验；正式 HTTPS 查询成功不单独标记 APK UI 通过。

## 最终产物与复验

- assembleQaRelease：BUILD SUCCESSFUL in 3m38s，252 tasks，15 executed。
- verify-android-release-binding.py：APK_PUBLIC_STATIC_ENDPOINT_FIELD_OK、ANDROID_RELEASE_ENDPOINT_REFLECTION_OK。R8 对未改名字段可省略 identity mapping；本轮直接解析打包 DEX，确认 DemoApplication 声明了名为 NewAddress 的 public static String 字段。
- APK：qch-business-closure-20261001-search-fix-qa.apk，84,731,822 字节。
- SHA256：`bef6f8711960fe0969e9d32b6222a05786ba96607a11b9ae753084fa13ff8fe6`。
- 累计源码包：business-closure-20261001-search-fix-source.zip；对应 SOURCE-MANIFEST.json 与 SHA256SUMS.txt 位于发布目录。
- 两个原下载链接会原子替换为此 APK；同时保留新文件名版本地址，旧 APK 在服务器 /root/IM/apk-backups 备份。邀请码选填修正继续保留。

```powershell
curl.exe -fL "https://admin.awhaha.com/downloads/bridge-qa-20260930/qch-business-final-20260929-qa.apk?v=bef6f8711960" -o "$env:USERPROFILE\Downloads\qch-latest-qa.apk"
Get-FileHash "$env:USERPROFILE\Downloads\qch-latest-qa.apk" -Algorithm SHA256
```

复验从“添加好友”入口输入完整账号名或用户码，勿使用只检索已有会话的搜索框。预期返回联系人头像/昵称，可进入申请界面；邀请码仍选填，页面不显示内部集成术语。

构建命令：qch 目录使用既有 JDK11/Android SDK，执行 `gradle.bat assembleQaRelease --no-daemon`。随后从发布目录执行 `python verify-android-release-binding.py`，必须输出 ANDROID_RELEASE_ENDPOINT_REFLECTION_OK，才允许上传。

服务器仅替换下载 APK，原文件备份后原子更新。服务端无需重启、无迁移。旧 APK 存在已确认搜索缺陷，不建议回退继续验收。
