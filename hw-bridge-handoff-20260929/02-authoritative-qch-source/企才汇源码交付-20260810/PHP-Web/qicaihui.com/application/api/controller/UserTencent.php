<?php

namespace app\api\controller;

use think\Controller;

class UserTencent extends Common
{
    //获取用户黑名单
    public function get_blacklist_list(){
        $page = input('page', 1);
        $page_limit = input('page_limit', 20);
        $rid = input('rid', 0);
        $reslut = model('api/UserTencent')->get_blacklist_list($this->uid, $rid, $page, $page_limit);
        return ajaxReturn($reslut['code'],$reslut['msg'],$reslut['data']);
    }

    //添加用户黑名单
    public function add_blacklist(){
        $user_id = input('user_id','');
        $rid = input('rid', 0);
        $reslut = model('api/UserTencent')->add_blacklist($this->uid, $rid, $user_id);
        return ajaxReturn($reslut['code'],$reslut['msg'],$reslut['data']);
    }
    
    //移除用户黑名单
    public function remove_blacklist(){
        $user_id = input('user_id','');
        $rid = input('rid', 0);
        $reslut = model('api/UserTencent')->remove_blacklist($this->uid, $rid, $user_id);
        return ajaxReturn($reslut['code'],$reslut['msg'],$reslut['data']);
    }
    
    //获取拉黑
    public function get_user_black_status()
    {
        $user_id = input('user_id','');
        $reslut = model('api/UserTencent')->get_user_black_status($this->uid, $user_id);
        return ajaxReturn($reslut['code'],$reslut['msg'],$reslut['data']);
    }
    
}
