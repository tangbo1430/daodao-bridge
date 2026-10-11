from pathlib import Path
root=Path('C:/Users/Administrator/Desktop/tx2daodao/hw-bridge-handoff-20260929/02-authoritative-qch-source/企才汇源码交付-20260810/Android/qch/app/src/main/java/com/qchm/xyz/bj')
for cls in ('RegistActivity','RegistActivity2'):
 p=root/(cls+'.java');s=p.read_text(encoding='utf-8')
 s=s.replace('    private boolean registering;','    private boolean registering;\n    private boolean strictBridgeRegistration;')
 s=s.replace('        registerButton = findViewById(R.id.tvLogin);','        registerButton = findViewById(R.id.tvLogin);\n        loadBridgeRegistrationRules();')
 needle='                    if (TextUtils.isEmpty(phone)) {'
 invite='pass2' if cls=='RegistActivity' else 'pass3'
 s=s.replace(needle,'''                    if (strictBridgeRegistration && (!phone.matches("(?:[A-Za-z][A-Za-z0-9_]{7,21}|[0-9]{8,22})") || TextUtils.isEmpty(INVITE))) {
                        new android.app.AlertDialog.Builder(CLASS.this).setTitle("注册信息")
                                .setMessage("用户名须为8至22位，英文字母开头，可含数字和下划线；纯数字账号需邀请码允许。请输入有效叨叨邀请码。")
                                .setPositiveButton("知道了", null).show();
                        return;
                    }
'''.replace('INVITE',invite).replace('CLASS',cls)+needle,1)
 # The first occurrence belongs to registration; login retains its own flow.
 needle='                                httpByLogin(phone, pass);'
 s=s.replace(needle,'''                                JSONObject registration = object.optJSONObject("data");
                                if (registration != null && "pending".equals(registration.optString("bridge_status"))) {
                                    setRegistering(false);
                                    Toast.makeText(CLASS.this, "注册已保存，叨叨桥接确认中。请稍后登录，无需重复注册", Toast.LENGTH_LONG).show();
                                    startActivity(new Intent(CLASS.this, LoginActivity.class));
                                    finish();
                                    return;
                                }
'''.replace('CLASS',cls)+needle,1)
 s=s.replace('                        Log.i("get", response.body());','                        // Registration/login responses may contain credentials; never log them.')
 method='''    private void loadBridgeRegistrationRules() {
        OkGo.<String>post(DemoApplication.NewAddress + "/api/login/show_reg").tag(this)
                .cacheMode(CacheMode.NO_CACHE).execute(new StringCallback() {
                    @Override public void onSuccess(Response<String> response) {
                        try {
                            JSONObject envelope = new JSONObject(response.body());
                            JSONObject rules = envelope.optJSONObject("data");
                            if (envelope.optInt("code") != 200 || rules == null) return;
                            strictBridgeRegistration = rules.optBoolean("bridge_strict_registration", false);
                            if (strictBridgeRegistration) {
                                etPass2.setHint("叨叨邀请码（必填）");
                                ACCOUNT_HINT
                            }
                        } catch (JSONException ignored) { /* Server validates every registration. */ }
                    }
                });
    }

'''
 hint='etPhone.setHint("用户名：8至22位，字母开头");\n                                etPhone.setFilters(new android.text.InputFilter[]{new android.text.InputFilter.LengthFilter(22)});\n                                etPhone.setInputType(android.text.InputType.TYPE_CLASS_TEXT);' if cls.endswith('2') else 'etPhone.setHint("手机号（需邀请码允许数字账号）");'
 s=s.replace('    private void setRegistering(boolean registering) {',method.replace('ACCOUNT_HINT',hint)+'    private void setRegistering(boolean registering) {')
 p.write_text(s,encoding='utf-8')
