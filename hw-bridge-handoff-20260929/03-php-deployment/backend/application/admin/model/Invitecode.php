<?php

namespace app\admin\model;

use think\Db;
use think\Model;

class Invitecode extends Model
{
    //获取邀请码列表
    public function get_invitecode_list($name, $order, $sort, $page = 1, $limit = 20)
    {
        $map = [];
        
        $map['system_type'] = config('app.system_type');
        if (!empty($name)) {
            $map[] = ['name', 'like', '%' . $name . '%'];
        }
    
     
        $list = db::name('invitecode')
            //->alias('a')->join('yy_user b', 'a.user_id = b.uid')
            /* ->field('a.*,b.base64_nick_name') */->where($map)->order($order, $sort)->page($page, $limit)->select();
        foreach ($list as $k => &$v) {
           // $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']), 'UTF-8', 'UTF-8');
            $v['nick_name'] ='';
            if($v['user_id']){
                if(strpos($v['user_id'],',')!==false){
                    $user_id = explode(',',$v['user_id']);
                }else{
                    $user_id = [$v['user_id']];
                }
                $nick_name = db::name('user')->where('uid','in',$user_id)->value('nick_name');
                if(is_array($nick_name)){
                    $v['nick_name'] = implode(',',$nick_name);
                }else{
                    $v['nick_name'] = $nick_name;
                }
            }
            //根据user_id获取使用次数
            if($v['invite_num']==0){
                $v['invite_num'] = '无限制';
            }
        }
        
        $data = [];
        $data['count'] = db::name('invitecode')->where($map)->count();
        $data['list'] = $list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    //获取举报详情
    public function get_group_info($id)
    {
        if (empty($id)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $room_type_info = db::name('group_v')->find($id);
        return ['code' => 200, 'msg' => '获取成功', 'data' => $room_type_info];
    }
    //修改举报信息
    public function add_group($group_name, $uid)
    {
        
        if(empty($group_name)) {
            return ['code' => 201, 'msg' => '请输入群组名称', 'data' => null];
        }
        if(empty($uid)) {
            return ['code' => 201, 'msg' => '请输入群组id', 'data' => null];
        }
        $user_info = Db::name('user')->find($uid);
        if(empty($user_info)) {
            return ['code' => 201, 'msg' => '群主iD用户信息不存在', 'data' => null];
        }
        $map = [];
        $map[] = ['group_name', '=', $group_name];
        $map[] = ['is_delete', '=', 1];
        $group_infos = db::name('group_v')->where($map)->find();
        if($group_infos) {
            return ['code' => 201, 'msg' => '群已存在', 'data' => null];
        }
        $group_id = model('api/GroupV')->get_available_group_id();
        $insert_data = [
            'group_name' => $group_name,
            'uid' => $uid,
            'add_time' => time(),
            'update_time' => time(),
            'id' => $group_id,
            'im_g_id' => $group_id,
            'system_type' => config('app.system_type'),
        ];
        Db::startTrans();
        try{
            db::name('group_v')->insert($insert_data);
            $result = model('api/Tencent')->create_family($group_id, $uid, $group_name);
            if($result['code'] == 201) {
                Db::rollback();
                return ['code' => 201, 'msg' => '创建失败1', 'data' => null];
            }
            
            Db::commit();
            return ['code' => 201, 'msg' => '创建成功', 'data' => null];
        } catch(\Exception $e) {
            Db::rollback();
            // dump($e);
            return ['code' => 201, 'msg' => '创建失败', 'data' => null];
        }
        

    }
    //修改举报信息
    public function edit_group($id, $group_name, $uid)
    {
        if(empty($group_name)) {
            return ['code' => 201, 'msg' => '请输入群组名称', 'data' => null];
        }
        if(empty($uid)) {
            return ['code' => 201, 'msg' => '请输入群组id', 'data' => null];
        }
        $user_info = Db::name('user')->find($uid);
        if(empty($user_info)) {
            return ['code' => 201, 'msg' => '群主iD用户信息不存在', 'data' => null];
        }
        $group_info = Db::name('group_v')->find($id);
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        $map = [];
        $map[] = ['group_name', '=', $group_name];
        $map[] = ['is_delete', '=', 1];
        $map[] = ['id', 'neq', $id];
        $group_infos = db::name('group_v')->where($map)->find();
        if($group_infos) {
            return ['code' => 201, 'msg' => '群已存在', 'data' => null];
        }
        
        //更改群主是否是群成功
        if($uid != $group_info['uid']) {
            $uid_str = strval($uid);
            $result = model('api/Tencent')->get_group_member_info($group_info['im_g_id'], [$uid_str], ['Member,Admin']);
            if($result['code'] == 201) {
                return ['code' => 201, 'msg' => '群成员不存在', 'data' => null];
            } else {
                if(empty($result['data'])) {
                    return ['code' => 201, 'msg' => '群成员不存在', 'data' => null];
                }
            }
        }
        
        Db::startTrans();
        try{
            db::name('group_v')->where('id', $id)->update(['uid' => $uid, 'group_name' => $group_name]);
            if($group_name != $group_info['group_name']) {
  
                $result = model('api/Tencent')->modify_group_info($group_info['im_g_id'], $group_name);
                
                if($result['code'] == 201) {
                    Db::rollback();
                    return ['code' => 201, 'msg' => '修改失败', 'data' => null];
                }
            }
            if($uid != $group_info['uid']) {
                $result = model('api/Tencent')->change_group_owner($group_info['im_g_id'], $uid_str);
                if($result['code'] == 201) {
                    Db::rollback();
                    return ['code' => 201, 'msg' => '变更群主id失败', 'data' => null];
                }
            }
            Db::commit();
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        } catch(\Exception $e) {
            Db::rollback();
            // dump($e);
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        }

    }

    //删除举报信息
    public function del_group($id)
    {
        $group_info = Db::name('group_v')->find($id);
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        if($group_info['is_delete'] == 2) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
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
            return ['code' => 201, 'msg' => '解散成功', 'data' => null];
        } catch(\Exception $e) {
            Db::rollback();
            return ['code' => 201, 'msg' => '解散失败', 'data' => null];
        }
    }

    //举报类型 列表
    public function get_user_list($uid, $group_id, $user_name, $order, $sort, $page = 1, $limit = 20)
    {
        $group_info = Db::name('group_v')->where(['id' => $group_id, 'is_delete' => 1])->find();
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群组不存在', 'data' => null];
        }
        $has_group_uid = model('api/Tencent')->get_group_member_list($group_info['im_g_id'], 1, 1000);
        if($has_group_uid['code'] == 201) {
            return ['code' => 201, 'msg' => '获取腾讯云数据错误', 'data' => null];
        } else {
            $member_list = $has_group_uid['data'];
            $has_group_uid = array_column($member_list, 'Member_Account');
        }
        $map = [];
        if($uid) {
            $map[] = ['uid', '=', $uid];
        }
        if($user_name) {
            $map[] = ['user_name', '=', $user_name];
        }
        $map[] = ['uid', 'not in', $has_group_uid];
        $user_list = Db::name('user')->where($map)->field('uid,base64_nick_name,user_name')->order($order, $sort)
        ->page($page, $limit)
        ->select();
        foreach($user_list as &$v) {
            $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']), 'UTF-8', 'UTF-8');
        }
        $data = [];
        $data['count'] = Db::name('user')->where($map)->count();
        $data['list'] = $user_list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    //举报类型 列表
    public function get_group_user_list($uid, $group_id, $user_name, $order, $sort, $page = 1, $limit = 20)
    {
        $group_info = Db::name('group_v')->where(['id' => $group_id, 'is_delete' => 1])->find();
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群组不存在', 'data' => null];
        }
        $has_group_uid = model('api/Tencent')->get_group_member_list($group_info['im_g_id'], 0, 0);
        if($has_group_uid['code'] == 201) {
            return ['code' => 201, 'msg' => '获取腾讯云数据错误', 'data' => null];
        } else {
            $member_list = $has_group_uid['data'];
            $has_group_uid = array_column($member_list, 'Member_Account');
        }
        $map = [];
        if($uid) {
            $map[] = ['uid', '=', $uid];
        }
        if($user_name) {
            $map[] = ['user_name', '=', $user_name];
        }
        $map[] = ['uid', 'in', $has_group_uid];
        $user_list = Db::name('user')->where($map)->field('uid,base64_nick_name,user_name,head_pic')->order($order, $sort)
        ->page($page, $limit)
        ->select();
        foreach($user_list as &$v) {
            $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']), 'UTF-8', 'UTF-8');
            $v['head_pic'] = localpath_to_netpath($v['head_pic']);
        }
        $data = [];
        $data['count'] = Db::name('user')->where($map)->count();
        $data['list'] = $user_list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    //添加举报类型
    public function add_group_user($group_id, $uid_arr)
    {
        if(empty($uid_arr)) {
            return ['code' => 201, 'msg' => '请选择添加人员', 'data' => null];
        }
        if(empty($group_id)) {
            return ['code' => 201, 'msg' => '请选择群组', 'data' => null];
        }
        $group_info = Db::name('group_v')->where(['id' => $group_id, 'is_delete' => 1])->find();
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群组不存在', 'data' => null];
        }
        $uid_arr = array_column($uid_arr, 'uid');
        return model('api/Tencent')->batch_create_group_member($group_info['im_g_id'], $uid_arr);
    }

    
    //删除举报类型
    public function del_group_user($uid, $group_id)
    {
        $group_info = Db::name('group_v')->where(['id' => $group_id, 'is_delete' => 1])->find();
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群组不存在', 'data' => null];
        }
        return model('api/Tencent')->delete_group_member($group_info['im_g_id'], $uid);

    }
}
