<?php

namespace app\admin\validate;

use think\Validate;

class UserZone extends Validate
{
    protected $rule = [
        'sound_duration' => 'number|between:0,60',
        'content' => 'length:1,300',
    ];

    /**
     * 定义错误信息
     * 格式：'字段名.规则名'    =>    '错误信息'
     *
     * @var array
     */
    protected $message = [
        'sound_duration.number' => '声音长度参数非法',
        'sound_duration.between' => '声音最大限制60s',
        // 'content.require' => '请填写内容',
        'content.length' => '内容字数超出限制',

    ];

    protected $scene = [
        'apiAdd' => ['sound_duration', 'content'],
    ];
}