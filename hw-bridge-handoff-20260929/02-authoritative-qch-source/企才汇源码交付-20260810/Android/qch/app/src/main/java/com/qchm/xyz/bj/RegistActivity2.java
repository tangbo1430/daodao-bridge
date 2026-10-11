package com.qchm.xyz.bj;

import android.content.Intent;
import android.os.Bundle;
import android.text.TextUtils;
import android.util.Log;
import android.view.View;
import android.widget.CheckBox;
import android.widget.EditText;
import android.widget.ImageView;
import android.widget.TextView;
import android.widget.Toast;

import com.blankj.utilcode.util.GsonUtils;
import com.bumptech.glide.Glide;
import com.qchm.xyz.DemoApplication;
import com.qchm.xyz.R;
import com.qchm.xyz.TIMAppService;
import com.qchm.xyz.bean.CaptchaBean;
import com.qchm.xyz.bean.UserInfo;
import com.qchm.xyz.bj.view.X5WebViewActivity;
import com.qchm.xyz.config.AppConfig;
import com.qchm.xyz.imaccess.ImLoginCoordinator;
import com.qchm.xyz.imaccess.ImSession;
import com.qchm.xyz.imaccess.ImSessionException;
import com.qchm.xyz.main.MainActivity;
import com.qchm.xyz.utils.DemoLog;
import com.qchm.xyz.utils.TUIUtils;
import com.lzy.okgo.OkGo;
import com.lzy.okgo.cache.CacheMode;
import com.lzy.okgo.callback.StringCallback;
import com.lzy.okgo.model.Response;
import com.tencent.imsdk.v2.V2TIMManager;
import com.tencent.imsdk.v2.V2TIMUserFullInfo;
import com.tencent.qcloud.tuicore.TUILogin;
import com.tencent.qcloud.tuicore.interfaces.TUICallback;
import com.tencent.qcloud.tuicore.util.ToastUtil;
import com.tencent.qcloud.tuikit.timcommon.component.activities.BaseLightActivity;

import org.jetbrains.annotations.Nullable;
import org.json.JSONException;
import org.json.JSONObject;

public class RegistActivity2 extends BaseLightActivity {


    CheckBox iv_check;
    EditText etPass;
    EditText etPass1;
    EditText etPhone;
    EditText etCode;
    EditText etPass2;
    View tvYS;
    View tvYH;
    ImageView imCode;
    TextView registerButton;
    TextView loginLink;

    private CaptchaBean captchaBean;
    private boolean registering;
    private boolean strictBridgeRegistration;


