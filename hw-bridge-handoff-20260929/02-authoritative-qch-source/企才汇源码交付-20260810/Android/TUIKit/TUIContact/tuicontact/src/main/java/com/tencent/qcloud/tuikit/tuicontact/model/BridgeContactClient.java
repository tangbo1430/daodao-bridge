package com.tencent.qcloud.tuikit.tuicontact.model;

import android.os.Handler;
import android.os.Looper;
import android.content.Context;
import com.tencent.qcloud.tuicore.TUILogin;
import com.tencent.qcloud.tuikit.tuicontact.TUIContactService;
import com.tencent.qcloud.tuikit.timcommon.component.interfaces.IUIKitCallback;

import org.json.JSONObject;
import org.json.JSONArray;

import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.regex.Pattern;
import java.util.concurrent.ConcurrentHashMap;
import java.util.List;
import java.util.ArrayList;
import com.tencent.qcloud.tuikit.tuicontact.bean.FriendApplicationBean;

/** Small adapter for the PHP bridge APIs. It keeps secrets out of the APK. */
public final class BridgeContactClient {
    private static final Pattern TS_UID = Pattern.compile("^[0-9a-fA-F]{32}$");
    private static final ConcurrentHashMap<String, String> VERCODE_CACHE = new ConcurrentHashMap<>();
    private BridgeContactClient() {}

    public static boolean isTangUser(String userId) { return userId != null && ("u_10000".equals(userId) || TS_UID.matcher(userId).matches()); }
    public static boolean isTangGroup(String groupId) { return groupId != null && groupId.startsWith("dd_") && groupId.length() > 3; }

    public static void group(String action, String groupId, String name, List<String> members,
                             String field, String value, String requestId, IUIKitCallback<String> callback) {
        groupDetails(action, groupId, name, members, field, value, requestId, new IUIKitCallback<JSONObject>() {
            @Override public void onSuccess(JSONObject value) { if (callback != null) callback.onSuccess(value.optString("group_id")); }
            @Override public void onError(String module, int code, String message) { if (callback != null) callback.onError(module, code, message); }
        });
    }

    public static void groupDetails(String action, String groupId, String name, List<String> members,
                                    String field, String value, String requestId, IUIKitCallback<JSONObject> callback) {
        new Thread(() -> {
            HttpURLConnection connection = null;
            try {
                Context context = TUIContactService.getAppContext();
                String base = (String) context.getClass().getClassLoader().loadClass("com.qchm.xyz.DemoApplication").getField("NewAddress").get(null);
                JSONObject model = new JSONObject(context.getSharedPreferences("userInfo", 0).getString("per_user_model", "{}"));
                String token = model.optString("userToken");
                if (token.isEmpty()) throw new IllegalStateException("登录已失效");
                JSONObject body = new JSONObject();
                body.put("login_token", token); body.put("group_id", groupId); body.put("name", name);
                body.put("members", new JSONArray(members == null ? new ArrayList<String>() : members));
                body.put("field", field); body.put("value", value); body.put("request_id", requestId);
                android.content.SharedPreferences attempts = context.getSharedPreferences("bridgeGroupAttempts", 0);
                String attemptKey = "";
                if ("create".equals(action)) {
                    java.util.List<String> sorted = new java.util.ArrayList<>(members == null ? new java.util.ArrayList<String>() : members);
                    java.util.Collections.sort(sorted);
                    byte[] digest = java.security.MessageDigest.getInstance("SHA-256").digest((TUILogin.getLoginUser()+"\n"+name+"\n"+new JSONArray(sorted)).getBytes(StandardCharsets.UTF_8));
                    StringBuilder key = new StringBuilder(); for (byte valueByte : digest) key.append(String.format("%02x", valueByte & 255));
                    attemptKey = key.toString();
                    synchronized (BridgeContactClient.class) {
                        String stableRequest = attempts.getString(attemptKey, "");
                        if (stableRequest.isEmpty()) { stableRequest = java.util.UUID.randomUUID().toString(); attempts.edit().putString(attemptKey, stableRequest).commit(); }
                        body.put("request_id", stableRequest);
                    }
                }
                connection = (HttpURLConnection) new URL(base + "/api/bridge/group_" + action).openConnection();
                connection.setRequestMethod("POST"); connection.setDoOutput(true);
                connection.setConnectTimeout(8000); connection.setReadTimeout(60000);
                connection.setRequestProperty("Content-Type", "application/json");
                try(OutputStream output = connection.getOutputStream()) { output.write(body.toString().getBytes(StandardCharsets.UTF_8)); }
                JSONObject result = readResponse(connection);
                if ("create".equals(action) && !isTangGroup(result.optString("group_id"))) {
                    throw new IllegalStateException("群创建结果确认中，请重试");
                }
                if (!attemptKey.isEmpty()) attempts.edit().remove(attemptKey).apply();
                new Handler(Looper.getMainLooper()).post(() -> callback.onSuccess(result));
            } catch(Exception error) {
                String message = operationErrorMessage(error, "群操作失败，请稍后重试");
                new Handler(Looper.getMainLooper()).post(() -> callback.onError("Bridge", 502, message));
            } finally { if(connection != null) connection.disconnect(); }
        }, "bridge-group").start();
    }

