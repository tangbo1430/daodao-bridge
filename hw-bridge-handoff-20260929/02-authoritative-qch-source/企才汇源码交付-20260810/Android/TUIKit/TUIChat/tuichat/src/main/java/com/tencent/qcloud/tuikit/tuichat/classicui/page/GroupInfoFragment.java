package com.tencent.qcloud.tuikit.tuichat.classicui.page;

import android.app.Activity;
import android.content.Intent;
import android.os.Bundle;
import android.text.TextUtils;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.CompoundButton;

import androidx.activity.result.ActivityResult;
import androidx.activity.result.ActivityResultCallback;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;

import com.baoyz.actionsheet.ActionSheet;
import com.blankj.utilcode.util.ToastUtils;
import com.tencent.imsdk.v2.V2TIMCallback;
import com.tencent.imsdk.v2.V2TIMGroupListener;
import com.tencent.imsdk.v2.V2TIMManager;
import com.tencent.imsdk.v2.V2TIMValueCallback;
import com.tencent.qcloud.tuicore.TUICore;
import com.tencent.qcloud.tuicore.util.ToastUtil;
import com.tencent.qcloud.tuikit.timcommon.component.activities.ImageSelectActivity;
import com.tencent.qcloud.tuikit.timcommon.util.ScreenUtil;
import com.tencent.qcloud.tuikit.tuichat.R;
import com.tencent.qcloud.tuikit.tuichat.TUIChatConstants;
import com.tencent.qcloud.tuikit.tuichat.TUIChatService;
import com.tencent.qcloud.tuikit.tuichat.classicui.widget.profile.GroupInfoLayout;
import com.tencent.qcloud.tuikit.tuichat.presenter.GroupProfilePresenter;
import com.tencent.qcloud.tuikit.tuichat.util.TUIChatLog;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

public class GroupInfoFragment extends Fragment {
    private static final String TAG = "GroupInfoFragment";

    private View baseView;
    private GroupInfoLayout groupInfoLayout;

    private String groupId;

    private String mChatBackgroundThumbnailUrl;

    private GroupProfilePresenter groupProfilePresenter = new GroupProfilePresenter();

    String groupGuard = ""; //群成员点击保护  1开


    @Nullable
    @Override
    public View onCreateView(LayoutInflater inflater, @Nullable ViewGroup container, Bundle savedInstanceState) {
        baseView = inflater.inflate(R.layout.group_info_fragment, container, false);
        initView();
        return baseView;
    }

    private void initView() {
        Bundle bundle = getArguments();
        if (bundle == null) {
            ToastUtil.toastShortMessage("groupId is empty. bundle is null");
            return;
        }
        groupId = bundle.getString(TUIChatConstants.Group.GROUP_ID);
        mChatBackgroundThumbnailUrl = bundle.getString(TUIChatConstants.CHAT_BACKGROUND_URI);
        groupInfoLayout = baseView.findViewById(R.id.group_info_layout);
        groupInfoLayout.setGroupProfilePresenter(groupProfilePresenter);
        groupProfilePresenter.loadGroupProfile(groupId);
        groupProfilePresenter.loadSelfInfo(groupId);

        groupInfoLayout.setOnChangeChatBackgroundListener(new GroupInfoLayout.OnChangeChatBackgroundListener() {
            @Override
            public void onChangeChatBackground() {
                startSetChatBackground();
            }
        });


        V2TIMManager.getGroupManager().getGroupAttributes(groupId, null, new V2TIMValueCallback<Map<String, String>>() {
            @Override
            public void onSuccess(Map<String, String> stringStringMap) {
                if (stringStringMap.containsKey("groupGuard")) {
                    groupGuard = stringStringMap.get("groupGuard");
                    groupInfoLayout.getChat_group_guard().setChecked(groupGuard.equals("1"));
                }
                groupInfoLayout.getChat_group_guard().setCheckListener(new CompoundButton.OnCheckedChangeListener() {
                    @Override
                    public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                        groupGuard = isChecked ? "1" : "2";
                        HashMap<String, String> attributeMap = new HashMap<>();
                        attributeMap.put("groupGuard", groupGuard);
                        V2TIMManager.getGroupManager().setGroupAttributes(groupId, attributeMap, new V2TIMCallback() {
                            @Override
                            public void onSuccess() {
                                Log.e("===", "更新成功");
                            }

                            @Override
                            public void onError(int code, String desc) {
                                Log.e("===", "更新失败" + code + desc);
                            }
                        });
                    }
                });
            }

            @Override
            public void onError(int code, String desc) {
                Log.e("===", "获取失败" + code + desc);
            }
        });

