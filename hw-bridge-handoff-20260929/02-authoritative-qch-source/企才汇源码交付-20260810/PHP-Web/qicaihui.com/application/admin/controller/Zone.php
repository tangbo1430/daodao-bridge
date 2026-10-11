<?php

namespace app\admin\controller;

use think\Controller;
use think\facade\Request;

class Zone extends Common
{
    //获取社区列表
    public function user_zone_data()
    {
        $uid = input('uid',0);
        $nick_name = input('nick_name','');
        $content = input('content','');
        $show_status = input('show_status','');
        $page = input('page',1);
        $limit = input('limit',20);
        $result = model('UserZone')->user_zone_data($uid,$nick_name,$content,$show_status,$page,$limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $result['data']['count'];
        $data['data'] = $result['data']['list'];
        return json($data);
    }
    
    //审核说说信息
    // public function zone_examine()
    // {
    //     $zid = input('zid', 0);
    //     $show_status = input('show_status', 0);
    //     $reslut = model('Userzone')->zone_examine($zid, $show_status);
    //     return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    // }
    
    public function get_zone_info()
    {
        $zid = input('zid', 0);
        $reslut = model('UserZone')->get_zone_info($zid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //编辑说说
    public function edit_user_zone()
    {
        $data = Request::only(['zid','show_status']);
        $result = model('UserZone')->edit_user_zone($data);
        return ajaxReturn($result['code'],$result['msg'],$result['data']);
    }
}
