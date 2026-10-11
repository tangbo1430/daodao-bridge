package com.tencent.qcloud.tuikit.tuicontact.classicui.pages;

import android.content.Intent;
import android.content.SharedPreferences;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.text.Spannable;
import android.text.SpannableString;
import android.text.TextPaint;
import android.text.TextUtils;
import android.text.method.LinkMovementMethod;
import android.text.style.ClickableSpan;
import android.text.style.ForegroundColorSpan;
import android.util.Log;
import android.view.Gravity;
import android.view.View;
import android.widget.TextView;
import android.widget.Toast;

import androidx.annotation.Nullable;

import com.blankj.utilcode.util.SPUtils;
import com.lzy.okgo.OkGo;
import com.lzy.okgo.cache.CacheMode;
import com.lzy.okgo.callback.StringCallback;
import com.lzy.okgo.model.Response;
import com.tencent.imsdk.v2.V2TIMManager;
import com.tencent.qcloud.tuicore.TUIConfig;
import com.tencent.qcloud.tuicore.TUIConstants;
import com.tencent.qcloud.tuicore.TUILogin;
import com.tencent.qcloud.tuicore.TUIThemeManager;
import com.tencent.qcloud.tuicore.util.ToastUtil;
import com.tencent.qcloud.tuikit.timcommon.component.LineControllerView;
import com.tencent.qcloud.tuikit.timcommon.component.PopupInputCard;
import com.tencent.qcloud.tuikit.timcommon.component.TitleBarLayout;
import com.tencent.qcloud.tuikit.timcommon.component.activities.BaseLightActivity;
import com.tencent.qcloud.tuikit.timcommon.component.activities.ImageSelectActivity;
import com.tencent.qcloud.tuikit.timcommon.component.activities.ImageSelectMinimalistActivity;
import com.tencent.qcloud.tuikit.timcommon.component.dialog.TUIKitDialog;
import com.tencent.qcloud.tuikit.timcommon.component.gatherimage.SynthesizedImageView;
import com.tencent.qcloud.tuikit.timcommon.component.impl.GlideEngine;
import com.tencent.qcloud.tuikit.timcommon.component.interfaces.ITitleBarLayout;
import com.tencent.qcloud.tuikit.timcommon.component.interfaces.IUIKitCallback;
import com.tencent.qcloud.tuikit.timcommon.util.ImageUtil;
import com.tencent.qcloud.tuikit.timcommon.util.ScreenUtil;
import com.tencent.qcloud.tuikit.tuicontact.R;
import com.tencent.qcloud.tuikit.tuicontact.TUIContactConstants;
import com.tencent.qcloud.tuikit.tuicontact.bean.ChatInfo;
import com.tencent.qcloud.tuikit.tuicontact.bean.GroupInfo;
import com.tencent.qcloud.tuikit.tuicontact.bean.GroupMemberInfo;
import com.tencent.qcloud.tuikit.tuicontact.classicui.util.ClassicUIUtils;
import com.tencent.qcloud.tuikit.tuicontact.presenter.ContactPresenter;
import com.tencent.qcloud.tuikit.tuicontact.util.TUIContactLog;

import org.json.JSONException;
import org.json.JSONArray;
import org.json.JSONObject;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.UUID;

public class CreateGroupActivity extends BaseLightActivity implements View.OnClickListener {
    private static final String TAG = CreateGroupActivity.class.getSimpleName();

    private static final int CHOOSE_AVATAR_REQUEST_CODE = 1;
    private static final int CHOOSE_GROUP_TYPE_REQUEST_CODE = 2;
    private static final String STATE_CREATE_REQUEST_ID = "create_request_id";
    private TitleBarLayout titleBarLayout;
    private SynthesizedImageView groupAvatar;
    private LineControllerView groupNameLv;
    private LineControllerView groupIdLv;
    private LineControllerView groupTypeLv;
    private View groupAvatarLayout;
    private TextView confirmButton;
    private TextView groupTypeContentView;
    private TextView groupTypeContentUrlView;
    private CharSequence confirmButtonText;

    private String groupName;
    private String groupId;
    private String groupType = V2TIMManager.GROUP_TYPE_WORK;
    private String groupAvatarUrl;
    private List<Object> groupAvatarUrlList;
    private int joinTypeIndex;
    private String groupAvatarImageId = "";

    private ArrayList<GroupMemberInfo> mGroupMembers = new ArrayList<>();
    private boolean mCreating;
    private String createRequestId;