    @Override
    protected void onCreate(@Nullable Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.bj_activity_regist2);
        imCode = findViewById(R.id.imCode);
        iv_check = findViewById(R.id.iv_check);
        etPhone = findViewById(R.id.etPhone);
        etPass = findViewById(R.id.etPass);
        etPass1 = findViewById(R.id.etPass1);
        etPass2 = findViewById(R.id.etPass2);
        tvYS = findViewById(R.id.tvYS);
        tvYH = findViewById(R.id.tvYH);
        etCode = findViewById(R.id.etCode);
        loginLink = findViewById(R.id.tvRegister);
        registerButton = findViewById(R.id.tvLogin);
        loadBridgeRegistrationRules();
        loginLink.setOnClickListener(v -> {
            if (registering) {
                return;
            }
            Intent intent = new Intent(RegistActivity2.this, LoginActivity.class);
            startActivity(intent);
        });
        tvYS.setOnClickListener(v -> {
            Intent intent = new Intent(RegistActivity2.this, X5WebViewActivity.class);
            intent.putExtra("title", "隐私协议");
            intent.putExtra("url", DemoApplication.NewAddress + "/index/index/page_show?type=privacy_policy");
            startActivity(intent);
        });
        tvYH.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                Intent intent = new Intent(RegistActivity2.this, X5WebViewActivity.class);
                intent.putExtra("title", "用户协议");
                intent.putExtra("url", DemoApplication.NewAddress + "/index/index/page_show?type=user_agreement");
                startActivity(intent);
            }
        });
        registerButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                if (registering) {
                    return;
                }
                if (iv_check.isChecked()) {
                    String phone = etPhone.getText().toString();
                    String pass = etPass.getText().toString();
                    String code = etCode.getText().toString();
                    String pass2 = etPass1.getText().toString();
                    String pass3 = etPass2.getText().toString();
                    if (strictBridgeRegistration && !phone.matches("(?:[A-Za-z][A-Za-z0-9_]{7,21}|[0-9]{8,22})")) {
                        new android.app.AlertDialog.Builder(RegistActivity2.this).setTitle("注册信息")
                                .setMessage("用户名须为8至22位；可使用纯数字，或以英文字母开头的字母、数字和下划线。")
                                .setPositiveButton("知道了", null).show();
                        return;
                    }
                    if (TextUtils.isEmpty(phone)) {
                        Toast.makeText(RegistActivity2.this, "请输入手机号...", Toast.LENGTH_SHORT).show();
                    } else if (TextUtils.isEmpty(pass)) {
                        Toast.makeText(RegistActivity2.this, "请输入密码...", Toast.LENGTH_SHORT).show();
                    } else if (TextUtils.isEmpty(code)) {
                        Toast.makeText(RegistActivity2.this, "验证码不能为空...", Toast.LENGTH_SHORT).show();
                    } else if (captchaBean == null || TextUtils.isEmpty(captchaBean.getKey())) {
                        Toast.makeText(RegistActivity2.this, "验证码加载中，请稍后重试", Toast.LENGTH_SHORT).show();
                    } else {
                        if(TextUtils.isEmpty(pass3)){
                            pass3="";
                        }
                        setRegistering(true);
                        httpByRegister(phone, pass, code, pass2,pass3);
                    }
                } else {
                    Toast.makeText(RegistActivity2.this, "请先阅读并同意用户协议和隐私协议...", Toast.LENGTH_SHORT).show();
                }
            }
        });
        imCode.setOnClickListener(v -> getCode());
        getCode();
    }

    private void getCode() {
        String getCodeUrl = DemoApplication.NewAddress + "/api/login/create_captcha";
        OkGo.<String>post(getCodeUrl)
                .tag(this)
                .cacheMode(CacheMode.NO_CACHE)
                .execute(new StringCallback() {
                    @Override
                    public void onSuccess(Response<String> response) {
                        try {
                            JSONObject object = new JSONObject(response.body());
                            if (object.has("code") && object.getInt("code") == 200) {
                                captchaBean = GsonUtils.fromJson(object.getString("data"), CaptchaBean.class);
                                Glide.with(RegistActivity2.this).load(captchaBean.getImage()).into(imCode);
                            }
                        } catch (JSONException e) {
                            e.printStackTrace();
                        }
                    }

                    @Override
                    public void onError(Response<String> response) {
                        super.onError(response);
                    }
                });
    }

    private void httpByRegister(String phone, String pass, String code, String pass2,String pass3) {
//        String url = DemoApplication.NewAddress + "/api/login/user_reg";
        String url = DemoApplication.NewAddress + "/api/login/user_reg_account";
        OkGo.<String>post(url)
                .tag(this)
                .params("user_login", phone)
                .params("password", pass)
                .params("system_type", 2)
//                .params("sms_code", code)
//                .params("reg_code", pass2)
                .params("repassword", pass2)
                .params("captcha_code", code)
                .params("captcha_key", captchaBean.getKey())
                .params("invite_code",  pass3)
                .cacheMode(CacheMode.NO_CACHE)
                .execute(new StringCallback() {

                    @Override
                    public void onSuccess(Response<String> response) {
                        // Registration/login responses may contain credentials; never log them.
                        try {
                            JSONObject object = new JSONObject(response.body());
                            if (object.has("code") && object.getInt("code") == 200) {
                                JSONObject registration = object.optJSONObject("data");
                                if (registration != null && "pending".equals(registration.optString("bridge_status"))) {
                                    setRegistering(false);
                                    Toast.makeText(RegistActivity2.this, "注册信息已保存，账号正在开通，请稍后登录，无需重复注册", Toast.LENGTH_LONG).show();
                                    startActivity(new Intent(RegistActivity2.this, LoginActivity.class));
                                    finish();
                                    return;
                                }
                                httpByLogin(phone, pass);
                            } else {
                                setRegistering(false);
                                if (object.has("msg")) {
                                    Toast.makeText(RegistActivity2.this, object.getString("msg"), Toast.LENGTH_SHORT).show();
                                }
                            }
                        } catch (JSONException e) {
                            setRegistering(false);
                            Toast.makeText(RegistActivity2.this, "注册响应异常", Toast.LENGTH_SHORT).show();
                        }
                    }

                    @Override
                    public void onError(Response<String> response) {
                        setRegistering(false);
                        Toast.makeText(RegistActivity2.this, "注册请求失败，请检查网络", Toast.LENGTH_SHORT).show();
                    }
                });
    }

    private void httpByLogin(String phone, String pass) {
        String url = DemoApplication.NewAddress + "/api/login/user_login";
        OkGo.<String>post(url)
                .tag(this)
                .params("user_login", phone)
                .params("password", pass)
                .params("last_login_device", "001")
                .cacheMode(CacheMode.NO_CACHE)
                .execute(new StringCallback() {

                    @Override
                    public void onSuccess(Response<String> response) {
                        try {
                            JSONObject object = new JSONObject(response.body());
                            if (object.has("code") && object.getInt("code") == 200) {
                                JSONObject data = object.getJSONObject("data");
                                loginIM(data.optString("nick_name"), data.getString("login_token"));
                            } else {
                                setRegistering(false);
                                if (object.has("msg")) {
                                    Toast.makeText(RegistActivity2.this, object.getString("msg"), Toast.LENGTH_SHORT).show();
                                }
                            }
                        } catch (JSONException e) {
                            setRegistering(false);
                            Toast.makeText(RegistActivity2.this, "登录响应异常", Toast.LENGTH_SHORT).show();
                        }
                    }

                    @Override
                    public void onError(Response<String> response) {
                        setRegistering(false);
                        Toast.makeText(RegistActivity2.this, "登录请求失败，请检查网络", Toast.LENGTH_SHORT).show();
                    }
                });
    }

    private void loginIM(String phone, String loginToken) {
        ImLoginCoordinator.getInstance().loginWithToken(this, loginToken, new ImLoginCoordinator.Callback() {
            @Override
            public void onError(ImSessionException error) {
                setRegistering(false);
                ToastUtil.toastLongMessage(getString(R.string.failed_login_tip) + ": " + error.getMessage());
                DemoLog.i(TIMAppService.TAG, "imLogin errorCode = " + error.getCode());
            }

            @Override
            public void onSuccess(ImSession session) {
                V2TIMUserFullInfo info = new V2TIMUserFullInfo();
                info.setNickname(phone);
                V2TIMManager.getInstance()
                        .setSelfInfo(info, null);

                UserInfo.getInstance().setDebugLogin(true);
                Intent intent;
                if (AppConfig.DEMO_UI_STYLE == 0) {
                    intent = new Intent(RegistActivity2.this, MainActivity.class);
                } else {
                    intent = new Intent(RegistActivity2.this, MainActivity.class);
                }
                startActivity(intent);
                TIMAppService.getInstance().registerPushManually();
                finish();
            }
        });
    }

    private void loadBridgeRegistrationRules() {
        OkGo.<String>post(DemoApplication.NewAddress + "/api/login/show_reg").tag(this)
                .cacheMode(CacheMode.NO_CACHE).execute(new StringCallback() {
                    @Override public void onSuccess(Response<String> response) {
                        try {
                            JSONObject envelope = new JSONObject(response.body());
                            JSONObject rules = envelope.optJSONObject("data");
                            if (envelope.optInt("code") != 200 || rules == null) return;
                            strictBridgeRegistration = rules.optBoolean("bridge_strict_registration", false);
                            if (strictBridgeRegistration) {
                                etPass2.setHint("邀请码（选填）");
                                etPhone.setHint("用户名：8至22位，字母开头");
                                etPhone.setFilters(new android.text.InputFilter[]{new android.text.InputFilter.LengthFilter(22)});
                                etPhone.setInputType(android.text.InputType.TYPE_CLASS_TEXT);
                            }
                        } catch (JSONException ignored) { /* Server validates every registration. */ }
                    }
                });
    }

    private void setRegistering(boolean registering) {
        this.registering = registering;
        registerButton.setEnabled(!registering);
        registerButton.setAlpha(registering ? 0.6f : 1f);
        registerButton.setText(registering ? "注册中..." : "注册");
        loginLink.setEnabled(!registering);
        imCode.setEnabled(!registering);
    }

}
