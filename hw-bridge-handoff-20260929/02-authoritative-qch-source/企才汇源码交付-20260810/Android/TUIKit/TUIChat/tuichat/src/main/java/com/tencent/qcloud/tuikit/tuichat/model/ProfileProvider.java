package com.tencent.qcloud.tuikit.tuichat.model;

import android.text.TextUtils;
import android.os.Bundle;
import com.tencent.imsdk.v2.V2TIMCallback;
import com.tencent.imsdk.v2.V2TIMConversation;
import com.tencent.imsdk.v2.V2TIMConversationOperationResult;
import com.tencent.imsdk.v2.V2TIMFriendCheckResult;
import com.tencent.imsdk.v2.V2TIMFriendInfo;
import com.tencent.imsdk.v2.V2TIMFriendInfoResult;
import com.tencent.imsdk.v2.V2TIMFriendOperationResult;
import com.tencent.imsdk.v2.V2TIMGroupInfo;
import com.tencent.imsdk.v2.V2TIMGroupInfoResult;
import com.tencent.imsdk.v2.V2TIMGroupMemberFullInfo;
import com.tencent.imsdk.v2.V2TIMManager;
import com.tencent.imsdk.v2.V2TIMReceiveMessageOptInfo;
import com.tencent.imsdk.v2.V2TIMUserFullInfo;
import com.tencent.imsdk.v2.V2TIMValueCallback;
import com.tencent.qcloud.tuicore.interfaces.TUICallback;
import com.tencent.qcloud.tuicore.TUIConstants;
import com.tencent.qcloud.tuicore.TUICore;
import com.tencent.qcloud.tuicore.interfaces.TUIServiceCallback;
import com.tencent.qcloud.tuicore.interfaces.TUIValueCallback;
import com.tencent.qcloud.tuikit.timcommon.bean.FriendProfileBean;
import com.tencent.qcloud.tuikit.timcommon.bean.GroupProfileBean;
import com.tencent.qcloud.tuikit.timcommon.util.TIMCommonUtil;
import com.tencent.qcloud.tuikit.tuichat.bean.GroupMemberBean;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ProfileProvider {

    public void getGroupsProfile(String groupID, TUIValueCallback<GroupProfileBean> callback) {
        List<String> groupIDs = new ArrayList<>();
        groupIDs.add(groupID);
        V2TIMManager.getGroupManager().getGroupsInfo(groupIDs, new V2TIMValueCallback<List<V2TIMGroupInfoResult>>() {
            @Override
            public void onSuccess(List<V2TIMGroupInfoResult> v2TIMGroupInfoResults) {
                if (v2TIMGroupInfoResults != null && !v2TIMGroupInfoResults.isEmpty()) {
                    V2TIMGroupInfoResult result = v2TIMGroupInfoResults.get(0);
                    if (result.getResultCode() == 0) {
                        V2TIMGroupInfo groupInfo = result.getGroupInfo();
                        GroupProfileBean groupProfileBean = new GroupProfileBean();
                        groupProfileBean.setGroupID(groupInfo.getGroupID());
                        groupProfileBean.setGroupName(groupInfo.getGroupName());
                        groupProfileBean.setGroupFaceUrl(groupInfo.getFaceUrl());
                        groupProfileBean.setRoleInGroup(groupInfo.getRole());
                        groupProfileBean.setGroupType(groupInfo.getGroupType());
                        groupProfileBean.setNotification(groupInfo.getNotification());
                        groupProfileBean.setMemberCount(groupInfo.getMemberCount());
                        groupProfileBean.setAllMuted(groupInfo.isAllMuted());
                        groupProfileBean.setApproveOpt(groupInfo.getGroupApproveOpt());
                        groupProfileBean.setAddOpt(groupInfo.getGroupAddOpt());
                        groupProfileBean.setRecvOpt(groupInfo.getRecvOpt());
                        if (groupID != null && groupID.startsWith("dd_")) {
                            Map<String, Object> params = new HashMap<>(); params.put("group_id", groupID);
                            TUICore.callService(TUIConstants.TUIContact.SERVICE_NAME, "bridgeGroupState", params, new TUIServiceCallback() {
                                @Override public void onServiceCallback(int code, String message, Bundle data) {
                                    if (code != 0 || data == null || !data.containsKey("invite")) {
                                        TUIValueCallback.onError(callback, code == 0 ? 502 : code, message); return;
                                    }
                                    int option = data.getInt("invite") == 1 ? V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH : V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY;
                                    groupProfileBean.setAddOpt(option); groupProfileBean.setApproveOpt(option);
                                    groupProfileBean.setAllMuted(data.getBoolean("all_muted"));
                                    if (!TextUtils.isEmpty(data.getString("name"))) groupProfileBean.setGroupName(data.getString("name"));
                                    TUIValueCallback.onSuccess(callback, groupProfileBean);
                                }
                            });
                        } else TUIValueCallback.onSuccess(callback, groupProfileBean);
                    } else {
                        TUIValueCallback.onError(callback, result.getResultCode(), result.getResultMessage());
                    }
                } else {
                    TUIValueCallback.onError(callback, -1, "getGroupInfo failed, result is null");
                }
            }

            @Override
            public void onError(int code, String desc) {
                TUIValueCallback.onError(callback, code, desc);
            }
        });
    }

    public void getGroupMembersProfile(String groupID, List<String> userIDs, TUIValueCallback<List<GroupMemberBean>> callback) {
        V2TIMManager.getGroupManager().getGroupMembersInfo(groupID, userIDs, new V2TIMValueCallback<List<V2TIMGroupMemberFullInfo>>() {
            @Override
            public void onSuccess(List<V2TIMGroupMemberFullInfo> v2TIMGroupMemberFullInfos) {
                List<GroupMemberBean> groupMemberBeans = new ArrayList<>();
                for (V2TIMGroupMemberFullInfo v2TIMGroupMemberFullInfo : v2TIMGroupMemberFullInfos) {
                    GroupMemberBean groupMemberBean = new GroupMemberBean();
                    groupMemberBean.setUserId(v2TIMGroupMemberFullInfo.getUserID());
                    groupMemberBean.setFriendRemark(v2TIMGroupMemberFullInfo.getFriendRemark());
                    groupMemberBean.setFaceUrl(v2TIMGroupMemberFullInfo.getFaceUrl());
                    groupMemberBean.setNickName(v2TIMGroupMemberFullInfo.getNickName());
                    groupMemberBean.setNameCard(v2TIMGroupMemberFullInfo.getNameCard());
                    groupMemberBean.setRole(v2TIMGroupMemberFullInfo.getRole());
                    groupMemberBeans.add(groupMemberBean);
                }
                TUIValueCallback.onSuccess(callback, groupMemberBeans);
            }

            @Override
            public void onError(int code, String desc) {
                TUIValueCallback.onError(callback, code, desc);
            }
        });
    }

    public void getFriendsProfile(List<String> userIDList, TUIValueCallback<List<FriendProfileBean>> callback) {
        V2TIMManager.getFriendshipManager().getFriendsInfo(userIDList, new V2TIMValueCallback<List<V2TIMFriendInfoResult>>() {
            @Override
            public void onSuccess(List<V2TIMFriendInfoResult> v2TIMFriendInfoResults) {
                List<FriendProfileBean> friendProfileBeans = new ArrayList<>();
                for (V2TIMFriendInfoResult result : v2TIMFriendInfoResults) {
                    if (result.getResultCode() == 0) {
                        V2TIMFriendInfo friendInfo = result.getFriendInfo();
                        FriendProfileBean friendProfileBean = new FriendProfileBean();
                        friendProfileBean.setUserId(friendInfo.getUserID());
                        friendProfileBean.setFriendRemark(friendInfo.getFriendRemark());
                        V2TIMUserFullInfo userFullInfo = friendInfo.getUserProfile();
                        friendProfileBean.setFaceUrl(userFullInfo.getFaceUrl());
                        friendProfileBean.setNickName(userFullInfo.getNickName());
                        friendProfileBean.setSignature(userFullInfo.getSelfSignature());
                        friendProfileBean.setAllowType(userFullInfo.getAllowType());
                        friendProfileBean.setBirthday(userFullInfo.getBirthday());
                        friendProfileBeans.add(friendProfileBean);
                    }
                }
                TUIValueCallback.onSuccess(callback, friendProfileBeans);
            }

            @Override
            public void onError(int code, String desc) {
                TUIValueCallback.onError(callback, code, desc);
            }
        });
    }

    public void checkFriend(List<String> userIDList, TUIValueCallback<Map<String, Integer>> callback) {
        V2TIMManager.getFriendshipManager().checkFriend(
            userIDList, V2TIMFriendInfo.V2TIM_FRIEND_TYPE_BOTH, new V2TIMValueCallback<List<V2TIMFriendCheckResult>>() {
                @Override
                public void onSuccess(List<V2TIMFriendCheckResult> v2TIMFriendCheckResults) {
                    Map<String, Integer> resultMap = new HashMap<>();
                    for (V2TIMFriendCheckResult result : v2TIMFriendCheckResults) {
                        if (result.getResultCode() == 0) {
                            resultMap.put(result.getUserID(), result.getResultType());
                        }
                    }
                    TUIValueCallback.onSuccess(callback, resultMap);
                }

                @Override
                public void onError(int code, String desc) {
                    TUIValueCallback.onError(callback, code, desc);
                }
            });
    }

    public void checkBlackList(String userID, TUIValueCallback<Boolean> callback) {
        Map<String, Object> param = new HashMap<>();
        param.put(TUIConstants.AppBlocklist.PARAM_PEER_ACCOUNT, userID);
        TUICore.callService(TUIConstants.Service.TUI_DEMO, TUIConstants.AppBlocklist.METHOD_STATUS, param, new TUIServiceCallback() {
            @Override
            public void onServiceCallback(int errorCode, String errorMessage, Bundle bundle) {
                if (errorCode != 0) {
                    TUIValueCallback.onError(callback, errorCode, errorMessage);
                    return;
                }
                TUIValueCallback.onSuccess(callback, bundle.getBoolean(TUIConstants.AppBlocklist.RESULT_BLOCKED, false));
            }
        });
    }

    private boolean bridgeGroupAction(String groupID, String action, String field, String value, TUICallback callback) {
        if (groupID == null || !groupID.startsWith("dd_")) return false;
        Map<String, Object> params = new HashMap<>();
        params.put("group_id", groupID); params.put("action", action); params.put("field", field); params.put("value", value);
        TUICore.callService(TUIConstants.TUIContact.SERVICE_NAME, "bridgeGroupAction", params, new TUIServiceCallback() {
            @Override public void onServiceCallback(int code, String message, Bundle bundle) {
                if (code == 0) TUICallback.onSuccess(callback); else TUICallback.onError(callback, code, message);
            }
        });
        return true;
    }

    public void modifyGroupName(String groupID, String name, TUICallback callback) {
        if (bridgeGroupAction(groupID, "update", "name", name, callback)) return;
        V2TIMGroupInfo groupInfo = new V2TIMGroupInfo();
        groupInfo.setGroupID(groupID);
        groupInfo.setGroupName(name);
        V2TIMManager.getGroupManager().setGroupInfo(groupInfo, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void modifyGroupFaceUrl(String groupID, String faceUrl, TUICallback callback) {
        if (bridgeGroupAction(groupID, "update", "avatar", faceUrl, callback)) return;
        V2TIMGroupInfo groupInfo = new V2TIMGroupInfo();
        groupInfo.setGroupID(groupID);
        groupInfo.setFaceUrl(faceUrl);
        V2TIMManager.getGroupManager().setGroupInfo(groupInfo, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void modifyGroupNotification(String groupID, String notification, TUICallback callback) {
        if (bridgeGroupAction(groupID, "update", "notice", notification, callback)) return;
        V2TIMGroupInfo groupInfo = new V2TIMGroupInfo();
        groupInfo.setGroupID(groupID);
        groupInfo.setNotification(notification);
        V2TIMManager.getGroupManager().setGroupInfo(groupInfo, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void modifyGroupApproveOpt(String groupID, int approveOpt, TUICallback callback) {
        if (groupID != null && groupID.startsWith("dd_")) {
            modifyGroupAddOpt(groupID, approveOpt, callback); return;
        }
        V2TIMGroupInfo groupInfo = new V2TIMGroupInfo();
        groupInfo.setGroupID(groupID);
        groupInfo.setGroupApproveOpt(approveOpt);
        V2TIMManager.getGroupManager().setGroupInfo(groupInfo, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void modifyGroupAddOpt(String groupID, int addOpt, TUICallback callback) {
        if (groupID != null && groupID.startsWith("dd_") && addOpt != V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH && addOpt != V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY) {
            TUICallback.onError(callback, 409, "该群支持邀请确认，不支持禁止加群"); return;
        }
        if (bridgeGroupAction(groupID, "update", "invite", addOpt == V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH ? "1" : "0", callback)) return;
        V2TIMGroupInfo groupInfo = new V2TIMGroupInfo();
        groupInfo.setGroupID(groupID);
        groupInfo.setGroupAddOpt(addOpt);
        V2TIMManager.getGroupManager().setGroupInfo(groupInfo, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void modifySelfNameCard(String groupID, String nameCard, TUICallback callback) {
        V2TIMGroupMemberFullInfo memberInfo = new V2TIMGroupMemberFullInfo();
        memberInfo.setUserID(V2TIMManager.getInstance().getLoginUser());
        memberInfo.setNameCard(nameCard);
        V2TIMManager.getGroupManager().setGroupMemberInfo(groupID, memberInfo, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void pinConversation(String conversationId, boolean isPinned, TUICallback callBack) {
        V2TIMManager.getConversationManager().pinConversation(conversationId, isPinned, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callBack);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callBack, code, desc);
            }
        });
    }

    public void setConversationFold(String conversationID, boolean isFolded, TUICallback callback) {
        List<String> conversationList = Collections.singletonList(conversationID);
        long markType = V2TIMConversation.V2TIM_CONVERSATION_MARK_TYPE_FOLD;
        V2TIMManager.getConversationManager().markConversation(
            conversationList, markType, isFolded, new V2TIMValueCallback<List<V2TIMConversationOperationResult>>() {
                @Override
                public void onSuccess(List<V2TIMConversationOperationResult> v2TIMConversationOperationResults) {
                    if (!v2TIMConversationOperationResults.isEmpty()) {
                        V2TIMConversationOperationResult result = v2TIMConversationOperationResults.get(0);
                        if (result.getResultCode() == 0) {
                            TUICallback.onSuccess(callback);
                        } else {
                            TUICallback.onError(callback, result.getResultCode(), result.getResultInfo());
                        }
                    }
                }

                @Override
                public void onError(int code, String desc) {
                    TUICallback.onError(callback, code, desc);
                }
            });
    }

    public void checkoutConversationPinned(String chatID, boolean isGroup, TUIValueCallback<Boolean> callback) {
        String conversationID = TIMCommonUtil.getConversationIdByID(chatID, isGroup);
        V2TIMManager.getConversationManager().getConversation(conversationID, new V2TIMValueCallback<V2TIMConversation>() {
            @Override
            public void onSuccess(V2TIMConversation v2TIMConversation) {
                TUIValueCallback.onSuccess(callback, v2TIMConversation.isPinned());
            }

            @Override
            public void onError(int code, String desc) {
                TUIValueCallback.onError(callback, code, desc);
            }
        });
    }

    public void getConversation(String conversationID, TUIValueCallback<V2TIMConversation> callback) {
        V2TIMManager.getConversationManager().getConversation(conversationID, new V2TIMValueCallback<V2TIMConversation>() {
            @Override
            public void onSuccess(V2TIMConversation v2TIMConversation) {
                TUIValueCallback.onSuccess(callback, v2TIMConversation);
            }

            @Override
            public void onError(int code, String desc) {
                TUIValueCallback.onError(callback, code, desc);
            }
        });
    }

    public void setGroupMessageRecvOpt(String groupID, int opt, TUICallback callback) {
        V2TIMManager.getMessageManager().setGroupReceiveMessageOpt(groupID, opt, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void getC2CReceiveMessageOpt(List<String> userIdList, TUIValueCallback<Map<String, Integer>> callback) {
        V2TIMManager.getMessageManager().getC2CReceiveMessageOpt(userIdList, new V2TIMValueCallback<List<V2TIMReceiveMessageOptInfo>>() {
            @Override
            public void onSuccess(List<V2TIMReceiveMessageOptInfo> v2TIMReceiveMessageOptInfos) {
                if (v2TIMReceiveMessageOptInfos == null || v2TIMReceiveMessageOptInfos.isEmpty()) {
                    return;
                }
                Map<String, Integer> resultMap = new HashMap<>();
                for (V2TIMReceiveMessageOptInfo v2TIMReceiveMessageOptInfo : v2TIMReceiveMessageOptInfos) {
                    resultMap.put(v2TIMReceiveMessageOptInfo.getUserID(), v2TIMReceiveMessageOptInfo.getC2CReceiveMessageOpt());
                }
                TUIValueCallback.onSuccess(callback, resultMap);
            }

            @Override
            public void onError(int code, String desc) {}
        });
    }

    public void setC2CReceiveMessageOpt(List<String> userIdList, int opt, TUICallback callback) {
        V2TIMManager.getMessageManager().setC2CReceiveMessageOpt(userIdList, opt, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void modifyFriendRemark(String userID, String remark, TUICallback callback) {
        V2TIMFriendInfo friendInfo = new V2TIMFriendInfo();
        friendInfo.setUserID(userID);
        friendInfo.setFriendRemark(remark);
        V2TIMManager.getFriendshipManager().setFriendInfo(friendInfo, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void removeFriends(List<String> userIDList, TUIValueCallback<Map<String, Integer>> callback) {
        if (userIDList == null || userIDList.isEmpty()) {
            TUIValueCallback.onSuccess(callback, new HashMap<>());
            return;
        }
        if (containsLoginUser(userIDList)) {
            TUIValueCallback.onError(callback, 400, "不能删除自己");
            return;
        }
        for (String peer : userIDList) {
            if (peer != null && ("u_10000".equals(peer) || peer.matches("^[0-9a-fA-F]{32}$"))) {
                Map<String, Object> param = new HashMap<>();
                param.put("peer_account", peer);
                TUICore.callService(TUIConstants.TUIContact.SERVICE_NAME, "bridgeFriendDelete", param, new TUIServiceCallback() {
                    @Override public void onServiceCallback(int code, String message, Bundle bundle) {
                        if (code != 0) { TUIValueCallback.onError(callback, code, message); return; }
                        List<String> remaining = new java.util.ArrayList<>(userIDList);
                        remaining.remove(peer);
                        removeFriends(remaining, new TUIValueCallback<Map<String, Integer>>() {
                            @Override public void onSuccess(Map<String, Integer> results) {
                                results.put(peer, 0); TUIValueCallback.onSuccess(callback, results);
                            }
                            @Override public void onError(int error, String desc) { TUIValueCallback.onError(callback, error, desc); }
                        });
                    }
                });
                return;
            }
        }
        V2TIMManager.getFriendshipManager().deleteFromFriendList(
            userIDList, V2TIMFriendInfo.V2TIM_FRIEND_TYPE_BOTH, new V2TIMValueCallback<List<V2TIMFriendOperationResult>>() {
                @Override
                public void onSuccess(List<V2TIMFriendOperationResult> v2TIMFriendOperationResults) {
                    Map<String, Integer> resultMap = new HashMap<>();
                    for (V2TIMFriendOperationResult v2TIMFriendOperationResult : v2TIMFriendOperationResults) {
                        resultMap.put(v2TIMFriendOperationResult.getUserID(), v2TIMFriendOperationResult.getResultCode());
                    }
                    TUIValueCallback.onSuccess(callback, resultMap);
                }

                @Override
                public void onError(int code, String desc) {
                    TUIValueCallback.onError(callback, code, desc);
                }
            });
    }

    public void deleteFromBlackList(List<String> idList, TUICallback callback) {
        if (idList == null || idList.isEmpty()) {
            TUICallback.onError(callback, 400, "用户账号无效");
            return;
        }
        if (containsLoginUser(idList)) {
            TUICallback.onError(callback, 400, "不能操作自己");
            return;
        }
        Map<String, Object> param = new HashMap<>();
        param.put(TUIConstants.AppBlocklist.PARAM_PEER_ACCOUNT, idList.get(0));
        param.put(TUIConstants.AppBlocklist.PARAM_BLOCKED, false);
        TUICore.callService(TUIConstants.Service.TUI_DEMO, TUIConstants.AppBlocklist.METHOD_CHANGE, param, new TUIServiceCallback() {
            @Override
            public void onServiceCallback(int errorCode, String errorMessage, Bundle bundle) {
                if (errorCode != 0) {
                    TUICallback.onError(callback, errorCode, errorMessage);
                } else {
                    TUICallback.onSuccess(callback);
                    // Older builds used Tencent's native blacklist. Remove that stale entry
                    // after the app-level unblock, but do not let migration cleanup alter UI state.
                    V2TIMManager.getFriendshipManager().deleteFromBlackList(idList, new V2TIMValueCallback<List<V2TIMFriendOperationResult>>() {
                        @Override
                        public void onSuccess(List<V2TIMFriendOperationResult> results) {}

                        @Override
                        public void onError(int code, String desc) {}
                    });
                }
            }
        });
    }

    public void addToBlackList(List<String> idList, TUIValueCallback<Map<String, Integer>> callback) {
        if (idList == null || idList.isEmpty()) {
            TUIValueCallback.onError(callback, 400, "用户账号无效");
            return;
        }
        if (containsLoginUser(idList)) {
            TUIValueCallback.onError(callback, 400, "不能屏蔽自己");
            return;
        }
        String userID = idList.get(0);
        Map<String, Object> param = new HashMap<>();
        param.put(TUIConstants.AppBlocklist.PARAM_PEER_ACCOUNT, userID);
        param.put(TUIConstants.AppBlocklist.PARAM_BLOCKED, true);
        TUICore.callService(TUIConstants.Service.TUI_DEMO, TUIConstants.AppBlocklist.METHOD_CHANGE, param, new TUIServiceCallback() {
            @Override
            public void onServiceCallback(int errorCode, String errorMessage, Bundle bundle) {
                if (errorCode != 0) {
                    TUIValueCallback.onError(callback, errorCode, errorMessage);
                } else {
                    Map<String, Integer> result = new HashMap<>();
                    result.put(userID, 0);
                    TUIValueCallback.onSuccess(callback, result);
                }
            }
        });
    }

    private boolean containsLoginUser(List<String> userIDs) {
        if (userIDs == null) {
            return false;
        }
        String loginUser = V2TIMManager.getInstance().getLoginUser();
        for (String userID : userIDs) {
            if (TextUtils.equals(loginUser, userID)) {
                return true;
            }
        }
        return false;
    }

    public void quitGroup(String groupID, TUICallback callback) {
        if (bridgeGroupAction(groupID, "exit", "", "", callback)) return;
        V2TIMManager.getInstance().quitGroup(groupID, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void dismissGroup(String groupID, TUICallback callback) {
        if (bridgeGroupAction(groupID, "disband", "", "", callback)) return;
        V2TIMManager.getInstance().dismissGroup(groupID, new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        });
    }

    public void clearHistoryMessage(String chatID, boolean isGroup, TUICallback callback) {
        V2TIMCallback v2TIMCallback = new V2TIMCallback() {
            @Override
            public void onSuccess() {
                TUICallback.onSuccess(callback);
            }

            @Override
            public void onError(int code, String desc) {
                TUICallback.onError(callback, code, desc);
            }
        };
        if (isGroup) {
            V2TIMManager.getMessageManager().clearGroupHistoryMessage(chatID, v2TIMCallback);
        } else {
            V2TIMManager.getMessageManager().clearC2CHistoryMessage(chatID, v2TIMCallback);
        }
    }
}
