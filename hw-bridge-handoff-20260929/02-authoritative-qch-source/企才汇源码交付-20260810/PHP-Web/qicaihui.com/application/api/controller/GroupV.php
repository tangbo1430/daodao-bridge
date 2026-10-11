<?php

namespace app\api\controller;

use think\Controller;

class GroupV extends Common
{
    //创建家族
    public function create_group()
    {
        $uid = $this->uid;
        $group_name = input('group_name', '');
        $key_name = "api:family:create_family:" . $uid;
        redis_lock_exit($key_name);
        $reslut = model('api/GroupV')->create_group($uid, $group_name);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //撤回消息
    public function group_msg_recall()
    {
        $group_id = input('group_id', 0);
        $msg_seq = input('msg_seq', 0);
        $uid = $this->uid;
        $reslut = model('api/GroupV')->group_msg_recall($uid, $group_id, $msg_seq);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //解散群
    public function del_group()
    {
        $group_id = input('group_id', 0);
        $key_name = "api:family:del_group:" . $group_id;
        redis_lock_exit($key_name);
        $reslut = model('api/GroupV')->del_group($this->uid, $group_id);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //设置群保护
    public function open_protect()
    {
        $group_id = input('group_id', 0);
        $key_name = "api:family:open_protect:" . $group_id;
        redis_lock_exit($key_name);
        $reslut = model('api/GroupV')->open_protect($this->uid, $group_id);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //获取群保护信息
    public function get_open_protect()
    {
        $group_id = input('group_id', 0);
        $reslut = model('api/GroupV')->get_open_protect($group_id);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    //获取群文件列表
    public function get_group_file_list()
    {
        $group_id = input('group_id', 0);
        $reslut = model('api/GroupV')->get_group_file_list($group_id);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
}