    public static void groupAction(String action, String groupId, List<String> members, String field, String value, IUIKitCallback<Void> callback) {
        group(action, groupId, "", members, field, value, "", new IUIKitCallback<String>() {
            @Override public void onSuccess(String result) { if (callback != null) callback.onSuccess(null); }
            @Override public void onError(String module, int code, String message) { if (callback != null) callback.onError(module, code, message); }
        });
    }

    static void apply(String targetUid, String remark, IUIKitCallback<PairResult> callback) {
        mutate("apply", targetUid, remark, callback);
    }

    static void mutate(String action, String targetUid, String remark, IUIKitCallback<PairResult> callback) {
        new Thread(() -> {
            try {
                Context context = TUIContactService.getAppContext();
                String base = (String) context.getClass().getClassLoader()
                        .loadClass("com.qchm.xyz.DemoApplication").getField("NewAddress").get(null);
                JSONObject model = new JSONObject(context.getSharedPreferences("userInfo", 0)
                        .getString("per_user_model", "{}"));
                JSONObject body = new JSONObject();
                body.put("tencent_user_id", TUILogin.getLoginUser());
                body.put("login_token", model.optString("userToken"));
                body.put("target_tsdd_uid", targetUid);
                body.put("remark", remark == null ? "" : remark);
                body.put("vercode", VERCODE_CACHE.getOrDefault(targetUid, ""));
                if (model.optString("userToken").isEmpty()) throw new IllegalStateException("登录已失效");
                HttpURLConnection connection = (HttpURLConnection) new URL(base + "/api/bridge/friend_" + action).openConnection();
                try {
                connection.setRequestMethod("POST"); connection.setDoOutput(true);
                connection.setConnectTimeout(8000); connection.setReadTimeout(15000);
                connection.setRequestProperty("Content-Type", "application/json");
                byte[] data = body.toString().getBytes(StandardCharsets.UTF_8);
                try (OutputStream output = connection.getOutputStream()) { output.write(data); }
                JSONObject result = readResponse(connection);
                post(callback, true, new PairResult(result.optString("msg", "操作成功"), result.optString("status")));
                } finally { connection.disconnect(); }
            } catch (Exception error) {
                post(callback, false, new PairResult(operationErrorMessage(error, "好友操作失败，请稍后重试")));
            }
        }, "bridge-friend-apply").start();
    }

    public static void action(String action, String peer, IUIKitCallback<Void> callback) {
        mutate(action, peer, "", new IUIKitCallback<PairResult>() {
            @Override public void onSuccess(PairResult result) {
                if ("accept".equals(action) && !"active".equals(result.status)) {
                    callback.onError("Bridge", 409, "申请已过期，已发送新的好友申请，等待对方验证"); return;
                }
                if (callback != null) callback.onSuccess(null);
            }
            @Override public void onError(String module, int code, String message) { callback.onError(module, code, message); }
        });
    }

    private static JSONObject readResponse(HttpURLConnection connection) throws Exception {
        int status = connection.getResponseCode();
        java.io.InputStream stream = status >= 200 && status < 300 ? connection.getInputStream() : connection.getErrorStream();
        if (stream == null) throw new IllegalStateException("服务响应异常，请稍后重试");
        JSONObject root;
        try (java.io.InputStream input=stream; java.io.ByteArrayOutputStream output=new java.io.ByteArrayOutputStream()) {
            byte[] buffer=new byte[4096]; int count;
            while ((count=input.read(buffer))!=-1) {
                if (output.size()+count>1024*1024) throw new IllegalStateException("服务响应异常，请稍后重试");
                output.write(buffer,0,count);
            }
            root=new JSONObject(output.toString("UTF-8"));
        }
        int code=root.optInt("code",200);
        if (status<200 || status>=300 || (code!=200 && code!=0) || root.has("error"))
            throw new IllegalStateException(root.optString("msg","操作失败，请稍后重试"));
        JSONObject data=root.optJSONObject("data"); return data==null?root:data;
    }

