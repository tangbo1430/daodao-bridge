package com.tencent.qcloud.tuikit.tuichat.classicui.page;

import android.content.Intent;
import android.os.Bundle;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;

import androidx.annotation.Nullable;

import com.tencent.imsdk.v2.V2TIMGroupInfoResult;
import com.tencent.imsdk.v2.V2TIMGroupListener;
import com.tencent.imsdk.v2.V2TIMManager;
import com.tencent.imsdk.v2.V2TIMValueCallback;
import com.tencent.qcloud.tuicore.TUIConstants;
import com.tencent.qcloud.tuicore.TUICore;
import com.tencent.qcloud.tuicore.interfaces.TUIExtensionEventListener;
import com.tencent.qcloud.tuicore.interfaces.TUIExtensionInfo;
import com.tencent.qcloud.tuicore.interfaces.TUIValueCallback;
import com.tencent.qcloud.tuikit.timcommon.bean.TUIMessageBean;
import com.tencent.qcloud.tuikit.tuichat.R;
import com.tencent.qcloud.tuikit.tuichat.TUIChatConstants;
import com.tencent.qcloud.tuikit.tuichat.bean.GroupChatInfo;
import com.tencent.qcloud.tuikit.tuichat.presenter.GroupChatPresenter;
import com.tencent.qcloud.tuikit.tuichat.util.TUIChatLog;
import com.tencent.qcloud.tuikit.tuichat.util.TUIChatUtils;

