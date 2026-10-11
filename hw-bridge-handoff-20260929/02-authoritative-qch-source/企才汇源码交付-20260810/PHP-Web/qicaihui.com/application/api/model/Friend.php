<?php

namespace app\api\model;

use think\Db;
use think\Model;

class Friend extends Model
{
    public function apply_friend($uid, $friend_uid)
    {
        if(empty($friend_uid)) {
            return ['code' => 201, 'msg' => '请输入申请好友id', 'data' => null];
        }
        $map = [];
        $map[] = ['status', '=', 1];
        $map[] = ['is_delete', '=', 1];
        $map[] = ['friend_uid', '=', $friend_uid];
        $count = Db::name('user_friend_apply')->where($map)->count();
        if($count > 0) {
            return ['code' => 201, 'msg' => '已申请请勿重复申请', 'data' => null];
        }
        $result = model('api/Tencent')->friend_check($uid, [strval($friend_uid)]);
        // dump($result);die;
        if($result['code'] != 200) {
            return $result;
        }
        $insert_data = [
            'uid' => $uid,
            'friend_uid' => $friend_uid,
            'add_time' => time(),
            'update_time' => time()
        ];
        $result = Db::name('user_friend_apply')->insert($insert_data);
        if($result) {
            return ['code' => 200, 'msg' => '申请成功', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '申请失败', 'data' => null];
        }
    }
    //获取好友申请记录
    public function get_apply_list($uid, $page, $page_limit)
    {
        $map = [];
        $map[] = ['a.friend_uid', '=', $uid];
        $map[] = ['a.is_delete', '=', 1];
        $list = Db::name('user_friend_apply')->alias('a')
            ->leftJoin('user b', 'a.friend_uid = b.uid')
            ->field('a.friend_uid,b.head_pic,b.base64_nick_name,status,a.id as apply_id')
            ->where($map)
            ->page($page, $page_limit)
            ->select();
        foreach($list as &$v) {
            $v['head_pic'] = localpath_to_netpath($v['head_pic']);
            $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']), 'UTF-8', 'UTF-8');
        }
        return ['code' => 200, 'msg' => '获取数据成功', 'data' => $list];
    }
    
    //邀请审核
    public function audit_apply($uid, $apply_id, $status)
    {
        if(!in_array($status, [2,3])) {
            return ['code' => 201, 'msg' => '审核状态错误', 'data' => null];
        }
        $map = [];
        $map[] = ['friend_uid', '=', $uid];
        $map[] = ['is_delete', '=', 1];
        $map[] = ['status', '=', 1];
        $map[] = ['id', '=', $apply_id];
        
        $info = Db::name('user_friend_apply')->where($map)->find();
        if(empty($info)) {
            return ['code' => 201, 'msg' => '已通过', 'data' => null];
        }
        
        $result = Db::name('user_friend_apply')->where('id', $apply_id)->update(['status' => $status, 'update_time' => time()]);
        if($result) {
            return ['code' => 200, 'msg' => '审核成功', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '审核失败', 'data' => null];
        }
    }
    //删除申请
    public function del_apply($uid, $apply_id)
    {
       $map = [];
        $map[] = ['friend_uid', '=', $uid];
        $map[] = ['is_delete', '=', 1];
        $map[] = ['id', '=', $apply_id];
        
        $info = Db::name('user_friend_apply')->where($map)->find();
        if(empty($info)) {
            return ['code' => 201, 'msg' => '已删除', 'data' => null];
        }
        $result = Db::name('user_friend_apply')->where('id', $apply_id)->update(['is_delete' => 2, 'update_time' => time()]);
        if($result) {
            return ['code' => 200, 'msg' => '删除成功', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '删除失败', 'data' => null];
        }
    }
    
}
