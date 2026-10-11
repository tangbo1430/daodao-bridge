<?php

namespace app\admin\validate;

use think\Validate;

class User extends Validate
{
    /**
     * 定义验证规则
     * 格式：'字段名'    =>    ['规则1','规则2'...]
     *
     * @var array
     */

    protected $rule = [
        'user_name' => 'require|unique:user,user_name,,uid|mobile',
        'password' => 'require|regex:[a-zA-Z0-9]{6,16}',
        'trade_password' => 'require|regex:\d{6}',
        'nick_name' => 'require|length:1,8',
        'nick_name' => 'require',
        'birthday' => 'require|regex:\d{4}-\d{2}-\d{2}',
        'constellation' => 'require|in:白羊座,金牛座,双子座,巨蟹座,狮子座,处女座,天秤座,天蝎座,射手座,摩羯座,双鱼座,水瓶座',
        'head_pic' => 'require',
        'sex' => 'require|in:1,2',
        'is_can_recharge' => 'require|in:1,2',
        'is_sign' => 'require|in:1,2',
        'login_status' => 'require|in:1,2',
        'system' => 'require|in:0,1,2',
        'special_uid' => 'unique:user,special_uid,,uid|integer',
    ];

    /**
     * 定义错误信息
     * 格式：'字段名.规则名'    =>    '错误信息'
     *
     * @var array
     */
    protected $message = [
        'user_name.require' => '用户名必须',
        'user_name.unique' => '用户名已被占用',
        'user_name.mobile' => '用户名必须是手机号码',
        'password.require' => '密码必须',
        'password.regex' => '密码必须6-16位字母或者数字组成',
        'trade_password.require' => '交易密码必须',
        'trade_password.regex' => '交易密码必须6数字组成',
        'nick_name.require' => '用户昵称必须',
        'nick_name.length' => '用户昵称必须1~8个字符',
        'birthday.require' => '出生日期必须',
        'birthday.regex' => '出生日期格式非法',
        'constellation.require' => '请填写星座',
        'constellation.in' => '星座类型错误',
        'head_pic.require' => '用户头像必须',
        'is_can_recharge.require' => '是否可代充状态必须',
        'is_can_recharge.in' => '是否可代充状态非法',
        'is_sign.require' => '是否签约状态必须',
        'is_sign.in' => '是否签约状态非法',
        'login_status.require' => '是否允许登录状态必须',
        'login_status.in' => '是否允许登录状态非法',
        'special_uid.unique' => '该靓号已占用',
        'special_uid.integer' => '靓号格式错误',
    ];

    protected $scene = [

        'adminEdit' => ['nick_name', 'sex',  'login_status'],
        'adminEditPassword' => ['password'],
        'apiEditPassword' => ['password'],
        'apiEditTradePassword' => ['trade_password'],
        'apiAdd' => ['user_name', 'password'],
        'apiEditInfo' => ['nick_name'],

    ];
}
