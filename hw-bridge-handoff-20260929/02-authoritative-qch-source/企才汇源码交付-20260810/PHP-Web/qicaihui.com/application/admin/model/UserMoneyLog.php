<?php

namespace app\admin\model;

use think\Model;

class UserMoneyLog extends Model
{
    public static function ChangeTypeLable()
    {
        return [
            '1' => '系统调节',
            '2' => '签到获得',
            '3' => '发红包',
            '4' => '抢红包',
            '5' => '提现',
            '6' => '提现返还',
            '7' => '红包回退'
        ];
    }
}