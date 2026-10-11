<?php

namespace app\api\controller;

use think\Controller;
use think\Db;
class GroupV extends Common
{
    //创建家族
    public function create_group()
    {
        $uid = $this->uid;
        $group_name = input('group_name', '');
        // JSON array quotes must bypass the global htmlspecialchars filter.
        $member_ids = $this->parseMemberIds(request()->post('member_ids', '', null));
        $request_id = trim((string) input('request_id', ''));
        $key_name = "api:group:create_group:" . $uid;
        redis_lock_exit($key_name, 1, 60);
        try {
            $result = model('api/GroupV')->create_group($uid, $group_name, $member_ids, $request_id);
            return json(['code' => $result['code'], 'msg' => $result['msg'], 'data' => $result['data']]);
        } finally {
            redis_unlock($key_name);
        }
    }

    private function parseMemberIds($memberIds)
    {
        if (is_string($memberIds)) {
            $memberIds = trim($memberIds);
            if ($memberIds === '') {
                return [];
            }
            $decoded = json_decode($memberIds, true);
            if (is_array($decoded)) {
                return $decoded;
            }
            return array_values(array_filter(array_map('trim', explode(',', $memberIds)), 'strlen'));
        }
        if (!is_array($memberIds)) {
            return [];
        }
        if (count($memberIds) === 1) {
            $first = reset($memberIds);
            if (is_string($first)) {
                $decoded = json_decode(trim($first), true);
                if (is_array($decoded)) {
                    return $decoded;
                }
            }
        }
        return array_values($memberIds);
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
        $group_id = input('group_id', '');
        $reslut = model('api/GroupV')->get_group_file_list($group_id);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
}
