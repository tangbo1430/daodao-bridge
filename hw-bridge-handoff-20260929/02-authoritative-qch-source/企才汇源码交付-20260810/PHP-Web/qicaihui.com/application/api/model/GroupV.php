<?php

namespace app\api\model;

use think\Db;
use think\Model;

class GroupV extends Model
{
    public function create_group($uid, $group_name)
    {
        if(empty($group_name)) {
            return ['code' => 201, 'msg' => '请输入群组名称', 'data' => null];
        }
	 //未使用邀请码注册用户：限制功能，只能拉一个群，只能加一个好友，第二个群第二个好友出现报错 

        //根据uid 对user_id字段进行find_in_set查询
        $where=[];
        $where[]=['exp',Db::raw("FIND_IN_SET('{$uid}',user_id)")];
        $user_info = Db::name('invitecode')->where($where)->find();
        if(empty($user_info)) {
            $group_count = Db::name('group_v')->where('uid', $uid)->count();
            if($group_count > 0) {
                return ['code' => 201, 'msg' => '只能创建一个群', 'data' => null];
            }
        }
        $group_id = $this->get_available_group_id();
        //获取用户信息
        $user_info = Db::name('user')->find($uid);
        $system_type = 2;
        if(in_array($user_info['system_type'], [3,4])) {
            $system_type = $user_info['system_type'];
        }
        $insert_data = [
            'id' => $group_id,
            'group_name' => $group_name,
            'add_time' => time(),
            'update_time' => time(),
            'im_g_id' => $group_id,
            'uid' => $uid,
            'system_type' => $system_type
        ];
        Db::startTrans();
        try{
            Db::name('group_v')->insert($insert_data);
            $result = model('api/Tencent')->create_family($group_id, $uid, $group_name);
            if($result['code'] == 201) {
                Db::rollback();
                return ['code' => 201, 'msg' => '创建失败1', 'data' => null];
            }
            Db::commit();
            return ['code' => 200, 'msg' => '创建成功', 'data' => ['group_id' => strval($group_id)]];
        } catch(\Exception $e) {
            Db::rollback();
            // dump($e);
            return ['code' => 201, 'msg' => '创建失败', 'data' => null];
        }
    }
    //过滤靓号
    public function get_available_group_id()
    {
        $group_id = mt_rand(10000, 99999);
        $group_info = db::name('group_v')->field('id')->where('id', $group_id)->find();
        if (!empty($group_info)) {
            return $this->get_available_group_id();
        } else {
            return $group_id;
        }
    }
    
    //撤回群消息
    public function group_msg_recall($uid, $group_id, $msg_seq)
    {
        $uid_str = strval($uid);
        $result = model('api/Tencent')->get_group_member_info($group_id, [$uid_str], ['Owner', 'Admin']);
        
        if($result['code'] == 201) {
            return ['code' => 201, 'msg' => '查找数据错误', 'data' => null];
        } else {
            if(empty($result['data'])) {
                return ['code' => 201, 'msg' => '不是管理或群主,不能撤回', 'data' => null];
            }
        }
        $result = model('api/Tencent')->group_msg_recall($group_id, $msg_seq);
        return $result;
    }
    //删除举报信息
    public function del_group($uid, $id)
    {
        $group_info = Db::name('group_v')->find($id);
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        if($group_info['is_delete'] == 2) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        if($group_info['uid'] != $uid) {
            return ['code' => 201, 'msg' => '不是群主，不能解散群', 'data' => null];
        }
        Db::startTrans();
        try{
            db::name('group_v')->where('id', $id)->update(['is_delete' => 2]);
            $result = model('api/Tencent')->destroy_group($group_info['im_g_id']);
            if($result['code'] == 201) {
                Db::rollback();
                return ['code' => 201, 'msg' => '解散失败', 'data' => null];
            }
            
            Db::commit();
            return ['code' => 200, 'msg' => '解散成功', 'data' => null];
        } catch(\Exception $e) {
            Db::rollback();
            return ['code' => 201, 'msg' => '解散失败', 'data' => null];
        }
    }
    //设置群保护
    public function open_protect($uid, $group_id)
    {
        $group_info = Db::name('group_v')->find($group_id);
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        if($group_info['uid'] != $uid) {
            return ['code' => 201, 'msg' => '不是群主，不能设置群保护', 'data' => null];
        }
        if($group_info['open_protect'] == 1) {
            $open_protect = 2;
        } else {
            $open_protect = 1;
        }
        $result = db::name('group_v')->where('id', $group_id)->update(['open_protect' => $open_protect, 'update_time' => time()]);
        if($result) {
            return ['code' => 200, 'msg' => '设置成功', 'data' => null];
        }
        return ['code' => 201, 'msg' => '设置失败', 'data' => null];
    }
    //获取群保护状态
    public function get_open_protect($group_id)
    {
        $group_info = Db::name('group_v')->find($group_id);
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        return ['code' => 200, 'msg' => '获取数据成功', 'data' => ['open_protect' => $group_info['open_protect']]];
    }
    //获取群文件列表
    public function get_group_file_list($group_id)
    {
        $group_info = Db::name('group_v')->find($group_id);
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        $file_list = Db::name('group_file')->where('group_id', $group_id)->select();
        foreach($file_list as $key => $value){
            $file_list[$key]['system_type'] = $group_info['system_type'];
        }
        return ['code' => 200, 'msg' => '获取数据成功', 'data' => ['file_list' => $file_list]];
    }
}
