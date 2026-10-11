# 历史联调辅助脚本

来自原工作区 toolchain 顶层的 24 个 Python/Go 脚本，供追溯分析和维护参考。

这些脚本并非统一 CLI，有些会修改测试资料、发送消息、执行修复或操作服务器，且包含历史账号、路径、业务条件。运行前逐个阅读、替换环境依赖并确认当前授权测试范围。正式业务实现以 `hw-bridge-handoff-20260929/01-bridge`、Android 和 PHP 源码为准；正式验收入口优先使用 `06-release-and-deploy/verify-*`。

旧的 `GroupV.server-before.php` 是历史备份，不作为当前源码再次收录。构建日志、服务端报文与本机工具链未复制。
