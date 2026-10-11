<?php

namespace app\api\model;

use think\Db;
use think\Model;
use think\facade\Log;

class UserTencent extends Model
{
    //获取黑名单
    public function get_blacklist_list($uid, $rid, $page, $page_limit){
        $page = intval($page);
        $page_limit = $page_limit < 30 ? $page_limit : 30;
        //黑名单列表
        $map = [];
        $map[] = ['a.uid', '=', $uid];
        $user_black = db::name('user_black')->alias('a')->join('yy_user b', 'a.receive_uid = b.uid')
        ->field('a.receive_uid as uid, b.nick_name, b.base64_nick_name,b.head_pic,b.sex')->where($map)
        ->page($page, $page_limit)
        ->select();
        foreach ($user_black as $k => &$v) {
            $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']), 'UTF-8', 'UTF-8');
            $v['head_pic'] = localpath_to_netpath($v['head_pic']);
            
        }
        return ['code' => 200, 'msg' => '获取成功', 'data' => $user_black];
    }
    
    //添加腾讯IM黑名单
    public function add_blacklist($uid, $rid, $user_id){
        
        if(!empty($rid)){
            $room_info = db::name('room')->find($rid);
            $uid = $room_info['room_owner_uid'];
        }
        
        if($uid == $user_id){
            return ['code' => 201, 'msg' => '拉黑用户不能为自己', 'data' => null];
        }
        
        $map = [];
        $map[] = ['uid', '=', $uid];
        $map[] = ['receive_uid', '=', $user_id];
        $user_black_info = db::name('user_black')->where($map)->find();
        if($user_black_info){
            return ['code' => 201, 'msg' => '该用户已被加入黑名单', 'data' => null];
        }
        Db::startTrans();
        try {
            //添加到黑名单
            $insert = [];
            $insert['uid'] = $uid;
            $insert['receive_uid'] = $user_id;
            $insert['add_time'] = time();
            $reslut = db::name('user_black')->insert($insert);
            if(!$reslut){
                Db::rollback();
                return ['code' => 201, 'msg' => '添加失败', 'data' => null];
            }
            
            //添加腾讯IM
            // $user_id = explode(',',$user_id);
            // $uid = strval($uid);
            // $reslut = model('Tencent')->black_list_add($uid,$user_id);
            // if($reslut['code'] != 200){
            //     Db::rollback();
            //     return ['code' => 201, 'msg' => $reslut['msg'], 'data' => null];
            // }
            
            Db::commit();
            return ['code' => 200, 'msg' => '添加成功', 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            dump($e);
            Db::rollback();
            return ['code' => 201, 'msg' => '添加失败', 'data' => null];
        }
    }
    
    //删除腾讯IM黑名单
    public function remove_blacklist($uid, $rid, $user_id){
        if(!empty($rid)){
            $room_info = db::name('room')->find($rid);
            $uid = $room_info['room_owner_uid'];
        }
        
        $map = [];
        $map[] = ['uid', '=', $uid];
        $map[] = ['receive_uid', '=', $user_id];
        $user_black_info = db::name('user_black')->where($map)->find();
        if(!$user_black_info){
            return ['code' => 201, 'msg' => '该用户没有被加入黑名单', 'data' => null];
        }
        
        Db::startTrans();
        try {
            //删除腾讯IM黑名单
            $reslut = db::name('user_black')->where('id', $user_black_info['id'])->delete();
            if(!$reslut){
                Db::rollback();
                return ['code' => 201, 'msg' => '移除失败', 'data' => null];
            }
            
            // $user_id = explode(',',$user_id);
            // $uid = strval($uid);
            // $reslut = model('Tencent')->black_list_delete($uid,$user_id);
            // if($reslut['code'] != 200){
            //     Db::rollback();
            //     return ['code' => 201, 'msg' => $reslut['msg'], 'data' => null];
            // }
            
            Db::commit();
            return ['code' => 200, 'msg' => '移除成功', 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            dump($e);
            Db::rollback();
            return ['code' => 201, 'msg' => '移除失败', 'data' => null];
        }
        
    }
    //是否拉黑
    public function get_user_black_status($uid, $receive_uid)
    {
        $map = [];
        $map[] = ['uid', '=', $uid];
        $map[] = ['receive_uid', '=', $receive_uid];
        $info = db::name('user_black')->where($map)->find();
        if($info) {
            $is_black = 1;
        } else {
            $is_black = 2;
        }
        return ['code' => 200, 'msg' => '获取拉黑数据成功', 'data' => ['is_black' => $is_black]];
    }
    //腾讯IM回调
    public function tencent_call_back($data,$sign_data){
        //回调鉴权
        // $token = '__SET_LEGACY_CALLBACK_TOKEN__';
        // $sign = sha256($token.$sign_data['RequestTime']);
        // // $datas = json_encode($data);
        // // error_log($datas, 3, '1.txt');
        // // $sign_datas = json_encode($sign_data);
        // if($sign != $sign_data['Sign']){
        //     return ['ActionStatus' => 'FAIL', 'ErrorInfo' => '', 'ErrorCode' => 1];
        // }
        // dump($data);exit;
        //加入群聊
        // if($data['CallbackCommand'] == 'Group.CallbackAfterNewMemberJoin'){
        //     $rid = $data['GroupId'];
        //     $member_list = $data['NewMemberList'];
        //     $reslut = $this->tencent_group_join_call_back($rid, $member_list[0]['Member_Account']);
        // }
        //群里前置回调
        if($data['CallbackCommand'] == "Group.CallbackBeforeSendMsg"){
            $from_uid = $data['From_Account'];
            $from_user_info = Db::name('user')->find($from_uid);
            if($from_user_info) {
                if($from_user_info['login_status'] != 1) {
                    return ['ActionStatus' => 'OK', 'ErrorInfo' => '您已被平台封禁，不能发送消息', 'ErrorCode' => 120001];
                }
            }
            return ['ActionStatus' => 'OK', 'ErrorInfo' => '', 'ErrorCode' => 0];
        }
        //退出群聊
        if($data['CallbackCommand'] == 'Group.CallbackAfterMemberExit'){
            
            $rid = $data['GroupId'];
            $member_list = $data['ExitMemberList'];
            $reslut = $this->tencent_group_quit_call_back($rid, $member_list[0]['Member_Account']);
        }
        if(isset($data['CallbackCommand']) && ($data['CallbackCommand'] == 'C2C.CallbackBeforeSendMsg')) {
            
            $from_uid = $data['From_Account'];
            $to_uid = $data['To_Account'];
            //是否被拉黑
            $to_is_black = Db::name('user_black')
                ->where(['uid' => $to_uid, 'receive_uid' => $from_uid])->value('id');
            if ($to_is_black) {
                return ['ActionStatus' => 'OK', 'ErrorInfo' => '您已被对方拉黑，不能发送消息', 'ErrorCode' => 120001];
            }
            return ['ActionStatus' => 'OK', 'ErrorInfo' => '', 'ErrorCode' => 0];
        }
        if(isset($data['CallbackCommand']) && ($data['CallbackCommand'] == 'Sns.CallbackFriendAdd')) {
            
            
            $PairList = $data['PairList'] ?? [];
            if($PairList) {
                $to_uid = $PairList[0]['From_Account'];
                $from_uid = $PairList[0]['To_Account'];
                Log::info($to_uid);
                Log::info($from_uid);
                if($to_uid && $from_uid) {
                    $from_nick_name = Db::name('user')->where('uid', $from_uid)->value('base64_nick_name');
                    $from_nick_name = mb_convert_encoding(base64_decode($from_nick_name), 'UTF-8', 'UTF-8');
                    $send_msg = '我是' . $from_nick_name . '已通过你的好友请求';
                    Log::info($send_msg);
                    $this->user_custom_sendmsg($from_uid, $to_uid, $send_msg);
                }
            }
            
            
        }
        //在线状态更新
        if($data['CallbackCommand'] == 'State.StateChange'){
            $info = $data['Info'];
            $reslut = $this->tencent_member_status_change($info);
        }
        
        return ['ActionStatus' => 'OK', 'ErrorInfo' => '', 'ErrorCode' => 0];
    }
    
    //腾讯IM监控加入群聊
    public function tencent_group_join_call_back($rid, $uid){
        //进入房间
        $in_room = db::name('room_visitor')->where(['uid' => $uid, 'rid' => $rid])->find();
        if(empty($in_room) && $rid != '123456789') {
            $insert_vistor_room_data = [
                'uid' => $uid,
                'rid' => $rid,
                'add_time' => time(),
                'update_time' => time(),
            ];
            db::name('room_visitor')->insert($insert_vistor_room_data);
        }
        $count = Db::name('room_visitor')->where('rid', $rid)->count();
        
        $data = [
            'onilne_num' => $count,
            ];
        $push_data = [];
        $push_data['code'] = 210;
        $push_data['msg'] = "用户进入房间";
        $push_data['data'] = $data;
        
        $result = model('WebSocketPush')->send_to_group($rid, $push_data);
    }
    //腾讯IM监控退出群聊
    public function tencent_group_quit_call_back($rid, $uid){
        model('api/room')->quit_room($uid, $rid);
        $count = Db::name('room_visitor')->where('rid', $rid)->count();
        
        $data = [
            'onilne_num' => $count,
            ];
        $push_data = [];
        $push_data['code'] = 210;
        $push_data['msg'] = "用户退出房间";
        $push_data['data'] = $data;
        $result = model('WebSocketPush')->send_to_group($rid, $push_data);
    }
    //腾讯IM监控在线状态更新
    public function tencent_member_status_change($info){
        $uid = $info['To_Account'];
        $action = $info['Action'];
        
        if($action == 'Disconnect') {
            Db::name('user')->where('uid', $uid)->update(['is_online' => 2, 'update_time' => time()]);
            $quit_room = Db::name('room_visitor')->where('uid', $uid)->order('vid', 'asc')->select();
        
            if($quit_room) {
                foreach ($quit_room as $k => $v){
                    model('room')->quit_room($uid, $v['rid']);
                    $count = Db::name('room_visitor')->where('rid', $v['rid'])->count();
                    $data = [
                        'onilne_num' => $count,
                        ];
                    $push_data = [];
                    $push_data['code'] = 210;
                    $push_data['msg'] = "用户退出房间";
                    $push_data['data'] = $data;
                    $result = model('WebSocketPush')->send_to_group($v['rid'], $push_data);
                }
            }
            
        } elseif ($action == 'Login') {
            
            Db::name('user')->where('uid', $uid)->update(['is_online' => 1, 'update_time' => time()]);
        }
        
        
        
    }
    
    //指定用户发送自定义消息给注册用户
    public function user_custom_sendmsg($from_uid, $receive_uid, $message){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = model('api/Tencent')->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        $postUrl = 'https://console.tim.qq.com/v4/openim/sendmsg?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';

        $curlPost = array(
            'SyncOtherMachine'      => 1,
            'OnlineOnlyFlag' => 0,
            'To_Account'            => strval($receive_uid),
            'From_Account' => strval($from_uid),
            'MsgRandom'             => time(),
            'MsgBody'               => array(
                array(
                    'MsgType'       => 'TIMTextElem',
                    'MsgContent'    => array(
                        'Text' => $message,
                        // 'Data'  => json_encode($message),
                        // // 'Desc'  => 'notification',
                    ),
                ),
            ),
        );
        $curlPost = json_encode($curlPost);
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        Log::info('xxxx');
        Log::info($reslut);
        return $reslut;
    }
    
    //腾讯IM请求封装方法
    public function tencent_post_url($postUrl, $curlPost){
        
        $headerArray =array(
            "Content-type:application/json",
            "Accept:application/json",
            );
        
        $ch = curl_init();//初始化curl
        curl_setopt($ch, CURLOPT_URL, $postUrl);//抓取指定网页
        // curl_setopt($ch, CURLOPT_HEADER, 0);//设置header 
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1);//要求结果为字符串且输出到屏幕上
        curl_setopt($ch, CURLOPT_POST, 1);//post提交方式
        curl_setopt($ch, CURLOPT_POSTFIELDS, $curlPost);
        curl_setopt($ch, CURLOPT_HTTPHEADER, $headerArray);
        curl_setopt($ch, CURLOPT_TIMEOUT, 20);
        $data = curl_exec($ch);//运行curl
        curl_close($ch);
        if ($data) {
            $data = json_decode($data, true);
        }
        // dump($data);exit;
        return $data;
    }
}
