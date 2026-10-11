package com.tencent.qcloud.tuikit.tuichat.component.album;

import android.net.Uri;
import androidx.activity.result.ActivityResultCaller;
import com.tencent.qcloud.tuicore.util.ToastUtil;
import com.tencent.qcloud.tuicore.interfaces.TUIValueCallback;
import com.tencent.qcloud.tuikit.timcommon.util.ActivityResultResolver;
import com.tencent.qcloud.tuikit.tuichat.interfaces.AlbumPickerListener;
import com.tencent.qcloud.tuikit.tuichat.interfaces.IAlbumPicker;
import java.util.List;

class SystemAlbumPickerImpl extends IAlbumPicker {
    private static final int MAX_MEDIA_PER_SELECTION = 9;

    @Override
    public void pickMedia(ActivityResultCaller activityResultCaller, AlbumPickerListener listener) {
        if (activityResultCaller == null || listener == null) {
            return;
        }
        // InputView owns MediaLaunchGate before it reaches the picker. Acquiring it again
        // would reject a normal album launch as if the user had double-tapped.
        ActivityResultResolver.getMultipleContentInMediaFlow(activityResultCaller,
            new String[] {ActivityResultResolver.CONTENT_TYPE_IMAGE, ActivityResultResolver.CONTENT_TYPE_VIDEO}, new TUIValueCallback<List<Uri>>() {
                @Override
                public void onSuccess(List<Uri> uris) {
                    if (uris == null || uris.isEmpty()) {
                        listener.onCancel();
                        return;
                    }
                    int mediaCount = Math.min(uris.size(), MAX_MEDIA_PER_SELECTION);
                    for (int index = 0; index < mediaCount; index++) {
                        Uri uri = uris.get(index);
                        listener.onFinished(uri, null);
                    }
                    if (uris.size() > MAX_MEDIA_PER_SELECTION) {
                        ToastUtil.toastShortMessage("一次最多发送9个媒体文件");
                    }
                }

                @Override
                public void onError(int errorCode, String errorMessage) {
                    listener.onCancel();
                }
            });
    }
}
