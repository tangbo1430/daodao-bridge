package com.tencent.qcloud.tuikit.tuichat.bean.message;

import android.text.TextUtils;

import com.google.gson.Gson;
import com.tencent.imsdk.v2.V2TIMMessage;
import com.tencent.qcloud.tuikit.timcommon.bean.TUIMessageBean;
import com.tencent.qcloud.tuikit.tuichat.R;
import com.tencent.qcloud.tuikit.tuichat.TUIChatService;
import com.tencent.qcloud.tuikit.tuichat.util.TUIChatLog;

public class CardBean extends TUIMessageBean {
    CardMessage cardMessage;

    @Override
    public void onProcessMessage(V2TIMMessage v2TIMMessage) {

        String data = new String(v2TIMMessage.getCustomElem().getData());
        if (!TextUtils.isEmpty(data)) {
            try {

                cardMessage = new Gson().fromJson(data, CardMessage.class);
            } catch (Exception e) {
                TUIChatLog.e("CustomEvaluationMessage", "exception e = " + e);
            }
        }
        if (cardMessage != null) {
            setExtra("[名片消息]");
        } else {
            String text = TUIChatService.getAppContext().getString(R.string.no_support_msg);
            setExtra(text);
        }
    }


    public String getUserID() {
        return cardMessage.getUserID();
    }

    public String getName() {
        return cardMessage.getName();
    }

    public String getAvatar() {
        return cardMessage.getAvatar();
    }
}
