package com.tencent.qcloud.tuikit.tuichat.util;

import android.Manifest;
import android.content.Context;
import android.content.pm.ApplicationInfo;
import android.os.Build;
import android.text.TextUtils;
import com.tencent.qcloud.tuicore.TUIConfig;
import com.tencent.qcloud.tuicore.permission.PermissionCallback;
import com.tencent.qcloud.tuicore.permission.PermissionRequester;

public final class MediaReadPermissionHelper {
    private static final int ANDROID_13 = 33;
    private static final String READ_MEDIA_IMAGES = "android.permission.READ_MEDIA_IMAGES";
    private static final String READ_MEDIA_VIDEO = "android.permission.READ_MEDIA_VIDEO";

    private MediaReadPermissionHelper() {}

    public static void requestImagePermission(Callback callback) {
        requestPermission(false, callback);
    }

    public static void requestImageVideoPermission(Callback callback) {
        requestPermission(true, callback);
    }

    private static void requestPermission(boolean includeVideo, Callback callback) {
        String[] permissions = buildPermissions(includeVideo);
        PermissionRequester.newInstance(permissions)
            .title(includeVideo ? "访问照片和视频权限" : "访问照片权限")
            .description(includeVideo ? "用于从相册选择图片或视频并发送到聊天中。" : "用于从相册选择图片。")
            .settingsTip("请在设置-应用-" + getAppName() + "-权限中开启"
                + (includeVideo ? "照片和视频权限。" : "照片权限。"))
            .callback(new PermissionCallback() {
                @Override
                public void onGranted() {
                    if (callback != null) {
                        callback.onGranted();
                    }
                }

                @Override
                public void onDenied() {
                    if (callback != null) {
                        callback.onDenied();
                    }
                }

                @Override
                public void onRequesting() {
                    if (callback != null) {
                        callback.onDenied();
                    }
                }
            })
            .request();
    }

    private static String[] buildPermissions(boolean includeVideo) {
        if (Build.VERSION.SDK_INT >= ANDROID_13) {
            if (includeVideo) {
                return new String[] {READ_MEDIA_IMAGES, READ_MEDIA_VIDEO};
            }
            return new String[] {READ_MEDIA_IMAGES};
        }
        return new String[] {Manifest.permission.READ_EXTERNAL_STORAGE};
    }

    private static String getAppName() {
        Context context = TUIConfig.getAppContext();
        if (context == null) {
            return "App";
        }
        ApplicationInfo applicationInfo = context.getApplicationInfo();
        CharSequence label = applicationInfo.loadLabel(context.getPackageManager());
        if (TextUtils.isEmpty(label)) {
            return "App";
        }
        return label.toString();
    }

    public interface Callback {
        void onGranted();

        void onDenied();
    }
}
