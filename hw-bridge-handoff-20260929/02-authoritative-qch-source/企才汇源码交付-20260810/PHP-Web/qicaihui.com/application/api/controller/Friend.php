<?php

namespace app\api\controller;

use think\Controller;

class Friend extends Common
{
    //创建家族
    public function apply_friend()
    {
        $uid = $this->uid;
        $friend_uid = input('friend_uid', '');
        $key_name = "api:Friend:apply_friend:" . $uid;
        redis_lock_exit($key_name);
        $reslut = model('api/Friend')->apply_friend($uid, $friend_uid);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //撤回消息
    public function get_apply_list()
    {
        $page = input('page', 1);
        $page_limit = input('page_limit', 20);
        $uid = $this->uid;
        $reslut = model('api/Friend')->get_apply_list($uid, $page, $page_limit);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //解散群
    public function del_apply()
    {
        $apply_id = input('apply_id', 0);
        $key_name = "api:friend:del_apply:" . $this->uid;
        redis_lock_exit($key_name);
        $reslut = model('api/Friend')->del_apply($this->uid, $apply_id);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //设置群保护
    public function audit_apply()
    {
        $apply_id = input('apply_id', 0);
        $status = input('status', 0);
        $key_name = "api:friend:audit_apply:" . $this->uid;
        redis_lock_exit($key_name);
        $reslut = model('api/Friend')->audit_apply($this->uid, $apply_id, $status);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    

}
