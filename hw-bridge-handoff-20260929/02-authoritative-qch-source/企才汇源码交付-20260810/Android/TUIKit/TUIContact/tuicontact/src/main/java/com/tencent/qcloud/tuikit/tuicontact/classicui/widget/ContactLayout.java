package com.tencent.qcloud.tuikit.tuicontact.classicui.widget;

import android.content.Context;
import android.content.Intent;
import android.graphics.Color;
import android.text.TextUtils;
import android.util.AttributeSet;
import android.text.Editable;
import android.text.TextWatcher;
import android.view.Gravity;
import android.view.View;
import android.widget.EditText;
import android.widget.ImageView;
import android.widget.Toast;
import android.widget.LinearLayout;
import android.widget.TextView;
import androidx.annotation.Nullable;
import com.tencent.imsdk.v2.V2TIMConversation;
import com.tencent.imsdk.v2.V2TIMConversationResult;
import com.tencent.imsdk.v2.V2TIMManager;
import com.tencent.imsdk.v2.V2TIMValueCallback;
import com.tencent.qcloud.tuikit.timcommon.component.TitleBarLayout;
import com.tencent.qcloud.tuikit.timcommon.component.interfaces.ILayout;
import com.tencent.qcloud.tuikit.timcommon.component.impl.GlideEngine;
import com.tencent.qcloud.tuikit.tuicontact.R;
import com.tencent.qcloud.tuikit.tuicontact.presenter.ContactPresenter;
import com.tencent.qcloud.tuikit.tuicontact.classicui.pages.GroupListActivity;
import com.tencent.qcloud.tuikit.tuicontact.classicui.pages.NewFriendActivity;
import com.tencent.qcloud.tuikit.tuicontact.classicui.util.ClassicUIUtils;

import java.util.List;

public class ContactLayout extends LinearLayout implements ILayout {
    private static final String TAG = ContactLayout.class.getSimpleName();

    private ContactListView mContactListView;
    private View recentSection;
    private LinearLayout recentList;

    private ContactPresenter presenter;

    public ContactLayout(Context context) {
        super(context);
        init();
    }

    public ContactLayout(Context context, @Nullable AttributeSet attrs) {
        super(context, attrs);
        init();
    }

    public ContactLayout(Context context, @Nullable AttributeSet attrs, int defStyleAttr) {
        super(context, attrs, defStyleAttr);
        init();
    }

    public void setPresenter(ContactPresenter presenter) {
        this.presenter = presenter;
    }

    private void init() {
        inflate(getContext(), R.layout.contact_contact_list_layout, this);
        mContactListView = findViewById(R.id.contact_listview);
        recentSection = findViewById(R.id.contact_recent_section);
        recentList = findViewById(R.id.contact_recent_list);
        EditText searchInput = findViewById(R.id.contact_search_input);
        searchInput.addTextChangedListener(new TextWatcher() {
            @Override
            public void beforeTextChanged(CharSequence s, int start, int count, int after) {}

            @Override
            public void onTextChanged(CharSequence s, int start, int before, int count) {
                mContactListView.filterContacts(s == null ? "" : s.toString());
            }

            @Override
            public void afterTextChanged(Editable s) {}
        });
        findViewById(R.id.contact_shortcut_new_friend).setOnClickListener(v ->
                getContext().startActivity(new Intent(getContext(), NewFriendActivity.class)));
        findViewById(R.id.contact_shortcut_group).setOnClickListener(v ->
                getContext().startActivity(new Intent(getContext(), GroupListActivity.class)));
        findViewById(R.id.contact_shortcut_tag).setOnClickListener(v ->
                Toast.makeText(getContext(), "暂无标签", Toast.LENGTH_SHORT).show());
        findViewById(R.id.contact_shortcut_service).setOnClickListener(v ->
                Toast.makeText(getContext(), "暂无客服", Toast.LENGTH_SHORT).show());
        loadRecentConversations();
    }

    public void initDefault() {
        mContactListView.setPresenter(presenter);
        presenter.setContactListView(mContactListView);
        presenter.setIsClassicStyle(true);
        mContactListView.loadDataSource(ContactListView.DataSource.CONTACT_LIST);
    }

    public void reloadData() {
        mContactListView.reloadContactList();
        loadRecentConversations();
    }

    public void loadRecentConversations() {
        V2TIMManager.getConversationManager().getConversationList(0, 12,
                new V2TIMValueCallback<V2TIMConversationResult>() {
                    @Override
                    public void onSuccess(V2TIMConversationResult result) {
                        post(() -> renderRecentConversations(result == null ? null : result.getConversationList()));
                    }

                    @Override
                    public void onError(int code, String desc) {
                        post(() -> recentSection.setVisibility(GONE));
                    }
                });
    }

    private void renderRecentConversations(List<V2TIMConversation> conversations) {
        recentList.removeAllViews();
        if (conversations == null || conversations.isEmpty()) {
            recentSection.setVisibility(GONE);
            return;
        }
        int count = 0;
        for (V2TIMConversation conversation : conversations) {
            if (conversation == null || (conversation.getType() != V2TIMConversation.V2TIM_C2C
                    && conversation.getType() != V2TIMConversation.V2TIM_GROUP)) {
                continue;
            }
            recentList.addView(createRecentConversationView(conversation));
            if (++count == 4) {
                break;
            }
        }
        recentSection.setVisibility(count == 0 ? GONE : VISIBLE);
    }

    private View createRecentConversationView(V2TIMConversation conversation) {
        LinearLayout item = new LinearLayout(getContext());
        item.setOrientation(VERTICAL);
        item.setGravity(Gravity.CENTER_HORIZONTAL);
        item.setPadding(dp(4), dp(4), dp(4), 0);
        LinearLayout.LayoutParams itemParams = new LinearLayout.LayoutParams(dp(76), dp(78));
        item.setLayoutParams(itemParams);

        ImageView avatar = new ImageView(getContext());
        avatar.setScaleType(ImageView.ScaleType.CENTER_CROP);
        item.addView(avatar, new LinearLayout.LayoutParams(dp(40), dp(40)));
        GlideEngine.loadUserIcon(avatar, conversation.getFaceUrl(), dp(6));

        TextView name = new TextView(getContext());
        name.setGravity(Gravity.CENTER);
        name.setSingleLine(true);
        name.setEllipsize(TextUtils.TruncateAt.END);
        name.setText(TextUtils.isEmpty(conversation.getShowName()) ? getConversationId(conversation) : conversation.getShowName());
        name.setTextColor(Color.parseColor("#31333A"));
        name.setTextSize(13);
        LinearLayout.LayoutParams nameParams = new LinearLayout.LayoutParams(dp(68), dp(26));
        nameParams.topMargin = dp(3);
        item.addView(name, nameParams);

        item.setOnClickListener(v -> ClassicUIUtils.startChatActivity(
                getConversationId(conversation), conversation.getType(), conversation.getShowName(), conversation.getGroupType()));
        return item;
    }

    private String getConversationId(V2TIMConversation conversation) {
        return conversation.getType() == V2TIMConversation.V2TIM_GROUP
                ? conversation.getGroupID() : conversation.getUserID();
    }

    private int dp(int value) {
        return Math.round(value * getResources().getDisplayMetrics().density);
    }

    public ContactListView getContactListView() {
        return mContactListView;
    }

    @Override
    public TitleBarLayout getTitleBar() {
        return null;
    }

    @Override
    public void setParentLayout(Object parent) {}
}
