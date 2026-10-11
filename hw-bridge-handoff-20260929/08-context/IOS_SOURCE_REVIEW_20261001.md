# iOS 参考源码核对（2026-10-01）

## 来源与范围

- 目录：工作区 `toolchain/upstream-tang-ios`，Git 提交 `6913a3b`（2026-09-28）。这是此前下载的上游参考工程；尚不能证明与测试手机安装包对应。
- 含 App、模块源码、Xcode 工程、workspace、Podfile。Podfile 引用本地 `Modules/WuKongIMiOSSDK`，Pods 工程 Sources 列表包含 `WKChatManager.m`。工作树无修改。
- 本轮只读核对 iOS 和原生服务源码。用户允许修改范围为 APK 和 Bridge；没有修改 iOS、PHP、原生服务或悟空核心，没有新部署。

## 1. 批量消息扩展同步可能误删同批消息

位置：`Modules/WuKongIMiOSSDK/WuKongIMSDK/Classes/manager/WKChatManager.m:1793`。

`syncMessageExtra` 先按返回扩展的 messageID 获取本地消息，然后双重循环。内层只检查 `messageExtra.isMutualDeleted`，没有比较当前 message.messageId 与该扩展的 messageID。一条扩展被双向删除，就会将查询到的同批消息全部加入 deletedMessages 并调用 deleteMessage。

最小控制流复现（合成数据，非真实报文、非 iOS 运行测试）：本地消息 101、102、103，扩展只有 102 删除。按源码循环得到删除集合 101、102、103，正确集合应仅为 102。Python 控制流复现断言通过。

这是参考源码中的明确缺陷，但不是已证明的本次图片丢失根因：仍需确认安装版本以及失败时同批扩展数据。原生历史存在、资源可下载与客户端本地误删并不矛盾；Bridge done 或已读回执不能证明特定手机展示成功。

参考原生服务 `toolchain/deployed-native-reference/libs/TangSengDaoDaoServer/modules/message/api.go` 的 revoke 实现也会写删除状态并发送 syncMessageExtra，与已观测到的静默删除路径相符。不能靠换成随机消息键重发修复本地删除，否则会造成重复消息且破坏撤回语义。

## 2. 撤回后的“新消息”计数风险

位置：`Modules/WuKongBase/WuKongBase/Classes/Sections/Conversation/WKMessageListView.m:544`。

`refreshNewMsgCount` 使用 lastMessageSeq - browseToMessageSeq。序号有删除空位时，差值并不等于仍可见消息数；例如浏览至 100，101/102/103 中 102 删除，序号差仍为 3，可见消息仅 2。合成算例验证通过；具体“残留 1 条”仍需设备复现，不能把算例当作实际 UI 验收。

Bridge 无权直接改原生接收端的本地数据库或浏览位置。不得冒用接收者清空整段未读，也不能把腾讯未读数覆盖到包含原生消息的会话。

## 3. 群踢人通知的证据纠正

参考原生项目源码来自桌面 `TangSengDaoDaoUltimateServer-Source-20260927.zip`，仅提取 Go 文件用于只读分析，未提取安全配置；未证明与现网二进制完全相同。

- `modules/group/api.go` 的群内移除提示由 `shouldSendGroupMemberRemoveSysmsg()` 控制，对应配置默认关闭。
- `join_group_remind` 不控制该提示；此前切换入群提醒的实验不能证明踢人私有通知缺失。
- `TangSengDaoDaoServerLib/config/msg_group.go` 的 `SendGroupMemberBeRemove` 单独向被踢成员发送 1020，带 Subscribers、visibles 和 NoUpdateConversation。
- iOS SDK 对 1000～2000 使用通用系统消息解码。不能因没有单独的 1020 switch 分支就断言不会显示；但专门的成员变化分支未包含 1020。
- 群主历史中没有 1020 不能证明被踢者没收到。仍需被踢设备的通知与成员状态证据。

## 4. 头像与完成口径

现有 Bridge 上传头像后刷新完整资料的修复保持部署状态，详见 `IOS_FEEDBACK_20261001.md`。本轮没有新增 APK 或服务包。

只修改 APK/Bridge 能处理发送协议、资源准备、映射与重试问题，无法直接修正 iOS 本地批量删除或计数算法。上述问题保留为参考源码风险；不能宣称已修复或已证明生产根因。

建议定向复验：在专用测试会话中发送“文本 A、图片 B、文本 C”，收件端短暂离线，撤回 A 后再上线，检查 B/C 是否仍在。必须保留同条消息的 client_msg_no/message_id、服务端历史和客户端结果。不要在现有业务会话批量重放或清空历史。
