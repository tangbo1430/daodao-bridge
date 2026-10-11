package com.tencent.qcloud.tuikit.tuichat.bean.message;

import com.tencent.imsdk.v2.V2TIMMessage;
import java.nio.charset.StandardCharsets;
import org.json.JSONObject;

/** Bridge notices contain display data only; application credentials stay on the server. */
public class BridgeNoticeMessageBean extends TextMessageBean {
    private boolean friendApplication;
    public boolean isFriendApplication() { return friendApplication; }
    @Override public void onProcessMessage(V2TIMMessage message) {
        String content = "[系统提示]";
        try {
            JSONObject data = new JSONObject(new String(message.getCustomElem().getData(), StandardCharsets.UTF_8));
            friendApplication = "bridge_friend_apply".equals(data.optString("businessID"));
            content = data.optString("content", content);
        } catch (Exception ignored) { friendApplication = false; }
        setText(content);
        setExtra(content);
    }
}
