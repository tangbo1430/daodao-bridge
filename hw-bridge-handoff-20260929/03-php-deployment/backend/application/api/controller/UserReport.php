<?php

namespace app\api\controller;

use think\Controller;

class UserReport extends Common
{
    //用户举报
    public function user_report()
    {
        $to_uid = input('to_uid', 0);
        $type_id = input('type_id', '');
        $image = input('image', '');
        $content = input('content', '');
        $reslut = model('user_report')->user_report($this->uid, $to_uid, $type_id, $image, $content);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    public function report_list(){
        $reslut = model('user_report')->report_list();
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
}