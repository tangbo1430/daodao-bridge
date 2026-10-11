<?php

namespace app\admin\controller;

use think\captcha\Captcha;
use think\Controller;
use think\Db;

class Cli extends Controller
{
    public function export_operate_log()
    {
        $sys_type = input('sys_type', 1);
        $time = input('time', 1);
        model('Admin')->export_operate_log($sys_type, $time);
    }
}
