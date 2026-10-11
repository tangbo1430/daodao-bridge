# 企才汇 PHP API 接口文档

版本：`1.0`  
实现：ThinkPHP 5.1  
源码：`wwwroot/qicaihui.com/application/api/controller`

## 1. 使用说明

### 1.1 基础地址

部署环境的基础地址由环境配置决定，例如：

```text
https://admin.awhaha.com
```

ThinkPHP 当前未配置完整路由表，API 使用控制器/方法自动路由：

```text
POST /api/{controller}/{action}
```

控制器名按实际名称填写，例如 `Login`、`UserTencent`、`GroupV`。

### 1.2 请求格式

- 普通接口：`application/x-www-form-urlencoded` 或 `application/json`
- 上传接口：`multipart/form-data`，文件字段名为 `file`
- 腾讯 IM 回调：`POST` JSON/表单请求，并携带腾讯回调查询参数
- 字符编码：UTF-8
- 分页参数统一使用 `page`、`page_limit`

### 1.3 登录鉴权

登录成功后返回 `login_token`。除登录、注册、验证码、系统配置和腾讯回调外，业务接口必须传：

```text
login_token: string
```

当前 PHP 公共控制器通过 `login_token` 校验用户身份，并将用户 ID 注入 `$this->uid`。建议客户端统一放在请求体中；后续可兼容 `Authorization: Bearer <token>`，但当前源码尚未实现该 Header 鉴权。

### 1.4 统一响应

大多数接口返回：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

| code | 含义 |
| --- | --- |
| `200` | 成功 |
| `0` | 参数缺失或上传失败等业务失败 |
| `201` | 业务校验失败/验证码错误/操作失败 |
| `301` | 未登录、登录失效或系统维护 |
| 其他 | 以 `msg` 为准，客户端不得只判断 HTTP 状态码 |

当前实现部分接口仍直接返回 JSON 或文本，正式对外发布前建议统一为上述响应结构，并将 HTTP 状态码与业务 `code` 分离。

## 2. 接口目录

### 2.1 登录与基础能力

| 方法 | HTTP | 鉴权 | 主要参数 |
| --- | --- | --- | --- |
| `Login/user_login` | POST | 否 | `user_login`, `password`, `system`, `last_login_device` |
| `Login/user_reg` | POST | 否 | `user_login`, `password`, `invite_code`, `repassword`, `captcha_code`, `captcha_key` |
| `Login/user_login_account` | POST | 否 | `user_login`, `password`, `system`, `last_login_device` |
| `Login/user_reg_account` | POST | 否 | `user_login`, `password`, `invite_code`, `repassword`, `sms_code` |
| `Login/send_sms` | POST | 否 | `mobile`, `type` |
| `Login/create_captcha` | GET/POST | 否 | 无固定参数 |
| `Login/show_reg` | GET | 否 | 无固定参数 |
| `Api/modify_password` | POST | 否 | `mobile`, `password`, `sms_code` |
| `Api/send_sms` | POST | 否 | `mobile`, `type` |
| `Api/verification_code` | POST | 否 | `mobile`, `code` |
| `Api/get_system_base_config` | GET | 否 | 无固定参数 |

### 2.2 用户与个人资料

| 方法 | HTTP | 鉴权 | 主要参数 |
| --- | --- | --- | --- |
| `User/get_user_info` | POST | 是 | `login_token` |
| `User/modify_user_info` | POST | 是 | `nick_name`, `birthday`, `sex`, `head_pic`, `city`, `autograph`, `hobby` |
| `User/get_user_constellation` | GET/POST | 是 | `birthday` |
| `User/user_log_out` | POST | 是 | `login_token` |
| `User/modify_password` | POST | 是 | `mobile`, `password`, `sms_code` |
| `User/modify_password_account` | POST | 是 | `password`, `old_password` |
| `User/modify_trade_password` | POST | 是 | `trade_password`, `sms_code` |
| `User/get_user_money_list` | POST | 是 | `page`, `page_limit` |
| `User/user_withdrawal` | POST | 是 | `address`, `money`, `trade_password` |
| `User/get_user_withdrawal_list` | POST | 是 | `page`, `page_limit` |
| `User/real_name_authentication` | POST | 是 | `real_name`, `card_id`, `identity1`, `identity2` |
| `User/cancel_account` | POST | 是 | 以源码校验为准 |

### 2.3 好友、黑名单与群组

