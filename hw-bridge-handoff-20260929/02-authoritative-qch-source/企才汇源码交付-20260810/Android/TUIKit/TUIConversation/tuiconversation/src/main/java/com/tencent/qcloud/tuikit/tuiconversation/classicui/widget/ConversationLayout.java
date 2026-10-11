package com.tencent.qcloud.tuikit.tuiconversation.classicui.widget;

import android.content.Context;
import android.graphics.drawable.Drawable;
import android.text.TextUtils;
import android.util.AttributeSet;
import android.util.TypedValue;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ImageButton;
import android.widget.LinearLayout;
import android.widget.RelativeLayout;
import android.widget.TextView;
import android.graphics.Color;
import com.tencent.qcloud.tuicore.TUICore;
import com.tencent.qcloud.tuikit.timcommon.component.TitleBarLayout;
import com.tencent.qcloud.tuikit.timcommon.component.interfaces.IUIKitCallback;
import com.tencent.qcloud.tuikit.tuiconversation.R;
import com.tencent.qcloud.tuikit.tuiconversation.bean.ConversationInfo;
import com.tencent.qcloud.tuikit.tuiconversation.classicui.interfaces.IConversationLayout;
import com.tencent.qcloud.tuikit.tuiconversation.config.classicui.TUIConversationConfigClassic;
import com.tencent.qcloud.tuikit.tuiconversation.interfaces.IConversationListAdapter;
import com.tencent.qcloud.tuikit.tuiconversation.presenter.ConversationPresenter;
import java.util.ArrayList;
import java.util.List;

public class ConversationLayout extends RelativeLayout implements IConversationLayout {
    private ConversationListLayout mConversationList;
    private ConversationPresenter presenter;
    private ConversationListAdapter conversationAdapter;
    private TextView filterAll;
    private TextView filterUnread;
    private TextView filterGroup;
    private TextView unreadSummary;
    private TextView quickTitle;
    private LinearLayout quickSection;
    private LinearLayout quickList;

    public ConversationLayout(Context context) {
        super(context);
        init();
    }

    public ConversationLayout(Context context, AttributeSet attrs) {
        super(context, attrs);
        init();
    }

    public ConversationLayout(Context context, AttributeSet attrs, int defStyleAttr) {
        super(context, attrs, defStyleAttr);
        init();
    }

    public void setPresenter(ConversationPresenter presenter) {
        this.presenter = presenter;
        if (mConversationList != null) {
            mConversationList.setPresenter(presenter);
        }
    }

    private void init() {
        inflate(getContext(), R.layout.conversation_layout, this);
        mConversationList = findViewById(R.id.conversation_list);
        filterAll = findViewById(R.id.conversation_filter_all);
        filterUnread = findViewById(R.id.conversation_filter_unread);
        filterGroup = findViewById(R.id.conversation_filter_group);
        unreadSummary = findViewById(R.id.conversation_unread_summary);
        quickTitle = findViewById(R.id.conversation_quick_title);
        quickSection = findViewById(R.id.conversation_quick_section);
        quickList = findViewById(R.id.conversation_quick_list);
        filterAll.setOnClickListener(v -> selectFilter(0));
        filterUnread.setOnClickListener(v -> selectFilter(1));
        filterGroup.setOnClickListener(v -> selectFilter(2));
        findViewById(R.id.conversation_search).setOnClickListener(v -> TUICore.startActivity("SearchMainActivity", null));
        findViewById(R.id.conversation_fab).setOnClickListener(v -> performMainMenuClick());
        applyCustomConfig();
    }

    private void applyCustomConfig() {
        Drawable listBackground = TUIConversationConfigClassic.getListBackground();
        if (listBackground != null) {
            mConversationList.setBackground(listBackground);
        }
    }

    public void initDefault() {
        final ConversationListAdapter adapter = new ConversationListAdapter();
        conversationAdapter = adapter;
        adapter.setQchDataObserver(this::renderConversationSummary);
        if (presenter != null) {
            adapter.setShowFoldedStyle(true);
        }
        mConversationList.setAdapter((IConversationListAdapter) adapter);
        if (presenter != null) {
            presenter.setAdapter(adapter);
        }
        mConversationList.loadConversation();
        mConversationList.loadMarkedConversation();
    }

    public void reloadConversation() {
        if (presenter != null) {
            presenter.reLoadConversation();
        }
    }

