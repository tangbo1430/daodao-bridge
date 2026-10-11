package com.tencent.qcloud.tuikit.tuichat.minimalistui.widget.message.viewholder;
import android.view.View;
import com.tencent.qcloud.tuicore.TUICore;
import com.tencent.qcloud.tuikit.timcommon.bean.TUIMessageBean;
import com.tencent.qcloud.tuikit.tuichat.bean.message.BridgeNoticeMessageBean;
public class BridgeNoticeMessageHolder extends TextMessageHolder {
    public BridgeNoticeMessageHolder(View itemView) { super(itemView); }
    @Override public void layoutVariableViews(TUIMessageBean message, int position) {
        super.layoutVariableViews(message, position);
        if (message instanceof BridgeNoticeMessageBean && ((BridgeNoticeMessageBean)message).isFriendApplication()) {
            timeInLineTextLayout.getTextView().setOnClickListener(v -> TUICore.startActivity("NewFriendApplicationMinimalistActivity", null));
        }
    }
}
