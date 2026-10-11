<?php

namespace app\admin\controller;

use think\Controller;
use think\Request;

class Statistics extends Common
{
    //首页基础数据
    public function welcome_data()
    {
        $reslut = model('Statistics')->welcome_data();
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
}