        V2TIMManager.getInstance().addGroupListener(new V2TIMGroupListener() {
            @Override
            public void onGroupAttributeChanged(String groupID, Map<String, String> groupAttributeMap) {
                // 群属性变更通知
                if (groupID.equals(groupId) && groupAttributeMap.containsKey("groupGuard")) {
                    groupGuard = groupAttributeMap.get("groupGuard");
                }
            }
        });
        groupInfoLayout.getReport().setOnClickListener(v -> ActionSheet.createBuilder(requireActivity(), getActivity().getSupportFragmentManager())
                .setCancelButtonTitle("取消")
                .setOtherButtonTitles("群成员存在赌博行为", "群成员存在欺诈骗钱行为", "群成员发布不适当内容对我造成骚扰", "群成员传播谣言信息")
                .setCancelableOnTouchOutside(true)
                .setListener(new ActionSheet.ActionSheetListener() {
                    @Override
                    public void onDismiss(ActionSheet actionSheet, boolean isCancel) {

                    }

                    @Override
                    public void onOtherButtonClick(ActionSheet actionSheet, int index) {
                        ToastUtils.showLong("举报成功，等待后台审核！");
                        ((Activity) getContext()).finish();
                    }
                })
                .show());

    }

    private void startSetChatBackground() {
        ArrayList<ImageSelectActivity.ImageBean> faceList = new ArrayList<>();
        ImageSelectActivity.ImageBean defaultFace = new ImageSelectActivity.ImageBean();
        defaultFace.setDefault(true);
        faceList.add(defaultFace);
        for (int i = 0; i < TUIChatConstants.CHAT_CONVERSATION_BACKGROUND_COUNT; i++) {
            ImageSelectActivity.ImageBean imageBean = new ImageSelectActivity.ImageBean();
            imageBean.setImageUri(String.format(TUIChatConstants.CHAT_CONVERSATION_BACKGROUND_URL, (i + 1) + ""));
            imageBean.setThumbnailUri(String.format(TUIChatConstants.CHAT_CONVERSATION_BACKGROUND_THUMBNAIL_URL, (i + 1) + ""));
            faceList.add(imageBean);
        }

        Intent intent = new Intent();
        intent.putExtra(ImageSelectActivity.TITLE, getResources().getString(R.string.chat_background_title));
        intent.putExtra(ImageSelectActivity.SPAN_COUNT, 2);
        int itemWidth = (int) (ScreenUtil.getScreenWidth(TUIChatService.getAppContext()) * 0.42f);
        int itemHeight = (int) (itemWidth / 1.5f);
        intent.putExtra(ImageSelectActivity.ITEM_WIDTH, itemWidth);
        intent.putExtra(ImageSelectActivity.ITEM_HEIGHT, itemHeight);
        intent.putExtra(ImageSelectActivity.DATA, faceList);
        if (TextUtils.isEmpty(mChatBackgroundThumbnailUrl)
                || TextUtils.equals(TUIChatConstants.CHAT_CONVERSATION_BACKGROUND_DEFAULT_URL, mChatBackgroundThumbnailUrl)) {
            intent.putExtra(ImageSelectActivity.SELECTED, defaultFace);
        } else {
            intent.putExtra(ImageSelectActivity.SELECTED, new ImageSelectActivity.ImageBean(mChatBackgroundThumbnailUrl, "", false));
        }
        intent.putExtra(ImageSelectActivity.NEED_DOWNLOAD_LOCAL, true);

        TUICore.startActivityForResult(this, ImageSelectActivity.class, intent.getExtras(), new ActivityResultCallback<ActivityResult>() {
            @Override
            public void onActivityResult(ActivityResult result) {
                if (result.getData() == null) {
                    return;
                }
                ImageSelectActivity.ImageBean imageBean = (ImageSelectActivity.ImageBean) result.getData().getSerializableExtra(ImageSelectActivity.DATA);
                if (imageBean == null) {
                    TUIChatLog.e("GroupInfoFragment", "onActivityResult imageBean is null");
                    return;
                }

                String backgroundUri = imageBean.getLocalPath();
                String thumbnailUri = imageBean.getThumbnailUri();
                TUIChatLog.d("GroupInfoFragment", "onActivityResult backgroundUri = " + backgroundUri);
                mChatBackgroundThumbnailUrl = thumbnailUri;
                String dataUri = thumbnailUri + "," + backgroundUri;
                TUIChatService.getInstance().setChatBackground(groupId, dataUri);
            }
        });
    }

    @Override public void onResume() {
        super.onResume();
        if (groupId != null && groupId.startsWith("dd_") && groupInfoLayout != null) groupProfilePresenter.loadGroupProfile(groupId);
    }

}
