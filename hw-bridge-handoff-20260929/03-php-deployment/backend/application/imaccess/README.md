# IM Access

Isolated Tencent IM account mapping and UserSig issuing module.

- Endpoint: `POST /imaccess/session`
- Required parameter: `login_token`
- Configuration: `IM_SDK_APP_ID`, `IM_SDK_SECRET_KEY`
- Optional: `IM_USER_SIG_TTL` (default 86400), `IM_ADMIN_ACCOUNT`, `IM_REST_BASE_URL`

The default REST domain is Singapore (`https://adminapisgp.im.qcloud.com`). Override it for applications in another region.

Apply `database/001_create_im_access_tables.sql` before use. This module does not update the legacy `yy_user.user_sig` column.

`tests/web_sdk_e2e.cjs` performs a real Web SDK login/logout. It reads the local business token from `IM_ACCESS_TEST_TOKEN` and never stores it in the test file.