| 方法 | HTTP | 鉴权 | 主要参数 |
| --- | --- | --- | --- |
| `Friend/apply_friend` | POST | 是 | `friend_uid` |
| `Friend/get_apply_list` | POST | 是 | `page`, `page_limit` |
| `Friend/del_apply` | POST | 是 | `apply_id` |
| `Friend/audit_apply` | POST | 是 | `apply_id`, `status` |
| `UserTencent/get_blacklist_list` | POST | 是 | `rid`, `page`, `page_limit` |
| `UserTencent/add_blacklist` | POST | 是 | `rid`, `user_id` |
| `UserTencent/remove_blacklist` | POST | 是 | `rid`, `user_id` |
| `UserTencent/get_user_black_status` | POST | 是 | `user_id` |
| `GroupV/create_group` | POST | 是 | `group_name` |
| `GroupV/group_msg_recall` | POST | 是 | `group_id`, `msg_seq` |
| `GroupV/del_group` | POST | 是 | `group_id` |
| `GroupV/open_protect` | POST | 是 | `group_id` |
| `GroupV/get_open_protect` | GET/POST | 是 | `group_id` |
| `GroupV/get_group_file_list` | GET/POST | 是 | `group_id`, 分页参数 |

### 2.4 动态/空间（Zone）

| 方法 | HTTP | 鉴权 | 主要参数 |
| --- | --- | --- | --- |
| `Zone/get_zone_list` | GET/POST | 是 | `is_follow`, `page`, `page_limit`, `status` |
| `Zone/get_zone_info` | GET/POST | 是 | `zid` |
| `Zone/publish_zone` | POST | 是 | `images`（JSON 字符串）, `content` |
| `Zone/praise_zone` | POST | 是 | `zid` |
| `Zone/follow_zone` | POST | 是 | `follow_uid` |
| `Zone/comment_zone` | POST | 是 | `zid`, `content` |
| `Zone/get_comment_list` | GET/POST | 是 | `zid`, `page`, `page_limit` |
| `Zone/delete_zone` | POST | 是 | `zid` |

### 2.5 红包、签到、上传和举报

| 方法 | HTTP | 鉴权 | 主要参数 |
| --- | --- | --- | --- |
| `Envelope/give_red_envelope` | POST | 是 | `group_id`, `money`, `num`, `title`, `interval`, `trade_password`, `is_special`, `receive_uid` |
| `Envelope/snatch_red_envelope` | POST | 是 | `group_id`, `envelope_id` |
| `Envelope/red_envelope_info` | GET/POST | 是 | `envelope_id` |
| `Envelope/snatch_red_envelope_info` | GET/POST | 是 | `envelope_id` |
| `SignConfig/get_sign_config` | GET/POST | 是 | 无固定参数 |
| `SignConfig/sign_in` | POST | 是 | `day_num` |
| `Upload/img_upload` | POST | 是 | `file`, `file_category` |
| `Upload/file_upload` | POST | 是 | `file`, `file_category` |
| `UserReport/user_report` | POST | 是 | `to_uid`, `type_id`, `image`, `content` |
| `UserReport/report_list` | GET/POST | 是 | 以源码校验为准 |

### 2.6 腾讯 IM 回调

| 方法 | HTTP | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `Tencent/call_back` | POST | 腾讯回调签名 | 接收腾讯 IM 回调事件；查询参数和请求体一起传入 |

## 3. 安全与上线检查

1. 生产环境关闭 `app_debug` 和数据库调试，禁止暴露异常堆栈。
2. 数据库、COS、短信、腾讯 IM 密钥全部通过环境变量或部署密钥注入，不进入代码包。
3. CORS 不应长期使用 `*`，应限制为明确的 Web 域名。
4. 上传接口必须限制扩展名、MIME、大小和存储路径，并禁止脚本执行。
5. 生产环境统一 HTTPS；登录 token、交易密码和身份证资料不得写入普通日志。
6. 建议补充 OpenAPI schema、字段类型、错误码字典和接口自动化测试后再对外发布正式版本。

## 4. 源码对应关系

- 控制器：`wwwroot/qicaihui.com/application/api/controller`
- 模型：`wwwroot/qicaihui.com/application/api/model`
- 公共鉴权：`wwwroot/qicaihui.com/application/api/controller/Common.php`
- 统一响应：`wwwroot/qicaihui.com/application/common.php` 的 `ajaxReturn`
- 路由：`wwwroot/qicaihui.com/route`
