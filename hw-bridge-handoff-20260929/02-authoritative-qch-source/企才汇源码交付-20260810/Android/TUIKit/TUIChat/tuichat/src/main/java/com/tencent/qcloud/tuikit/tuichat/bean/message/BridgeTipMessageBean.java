package com.tencent.qcloud.tuikit.tuichat.bean.message;
import com.tencent.imsdk.v2.V2TIMMessage;
import java.nio.charset.StandardCharsets;
import org.json.JSONObject;
/** Uses the existing centered system-tip layout and conversation summary. */
public class BridgeTipMessageBean extends TipsMessageBean {
    @Override public void onProcessMessage(V2TIMMessage message) {
        String text = "[系统提示]";
        try {
            JSONObject body = new JSONObject(new String(message.getCustomElem().getData(), StandardCharsets.UTF_8));
            text = body.optString("text", body.optString("content", text));
        } catch (Exception ignored) { }
        setText(text);
        setExtra(text);
    }
}
