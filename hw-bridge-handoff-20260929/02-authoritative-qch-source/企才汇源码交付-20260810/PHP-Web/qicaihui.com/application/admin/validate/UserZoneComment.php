<?php

namespace app\admin\validate;

use think\Validate;

class UserZoneComment extends Validate
{
    protected $rule = [

        'content' => 'require|length:1,300',
    ];

    /**
     * 定义错误信息
     * 格式：'字段名.规则名'    =>    '错误信息'
     *
     * @var array
     */
    protected $message = [
        'content.require' => '请填写内容',
        'content.length' => '内容字数超出限制',

    ];

    protected $scene = [
        'apiAdd' => ['content'],
    ];
}