package com.tencent.qcloud.tuikit.tuichat.classicui.page;

import android.content.Intent;
import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.PopupMenu;
import androidx.annotation.Nullable;
import com.tencent.qcloud.tuicore.TUIConstants;
import com.tencent.qcloud.tuicore.TUICore;
import com.tencent.qcloud.tuicore.interfaces.TUIExtensionEventListener;
import com.tencent.qcloud.tuicore.interfaces.TUIExtensionInfo;
import com.tencent.qcloud.tuicore.interfaces.TUIValueCallback;
import com.tencent.qcloud.tuikit.tuichat.R;
import com.tencent.qcloud.tuikit.tuichat.TUIChatConstants;
import com.tencent.qcloud.tuikit.tuichat.bean.C2CChatInfo;
import com.tencent.qcloud.tuikit.tuichat.bean.ChatInfo;
import com.tencent.qcloud.tuikit.tuichat.presenter.C2CChatPresenter;
import com.tencent.qcloud.tuikit.tuichat.presenter.FriendProfilePresenter;
import com.tencent.qcloud.tuikit.tuichat.util.TUIChatLog;
import java.util.Collections;
import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class TUIC2CChatFragment extends TUIBaseChatFragment {
    private static final String TAG = TUIC2CChatFragment.class.getSimpleName();

    private final C2CChatPresenter presenter;
    private final FriendProfilePresenter friendProfilePresenter = new FriendProfilePresenter();

    public TUIC2CChatFragment() {
        presenter = new C2CChatPresenter();
        presenter.initListener();
    }

    @Nullable
    @Override
    public View onCreateView(LayoutInflater inflater, @Nullable ViewGroup container, Bundle savedInstanceState) {
        TUIChatLog.i(TAG, "oncreate view " + this);

        baseView = super.onCreateView(inflater, container, savedInstanceState);
        if (!(chatInfo instanceof C2CChatInfo)) {
            return baseView;
        }

        initView();

        return baseView;
    }

    @Override
    protected void initView() {
        super.initView();

        setTitleBarExtension();

        chatView.setPresenter(presenter);
        presenter.setChatInfo((C2CChatInfo) chatInfo);
        presenter.setTypingListener(chatView.mTypingListener);
        chatView.setChatInfo(chatInfo);
    }

    private void setTitleBarExtension() {
        titleBar.setRightIcon(chatInfo.isBridgeConversation()
                ? R.drawable.chat_title_bar_more_menu_icon : R.drawable.qch_chat_call_icon);
        titleBar.getMiddleTitle().setOnClickListener(v -> openFriendProfile(chatInfo.getId()));
        titleBar.setOnRightClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                if (chatInfo.isBridgeConversation()) {
                    openFriendProfile(chatInfo.getId());
                    return;
                }
                Map<String, Object> param = new HashMap<>();
                param.put(TUIConstants.TUIChat.Extension.InputMore.USER_ID, chatInfo.getId());
                param.put(TUIConstants.TUIChat.Extension.InputMore.CONTEXT, TUIC2CChatFragment.this);
                param.put(TUIConstants.TUIChat.Extension.InputMore.FILTER_VOICE_CALL, !chatInfo.isEnableAudioCall());
                param.put(TUIConstants.TUIChat.Extension.InputMore.FILTER_VIDEO_CALL, !chatInfo.isEnableVideoCall());
                List<TUIExtensionInfo> extensionInfoList =
                    TUICore.getExtensionList(TUIConstants.TUIChat.Extension.InputMore.CLASSIC_EXTENSION_ID, param);
                if (!extensionInfoList.isEmpty()) {
                    Collections.sort(extensionInfoList, new Comparator<TUIExtensionInfo>() {
                        @Override
                        public int compare(TUIExtensionInfo o1, TUIExtensionInfo o2) {
                            return o2.getWeight() - o1.getWeight();
                        }
                    });
                    PopupMenu callMenu = new PopupMenu(requireContext(), titleBar.getRightGroup());
                    for (int i = 0; i < extensionInfoList.size(); i++) {
                        TUIExtensionInfo extensionInfo = extensionInfoList.get(i);
                        callMenu.getMenu().add(0, i, i, extensionInfo.getText());
                    }
                    callMenu.getMenu().add(0, extensionInfoList.size(), extensionInfoList.size(), R.string.profile_detail);
                    callMenu.setOnMenuItemClickListener(item -> {
                        int index = item.getItemId();
                        if (index == extensionInfoList.size()) {
                            openFriendProfile(chatInfo.getId());
                            return true;
                        }
                        TUIExtensionEventListener eventListener = extensionInfoList.get(index).getExtensionListener();
                        if (eventListener != null) {
                            eventListener.onClicked(null);
                        }
                        return true;
                    });
                    callMenu.show();
                } else {
                    openFriendProfile(chatInfo.getId());
                }
            }
        });
    }

    private void openFriendProfile(String userId) {
        Intent intent = new Intent(getContext(), FriendProfileActivity.class);
        intent.putExtra(TUIConstants.TUIChat.CHAT_ID, userId);
        intent.putExtra(TUIChatConstants.CHAT_BACKGROUND_URI, mChatBackgroundThumbnailUrl);
        intent.putExtra("qch_chat_info_mode", true);
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        startActivity(intent);
    }

    @Override
    public C2CChatPresenter getPresenter() {
        return presenter;
    }

    @Override
    public C2CChatInfo getChatInfo() {
        return (C2CChatInfo) chatInfo;
    }

    public void setChatInfo(C2CChatInfo c2CChatInfo) {
        this.chatInfo = c2CChatInfo;
    }
}
