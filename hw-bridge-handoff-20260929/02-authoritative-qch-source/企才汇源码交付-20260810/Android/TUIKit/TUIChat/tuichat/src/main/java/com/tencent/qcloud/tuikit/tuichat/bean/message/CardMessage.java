package com.tencent.qcloud.tuikit.tuichat.bean.message;

import com.tencent.qcloud.tuikit.tuichat.TUIChatConstants;

import java.io.Serializable;

public class CardMessage implements Serializable {
    public String businessID = TUIChatConstants.BUSINESS_ID_CUSTOM_CARD;
    public String userID;
    public String name;
    public String avatar;
    public int version = TUIChatConstants.JSON_VERSION_UNKNOWN;

    public String getUserID() {
        return userID;
    }

    public void setUserID(String userID) {
        this.userID = userID;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getAvatar() {
        return avatar;
    }

    public void setAvatar(String avatar) {
        this.avatar = avatar;
    }

    public static CardMessage create(String userID, String name, String avatar) {
        CardMessage message = new CardMessage();
        message.userID = userID;
        message.name = name;
        message.avatar = avatar;
        return message;
    }
}
