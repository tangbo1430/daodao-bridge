package com.qchm.xyz.push;

import android.content.Context;
import android.content.IntentFilter;
import android.os.Bundle;
import android.text.TextUtils;
import android.util.Log;
import androidx.localbroadcastmanager.content.LocalBroadcastManager;
import com.google.gson.Gson;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import com.qchm.xyz.BuildConfig;
import com.qchm.xyz.utils.DemoLog;
import com.qchm.xyz.utils.TUIUtils;
import com.tencent.qcloud.tuicore.TUIConstants;
import com.tencent.qcloud.tuicore.TUICore;
import com.tencent.qcloud.tuicore.interfaces.ITUINotification;
import com.tencent.qcloud.tuicore.interfaces.TUIServiceCallback;
import com.tencent.qcloud.tuikit.tuichat.util.OfflinePushInfoUtils;

import java.util.HashMap;
import java.util.Map;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;

public class OfflinePushAPIDemo {
    public static final String TAG = OfflinePushAPIDemo.class.getSimpleName();
    /**
     *  Register offline push service, called when IM account login is successful.Calling this interface does not require filling parameters into the
     * component's PrivateConstants.
     *
     *  @note  json：
     * {
     *    // huawei
     *   "huaweiPushBussinessId": "",    
     *   "huaweiBadgeClassName": "", 
     * launcher interface
     *
     *   // xiaomi
     *   "xiaomiPushBussinessId": "",
     *   "xiaomiPushAppId": "",
     *   "xiaomiPushAppKey": "",
     *
     *   // meizu
     *   "meizuPushBussinessId": "", 
     *   "meizuPushAppId": "",
     *   "meizuPushAppKey": "",
     *
     *   // vivo
     *   "vivoPushBussinessId": "", 
     *
     *   // google
     *   "fcmPushBussinessId": "", 
     *
     *   // oppo
     *   "oppoPushBussinessId": "", 
     *   "oppoPushAppKey": "",
     *   "oppoPushAppSecret": "",
     *
     *   // honor
     *   "honorPushBussinessId": "",    
     *  }
     *
     */
    public void registerPush(Context context) {
        OfflinePushParamBean offlinePushParamBean = loadPushConfig(context);
        if (offlinePushParamBean == null) {
            DemoLog.e(TAG, "registerPush config is missing");
            return;
        }
        offlinePushParamBean.setHuaweiBadgeClassName("com.qchm.xyz.SplashActivity");
        offlinePushParamBean.setFcmPushChannelId(OfflinePushInfoUtils.FCM_PUSH_CHANNEL_ID);
        offlinePushParamBean.setFcmPushChannelSoundName(OfflinePushInfoUtils.PRIVATE_RING_NAME);

        String jsonStr = new Gson().toJson(offlinePushParamBean);
        if (TextUtils.isEmpty(jsonStr)) {
            DemoLog.e(TAG, "registerPush json is null");
            return;
        }

        DemoLog.d(TAG, "registerPush json = " + jsonStr);
        Map<String, Object> param = new HashMap<>();
        param.put(TUIConstants.TIMPush.SET_CUSTOM_CONFIG_JSON_KEY, jsonStr);
        TUICore.callService(TUIConstants.TIMPush.SERVICE_NAME, TUIConstants.TIMPush.METHOD_SET_CUSTOM_CONFIG_JSON, param);

        Map<String, Object> registerParam = new HashMap<>();
        registerParam.put(TUIConstants.TIMPush.CONTEXT, context);
        TUICore.callService(TUIConstants.TIMPush.SERVICE_NAME, TUIConstants.TIMPush.METHOD_REGISTER_PUSH, registerParam, new TUIServiceCallback() {
            @Override
            public void onServiceCallback(int errorCode, String errorMessage, Bundle bundle) {
                DemoLog.d(TAG, "registerPush errorCode = " + errorCode + ", errorMessage = " + errorMessage);
            }
        });
    }

    private OfflinePushParamBean loadPushConfig(Context context) {
        try (InputStreamReader reader = new InputStreamReader(
                context.getAssets().open("timpush-configs.json"), StandardCharsets.UTF_8)) {
            JsonObject root = JsonParser.parseReader(reader).getAsJsonObject();
            JsonObject config = root.has(BuildConfig.APPLICATION_ID)
                    ? root.getAsJsonObject(BuildConfig.APPLICATION_ID) : null;
            if (config == null) {
                for (Map.Entry<String, JsonElement> entry : root.entrySet()) {
                    if (entry.getValue().isJsonObject()
                            && entry.getValue().getAsJsonObject().has("huaweiPushBussinessId")) {
                        config = entry.getValue().getAsJsonObject();
                        DemoLog.i(TAG, "use legacy push config for " + BuildConfig.APPLICATION_ID);
                        break;
                    }
                }
            }
            return config == null ? null : new Gson().fromJson(config, OfflinePushParamBean.class);
        } catch (Exception e) {
            DemoLog.e(TAG, "load push config failed: " + e.getMessage());
            return null;
        }
    }

    /**
     * When the IM console sets the click follow-up action to "use push component callback to jump"，After the push is successful, register NotifyEvent through
     * TUICore, and click the notification bar event will be returned through the TUICore.onNotifyEvent callback.
     */
    public void registerNotifyEvent() {
        TUICore.registerEvent(TUIConstants.TIMPush.EVENT_NOTIFY, TUIConstants.TIMPush.EVENT_NOTIFY_NOTIFICATION, new ITUINotification() {
            @Override
            public void onNotifyEvent(String key, String subKey, Map<String, Object> param) {
                Log.d(TAG, "onNotifyEvent key = " + key + "subKey = " + subKey);
                if (TUIConstants.TIMPush.EVENT_NOTIFY.equals(key)) {
                    if (TUIConstants.TIMPush.EVENT_NOTIFY_NOTIFICATION.equals(subKey)) {
                        if (param != null) {
                            String extString = (String) param.get(TUIConstants.TIMPush.NOTIFICATION_EXT_KEY);
                            TUIUtils.handleOfflinePush(extString, null);
                        }
                    }
                }
            }
        });
    }

    /**
     * When the IM console sets the click follow-up action to "use push component callback to jump"，After the push is successful, register the broadcast
     * receiver, click the notification bar event to return by sending the broadcast callback. IntentFilter 
     * TUIConstants.TIMPush.NOTIFICATION_BROADCAST_ACTION.
     */
    public void registerNotificationReceiver(Context context, OfflinePushLocalReceiver localReceiver) {
        DemoLog.d(TAG, "registerNotificationReceiver ");
        IntentFilter intentFilter = new IntentFilter();
        intentFilter.addAction(TUIConstants.TIMPush.NOTIFICATION_BROADCAST_ACTION);
        intentFilter.addAction(TUIConstants.TIMPush.BROADCAST_IM_LOGIN_AFTER_APP_WAKEUP);
        LocalBroadcastManager.getInstance(context).registerReceiver(localReceiver, intentFilter);
    }
}
