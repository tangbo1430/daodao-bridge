package com.tencent.qcloud.tuikit.tuichat.model;

import static com.tencent.imsdk.BaseConstants.ERR_SDK_MSG_MODIFY_CONFLICT;
import static com.tencent.imsdk.BaseConstants.ERR_SUCC;
import static com.tencent.imsdk.v2.V2TIMMessage.V2TIM_MSG_STATUS_HAS_DELETED;
import static com.tencent.imsdk.v2.V2TIMMessage.V2TIM_MSG_STATUS_SEND_FAIL;

import android.text.TextUtils;
import android.util.Pair;
import com.google.gson.Gson;
import com.tencent.imsdk.BaseConstants;
import com.tencent.imsdk.v2.V2TIMCallback;
import com.tencent.imsdk.v2.V2TIMCompleteCallback;
import com.tencent.imsdk.v2.V2TIMConversation;
import com.tencent.imsdk.v2.V2TIMConversationOperationResult;
import com.tencent.imsdk.v2.V2TIMFriendInfo;
import com.tencent.imsdk.v2.V2TIMFriendInfoResult;
import com.tencent.imsdk.v2.V2TIMGroupApplication;
import com.tencent.imsdk.v2.V2TIMGroupApplicationResult;
import com.tencent.imsdk.v2.V2TIMGroupChangeInfo;
import com.tencent.imsdk.v2.V2TIMGroupInfo;
import com.tencent.imsdk.v2.V2TIMGroupInfoResult;
import com.tencent.imsdk.v2.V2TIMGroupMemberFullInfo;
import com.tencent.imsdk.v2.V2TIMGroupMemberInfo;
import com.tencent.imsdk.v2.V2TIMGroupMemberInfoResult;
import com.tencent.imsdk.v2.V2TIMGroupMessageReadMemberList;
import com.tencent.imsdk.v2.V2TIMGroupTipsElem;
import com.tencent.imsdk.v2.V2TIMManager;
import com.tencent.imsdk.v2.V2TIMMergerElem;
import com.tencent.imsdk.v2.V2TIMMessage;
import com.tencent.imsdk.v2.V2TIMMessageListGetOption;
import com.tencent.imsdk.v2.V2TIMMessageReceipt;
import com.tencent.imsdk.v2.V2TIMOfflinePushInfo;
import com.tencent.imsdk.v2.V2TIMSendCallback;
import com.tencent.imsdk.v2.V2TIMUserFullInfo;
import com.tencent.imsdk.v2.V2TIMUserStatus;
import com.tencent.imsdk.v2.V2TIMValueCallback;
import com.tencent.qcloud.tuicore.TUIConfig;
import com.tencent.qcloud.tuicore.TUIConstants;
import com.tencent.qcloud.tuicore.TUICore;
import com.tencent.qcloud.tuicore.TUILogin;
import com.tencent.qcloud.tuicore.interfaces.TUICallback;
import com.tencent.qcloud.tuicore.interfaces.TUIValueCallback;
import com.tencent.qcloud.tuicore.push.OfflinePushExtConfigInfo;
import com.tencent.qcloud.tuicore.push.OfflinePushExtInfo;
import com.tencent.qcloud.tuicore.util.ErrorMessageConverter;
import com.tencent.qcloud.tuikit.timcommon.bean.MessageFeature;
import com.tencent.qcloud.tuikit.timcommon.bean.MessageReceiptInfo;
import com.tencent.qcloud.tuikit.timcommon.bean.TUIMessageBean;
import com.tencent.qcloud.tuikit.timcommon.bean.UserBean;
import com.tencent.qcloud.tuikit.timcommon.component.face.FaceManager;
import com.tencent.qcloud.tuikit.timcommon.component.interfaces.IUIKitCallback;
import com.tencent.qcloud.tuikit.tuichat.TUIChatConstants;
import com.tencent.qcloud.tuikit.tuichat.bean.ChatInfo;
import com.tencent.qcloud.tuikit.tuichat.bean.GroupApplyInfo;
import com.tencent.qcloud.tuikit.tuichat.bean.GroupMemberInfo;
import com.tencent.qcloud.tuikit.tuichat.bean.GroupMessageReadMembersInfo;
import com.tencent.qcloud.tuikit.tuichat.bean.OfflinePushInfo;
import com.tencent.qcloud.tuikit.tuichat.bean.UserStatusBean;
import com.tencent.qcloud.tuikit.tuichat.bean.message.MergeMessageBean;
import com.tencent.qcloud.tuikit.tuichat.bean.message.TipsMessageBean;
import com.tencent.qcloud.tuikit.tuichat.config.TUIChatConfigs;
import com.tencent.qcloud.tuikit.tuichat.util.ChatMessageParser;
import com.tencent.qcloud.tuikit.tuichat.util.OfflinePushInfoUtils;
import com.tencent.qcloud.tuikit.tuichat.util.TUIChatLog;
import com.tencent.qcloud.tuikit.tuichat.util.TUIChatUtils;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ChatProvider {
    private static final String TAG = ChatProvider.class.getSimpleName();
    private boolean lastInitialLoadUsedLocal;

    public static final int ERR_REVOKE_TIME_LIMIT_EXCEED = BaseConstants.ERR_REVOKE_TIME_LIMIT_EXCEED;
    public static final int ERR_REVOKE_TIME_LIMIT_SVR_GROUP = BaseConstants.ERR_SVR_GROUP_REVOKE_MSG_TIME_LIMIT;
    public static final int ERR_REVOKE_TIME_LIMIT_SVR_MESSAGE = BaseConstants.ERR_SVR_MSG_REVOKE_TIME_LIMIT;

    public void loadC2CMessage(String userId, int count, TUIMessageBean lastMessageInfo, IUIKitCallback<Pair<List<TUIMessageBean>, Integer>> callBack) {
        V2TIMMessage lastTIMMsg = null;
        if (lastMessageInfo != null) {
            lastTIMMsg = lastMessageInfo.getV2TIMMessage();
        }
        V2TIMManager.getMessageManager().getC2CHistoryMessageList(userId, count, lastTIMMsg, new V2TIMValueCallback<List<V2TIMMessage>>() {
            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
                TUIChatLog.e(
                    TAG, "loadChatMessages getC2CHistoryMessageList failed, code = " + code + ", desc = " + ErrorMessageConverter.convertIMError(code, desc));
            }

            @Override
            public void onSuccess(List<V2TIMMessage> v2TIMMessages) {
                List<TUIMessageBean> messageInfoList = ChatMessageParser.parsePresentMessageList(v2TIMMessages);
                TUIChatUtils.callbackOnSuccess(callBack, Pair.create(messageInfoList, v2TIMMessages.size()));
            }
        });
    }

    public void loadGroupMessage(String groupId, int count, TUIMessageBean lastMessageInfo, IUIKitCallback<Pair<List<TUIMessageBean>, Integer>> callBack) {
        V2TIMMessage lastTimMsg = null;
        if (lastMessageInfo != null) {
            lastTimMsg = lastMessageInfo.getV2TIMMessage();
        }
        V2TIMManager.getMessageManager().getGroupHistoryMessageList(groupId, count, lastTimMsg, new V2TIMValueCallback<List<V2TIMMessage>>() {
            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
                TUIChatLog.e(
                    TAG, "loadChatMessages getC2CHistoryMessageList failed, code = " + code + ", desc = " + ErrorMessageConverter.convertIMError(code, desc));
            }

            @Override
            public void onSuccess(List<V2TIMMessage> v2TIMMessages) {
                List<TUIMessageBean> messageInfoList = ChatMessageParser.parsePresentMessageList(v2TIMMessages);
                TUIChatUtils.callbackOnSuccess(callBack, Pair.create(messageInfoList, v2TIMMessages.size()));
            }
        });
    }

    /** Reads the SDK cache first; the presenter may pull newer cloud messages after the first frame. */
    public void loadInitialMessage(String chatId, boolean isGroup, int count,
                                   IUIKitCallback<Pair<List<TUIMessageBean>, Integer>> callBack) {
        lastInitialLoadUsedLocal = false;
        V2TIMMessageListGetOption option = new V2TIMMessageListGetOption();
        option.setCount(count);
        option.setGetType(V2TIMMessageListGetOption.V2TIM_GET_LOCAL_OLDER_MSG);
        if (isGroup) {
            option.setGroupID(chatId);
        } else {
            option.setUserID(chatId);
        }

        V2TIMManager.getMessageManager().getHistoryMessageList(option, new V2TIMValueCallback<List<V2TIMMessage>>() {
            @Override
            public void onSuccess(List<V2TIMMessage> messages) {
                if (messages != null && !messages.isEmpty()) {
                    lastInitialLoadUsedLocal = true;
                    TUIChatLog.i(TAG, "initial local message hit, group=" + isGroup + ", count=" + messages.size());
                    List<TUIMessageBean> parsed = ChatMessageParser.parsePresentMessageList(messages);
                    TUIChatUtils.callbackOnSuccess(callBack, Pair.create(parsed, messages.size()));
                    return;
                }
                TUIChatLog.i(TAG, "initial local message empty, online=" + isOnline());
                loadInitialCloudIfOnline(chatId, isGroup, count, callBack);
            }

            @Override
            public void onError(int code, String desc) {
                TUIChatLog.w(TAG, "initial local message failed, code=" + code + ", online=" + isOnline());
                if (isOnline()) {
                    loadInitialCloudIfOnline(chatId, isGroup, count, callBack);
                } else {
                    TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
                }
            }
        });
    }

    public boolean wasLastInitialLoadLocal() {
        return lastInitialLoadUsedLocal;
    }

    /** Loads only messages newer than the SDK-local newest message. */
    public void loadNewerMessageList(String chatId, boolean isGroup, int count, TUIMessageBean lastMessage,
                                     IUIKitCallback<Pair<List<TUIMessageBean>, Integer>> callBack) {
        V2TIMMessageListGetOption option = new V2TIMMessageListGetOption();
        option.setCount(count);
        option.setGetType(V2TIMMessageListGetOption.V2TIM_GET_CLOUD_NEWER_MSG);
        if (lastMessage != null) {
            option.setLastMsg(lastMessage.getV2TIMMessage());
        }
        if (isGroup) {
            option.setGroupID(chatId);
        } else {
            option.setUserID(chatId);
        }
        V2TIMManager.getMessageManager().getHistoryMessageList(option, new V2TIMValueCallback<List<V2TIMMessage>>() {
            @Override
            public void onSuccess(List<V2TIMMessage> messages) {
                List<TUIMessageBean> parsed = messages == null
                    ? new ArrayList<>() : ChatMessageParser.parsePresentMessageList(messages);
                TUIChatUtils.callbackOnSuccess(callBack, Pair.create(parsed, messages == null ? 0 : messages.size()));
            }

            @Override
            public void onError(int code, String desc) {
                TUIChatLog.w(TAG, "load newer message failed, code=" + code);
                TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
            }
        });
    }

    private void loadInitialCloudIfOnline(String chatId, boolean isGroup, int count,
                                          IUIKitCallback<Pair<List<TUIMessageBean>, Integer>> callBack) {
        if (!isOnline()) {
            TUIChatUtils.callbackOnError(callBack, TAG, -1, "本地没有可用消息");
            return;
        }
        if (isGroup) {
            loadGroupMessage(chatId, count, null, callBack);
        } else {
            loadC2CMessage(chatId, count, null, callBack);
        }
    }

    private boolean isOnline() {
        return V2TIMManager.getInstance().getLoginStatus() == V2TIMManager.V2TIM_STATUS_LOGINED;
    }

    public void loadHistoryMessageList(String chatId, boolean isGroup, int loadCount, TUIMessageBean locateMessageInfo, int getType,
        IUIKitCallback<Pair<List<TUIMessageBean>, Integer>> callBack) {
        V2TIMMessageListGetOption optionBackward = new V2TIMMessageListGetOption();
        optionBackward.setCount(loadCount);
        boolean online = isOnline();
        if (getType == TUIChatConstants.GET_MESSAGE_FORWARD) {
            optionBackward.setGetType(online
                ? V2TIMMessageListGetOption.V2TIM_GET_CLOUD_OLDER_MSG
                : V2TIMMessageListGetOption.V2TIM_GET_LOCAL_OLDER_MSG);
        } else if (getType == TUIChatConstants.GET_MESSAGE_BACKWARD) {
            optionBackward.setGetType(online
                ? V2TIMMessageListGetOption.V2TIM_GET_CLOUD_NEWER_MSG
                : V2TIMMessageListGetOption.V2TIM_GET_LOCAL_NEWER_MSG);
        }
        if (locateMessageInfo != null) {
            optionBackward.setLastMsg(locateMessageInfo.getV2TIMMessage());
        }
        if (isGroup) {
            optionBackward.setGroupID(chatId);
        } else {
            optionBackward.setUserID(chatId);
        }

        V2TIMManager.getMessageManager().getHistoryMessageList(optionBackward, new V2TIMValueCallback<List<V2TIMMessage>>() {
            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
                TUIChatLog.e(TAG,
                    "loadChatMessages getHistoryMessageList optionBackward failed, code = " + code
                        + ", desc = " + ErrorMessageConverter.convertIMError(code, desc));
            }

            @Override
            public void onSuccess(List<V2TIMMessage> v2TIMMessages) {
                List<TUIMessageBean> messageInfoList = ChatMessageParser.parsePresentMessageList(v2TIMMessages);
                TUIChatUtils.callbackOnSuccess(callBack, Pair.create(messageInfoList, v2TIMMessages.size()));
            }
        });
    }

    public void loadLastMessage(String chatId, boolean isGroup, boolean isGetLocalMessage, IUIKitCallback<TUIMessageBean> callBack) {
        V2TIMMessageListGetOption optionBackward = new V2TIMMessageListGetOption();
        optionBackward.setCount(1);
        optionBackward.setGetType(
            isGetLocalMessage ? V2TIMMessageListGetOption.V2TIM_GET_LOCAL_OLDER_MSG : V2TIMMessageListGetOption.V2TIM_GET_CLOUD_OLDER_MSG);
        if (isGroup) {
            optionBackward.setGroupID(chatId);
        } else {
            optionBackward.setUserID(chatId);
        }

        V2TIMManager.getMessageManager().getHistoryMessageList(optionBackward, new V2TIMValueCallback<List<V2TIMMessage>>() {
            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
                TUIChatLog.e(TAG, "loadLastMessage failed, code = " + code + ", desc = " + ErrorMessageConverter.convertIMError(code, desc));
            }

            @Override
            public void onSuccess(List<V2TIMMessage> v2TIMMessages) {
                List<TUIMessageBean> messageInfoList = ChatMessageParser.parsePresentMessageList(v2TIMMessages);
                if (messageInfoList == null || messageInfoList.isEmpty()) {
                    TUIChatUtils.callbackOnError(callBack, TAG, -1, "load last message failed.");
                } else {
                    TUIChatUtils.callbackOnSuccess(callBack, messageInfoList.get(0));
                }
            }
        });
    }

    public void c2cReadReport(String userId) {
        String conversationID = TUIChatUtils.getConversationIdByChatId(userId, false);
        V2TIMManager.getConversationManager().cleanConversationUnreadMessageCount(conversationID, 0, 0, new V2TIMCallback() {
            @Override
            public void onError(int code, String desc) {
                TUIChatLog.e(
                    TAG, "cleanConversationUnreadMessageCount failed, code = " + code + ", desc = " + ErrorMessageConverter.convertIMError(code, desc));
            }

            @Override
            public void onSuccess() {
                TUIChatLog.d(TAG, "cleanConversationUnreadMessageCount success");
            }
        });

        List<String> conversationIDList = new ArrayList<>();
        conversationIDList.add(conversationID);
        V2TIMManager.getConversationManager().markConversation(
            conversationIDList, V2TIMConversation.V2TIM_CONVERSATION_MARK_TYPE_UNREAD, false, new V2TIMValueCallback<List<V2TIMConversationOperationResult>>() {
                @Override
                public void onSuccess(List<V2TIMConversationOperationResult> v2TIMConversationOperationResults) {
                    if (v2TIMConversationOperationResults.size() > 0) {
                        V2TIMConversationOperationResult result = v2TIMConversationOperationResults.get(0);
                        TUIChatLog.d(TAG, "mark C2C conversation unread disable success, code:" + result.getResultCode() + "|msg:" + result.getResultInfo());
                    } else {
                        TUIChatLog.e(TAG, "mark C2C conversation unread disable failed, results size = 0");
                    }
                }

                @Override
                public void onError(int code, String desc) {
                    TUIChatLog.e(
                        TAG, "mark C2C conversation unread disable failed, code = " + code + ", desc = " + ErrorMessageConverter.convertIMError(code, desc));
                }
            });
    }

    public void groupReadReport(String groupId) {
        String conversationID = TUIChatUtils.getConversationIdByChatId(groupId, true);
        V2TIMManager.getConversationManager().cleanConversationUnreadMessageCount(conversationID, 0, 0, new V2TIMCallback() {
            @Override
            public void onError(int code, String desc) {
                TUIChatLog.e(
                    TAG, "cleanConversationUnreadMessageCount failed, code = " + code + ", desc = " + ErrorMessageConverter.convertIMError(code, desc));
            }

            @Override
            public void onSuccess() {
                TUIChatLog.d(TAG, "cleanConversationUnreadMessageCount success");
            }
        });

        List<String> conversationIDList = new ArrayList<>();
        conversationIDList.add(conversationID);
        V2TIMManager.getConversationManager().markConversation(
            conversationIDList, V2TIMConversation.V2TIM_CONVERSATION_MARK_TYPE_UNREAD, false, new V2TIMValueCallback<List<V2TIMConversationOperationResult>>() {
                @Override
                public void onSuccess(List<V2TIMConversationOperationResult> v2TIMConversationOperationResults) {
                    if (v2TIMConversationOperationResults.size() > 0) {
                        V2TIMConversationOperationResult result = v2TIMConversationOperationResults.get(0);
                        TUIChatLog.d(TAG, "mark group conversation unread disable success, code:" + result.getResultCode() + "|msg:" + result.getResultInfo());
                    } else {
                        TUIChatLog.e(TAG, "mark group conversation unread disable failed, results size = 0");
                    }
                }

                @Override
                public void onError(int code, String desc) {
                    TUIChatLog.e(
                        TAG, "mark group conversation unread disable failed, code = " + code + ", desc = " + ErrorMessageConverter.convertIMError(code, desc));
                }
            });
    }

    public void loadApplyInfo(final IUIKitCallback<List<GroupApplyInfo>> callBack) {
        final List<GroupApplyInfo> applies = new ArrayList<>();

        V2TIMManager.getGroupManager().getGroupApplicationList(new V2TIMValueCallback<V2TIMGroupApplicationResult>() {
            @Override
            public void onError(int code, String desc) {
                TUIChatLog.e(TAG, "getGroupPendencyList failed, code: " + code + "|desc: " + ErrorMessageConverter.convertIMError(code, desc));
                callBack.onError(TAG, code, ErrorMessageConverter.convertIMError(code, desc));
            }

            @Override
            public void onSuccess(V2TIMGroupApplicationResult v2TIMGroupApplicationResult) {
                List<V2TIMGroupApplication> v2TIMGroupApplicationList = v2TIMGroupApplicationResult.getGroupApplicationList();
                for (int i = 0; i < v2TIMGroupApplicationList.size(); i++) {
                    GroupApplyInfo info = new GroupApplyInfo(v2TIMGroupApplicationList.get(i));
                    applies.add(info);
                }
                callBack.onSuccess(applies);
            }
        });
    }

    private void setMessageTypingFeature(TUIMessageBean messageBean) {
        MessageFeature messageFeature = new MessageFeature();
        messageFeature.setNeedTyping(1);
        messageBean.setMessageTypingFeature(messageFeature);
    }

    public String sendTypingStatusMessage(TUIMessageBean message, String receiver, IUIKitCallback<TUIMessageBean> callBack) {
        V2TIMMessage v2TIMMessage = message.getV2TIMMessage();
        v2TIMMessage.setExcludedFromContentModeration(true);
        String msgID = V2TIMManager.getMessageManager().sendMessage(
            v2TIMMessage, receiver, null, V2TIMMessage.V2TIM_PRIORITY_DEFAULT, true, null, new V2TIMSendCallback<V2TIMMessage>() {
                @Override
                public void onError(int code, String desc) {
                    TUIChatLog.v(TAG, "sendMessage fail:" + code + "=" + ErrorMessageConverter.convertIMError(code, desc));
                    TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
                }

                @Override
                public void onSuccess(V2TIMMessage v2TIMMessage) {
                    TUIChatLog.v(TAG, "sendMessage onSuccess:" + v2TIMMessage.getMsgID());
                    message.setStatus(TUIMessageBean.MSG_STATUS_SEND_SUCCESS);
                    message.setV2TIMMessage(v2TIMMessage);
                    TUIChatUtils.callbackOnSuccess(callBack, message);
                    Map<String, Object> param = new HashMap<>();
                    param.put(TUIConstants.TUIChat.CHAT_ID, receiver);
                    TUICore.notifyEvent(TUIConstants.TUIChat.EVENT_KEY_MESSAGE_EVENT, TUIConstants.TUIChat.EVENT_SUB_KEY_SEND_MESSAGE_SUCCESS, param);
                }

                @Override
                public void onProgress(int progress) {}
            });
        return msgID;
    }

    private V2TIMOfflinePushInfo createOfflinePushInfo(TUIMessageBean message, ChatInfo chatInfo) {
        String description = FaceManager.emojiJudge(message.getExtra()).toString();

        OfflinePushExtInfo offlinePushExtInfo = new OfflinePushExtInfo();
        String sender = message.getSender();
        offlinePushExtInfo.getBusinessInfo().setDesc(description);
        offlinePushExtInfo.getBusinessInfo().setSenderId(sender);
        offlinePushExtInfo.getBusinessInfo().setFaceUrl(TUIConfig.getSelfFaceUrl());
        String senderNickName = chatInfo.getChatName();

        String groupID = "";
        if (chatInfo.getType() == V2TIMConversation.V2TIM_GROUP) {
            groupID = chatInfo.getId();
            offlinePushExtInfo.getBusinessInfo().setChatType(V2TIMConversation.V2TIM_GROUP);
            offlinePushExtInfo.getBusinessInfo().setSenderId(groupID);
            if (TextUtils.isEmpty(senderNickName)) {
                senderNickName = groupID;
            }
        } else {
            String selfNickName = TUIConfig.getSelfNickName();
            if (TextUtils.isEmpty(selfNickName)) {
                selfNickName = TUILogin.getLoginUser();
            }
            senderNickName = selfNickName;
        }
        offlinePushExtInfo.getBusinessInfo().setSenderNickName(senderNickName);
        offlinePushExtInfo.getConfigInfo().setFCMPushType(OfflinePushExtConfigInfo.FCM_PUSH_TYPE_DATA);
        offlinePushExtInfo.getConfigInfo().setFCMNotificationType(OfflinePushExtConfigInfo.FCM_NOTIFICATION_TYPE_TIMPUSH);

        // setCustomData Example
        /*ChatInfo info = new ChatInfo();
        info.setChatName("eeeee");
        info.setId("32323");
        offlinePushExtInfo.getEntity().setCustomData(new Gson().toJson(info).getBytes());*/

        V2TIMOfflinePushInfo v2TIMOfflinePushInfo = new V2TIMOfflinePushInfo();
        v2TIMOfflinePushInfo.setTitle(senderNickName);

        if (!TextUtils.isEmpty(description)) {
            v2TIMOfflinePushInfo.setDesc(description);
        }
        v2TIMOfflinePushInfo.setExt(new Gson().toJson(offlinePushExtInfo).getBytes());
        // OPPO must set a ChannelID to receive push messages. This channelID needs to be the same as the console.
        v2TIMOfflinePushInfo.setAndroidOPPOChannelID("tuikit");
        if (TUIChatConfigs.getConfigs().getGeneralConfig().isEnableAndroidPrivateRing()) {
            v2TIMOfflinePushInfo.setAndroidSound(OfflinePushInfoUtils.PRIVATE_RING_NAME);
            v2TIMOfflinePushInfo.setAndroidFCMChannelID(OfflinePushInfoUtils.FCM_PUSH_CHANNEL_ID);
        }
        v2TIMOfflinePushInfo.setAndroidHuaWeiCategory("IM");
        v2TIMOfflinePushInfo.setAndroidVIVOCategory("IM");
        v2TIMOfflinePushInfo.setIOSInterruptionLevel("time-sensitive");
        v2TIMOfflinePushInfo.enableIOSBackgroundNotification(false);
        v2TIMOfflinePushInfo.setAndroidHonorImportance("NORMAL");

        return v2TIMOfflinePushInfo;
    }

    public String sendMessage(TUIMessageBean message, ChatInfo chatInfo, boolean onlineUserOnly, IUIKitCallback<TUIMessageBean> callBack) {
        final V2TIMMessage v2TIMMessage = message.getV2TIMMessage();
        if (v2TIMMessage == null) {
            return null;
        }
        // support message typing flag
        setMessageTypingFeature(message);

        String userID = "";
        String groupID = "";
        boolean isGroup = false;
        if (chatInfo.getType() == V2TIMConversation.V2TIM_GROUP) {
            groupID = chatInfo.getId();
            isGroup = true;
        } else {
            userID = chatInfo.getId();
        }

        V2TIMOfflinePushInfo v2TIMOfflinePushInfo = createOfflinePushInfo(message, chatInfo);

        v2TIMMessage.setExcludedFromUnreadCount(TUIChatConfigs.getConfigs().getGeneralConfig().isExcludedFromUnreadCount());
        v2TIMMessage.setExcludedFromLastMessage(TUIChatConfigs.getConfigs().getGeneralConfig().isExcludedFromLastMessage());

        String targetUserID = isGroup ? null : userID;
        String targetGroupID = isGroup ? groupID : null;
        // Every primary outgoing message uses the same account dispatcher. Media keeps the SDK's
        // existing upload/progress callbacks, but no longer bypasses per-conversation FIFO.
        return enqueueDispatchedMessage(message, chatInfo, v2TIMMessage, targetUserID, targetGroupID,
            onlineUserOnly, v2TIMOfflinePushInfo, callBack);
    }

    private static boolean isAudioMessage(V2TIMMessage message) {
        return message != null && message.getSoundElem() != null;
    }

    private static boolean isShortMessage(V2TIMMessage message) {
        return message != null && (message.getTextElem() != null || message.getFaceElem() != null);
    }

    private String enqueueDispatchedMessage(TUIMessageBean message, ChatInfo chatInfo, V2TIMMessage v2TIMMessage,
                                              String userID, String groupID, boolean onlineUserOnly,
                                              V2TIMOfflinePushInfo offlinePushInfo,
                                              IUIKitCallback<TUIMessageBean> callBack) {
        MessageSendDispatcher dispatcher = MessageSendDispatcher.getActiveDispatcher();
        if (dispatcher == null) {
            dispatcher = MessageSendDispatcher.activateAccount(TUILogin.getLoginUser());
        }
        if (dispatcher == null) {
            TUIChatUtils.callbackOnError(callBack, TAG, -1, "IM账号未登录");
            return null;
        }

        MessageSendDispatcher.MessageType type = isAudioMessage(v2TIMMessage)
            ? MessageSendDispatcher.MessageType.AUDIO
            : (isShortMessage(v2TIMMessage) ? MessageSendDispatcher.MessageType.SHORT : MessageSendDispatcher.MessageType.MEDIA);
        MessageSendDispatcher.Submission submission = dispatcher.submit(
            chatInfo.getType() + ":" + chatInfo.getId(),
            type,
            attemptCallback -> V2TIMManager.getMessageManager().sendMessage(v2TIMMessage, userID, groupID,
                V2TIMMessage.V2TIM_PRIORITY_DEFAULT, onlineUserOnly, offlinePushInfo,
                new V2TIMSendCallback<V2TIMMessage>() {
                    @Override
                    public void onProgress(int progress) {
                        attemptCallback.onProgress(progress);
                    }

                    @Override
                    public void onError(int code, String desc) {
                        attemptCallback.onError(code, desc);
                    }

                    @Override
                    public void onSuccess(V2TIMMessage sentMessage) {
                        attemptCallback.onSuccess(sentMessage);
                    }
                }),
            new MessageSendDispatcher.Listener() {
                @Override
                public void onQueued(String clientMessageId) {
                    message.setClientMessageId(clientMessageId);
                    message.setSendStage(TUIMessageBean.SendStage.QUEUED);
                    message.setStatus(TUIMessageBean.MSG_STATUS_SENDING);
                }

                @Override
                public void onSending(String clientMessageId, int retryCount) {
                    message.setSendStage(TUIMessageBean.SendStage.SENDING);
                }

                @Override
                public void onProgress(String clientMessageId, int progress) {
                    TUIChatUtils.callbackOnProgress(callBack, progress);
                }

                @Override
                public void onSdkMessageId(String clientMessageId, String sdkMessageId) {
                    if (!TextUtils.isEmpty(sdkMessageId)) {
                        message.setId(sdkMessageId);
                    }
                }

                @Override
                public void onSuccess(String clientMessageId, V2TIMMessage sentMessage) {
                    if (sentMessage == null) {
                        onError(clientMessageId, -1, "腾讯IM返回空消息");
                        return;
                    }
                    message.setV2TIMMessage(sentMessage);
                    message.setSendStage(TUIMessageBean.SendStage.SUCCESS);
                    TUIChatUtils.callbackOnSuccess(callBack, message);
                    notifySendSuccess(chatInfo.getId(), sentMessage);
                }

                @Override
                public void onError(String clientMessageId, int code, String desc) {
                    message.setSendStage(TUIMessageBean.SendStage.FAILED);
                    TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
                    notifySendFailure(v2TIMMessage);
                }
            });
        if (!submission.isAccepted()) {
            TUIChatUtils.callbackOnError(callBack, TAG, -2, "发送过快，请稍后重试");
            return null;
        }
        return submission.getClientMessageId();
    }

    private String sendMessageDirect(TUIMessageBean message, ChatInfo chatInfo, V2TIMMessage v2TIMMessage,
                                     String userID, String groupID, boolean onlineUserOnly,
                                     V2TIMOfflinePushInfo offlinePushInfo, IUIKitCallback<TUIMessageBean> callBack) {
        DirectMessageSendRetry retry = new DirectMessageSendRetry(
            attemptCallback -> V2TIMManager.getMessageManager().sendMessage(v2TIMMessage, userID, groupID,
                V2TIMMessage.V2TIM_PRIORITY_DEFAULT, onlineUserOnly, offlinePushInfo, new V2TIMSendCallback<V2TIMMessage>() {
                @Override
                public void onProgress(int progress) {
                    attemptCallback.onProgress(progress);
                }

                @Override
                public void onError(int code, String desc) {
                    attemptCallback.onError(code, desc);
                }

                @Override
                public void onSuccess(V2TIMMessage sentMessage) {
                    attemptCallback.onSuccess(sentMessage);
                }
            }),
            new DirectMessageSendRetry.Listener() {
                @Override
                public void onProgress(int progress) {
                    TUIChatUtils.callbackOnProgress(callBack, progress);
                }

                @Override
                public void onError(int code, String desc) {
                    message.setSendStage(TUIMessageBean.SendStage.FAILED);
                    TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
                    notifySendFailure(v2TIMMessage);
                }

                @Override
                public void onSuccess(V2TIMMessage sentMessage) {
                    message.setV2TIMMessage(sentMessage);
                    message.setSendStage(TUIMessageBean.SendStage.SUCCESS);
                    TUIChatUtils.callbackOnSuccess(callBack, message);
                    notifySendSuccess(chatInfo.getId(), sentMessage);
                }
            });
        return retry.start();
    }

    private static void notifySendFailure(V2TIMMessage message) {
        Map<String, Object> param = new HashMap<>();
        param.put(TUIConstants.TUIChat.V2TIMMESSAGE, message);
        TUICore.notifyEvent(TUIConstants.TUIChat.EVENT_KEY_MESSAGE_EVENT,
            TUIConstants.TUIChat.EVENT_SUB_KEY_SEND_MESSAGE_FAILED, param);
    }

    private static void notifySendSuccess(String chatId, V2TIMMessage message) {
        Map<String, Object> param = new HashMap<>();
        param.put(TUIConstants.TUIChat.CHAT_ID, chatId);
        param.put(TUIConstants.TUIChat.V2TIMMESSAGE, message);
        TUICore.notifyEvent(TUIConstants.TUIChat.EVENT_KEY_MESSAGE_EVENT,
            TUIConstants.TUIChat.EVENT_SUB_KEY_SEND_MESSAGE_SUCCESS, param);
    }

    public String sendMessage(
        TUIMessageBean messageInfo, boolean isGroup, String id, OfflinePushInfo offlinePushInfo, IUIKitCallback<TUIMessageBean> callBack) {
        V2TIMMessage forwardMessage = messageInfo.getV2TIMMessage();
        if (forwardMessage == null || TextUtils.isEmpty(id)) {
            TUIChatUtils.callbackOnError(callBack, TAG, -1, "转发消息无效");
            return null;
        }
        forwardMessage.setExcludedFromUnreadCount(TUIChatConfigs.getConfigs().getGeneralConfig().isExcludedFromUnreadCount());
        forwardMessage.setExcludedFromLastMessage(TUIChatConfigs.getConfigs().getGeneralConfig().isExcludedFromLastMessage());

        V2TIMOfflinePushInfo v2TIMOfflinePushInfo = OfflinePushInfoUtils.convertOfflinePushInfoToV2PushInfo(offlinePushInfo);
        ChatInfo targetChat = new ChatInfo();
        targetChat.setId(id);
        targetChat.setType(isGroup ? ChatInfo.TYPE_GROUP : ChatInfo.TYPE_C2C);
        return enqueueDispatchedMessage(messageInfo, targetChat, forwardMessage, isGroup ? null : id, isGroup ? id : null,
            false, v2TIMOfflinePushInfo, callBack);
    }

    public String insertMessage(TUIMessageBean message, String receiver, boolean isGroup, TUIValueCallback<TUIMessageBean> callback)  {
        V2TIMMessage v2TIMMessage = message.getV2TIMMessage();
        if (v2TIMMessage == null) {
            return null;
        }
        String sender =  TUILogin.getLoginUser();
        String messageID;
        if (isGroup) {
            messageID = V2TIMManager.getMessageManager().insertGroupMessageToLocalStorage(
                    v2TIMMessage, receiver, sender, new V2TIMValueCallback<V2TIMMessage>() {
                @Override
                public void onSuccess(V2TIMMessage v2TIMMessage) {
                    TUIValueCallback.onSuccess(callback, ChatMessageParser.parsePresentMessage(v2TIMMessage));
                }

                @Override
                public void onError(int code, String desc) {
                    TUIValueCallback.onError(callback, code, desc);
                }
            });
        } else {
            messageID = V2TIMManager.getMessageManager().insertC2CMessageToLocalStorage(
                    v2TIMMessage, receiver, sender, new V2TIMValueCallback<V2TIMMessage>() {
                @Override
                public void onSuccess(V2TIMMessage v2TIMMessage) {
                    TUIValueCallback.onSuccess(callback, ChatMessageParser.parsePresentMessage(v2TIMMessage));
                }

                @Override
                public void onError(int code, String desc) {
                    TUIValueCallback.onError(callback, code, desc);
                }
            });
        }
        return messageID;
    }

    public void revokeMessage(TUIMessageBean messageInfo, IUIKitCallback<Void> callBack) {
        if (messageInfo == null || messageInfo.getV2TIMMessage() == null) {
            TUIChatUtils.callbackOnError(callBack, TAG, -1, "message is unavailable");
            return;
        }
        V2TIMManager.getMessageManager().revokeMessage(messageInfo.getV2TIMMessage(), new V2TIMCallback() {
            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
            }

            @Override
            public void onSuccess() {
                TUIChatUtils.callbackOnSuccess(callBack, null);
            }
        });
    }

    public void deleteMessages(List<TUIMessageBean> messageInfoList, IUIKitCallback<Void> callBack) {
        List<V2TIMMessage> v2TIMDeleteMessages = new ArrayList<>();
        for (int i = 0; i < messageInfoList.size(); i++) {
            v2TIMDeleteMessages.add(messageInfoList.get(i).getV2TIMMessage());
        }

        V2TIMManager.getMessageManager().deleteMessages(v2TIMDeleteMessages, new V2TIMCallback() {
            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callBack, TAG, code, desc);
                TUIChatLog.w(TAG, "deleteMessages code:" + code + "|desc:" + ErrorMessageConverter.convertIMError(code, desc));
            }

            @Override
            public void onSuccess() {
                TUIChatLog.i(TAG, "deleteMessages success");
                TUIChatUtils.callbackOnSuccess(callBack, null);
            }
        });
    }

    public void setDraft(String conversationId, String draft) {
        V2TIMManager.getConversationManager().setConversationDraft(conversationId, draft, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUIChatLog.i(TAG, "set draft success " + draft);
            }

            @Override
            public void onError(int code, String desc) {
                TUIChatLog.e(TAG, "set drafts error : " + code + " " + ErrorMessageConverter.convertIMError(code, desc));
            }
        });
    }

    public void addJoinGroupMessage(TUIMessageBean messageInfo, IUIKitCallback<List<GroupMemberInfo>> callback) {
        V2TIMGroupTipsElem groupTips = null;
        V2TIMMessage v2TIMMessage = messageInfo.getV2TIMMessage();
        if (v2TIMMessage != null) {
            groupTips = v2TIMMessage.getGroupTipsElem();
        }
        if (groupTips == null) {
            TUIChatUtils.callbackOnError(callback, TAG, -1, "groupTips is null");
            return;
        }
        List<V2TIMGroupMemberInfo> memberInfos = groupTips.getMemberList();
        List<GroupMemberInfo> groupMemberInfoList = new ArrayList<>();
        if (memberInfos.size() > 0) {
            for (V2TIMGroupMemberInfo v2TIMGroupMemberInfo : memberInfos) {
                GroupMemberInfo member = new GroupMemberInfo();
                member.covertTIMGroupMemberInfo(v2TIMGroupMemberInfo);
                groupMemberInfoList.add(member);
            }
        } else {
            GroupMemberInfo member = new GroupMemberInfo();
            member.covertTIMGroupMemberInfo(groupTips.getOpMember());
            groupMemberInfoList.add(member);
        }
        TUIChatUtils.callbackOnSuccess(callback, groupMemberInfoList);
    }

    public void addLeaveGroupMessage(TUIMessageBean messageInfo, IUIKitCallback<List<String>> callback) {
        V2TIMGroupTipsElem groupTips = null;
        V2TIMMessage v2TIMMessage = messageInfo.getV2TIMMessage();
        if (v2TIMMessage != null) {
            groupTips = v2TIMMessage.getGroupTipsElem();
        }
        if (groupTips == null) {
            TUIChatUtils.callbackOnError(callback, TAG, -1, "groupTips is null");
            return;
        }
        List<V2TIMGroupMemberInfo> memberInfos = groupTips.getMemberList();
        List<String> memberUserIds = new ArrayList<>();
        if (memberInfos.size() > 0) {
            for (V2TIMGroupMemberInfo v2TIMGroupMemberInfo : memberInfos) {
                String memberUserID = v2TIMGroupMemberInfo.getUserID();
                memberUserIds.add(memberUserID);
            }
        } else {
            V2TIMGroupMemberInfo memberInfo = groupTips.getOpMember();
            memberUserIds.add(memberInfo.getUserID());
        }
        TUIChatUtils.callbackOnSuccess(callback, memberUserIds);
    }

    public void addModifyGroupMessage(TUIMessageBean message, IUIKitCallback<Pair<Integer, String>> callback) {
        V2TIMGroupTipsElem groupTips = null;
        V2TIMMessage v2TIMMessage = message.getV2TIMMessage();
        if (v2TIMMessage != null) {
            groupTips = v2TIMMessage.getGroupTipsElem();
        }
        if (groupTips == null) {
            TUIChatUtils.callbackOnError(callback, TAG, -1, "groupTips is null");
            return;
        }
        List<V2TIMGroupChangeInfo> modifyList = groupTips.getGroupChangeInfoList();
        if (modifyList.size() > 0) {
            V2TIMGroupChangeInfo modifyInfo = modifyList.get(0);
            int modifyType = modifyInfo.getType();
            if (modifyType == V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_NAME) {
                TUIChatUtils.callbackOnSuccess(callback, new Pair<>(TipsMessageBean.MSG_TYPE_GROUP_MODIFY_NAME, modifyInfo.getValue()));
            } else if (modifyType == V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_NOTIFICATION) {
                TUIChatUtils.callbackOnSuccess(callback, new Pair<>(TipsMessageBean.MSG_TYPE_GROUP_MODIFY_NOTICE, modifyInfo.getValue()));
            }
        }
    }

    public boolean checkFailedMessageInfo(TUIMessageBean message) {
        if (message == null) {
            return true;
        }
        V2TIMMessage v2TIMMessage = message.getV2TIMMessage();
        if (v2TIMMessage.getStatus() == V2TIM_MSG_STATUS_SEND_FAIL) {
            return true;
        } else {
            return false;
        }
    }

    public void getFriendName(String id, IUIKitCallback<String[]> callback) {
        List<String> userIdList = new ArrayList<>();
        userIdList.add(id);
        V2TIMManager.getFriendshipManager().getFriendsInfo(userIdList, new V2TIMValueCallback<List<V2TIMFriendInfoResult>>() {
            @Override
            public void onSuccess(List<V2TIMFriendInfoResult> v2TIMFriendInfoResults) {
                V2TIMFriendInfoResult result = v2TIMFriendInfoResults.get(0);
                String[] nameArray = {result.getFriendInfo().getFriendRemark(), result.getFriendInfo().getUserProfile().getNickName()};
                TUIChatUtils.callbackOnSuccess(callback, nameArray);
            }

            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callback, code, desc);
            }
        });
    }

    public void findPresentMessages(List<String> msgIds, IUIKitCallback<List<TUIMessageBean>> callback) {
        findMessages(msgIds, new TUIValueCallback<List<TUIMessageBean>>() {
            @Override
            public void onSuccess(List<TUIMessageBean> messageBeans) {
                List<TUIMessageBean> presentMessageList = new ArrayList<>();
                if (messageBeans != null) {
                    for (TUIMessageBean message : messageBeans) {
                        if (message != null && message.getV2TIMMessage() != null
                                && message.getV2TIMMessage().getStatus() != V2TIM_MSG_STATUS_HAS_DELETED) {
                            presentMessageList.add(message);
                        }
                    }
                }
                TUIChatUtils.callbackOnSuccess(callback, presentMessageList);
            }

            @Override
            public void onError(int errorCode, String errorMessage) {
                TUIChatUtils.callbackOnError(callback, errorCode, errorMessage);
            }
        });
    }

    public void findMessages(List<String> msgIds, TUIValueCallback<List<TUIMessageBean>> callback) {
        V2TIMManager.getMessageManager().findMessages(msgIds, new V2TIMValueCallback<List<V2TIMMessage>>() {
            @Override
            public void onSuccess(List<V2TIMMessage> messages) {
                List<TUIMessageBean> messageBeans = ChatMessageParser.parseMessageList(messages);
                TUIValueCallback.onSuccess(callback, messageBeans);
            }

            @Override
            public void onError(int code, String desc) {
                TUIValueCallback.onError(callback, code, desc);
            }
        });
    }

    public void getGroupMessageBySeq(String chatId, long seq, IUIKitCallback<Pair<List<TUIMessageBean>, Integer>> callback) {
        V2TIMMessageListGetOption optionBackward = new V2TIMMessageListGetOption();
        optionBackward.setCount(1);
        optionBackward.setGetType(V2TIMMessageListGetOption.V2TIM_GET_CLOUD_OLDER_MSG);
        optionBackward.setLastMsgSeq(seq);
        optionBackward.setGroupID(chatId);

        V2TIMManager.getMessageManager().getHistoryMessageList(optionBackward, new V2TIMValueCallback<List<V2TIMMessage>>() {
            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callback, TAG, code, desc);
                TUIChatLog.e(TAG, "loadChatMessages getHistoryMessageList optionBackward failed, code = " + code + ", desc = " + desc);
            }

            @Override
            public void onSuccess(List<V2TIMMessage> v2TIMMessages) {
                List<TUIMessageBean> messageInfoList = ChatMessageParser.parsePresentMessageList(v2TIMMessages);
                TUIChatUtils.callbackOnSuccess(callback, Pair.create(messageInfoList, v2TIMMessages.size()));
            }
        });
    }

    public void downloadMergerMessage(MergeMessageBean messageBean, IUIKitCallback<List<TUIMessageBean>> callback) {
        V2TIMMergerElem mergerElem = messageBean.getMergerElem();
        if (mergerElem != null) {
            mergerElem.downloadMergerMessage(new V2TIMValueCallback<List<V2TIMMessage>>() {
                @Override
                public void onSuccess(List<V2TIMMessage> messageList) {
                    List<TUIMessageBean> messageInfoList = ChatMessageParser.parsePresentMessageList(messageList);
                    TUIChatUtils.callbackOnSuccess(callback, messageInfoList);
                }

                @Override
                public void onError(int code, String desc) {
                    TUIChatUtils.callbackOnError(callback, "MergeMessageElemBean", code, desc);
                }
            });
        }
    }

    public void getMessageReadReceipt(List<TUIMessageBean> messageBeanList, IUIKitCallback<List<MessageReceiptInfo>> callback) {
        List<V2TIMMessage> messageList = new ArrayList<>();
        for (TUIMessageBean messageBean : messageBeanList) {
            messageList.add(messageBean.getV2TIMMessage());
        }

        V2TIMManager.getMessageManager().getMessageReadReceipts(messageList, new V2TIMValueCallback<List<V2TIMMessageReceipt>>() {
            @Override
            public void onSuccess(List<V2TIMMessageReceipt> v2TIMGroupMessageReceipts) {
                List<MessageReceiptInfo> messageReceiptInfoList = new ArrayList<>();
                for (V2TIMMessageReceipt receipt : v2TIMGroupMessageReceipts) {
                    MessageReceiptInfo messageReceiptInfo = new MessageReceiptInfo();
                    messageReceiptInfo.setMessageReceipt(receipt);
                    messageReceiptInfoList.add(messageReceiptInfo);
                }
                TUIChatUtils.callbackOnSuccess(callback, messageReceiptInfoList);
            }

            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callback, code, desc);
            }
        });
    }

    public void sendMessageReadReceipt(List<TUIMessageBean> messageBeanList, IUIKitCallback<Void> callback) {
        List<V2TIMMessage> messageList = new ArrayList<>();
        for (TUIMessageBean messageBean : messageBeanList) {
            messageList.add(messageBean.getV2TIMMessage());
        }

        V2TIMManager.getMessageManager().sendMessageReadReceipts(messageList, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUIChatUtils.callbackOnSuccess(callback, null);
            }

            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callback, code, desc);
            }
        });
    }

    public void getGroupMessageReadMembers(
        TUIMessageBean messageBean, boolean isRead, int count, long nextSeq, IUIKitCallback<GroupMessageReadMembersInfo> callback) {
        V2TIMMessage message = messageBean.getV2TIMMessage();
        int filter = isRead ? V2TIMMessage.V2TIM_GROUP_MESSAGE_READ_MEMBERS_FILTER_READ : V2TIMMessage.V2TIM_GROUP_MESSAGE_READ_MEMBERS_FILTER_UNREAD;
        V2TIMManager.getMessageManager().getGroupMessageReadMemberList(
            message, filter, nextSeq, count, new V2TIMValueCallback<V2TIMGroupMessageReadMemberList>() {
                @Override
                public void onSuccess(V2TIMGroupMessageReadMemberList v2TIMGroupMessageReadMembers) {
                    GroupMessageReadMembersInfo messageReadMembersInfo = new GroupMessageReadMembersInfo();
                    messageReadMembersInfo.setReadMembers(v2TIMGroupMessageReadMembers);
                    TUIChatUtils.callbackOnSuccess(callback, messageReadMembersInfo);
                }

                @Override
                public void onError(int code, String desc) {
                    TUIChatUtils.callbackOnError(callback, code, desc);
                }
            });
    }

    public void modifyMessage(TUIMessageBean messageBean, IUIKitCallback<TUIMessageBean> callback) {
        V2TIMMessage v2TIMMessage = messageBean.getV2TIMMessage();
        V2TIMManager.getMessageManager().modifyMessage(v2TIMMessage, new V2TIMCompleteCallback<V2TIMMessage>() {
            @Override
            public void onComplete(int code, String desc, V2TIMMessage v2TIMMessage) {
                TUIMessageBean modifiedMessage = ChatMessageParser.parsePresentMessage(v2TIMMessage);
                if (code == ERR_SDK_MSG_MODIFY_CONFLICT) {
                    TUIChatUtils.callbackOnError(callback, code, desc, modifiedMessage);
                } else {
                    TUIChatUtils.callbackOnSuccess(callback, modifiedMessage);
                }
            }
        });
    }

    public void getUserBean(List<String> userIds, IUIKitCallback<List<UserBean>> callback) {
        V2TIMManager.getFriendshipManager().getFriendsInfo(userIds, new V2TIMValueCallback<List<V2TIMFriendInfoResult>>() {
            @Override
            public void onSuccess(List<V2TIMFriendInfoResult> v2TIMFriendInfoResults) {
                List<UserBean> reactUserBeanList = new ArrayList<>();
                for (V2TIMFriendInfoResult result : v2TIMFriendInfoResults) {
                    UserBean reactUserBean = new UserBean();
                    reactUserBean.setUserId(result.getFriendInfo().getUserID());
                    reactUserBean.setFriendRemark(result.getFriendInfo().getFriendRemark());
                    reactUserBean.setFaceUrl(result.getFriendInfo().getUserProfile().getFaceUrl());
                    if (result.getFriendInfo().getUserProfile() != null) {
                        reactUserBean.setNickName(result.getFriendInfo().getUserProfile().getNickName());
                    }
                    reactUserBeanList.add(reactUserBean);
                }
                TUIChatUtils.callbackOnSuccess(callback, reactUserBeanList);
            }

            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callback, code, desc);
            }
        });
    }

    public void getGroupMembersInfo(String groupId, List<String> groupMemberIds, IUIKitCallback<List<GroupMemberInfo>> callback) {
        if (TUIChatUtils.isTopicGroup(groupId)) {
            groupId = TUIChatUtils.getGroupIDFromTopicID(groupId);
        }
        V2TIMManager.getGroupManager().getGroupMembersInfo(groupId, groupMemberIds, new V2TIMValueCallback<List<V2TIMGroupMemberFullInfo>>() {
            @Override
            public void onSuccess(List<V2TIMGroupMemberFullInfo> v2TIMGroupMemberFullInfos) {
                List<GroupMemberInfo> groupMemberInfos = new ArrayList<>();
                for (V2TIMGroupMemberFullInfo fullInfo : v2TIMGroupMemberFullInfos) {
                    groupMemberInfos.add(new GroupMemberInfo().covertTIMGroupMemberInfo(fullInfo));
                }
                TUIChatUtils.callbackOnSuccess(callback, groupMemberInfos);
            }

            @Override
            public void onError(int code, String desc) {
                TUIChatUtils.callbackOnError(callback, code, desc);
            }
        });
    }

    public void loadUserStatus(List<String> userIDs, IUIKitCallback<Map<String, UserStatusBean>> callback) {
        if (userIDs == null || userIDs.size() == 0) {
            TUIChatLog.d(TAG, "loadContactUserStatus datasource is null");
            TUIChatUtils.callbackOnError(callback, -1, "data list is empty");
            return;
        }

        V2TIMManager.getInstance().getUserStatus(userIDs, new V2TIMValueCallback<List<V2TIMUserStatus>>() {
            @Override
            public void onSuccess(List<V2TIMUserStatus> v2TIMUserStatuses) {
                TUIChatLog.i(TAG, "getUserStatus success");
                Map<String, UserStatusBean> userStatusBeanMap = new HashMap<>();
                for (V2TIMUserStatus item : v2TIMUserStatuses) {
                    UserStatusBean statusBean = new UserStatusBean();
                    statusBean.setUserID(item.getUserID());
                    statusBean.setOnlineStatus(item.getStatusType());
                    userStatusBeanMap.put(item.getUserID(), statusBean);
                }
                TUIChatUtils.callbackOnSuccess(callback, userStatusBeanMap);
            }

            @Override
            public void onError(int code, String desc) {
                TUIChatLog.e(TAG, "getUserStatus error code = " + code + ",des = " + desc);
                TUIChatUtils.callbackOnError(callback, code, desc);
            }
        });
    }

    public void loadGroupMembers(String groupID, long nextSeq, final IUIKitCallback<List<GroupMemberInfo>> callBack) {
        int filter = V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_FILTER_ALL;
        V2TIMManager.getGroupManager().getGroupMemberList(groupID, filter, nextSeq, new V2TIMValueCallback<V2TIMGroupMemberInfoResult>() {
            @Override
            public void onError(int code, String desc) {
                TUIChatLog.e(TAG, "loadGroupMembers failed, code: " + code + "|desc: " + ErrorMessageConverter.convertIMError(code, desc));
                TUIChatUtils.callbackOnError(callBack, code, desc);
            }

            @Override
            public void onSuccess(V2TIMGroupMemberInfoResult v2TIMGroupMemberInfoResult) {
                List<GroupMemberInfo> members = new ArrayList<>();
                for (int i = 0; i < v2TIMGroupMemberInfoResult.getMemberInfoList().size(); i++) {
                    GroupMemberInfo member = new GroupMemberInfo();
                    members.add(member.covertTIMGroupMemberInfo(v2TIMGroupMemberInfoResult.getMemberInfoList().get(i)));
                }
                TUIChatUtils.callbackOnSuccess(callBack, members);
            }
        });
    }

    public void getChatName(String chatID, boolean isGroup, IUIKitCallback<String> callback) {
        if (!isGroup) {
            V2TIMManager.getFriendshipManager().getFriendsInfo(Collections.singletonList(chatID), new V2TIMValueCallback<List<V2TIMFriendInfoResult>>() {
                @Override
                public void onSuccess(List<V2TIMFriendInfoResult> v2TIMFriendInfoResults) {
                    if (v2TIMFriendInfoResults != null && !v2TIMFriendInfoResults.isEmpty()) {
                        V2TIMFriendInfo v2TIMFriendInfo = v2TIMFriendInfoResults.get(0).getFriendInfo();
                        String result;
                        if (v2TIMFriendInfo != null) {
                            String remark = v2TIMFriendInfo.getFriendRemark();
                            String nickName = null;
                            String id = v2TIMFriendInfo.getUserID();
                            V2TIMUserFullInfo fullInfo = v2TIMFriendInfo.getUserProfile();
                            if (fullInfo != null) {
                                nickName = fullInfo.getNickName();
                            }
                            if (!TextUtils.isEmpty(remark)) {
                                result = remark;
                            } else if (!TextUtils.isEmpty(nickName)) {
                                result = nickName;
                            } else {
                                result = id;
                            }
                        } else {
                            result = chatID;
                        }
                        TUIChatUtils.callbackOnSuccess(callback, result);
                    } else {
                        V2TIMManager.getInstance().getUsersInfo(Collections.singletonList(chatID), new V2TIMValueCallback<List<V2TIMUserFullInfo>>() {
                            @Override
                            public void onSuccess(List<V2TIMUserFullInfo> v2TIMUserFullInfos) {
                                if (v2TIMUserFullInfos != null && !v2TIMUserFullInfos.isEmpty()) {
                                    TUIChatUtils.callbackOnSuccess(callback, v2TIMUserFullInfos.get(0).getUserID());
                                } else {
                                    TUIChatUtils.callbackOnError(callback, -1, "get userInfo failed");
                                }
                            }

                            @Override
                            public void onError(int code, String desc) {
                                TUIChatUtils.callbackOnError(callback, code, desc);
                            }
                        });
                    }
                }

                @Override
                public void onError(int code, String desc) {
                    TUIChatUtils.callbackOnError(callback, code, desc);
                }
            });
        } else {
            V2TIMManager.getGroupManager().getGroupsInfo(Collections.singletonList(chatID), new V2TIMValueCallback<List<V2TIMGroupInfoResult>>() {
                @Override
                public void onSuccess(List<V2TIMGroupInfoResult> v2TIMGroupInfoResults) {
                    if (v2TIMGroupInfoResults != null && !v2TIMGroupInfoResults.isEmpty()) {
                        V2TIMGroupInfoResult v2TIMGroupInfoResult = v2TIMGroupInfoResults.get(0);
                        if (v2TIMGroupInfoResult.getResultCode() == ERR_SUCC) {
                            V2TIMGroupInfo groupInfo = v2TIMGroupInfoResults.get(0).getGroupInfo();
                            String result;
                            if (groupInfo != null) {
                                result = groupInfo.getGroupName();
                            } else {
                                result = chatID;
                            }
                            TUIChatUtils.callbackOnSuccess(callback, result);
                        } else {
                            TUIChatUtils.callbackOnError(callback, v2TIMGroupInfoResult.getResultCode(), v2TIMGroupInfoResult.getResultMessage());
                        }
                    } else {
                        TUIChatUtils.callbackOnError(callback, -1, "getGroupsInfo failed");
                    }
                }

                @Override
                public void onError(int code, String desc) {
                    TUIChatUtils.callbackOnError(callback, code, desc);
                }
            });
        }
    }

    public void getGroupInfo(String groupID, TUIValueCallback<V2TIMGroupInfo> callback) {
        V2TIMManager.getGroupManager().getGroupsInfo(Collections.singletonList(groupID), new V2TIMValueCallback<List<V2TIMGroupInfoResult>>() {
            @Override
            public void onSuccess(List<V2TIMGroupInfoResult> v2TIMGroupInfoResults) {
                if (!v2TIMGroupInfoResults.isEmpty()) {
                    V2TIMGroupInfoResult result = v2TIMGroupInfoResults.get(0);
                    if (result.getResultCode() == ERR_SUCC) {
                        TUIValueCallback.onSuccess(callback, result.getGroupInfo());
                    } else {
                        TUIValueCallback.onError(callback, result.getResultCode(), result.getResultMessage());
                    }
                }
            }

            @Override
            public void onError(int code, String desc) {
                TUIValueCallback.onError(callback, code, desc);
            }
        });
    }

    public void getChatFaceUrl(String chatID, boolean isGroup, IUIKitCallback<String> callback) {
        if (isGroup) {
            V2TIMManager.getGroupManager().getGroupsInfo(Collections.singletonList(chatID), new V2TIMValueCallback<List<V2TIMGroupInfoResult>>() {
                @Override
                public void onSuccess(List<V2TIMGroupInfoResult> v2TIMGroupInfoResults) {
                    V2TIMGroupInfoResult result = v2TIMGroupInfoResults.get(0);
                    if (result.getResultCode() == ERR_SUCC) {
                        TUIChatUtils.callbackOnSuccess(callback, result.getGroupInfo().getFaceUrl());
                    } else {
                        TUIChatUtils.callbackOnError(callback, result.getResultCode(), result.getResultMessage());
                    }
                }

                @Override
                public void onError(int code, String desc) {
                    TUIChatUtils.callbackOnError(callback, code, desc);
                }
            });
        } else {
            V2TIMManager.getInstance().getUsersInfo(Collections.singletonList(chatID), new V2TIMValueCallback<List<V2TIMUserFullInfo>>() {
                @Override
                public void onSuccess(List<V2TIMUserFullInfo> v2TIMUserFullInfos) {
                    V2TIMUserFullInfo result = v2TIMUserFullInfos.get(0);
                    TUIChatUtils.callbackOnSuccess(callback, result.getFaceUrl());
                }

                @Override
                public void onError(int code, String desc) {
                    TUIChatUtils.callbackOnError(callback, code, desc);
                }
            });
        }
    }

    public void getChatGridFaceUrls(String groupID, IUIKitCallback<List<Object>> callback) {
        V2TIMManager.getGroupManager().getGroupMemberList(
            groupID, V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_FILTER_ALL, 0, new V2TIMValueCallback<V2TIMGroupMemberInfoResult>() {
                @Override
                public void onSuccess(V2TIMGroupMemberInfoResult v2TIMGroupMemberInfoResult) {
                    List<V2TIMGroupMemberFullInfo> infoList = v2TIMGroupMemberInfoResult.getMemberInfoList();
                    int max = 9;
                    List<Object> faceUrls = new ArrayList<>();
                    for (V2TIMGroupMemberFullInfo fullInfo : infoList) {
                        if (max > 0) {
                            faceUrls.add(fullInfo.getFaceUrl());
                            max--;
                        } else {
                            break;
                        }
                    }
                    TUIChatUtils.callbackOnSuccess(callback, faceUrls);
                }

                @Override
                public void onError(int code, String desc) {
                    TUIChatUtils.callbackOnError(callback, code, desc);
                }
            });
    }

    public void pinGroupMessage(String groupID, TUIMessageBean messageBean, boolean isPinned, TUICallback callback) {
        if (messageBean == null || messageBean.getV2TIMMessage() == null) {
            TUICallback.onError(callback, -1, "message is null");
            return;
        }
        V2TIMMessage v2TIMMessage = messageBean.getV2TIMMessage();
        V2TIMManager.getMessageManager().pinGroupMessage(groupID, v2TIMMessage, isPinned, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void getGroupPinnedMessageList(String groupID, TUIValueCallback<List<TUIMessageBean>> callback) {
        V2TIMManager.getMessageManager().getPinnedGroupMessageList(groupID, new V2TIMValueCallback<List<V2TIMMessage>>() {
            @Override
            public void onSuccess(List<V2TIMMessage> v2TIMMessages) {
                List<TUIMessageBean> pinnedMessages = ChatMessageParser.parseMessageList(v2TIMMessages);
                TUIValueCallback.onSuccess(callback, pinnedMessages);
            }

            @Override
            public void onError(int code, String desc) {
                TUIValueCallback.onError(callback, code, desc);
            }
        });
    }
}