    private ContactPresenter presenter;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.contact_create_group_layout);
        createRequestId = savedInstanceState == null ? null : savedInstanceState.getString(STATE_CREATE_REQUEST_ID);
        if (TextUtils.isEmpty(createRequestId)) {
            createRequestId = UUID.randomUUID().toString();
        }

        titleBarLayout = findViewById(R.id.create_group_bar);
        groupNameLv = findViewById(R.id.group_name_layout);
        groupIdLv = findViewById(R.id.group_id_layout);
        groupTypeLv = findViewById(R.id.group_type_layout);
        groupAvatarLayout = findViewById(R.id.group_avatar_layout);
        groupAvatar = findViewById(R.id.group_avatar);
        confirmButton = findViewById(R.id.confirm_button);
        confirmButtonText = confirmButton.getText();
        groupTypeContentView = findViewById(R.id.group_type_text);
        groupTypeContentUrlView = findViewById(R.id.group_type_text_url);

        initData();
        setupViews();
    }

    private void initData() {
        Intent intent = getIntent();
        if (intent == null) {
            TUIContactLog.e(TAG, "intent is null");
            return;
        }
        Bundle bundle = intent.getExtras();
        if (bundle == null) {
            TUIContactLog.e(TAG, "bundle is null");
            return;
        }

        Object memberData = bundle.getSerializable(TUIConstants.TUIContact.GROUP_MEMBER_ID_LIST);
        if (memberData instanceof ArrayList) {
            mGroupMembers = (ArrayList<GroupMemberInfo>) memberData;
        }

        if (mGroupMembers == null) {
            mGroupMembers = new ArrayList<>();
        }
        if (mGroupMembers.size() <= 1) {
            TUIContactLog.e(TAG, "mGroupMemberIcons <= 1");
        }

        int num = mGroupMembers.size();
        if (num > 9) {
            num = 9;
        }
        for (int i = 0; i < num; i++) {
            GroupMemberInfo member = mGroupMembers.get(i);
            if (member != null && !TextUtils.isEmpty(member.getUserId())) {
                groupAvatarImageId += member.getUserId();
            }
        }

        groupName = bundle.getString(TUIConstants.TUIContact.GROUP_NAME);
        groupNameLv.setContent(groupName);

        groupTypeLv.setContent(groupType);

        if (TUIConfig.isEnableGroupGridAvatar()) {
            groupAvatarUrlList = fillGroupGridAvatar();
            groupAvatar.setImageId(groupAvatarImageId);
            groupAvatar.displayImage(groupAvatarUrlList).load(groupAvatarImageId);
            groupAvatarUrl = null;
        }

        joinTypeIndex = bundle.getInt(TUIConstants.TUIContact.JOIN_TYPE_INDEX, 2);
    }

    private void setupViews() {
        titleBarLayout.getRightIcon().setVisibility(View.GONE);
        titleBarLayout.setTitle(getString(R.string.create_group_chat), ITitleBarLayout.Position.MIDDLE);
        titleBarLayout.setOnLeftClickListener(this);

        int radius = getResources().getDimensionPixelSize(R.dimen.contact_profile_face_radius);
        groupAvatar.setRadius(radius);
        groupAvatar.setOnClickListener(this);
        groupAvatarLayout.setOnClickListener(this);
        groupNameLv.setOnClickListener(this);
        groupIdLv.setOnClickListener(this);
        groupTypeLv.setOnClickListener(this);
        groupAvatarLayout.setOnClickListener(this);

        confirmButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                if (mCreating) {
                    return;
                }
                if (TextUtils.isEmpty(groupName)) {
                    ToastUtil.toastLongMessage(getResources().getString(R.string.group_name_edit_null_tips));
                    return;
                }
                setCreating(true);
                new TUIKitDialog(CreateGroupActivity.this)
                        .builder()
                        .setCancelable(false)
                        .setCancelOutside(false)
                        .setTitle("确定创建群聊吗？")
                        .setDialogWidth(0.75f)
                        .setPositiveButton(getString(com.tencent.qcloud.tuicore.R.string.sure),
                                new View.OnClickListener() {
                                    @Override
                                    public void onClick(View v) {
                                        create_group();
                                    }
                                })
                        .setNegativeButton(getString(com.tencent.qcloud.tuicore.R.string.cancel),
                                new View.OnClickListener() {
                                    @Override
                                    public void onClick(View v) {
                                        setCreating(false);
                                    }
                                })
                        .show();
            }
        });

        presenter = new ContactPresenter();
        initGroupTypeContentView();
    }

    private void initGroupTypeContentView() {
        switch (groupType) {
            case V2TIMManager.GROUP_TYPE_WORK:
                groupTypeContentView.setText(getString(R.string.group_work_content));
                break;
            case V2TIMManager.GROUP_TYPE_PUBLIC:
                groupTypeContentView.setText(getString(R.string.group_public_des));
                break;
            case V2TIMManager.GROUP_TYPE_MEETING:
                groupTypeContentView.setText(getString(R.string.group_meeting_des));
                break;
            case V2TIMManager.GROUP_TYPE_COMMUNITY:
                groupTypeContentView.setText(getString(R.string.group_commnity_des));
                break;
            default:
                groupTypeContentView.setText(getString(R.string.group_work_content));
                break;
        }

        String groupTypeContent = getResources().getString(R.string.group_type_content_title);
        String urlContent = getResources().getString(R.string.group_type_content_url);
        int urlContentIndex = groupTypeContent.lastIndexOf(urlContent);

        final int foregroundColor;
        int currentTheme = TUIThemeManager.getInstance().getCurrentTheme();
        if (currentTheme == TUIThemeManager.THEME_LIVELY) {
            foregroundColor = getResources().getColor(com.tencent.qcloud.tuicore.R.color.core_primary_color_lively);
        } else if (currentTheme == TUIThemeManager.THEME_SERIOUS) {
            foregroundColor = getResources().getColor(com.tencent.qcloud.tuicore.R.color.core_primary_color_serious);
        } else {
            foregroundColor = getResources().getColor(com.tencent.qcloud.tuicore.R.color.core_primary_color_light);
        }

        SpannableString spannedString = new SpannableString(groupTypeContent);
        ForegroundColorSpan colorSpan = new ForegroundColorSpan(foregroundColor);
        spannedString.setSpan(colorSpan, urlContentIndex, urlContentIndex + urlContent.length(), Spannable.SPAN_EXCLUSIVE_INCLUSIVE);

        ClickableSpan clickableSpan = new ClickableSpan() {
            @Override
            public void onClick(View view) {
                if (TextUtils.equals(TUIThemeManager.getInstance().getCurrentLanguage(), "zh")) {
                    openWebUrl(TUIContactConstants.IM_PRODUCT_DOC_URL);
                } else {
                    openWebUrl(TUIContactConstants.IM_PRODUCT_DOC_URL_EN);
                }
            }

            @Override
            public void updateDrawState(TextPaint ds) {
                ds.setUnderlineText(false);
            }
        };
        spannedString.setSpan(clickableSpan, urlContentIndex, urlContentIndex + urlContent.length(), Spannable.SPAN_EXCLUSIVE_INCLUSIVE);
        groupTypeContentUrlView.setText(spannedString);
        groupTypeContentUrlView.setMovementMethod(LinkMovementMethod.getInstance());
    }

    @Override
    public void onClick(View v) {
        if (v == groupAvatar || v == groupAvatarLayout) {
            ArrayList<ImageSelectActivity.ImageBean> faceList = new ArrayList<>();
            if (TUIConfig.isEnableGroupGridAvatar()) {
                ImageSelectActivity.ImageBean imageBean = new ImageSelectActivity.ImageBean();
                imageBean.setGroupGridAvatar(fillGroupGridAvatar());
                imageBean.setImageId(groupAvatarImageId);
                faceList.add(imageBean);
            }

            for (int i = 0; i < TUIContactConstants.GROUP_FACE_COUNT; i++) {
                ImageSelectActivity.ImageBean imageBean = new ImageSelectActivity.ImageBean();
                imageBean.setThumbnailUri(String.format(TUIContactConstants.GROUP_FACE_URL, (i + 1) + ""));
                faceList.add(imageBean);
            }

            Intent intent = new Intent(CreateGroupActivity.this, ImageSelectActivity.class);
            intent.putExtra(ImageSelectActivity.TITLE, getString(R.string.group_choose_avatar));
            intent.putExtra(ImageSelectActivity.SPAN_COUNT, 4);
            intent.putExtra(ImageSelectActivity.PLACEHOLDER, com.tencent.qcloud.tuikit.timcommon.R.drawable.core_default_user_icon_light);
            int itemWidth = (int) (ScreenUtil.getScreenWidth(this) * 0.2f);
            intent.putExtra(ImageSelectMinimalistActivity.ITEM_WIDTH, itemWidth);
            intent.putExtra(ImageSelectMinimalistActivity.ITEM_HEIGHT, itemWidth);
            intent.putExtra(ImageSelectActivity.DATA, faceList);
            intent.putExtra(ImageSelectActivity.SELECTED, faceList.get(0));
            startActivityForResult(intent, CHOOSE_AVATAR_REQUEST_CODE);
        } else if (v == groupNameLv) {
            PopupInputCard popupInputCard = new PopupInputCard(this);
            popupInputCard.setContent(groupNameLv.getContent().toString());
            popupInputCard.setTitle(getResources().getString(R.string.modify_group_name));
            popupInputCard.setMaxLimit(30);
            popupInputCard.setOnPositive(new PopupInputCard.OnClickListener() {
                @Override
                public void onClick(String result) {
                    if (!TextUtils.isEmpty(result)) {
                        groupName = result;
                        groupNameLv.setContent(groupName);

                    } else {
                        ToastUtil.toastLongMessage(getResources().getString(R.string.group_name_edit_null_tips));
                    }
                }
            });
            popupInputCard.setTextExceedListener(new PopupInputCard.OnTextExceedListener() {
                @Override
                public void onTextExceedMax() {
                    ToastUtil.toastLongMessage(getResources().getString(R.string.group_name_edit_exceed_tips));
                }
            });
            popupInputCard.show(groupNameLv, Gravity.BOTTOM);
        } else if (v == groupIdLv) {
            PopupInputCard popupInputCard = new PopupInputCard(this);
            popupInputCard.setContent(groupIdLv.getContent().toString());
            popupInputCard.setTitle(getResources().getString(R.string.modify_group_id));
            popupInputCard.setMaxLimit(48);
            popupInputCard.setOnPositive(new PopupInputCard.OnClickListener() {
                @Override
                public void onClick(String result) {
                    if (TextUtils.equals(groupType, V2TIMManager.GROUP_TYPE_COMMUNITY)) {
                        if (!TextUtils.isEmpty(result) && result.startsWith("@TGS#_") == false) {
                            ToastUtil.toastLongMessage(getResources().getString(R.string.community_id_edit_format_tips));
                            groupIdLv.setContent("");
                            groupId = "";
                            return;
                        }
                    } else {
                        if (!TextUtils.isEmpty(result) && result.startsWith("@TGS#")) {
                            ToastUtil.toastLongMessage(getResources().getString(R.string.group_id_edit_format_tips));
                            groupIdLv.setContent("");
                            groupId = "";
                            return;
                        }
                    }

                    groupIdLv.setContent(result);
                    groupId = result;
                }
            });
            popupInputCard.setTextExceedListener(new PopupInputCard.OnTextExceedListener() {
                @Override
                public void onTextExceedMax() {
                    ToastUtil.toastLongMessage(getResources().getString(R.string.group_id_edit_exceed_tips));
                }
            });
            popupInputCard.show(groupIdLv, Gravity.BOTTOM);
        } else if (v == groupTypeLv) {
            Intent intent = new Intent(this, GroupTypeSelectActivity.class);
            startActivityForResult(intent, CHOOSE_GROUP_TYPE_REQUEST_CODE);
        } else if (v == titleBarLayout.getLeftGroup()) {
            finish();
        }
    }

    private List<Object> fillGroupGridAvatar() {
        final List<Object> urlList = new ArrayList<>();
        String savedIcon = ImageUtil.getGroupConversationAvatar(groupAvatarImageId);
        if (TextUtils.isEmpty(savedIcon)) {
            int faceSize = Math.min(mGroupMembers.size(), 9);
            for (int i = 0; i < faceSize; i++) {
                GroupMemberInfo member = mGroupMembers.get(i);
                String iconUrl = member == null ? null : member.getFaceUrl();
                urlList.add(TextUtils.isEmpty(iconUrl) ? TUIConfig.getDefaultAvatarImage() : iconUrl);
            }
        } else {
            urlList.add(savedIcon);
        }

        return urlList;
    }

    private void createGroupChat() {
        if (mCreating) {
            return;
        }

        final GroupInfo groupInfo = new GroupInfo();
        groupInfo.setChatName(groupName);
        groupInfo.setGroupName(groupName);
        groupInfo.setMemberDetails(mGroupMembers);
        groupInfo.setGroupType(groupType);
        groupInfo.setJoinType(joinTypeIndex);
        groupInfo.setCommunitySupportTopic(false);
        groupInfo.setFaceUrl(groupAvatarUrl);
        groupInfo.setId(groupId);

        mCreating = true;

        presenter.createGroupChat(groupInfo, new IUIKitCallback<String>() {
            @Override
            public void onSuccess(String data) {
                ClassicUIUtils.startChatActivity(data, ChatInfo.TYPE_GROUP, groupInfo.getGroupName(), groupInfo.getGroupType());
                finish();
            }

            @Override
            public void onError(String module, int errCode, String errMsg) {
                mCreating = false;
                if (errCode == TUIConstants.BuyingFeature.ERR_SDK_INTERFACE_NOT_SUPPORT || errCode == 11000) {
                    showNotSupportDialog();
                }
                ToastUtil.toastLongMessage("createGroupChat fail:" + errCode + "=" + errMsg);
            }
        });
    }

    private void showNotSupportDialog() {
        String string = getResources().getString(R.string.contact_im_flagship_edition_update_tip, getString(R.string.contact_community));
        String buyingGuidelines = getResources().getString(R.string.contact_buying_guidelines);
        int buyingGuidelinesIndex = string.lastIndexOf(buyingGuidelines);
        final int foregroundColor = getResources().getColor(TUIThemeManager.getAttrResId(this, com.tencent.qcloud.tuicore.R.attr.core_primary_color));
        SpannableString spannedString = new SpannableString(string);
        ForegroundColorSpan colorSpan2 = new ForegroundColorSpan(foregroundColor);
        spannedString.setSpan(colorSpan2, buyingGuidelinesIndex, buyingGuidelinesIndex + buyingGuidelines.length(), Spannable.SPAN_EXCLUSIVE_INCLUSIVE);

        ClickableSpan clickableSpan2 = new ClickableSpan() {
            @Override
            public void onClick(View view) {
                if (TextUtils.equals(TUIThemeManager.getInstance().getCurrentLanguage(), "zh")) {
                    openWebUrl(TUIConstants.BuyingFeature.BUYING_PRICE_DESC);
                } else {
                    openWebUrl(TUIConstants.BuyingFeature.BUYING_PRICE_DESC_EN);
                }
            }

            @Override
            public void updateDrawState(TextPaint ds) {
                ds.setUnderlineText(false);
            }
        };
        spannedString.setSpan(clickableSpan2, buyingGuidelinesIndex, buyingGuidelinesIndex + buyingGuidelines.length(), Spannable.SPAN_EXCLUSIVE_INCLUSIVE);

        TUIKitDialog.TUIIMUpdateDialog.getInstance()
            .createDialog(this)
            .setMovementMethod(LinkMovementMethod.getInstance())
            .setShowOnlyDebug(true)
            .setCancelable(true)
            .setCancelOutside(true)
            .setTitle(spannedString)
            .setDialogWidth(0.75f)
            .setDialogFeatureName(TUIConstants.BuyingFeature.BUYING_FEATURE_COMMUNITY)
            .setPositiveButton(getString(R.string.contact_no_more_reminders),
                new View.OnClickListener() {
                    @Override
                    public void onClick(View v) {
                        TUIKitDialog.TUIIMUpdateDialog.getInstance().dismiss();
                        TUIKitDialog.TUIIMUpdateDialog.getInstance().setNeverShow(true);
                    }
                })
            .setNegativeButton(getString(R.string.contact_i_know),
                new View.OnClickListener() {
                    @Override
                    public void onClick(View v) {
                        TUIKitDialog.TUIIMUpdateDialog.getInstance().dismiss();
                    }
                })
            .show();
    }

    private void openWebUrl(String url) {
        Intent intent = new Intent();
        intent.setAction(Intent.ACTION_VIEW);
        Uri contentUrl = Uri.parse(url);
        intent.setData(contentUrl);
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        startActivity(intent);
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, @Nullable Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (requestCode == CHOOSE_AVATAR_REQUEST_CODE && resultCode == ImageSelectActivity.RESULT_CODE_SUCCESS) {
            if (data != null) {
                ImageSelectActivity.ImageBean imageBean = (ImageSelectActivity.ImageBean) data.getSerializableExtra(ImageSelectActivity.DATA);
                if (imageBean == null) {
                    return;
                }

                List<Object> faceUrl = imageBean.getGroupGridAvatar();
                if (faceUrl != null && !faceUrl.isEmpty()) {
                    groupAvatar.setImageId(groupAvatarImageId);
                    groupAvatar.displayImage(faceUrl).load(groupAvatarImageId);
                    groupAvatarUrlList = faceUrl;
                    groupAvatarUrl = null;
                } else {
                    groupAvatarUrl = imageBean.getThumbnailUri();
                    GlideEngine.loadImage(groupAvatar, groupAvatarUrl);
                }
            }
        } else if (requestCode == CHOOSE_GROUP_TYPE_REQUEST_CODE) {
            if (data != null) {
                String type = data.getStringExtra(TUIContactConstants.Selection.TYPE);
                if (TextUtils.isEmpty(type)) {
                    TUIContactLog.e(TAG, "onActivityResult type is null");
                    return;
                }

                groupType = type;
                groupTypeLv.setContent(type);
                initGroupTypeContentView();
            }
        }
    }


    private void create_group() {
        if (mGroupMembers == null || mGroupMembers.size() <= 1) {
            setCreating(false);
            ToastUtil.toastLongMessage(getString(R.string.tips_empty_group_member));
            return;
        }
        SharedPreferences shareInfo = getSharedPreferences("userInfo", 0);
        String json = shareInfo.getString("per_user_model", "");
        String token = "";
        try {
            JSONObject object = new JSONObject(json);
            token = object.getString("userToken");
        } catch (JSONException e) {
            setCreating(false);
            ToastUtil.toastLongMessage("登录状态异常，请重新登录");
            return;
        }
        String address="";
        if(!TextUtils.isEmpty(SPUtils.getInstance().getString("Address"))){
            address=SPUtils.getInstance().getString("Address");
        }else {
            address="https://admin.awhaha.com";
        }
        String url = address+"/api/group_v/create_group";
        JSONArray memberIds = new JSONArray();
        Set<String> uniqueMemberIds = new HashSet<>();
        String loginUser = TUILogin.getLoginUser();
        for (GroupMemberInfo member : mGroupMembers) {
            if (member == null) {
                continue;
            }
            String memberId = member.getUserId();
            if (!TextUtils.isEmpty(memberId)
                    && !TextUtils.equals(loginUser, memberId)
                    && uniqueMemberIds.add(memberId)) {
                memberIds.put(memberId);
            }
        }

        OkGo.<String>post(url)
                .tag(this)
                .params("login_token", token)
                .params("group_name", groupName)
                .params("member_ids", memberIds.toString())
                .params("request_id", createRequestId)
                .cacheMode(CacheMode.NO_CACHE)
                .execute(new StringCallback() {
                    @Override
                    public void onSuccess(Response<String> response) {
                        if (!isActivityUsable()) {
                            return;
                        }
                        Log.i("get", response.body());
                        try {
                            if (TextUtils.isEmpty(response.body())) {
                                throw new JSONException("empty response");
                            }
                            JSONObject object = new JSONObject(response.body());
                            if (object.has("code") && object.getInt("code") == 200) {
                                JSONObject data = object.optJSONObject("data");
                                String group_id = data == null ? "" : data.optString("group_id");
                                if (TextUtils.isEmpty(group_id)) {
                                    throw new JSONException("group_id missing");
                                }
                                String actualGroupType = data.optString("group_type", V2TIMManager.GROUP_TYPE_PUBLIC);
                                ToastUtil.toastLongMessage("创建群聊成功！");
                                ClassicUIUtils.startChatActivity(group_id, ChatInfo.TYPE_GROUP, groupName, actualGroupType);
                                finish();
                            } else {
                                setCreating(false);
                                Toast.makeText(CreateGroupActivity.this, object.getString("msg"), Toast.LENGTH_LONG).show();
                            }
                        } catch (JSONException e) {
                            setCreating(false);
                            ToastUtil.toastLongMessage("创建群聊响应异常");
                        }
                    }

                    @Override
                    public void onError(Response<String> response) {
                        if (!isActivityUsable()) {
                            return;
                        }
                        setCreating(false);
                        ToastUtil.toastLongMessage("创建群聊失败，请检查网络");
                    }
                });
    }

    private boolean isActivityUsable() {
        return !isFinishing() && (Build.VERSION.SDK_INT < Build.VERSION_CODES.JELLY_BEAN_MR1 || !isDestroyed());
    }

    @Override
    protected void onSaveInstanceState(Bundle outState) {
        outState.putString(STATE_CREATE_REQUEST_ID, createRequestId);
        super.onSaveInstanceState(outState);
    }

    private void setCreating(boolean creating) {
        mCreating = creating;
        confirmButton.setEnabled(!creating);
        confirmButton.setAlpha(creating ? 0.6f : 1f);
        confirmButton.setText(creating ? "创建中..." : confirmButtonText);
    }

}
