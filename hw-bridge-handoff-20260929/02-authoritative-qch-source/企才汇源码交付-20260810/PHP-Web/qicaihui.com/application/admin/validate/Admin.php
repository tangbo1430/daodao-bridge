<?php

namespace app\admin\validate;

use think\Validate;

class Admin extends Validate
{
    /**
     * 定义验证规则
     * 格式：'字段名'    =>    ['规则1','规则2'...]
     *
     * @var array
     */

    protected $rule = [
        'user_name' => 'require|unique:admin,user_name,,aid',
        // 'password' => 'require|regex:[a-zA-Z0-9]{6,16}',
        'password' => 'require',

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

        'password.require' => '密码必须',
        // 'password.regex' => '密码必须6-16位字母或者数字组成',

    ];

    protected $scene = [
        'adminEditPassword' => ['password'],
        'adminAdd' => ['user_name', 'password'],
    ];
}