    public static void requests(IUIKitCallback<List<FriendApplicationBean>> callback) {
        new Thread(() -> {
            HttpURLConnection connection=null;
            try {
                Context context=TUIContactService.getAppContext();
                String base=(String)context.getClass().getClassLoader().loadClass("com.qchm.xyz.DemoApplication").getField("NewAddress").get(null);
                JSONObject model=new JSONObject(context.getSharedPreferences("userInfo",0).getString("per_user_model","{}"));
                String token=model.optString("userToken");
                if (token.isEmpty()) throw new IllegalStateException("登录已失效");
                connection=(HttpURLConnection)new URL(base+"/api/bridge/friend_requests").openConnection();
                connection.setConnectTimeout(8000);connection.setReadTimeout(15000);
                JSONObject request = new JSONObject(); request.put("login_token", token);
                writeJson(connection, request);
                JSONArray rows=readResponse(connection).optJSONArray("requests");
                List<FriendApplicationBean> list=new ArrayList<>();
                if (rows!=null) for(int i=0;i<rows.length();i++) {
                    JSONObject row=rows.getJSONObject(i);FriendApplicationBean bean=new FriendApplicationBean();
                    bean.setUserId(row.getString("user_id"));bean.setNickName(row.optString("nick"));
                    bean.setFaceUrl(row.optString("face_url"));bean.setAddWording(row.optString("remark"));
                    bean.setBridgeStatus(row.optString("status"));
                    bean.setAccept("active".equals(row.optString("status")));
                    bean.setAddType(FriendApplicationBean.FRIEND_APPLICATION_COME_IN);list.add(bean);
                }
                new Handler(Looper.getMainLooper()).post(() -> callback.onSuccess(list));
            } catch(Exception error) {
                new Handler(Looper.getMainLooper()).post(() -> callback.onError("Bridge",502,"好友申请同步失败"));
            } finally {if(connection!=null)connection.disconnect();}
        },"bridge-friend-requests").start();
    }

    public interface SearchCallback {
        void onSuccess(SearchResult result);
        void onError(String message);
    }

    public static final class SearchResult {
        public final String userId;
        public final String nick;
        public final String faceUrl;
        public final String vercode;

        SearchResult(String userId, String nick, String faceUrl, String vercode) {
            this.userId = userId;
            this.nick = nick;
            this.faceUrl = faceUrl;
            this.vercode = vercode;
        }
    }

    public static void search(String keyword, SearchCallback callback) {
        new Thread(() -> {
            HttpURLConnection connection = null;
            try {
                Context context = TUIContactService.getAppContext();
                String base = (String) context.getClass().getClassLoader()
                        .loadClass("com.qchm.xyz.DemoApplication").getField("NewAddress").get(null);
                JSONObject model = new JSONObject(context.getSharedPreferences("userInfo", 0).getString("per_user_model", "{}"));
                String token = model.optString("userToken");
                if (token.isEmpty()) throw new IllegalStateException("登录已失效");
                connection = (HttpURLConnection) new URL(base + "/api/bridge/search_user").openConnection();
                connection.setConnectTimeout(8000); connection.setReadTimeout(15000);
                JSONObject request = new JSONObject(); request.put("login_token", token); request.put("keyword", keyword);
                writeJson(connection, request);
                JSONObject payload = readResponse(connection);
                JSONArray users = payload.optJSONArray("users");
                if (users == null || users.length() == 0) throw new IllegalStateException("未找到该用户");
                JSONObject user = users.getJSONObject(0);
                String userId = user.optString("user_id");
                if (!isTangUser(userId)) throw new IllegalStateException("用户资料无效");
                SearchResult result = new SearchResult(userId, user.optString("nick"), user.optString("face_url"), user.optString("vercode"));
                if (!result.vercode.isEmpty()) VERCODE_CACHE.put(userId, result.vercode);
                new Handler(Looper.getMainLooper()).post(() -> callback.onSuccess(result));
            } catch (Exception error) {
                String message = searchErrorMessage(error);
                new Handler(Looper.getMainLooper()).post(() -> callback.onError(message));
            } finally { if (connection != null) connection.disconnect(); }
        }, "bridge-user-search").start();
    }

    private static String searchErrorMessage(Exception error) {
        return operationErrorMessage(error, "用户搜索失败，请稍后重试");
    }

    private static String operationErrorMessage(Exception error, String fallback) {
        if (error instanceof ReflectiveOperationException) return "服务初始化失败，请更新应用后重试";
        if (error instanceof java.io.IOException) return "网络连接失败，请稍后重试";
        if (error instanceof org.json.JSONException) return "服务响应异常，请稍后重试";
        return error instanceof IllegalStateException && error.getMessage() != null
                ? error.getMessage() : fallback;
    }

    private static void post(IUIKitCallback<PairResult> callback, boolean success, PairResult result) {
        new Handler(Looper.getMainLooper()).post(() -> {
            if (success) callback.onSuccess(result); else callback.onError("Bridge", 502, result.message);
        });
    }

    private static void writeJson(HttpURLConnection connection, JSONObject body) throws Exception {
        connection.setRequestMethod("POST"); connection.setDoOutput(true);
        connection.setRequestProperty("Content-Type", "application/json");
        try (OutputStream output = connection.getOutputStream()) { output.write(body.toString().getBytes(StandardCharsets.UTF_8)); }
    }

    static final class PairResult {
        final String message, status;
        PairResult(String message) { this(message, ""); }
        PairResult(String message, String status) { this.message=message; this.status=status; }
    }
}
