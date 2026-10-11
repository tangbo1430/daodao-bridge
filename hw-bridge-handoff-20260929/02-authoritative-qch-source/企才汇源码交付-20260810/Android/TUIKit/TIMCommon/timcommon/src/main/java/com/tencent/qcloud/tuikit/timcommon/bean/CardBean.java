package com.tencent.qcloud.tuikit.timcommon.bean;

public class CardBean {
    public String userID;
    public String name;
    public String avatar;

    public CardBean(String id, String name, String avatar) {
        this.name = name;
        this.avatar = avatar;
        this.userID = id;
    }

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
}
