package com.tencent.qcloud.tuikit.tuichat.classicui.widget.message.viewholder;

import android.content.Intent;
import android.text.TextUtils;
import android.view.View;
import android.widget.ImageView;
import android.widget.TextView;

import com.tencent.qcloud.tuikit.tuichat.R;
import com.bumptech.glide.Glide;
import com.bumptech.glide.load.resource.bitmap.CircleCrop;
import com.bumptech.glide.request.RequestOptions;
import com.tencent.qcloud.tuicore.TUIConstants;
import com.tencent.qcloud.tuicore.TUIThemeManager;
import com.tencent.qcloud.tuikit.timcommon.bean.TUIMessageBean;
import com.tencent.qcloud.tuikit.timcommon.classicui.widget.message.MessageContentHolder;
import com.tencent.qcloud.tuikit.tuichat.TUIChatService;
import com.tencent.qcloud.tuikit.tuichat.bean.message.CardBean;
import com.tencent.qcloud.tuikit.tuichat.classicui.page.FriendProfileActivity;

public class CardMessageHolder extends MessageContentHolder {
    ImageView imHead;
    TextView tvName;
    View view;

    public CardMessageHolder(View itemView) {
        super(itemView);
        view = itemView.findViewById(R.id.rlLayout);
        imHead = itemView.findViewById(R.id.imHead);
        tvName = itemView.findViewById(R.id.tvName);
    }

    @Override
    public void layoutVariableViews(TUIMessageBean msg, int position) {
        msgArea.setBackground(TUIChatService.getAppContext().getResources().getDrawable(com.tencent.qcloud.tuikit.timcommon.R.color.transparent));
        CardBean bean = (CardBean) msg;
        int defaultAvatar = TUIThemeManager.getAttrResId(imHead.getContext(), com.tencent.qcloud.tuikit.timcommon.R.attr.core_default_user_icon);
        Glide.with(imHead.getContext())
                .load(TextUtils.isEmpty(bean.getAvatar()) ? defaultAvatar : bean.getAvatar())
                .placeholder(defaultAvatar)
                .error(defaultAvatar)
                .apply(RequestOptions.bitmapTransform(new CircleCrop()))
                .into(imHead);
        tvName.setText(bean.getName());
        view.setOnClickListener(v -> {
            Intent intent = new Intent(view.getContext(), FriendProfileActivity.class);
            intent.putExtra(TUIConstants.TUIChat.CHAT_ID, bean.getUserID());
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            view.getContext().startActivity(intent);
        });

    }

    @Override
    public int getVariableLayout() {
        return R.layout.layout_message_card;
    }
}