    private void renderConversationSummary(List<ConversationInfo> conversations) {
        long unread = 0;
        List<ConversationInfo> pinned = new ArrayList<>();
        List<ConversationInfo> fallback = new ArrayList<>();
        for (ConversationInfo info : conversations) {
            if (info == null || info.isMarkFold()) {
                continue;
            }
            unread += Math.max(0, info.getUnRead());
            if (info.isTop()) {
                pinned.add(info);
            }
            if (fallback.size() < 4) {
                fallback.add(info);
            }
        }
        unreadSummary.setVisibility(unread > 0 ? VISIBLE : GONE);
        unreadSummary.setText(unread > 99 ? "99+条未读" : unread + "条未读");

        List<ConversationInfo> shortcuts = pinned.isEmpty() ? fallback : pinned;
        quickTitle.setText("置顶联系人");
        quickList.removeAllViews();
        int count = Math.min(4, shortcuts.size());
        for (int i = 0; i < count; i++) {
            quickList.addView(createQuickConversationView(shortcuts.get(i)));
        }
        quickSection.setVisibility(count == 0 ? GONE : VISIBLE);
    }

    private View createQuickConversationView(ConversationInfo info) {
        LinearLayout item = new LinearLayout(getContext());
        item.setOrientation(LinearLayout.VERTICAL);
        item.setGravity(Gravity.CENTER_HORIZONTAL);
        item.setPadding(dp(4), 0, dp(4), 0);
        item.setLayoutParams(new LinearLayout.LayoutParams(dp(72), ViewGroup.LayoutParams.MATCH_PARENT));

        ConversationIconView icon = new ConversationIconView(getContext());
        icon.setConversation(info);
        icon.setRadius(dp(6));
        item.addView(icon, new LinearLayout.LayoutParams(dp(38), dp(38)));

        TextView name = new TextView(getContext());
        name.setText(TextUtils.isEmpty(info.getTitle()) ? info.getId() : info.getTitle());
        name.setTextColor(Color.parseColor("#303139"));
        name.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        name.setGravity(Gravity.CENTER);
        name.setSingleLine(true);
        name.setEllipsize(TextUtils.TruncateAt.END);
        LinearLayout.LayoutParams nameParams = new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(24));
        nameParams.topMargin = dp(4);
        item.addView(name, nameParams);
        item.setOnClickListener(v -> conversationAdapter.performConversationClick(v, info));
        return item;
    }

    private void performMainMenuClick() {
        int id = getResources().getIdentifier("page_title_right_group", "id", getContext().getPackageName());
        View menuView = getRootView().findViewById(id);
        if (menuView != null) {
            menuView.performClick();
        }
    }

    private int dp(int value) {
        return (int) (value * getResources().getDisplayMetrics().density + 0.5f);
    }

    private void selectFilter(int mode) {
        if (conversationAdapter == null) {
            return;
        }
        conversationAdapter.setFilterMode(mode);
        TextView[] filters = {filterAll, filterUnread, filterGroup};
        for (int i = 0; i < filters.length; i++) {
            filters[i].setTextColor(Color.parseColor(i == mode ? "#7C35FF" : "#666A73"));
            filters[i].setTypeface(null, i == mode ? android.graphics.Typeface.BOLD : android.graphics.Typeface.NORMAL);
            filters[i].setBackgroundResource(i == mode
                    ? R.drawable.conversation_qch_filter_selected_bg
                    : android.R.color.transparent);
        }
    }

    @Override
    public void setParentLayout(Object parent) {}

    @Override
    public ConversationListLayout getConversationList() {
        return mConversationList;
    }

    @Override
    public void setConversationTop(ConversationInfo conversation, IUIKitCallback callBack) {
        if (presenter != null) {
            presenter.setConversationTop(conversation, callBack);
        }
    }

    @Override
    public void deleteConversation(ConversationInfo conversation) {
        if (presenter != null) {
            presenter.deleteConversation(conversation);
        }
    }

    @Override
    public void clearConversationMessage(ConversationInfo conversation) {
        if (presenter != null) {
            presenter.clearConversationMessage(conversation);
        }
    }

    @Override
    public void markConversationHidden(ConversationInfo conversation) {
        if (presenter != null) {
            presenter.markConversationHidden(conversation, true);
        }
    }

    @Override
    public void markConversationUnread(ConversationInfo conversationInfo, boolean markUnread) {
        if (presenter != null) {
            presenter.markConversationUnreadAndCleanUnreadCount(conversationInfo, markUnread);
        }
    }

    @Override
    public void hideFoldedItem(boolean needHide) {
        if (presenter != null) {
            presenter.hideFoldItem(needHide);
        }
    }

    @Override
    public void clearUnreadStatusOfFoldItem() {
        if (presenter != null) {
            presenter.setUnreadStatusOfFoldItem(false);
        }
    }

    @Override
    public TitleBarLayout getTitleBar() {
        return null;
    }
}