import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class TUIGroupChatFragment extends TUIBaseChatFragment {
    private static final String TAG = TUIGroupChatFragment.class.getSimpleName();

    String groupGuard = ""; //群成员点击保护  1开

    int myRole = 0;
    private V2TIMGroupListener groupStateListener;
    private int viewGeneration;
    private boolean viewAlive;
    private boolean allMuted;
    private final android.os.Handler uiHandler = new android.os.Handler(android.os.Looper.getMainLooper());
    private final GroupChatPresenter presenter;

    public TUIGroupChatFragment() {
        presenter = new GroupChatPresenter();
        presenter.initListener();
    }


    @Nullable
    @Override
    public View onCreateView(LayoutInflater inflater, @Nullable ViewGroup container, Bundle savedInstanceState) {
        TUIChatLog.i(TAG, "oncreate view " + this);

        baseView = super.onCreateView(inflater, container, savedInstanceState);
        if (!(chatInfo instanceof GroupChatInfo)) {
            return baseView;
        }
        initView();
        return baseView;
    }

    @Override
    protected void initView() {
        super.initView();
        viewAlive = true;
        final int generation = ++viewGeneration;
        chatView.setPresenter(presenter);
        presenter.setGroupInfo((GroupChatInfo) chatInfo);
        chatView.setChatInfo(chatInfo);

        presenter.getGroupType(chatInfo.getId(), new TUIValueCallback<String>() {
            @Override
            public void onSuccess(String type) {
                if (!viewAlive || generation != viewGeneration) return;
                ((GroupChatInfo) chatInfo).setGroupType(type);
                setTitleBarExtension();
            }

            @Override
            public void onError(int errorCode, String errorMessage) {
                if (viewAlive && generation == viewGeneration) setTitleBarExtension();
            }
        });


        V2TIMManager.getGroupManager().getGroupAttributes(chatInfo.getId(), null, new V2TIMValueCallback<Map<String, String>>() {
            @Override
            public void onSuccess(Map<String, String> stringStringMap) {
                if (!viewAlive || generation != viewGeneration || stringStringMap == null) return;
                if (stringStringMap.containsKey("groupGuard")) {
                    groupGuard = stringStringMap.get("groupGuard");
                }
            }

            @Override
            public void onError(int code, String desc) {
                Log.e("===", "获取失败" + code + desc);
            }
        });

        if (groupStateListener != null) V2TIMManager.getInstance().removeGroupListener(groupStateListener);
        groupStateListener = new V2TIMGroupListener() {
            @Override public void onGroupAttributeChanged(String groupID, Map<String, String> attributes) {
                if (viewAlive && generation == viewGeneration && chatInfo != null && android.text.TextUtils.equals(groupID, chatInfo.getId()) && attributes != null && attributes.containsKey("groupGuard")) groupGuard = attributes.get("groupGuard");
            }
            @Override public void onAllGroupMembersMuted(String groupID, boolean muted) {
                uiHandler.post(() -> {
                    if (!viewAlive || generation != viewGeneration || chatInfo == null || !android.text.TextUtils.equals(groupID, chatInfo.getId())) return;
                    allMuted = muted; applyMuteState();
                });
            }
            @Override public void onGrantAdministrator(String groupID, com.tencent.imsdk.v2.V2TIMGroupMemberInfo operator, List<com.tencent.imsdk.v2.V2TIMGroupMemberInfo> members) { updateOwnRole(groupID, members, 300); }
            @Override public void onRevokeAdministrator(String groupID, com.tencent.imsdk.v2.V2TIMGroupMemberInfo operator, List<com.tencent.imsdk.v2.V2TIMGroupMemberInfo> members) { updateOwnRole(groupID, members, 200); }
        };
        V2TIMManager.getInstance().addGroupListener(groupStateListener);

        List<String> groupIDList = new ArrayList<String>();
        groupIDList.add(chatInfo.getId());
        V2TIMManager.getGroupManager().getGroupsInfo(groupIDList, new V2TIMValueCallback<List<V2TIMGroupInfoResult>>() {
            @Override
            public void onSuccess(List<V2TIMGroupInfoResult> v2TIMGroupInfoResults) {
                if (!viewAlive || generation != viewGeneration || v2TIMGroupInfoResults == null || v2TIMGroupInfoResults.isEmpty()) return;
                V2TIMGroupInfoResult result = v2TIMGroupInfoResults.get(0);
                if (result == null || result.getResultCode() != 0 || result.getGroupInfo() == null) return;
                myRole = result.getGroupInfo().getRole();
                allMuted = result.getGroupInfo().isAllMuted();
                uiHandler.post(() -> { if (viewAlive && generation == viewGeneration) applyMuteState(); });
            }

            @Override
            public void onError(int code, String desc) {
                // 获取群资料失败
            }
        });

    }

    private void updateOwnRole(String groupID, List<com.tencent.imsdk.v2.V2TIMGroupMemberInfo> members, int role) {
        final int generation = viewGeneration;
        uiHandler.post(() -> {
            if (!viewAlive || generation != viewGeneration || chatInfo == null || !android.text.TextUtils.equals(groupID, chatInfo.getId()) || members == null) return;
            for (com.tencent.imsdk.v2.V2TIMGroupMemberInfo member : members) {
                if (member != null && android.text.TextUtils.equals(member.getUserID(), V2TIMManager.getInstance().getLoginUser())) { myRole = role; applyMuteState(); break; }
            }
        });
    }

    private void applyMuteState() {
        if (!viewAlive || !isAdded() || chatView == null || chatView.getInputLayout() == null || chatView.getInputLayout().getInputText() == null) return;
        boolean blocked = myRole == 200 && allMuted;
        chatView.getInputLayout().setEnabled(!blocked);
        chatView.getInputLayout().getInputText().setEnabled(!blocked);
        chatView.getInputLayout().getInputText().setHint(blocked ? "全员禁言中…" : "");
    }

    @Override public void onDestroyView() {
        viewAlive = false; ++viewGeneration;
        if (groupStateListener != null) { V2TIMManager.getInstance().removeGroupListener(groupStateListener); groupStateListener = null; }
        uiHandler.removeCallbacksAndMessages(null);
        super.onDestroyView();
    }

    private void setTitleBarExtension() {
        titleBar.setRightIcon(R.drawable.chat_title_bar_more_menu_icon);
        titleBar.setOnRightClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                Map<String, Object> param = new HashMap<>();
                if (TUIChatUtils.isTopicGroup(chatInfo.getId())) {
                    param.put(TUIConstants.TUIChat.Extension.ChatNavigationMoreItem.TOPIC_ID, chatInfo.getId());
                } else {
                    param.put(TUIConstants.TUIChat.Extension.ChatNavigationMoreItem.GROUP_ID, chatInfo.getId());
                }
                List<TUIExtensionInfo> extensionInfoList =
                        TUICore.getExtensionList(TUIConstants.TUIChat.Extension.ChatNavigationMoreItem.CLASSIC_EXTENSION_ID, param);
                if (!extensionInfoList.isEmpty()) {
                    Collections.sort(extensionInfoList, (o1, o2) -> o2.getWeight() - o1.getWeight());
                    TUIExtensionInfo extensionInfo = extensionInfoList.get(0);
                    TUIExtensionEventListener eventListener = extensionInfo.getExtensionListener();
                    if (eventListener != null) {
                        Map<String, Object> map = new HashMap<>();
                        if (TUIChatUtils.isTopicGroup(chatInfo.getId())) {
                            map.put(TUIConstants.TUIChat.Extension.ChatNavigationMoreItem.TOPIC_ID, chatInfo.getId());
                        } else {
                            map.put(TUIConstants.TUIChat.Extension.ChatNavigationMoreItem.GROUP_ID, chatInfo.getId());
                        }
                        map.put(TUIChatConstants.CHAT_BACKGROUND_URI, mChatBackgroundThumbnailUrl);
                        eventListener.onClicked(map);
                    }
                } else {
                    Intent intent = new Intent(getContext(), GroupInfoActivity.class);
                    intent.putExtra(TUIChatConstants.GROUP_ID, chatInfo.getId());
                    intent.putExtra(TUIChatConstants.CHAT_BACKGROUND_URI, mChatBackgroundThumbnailUrl);
                    intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
                    startActivity(intent);
                }
            }
        });
    }

    @Override
    protected void onUserIconClicked(TUIMessageBean message) {
        //消息列表头像
        if (null == message || groupGuard.equals("1")) {
            return;
        }

        Bundle bundle = new Bundle();
        bundle.putString(TUIConstants.TUIChat.CHAT_ID, message.getSender());
        TUICore.startActivity("FriendProfileActivity", bundle);
    }

    @Override
    protected void onUserIconLongClicked(TUIMessageBean messageBean) {
        String resultId = messageBean.getSender();
        String resultName = messageBean.getUserDisplayName();
        chatView.getInputLayout().addInputText(resultName, resultId);
    }

    @Override
    public GroupChatPresenter getPresenter() {
        return presenter;
    }

    public GroupChatInfo getChatInfo() {
        return ((GroupChatInfo) chatInfo);
    }

    public void setChatInfo(GroupChatInfo groupChatInfo) {
        this.chatInfo = groupChatInfo;
    }
}
