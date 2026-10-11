<?php

namespace app\api\model;
use think\facade\Log;

class WebSocketPushIm
{
    private $md5_key;
    public $str_key = '3651gsdfaj';
    public $send_time;
    public $server_rid = '@TGS#aY4PGONQD';  //腾讯im手动创建直播群
    /*
    200 通用数据
    
    房间PK
    220  推送PK当前结果数据

    300
    用户进入房间推送消息
    301
    房间麦位信息
    302
    房间赠送礼物推送数据
    303
    开宝箱房间播报数据
    304
    开宝箱全服播报数据
    400
    房间相亲数据
    */
    
    public function  __construct()
    {
        $this->send_time = time();
        $key = $this->str_key . '_' . $this->send_time;
        $this->md5_key = md5($key);
        
    }

    //发送消息给个人
    public function send_to_one($uid, $content)
    {
        $content['push_type'] = 'person_push';
        $content['md5_key'] = $this->md5_key;
        $content['send_time'] = $this->send_time;
        $content['businessID'] = 'WebSocket';
        return $this->user_custom_sendmsg($uid, $content);
    }
    //发送消息到指定房间
    public function send_to_group($rid, $content, $from_account, $to_account = [])
    {
        $content['push_type'] = 'group_push';
        $content['md5_key'] = $this->md5_key;
        $content['send_time'] = $this->send_time;
        $rid = strval($rid);
        $rid = $rid;
        return $this->send_group_msg_custom($content, $rid, $from_account, $to_account);
    }
    //发送群体消息
    public function send_to_all($content)
    {
        $rid = $this->server_rid;
        $content['push_type'] = 'server_push';
        $content['md5_key'] = $this->md5_key;
        $content['send_time'] = $this->send_time;
        return $this->send_group_msg_custom($content, $rid);
    }
    //验签
    public function decrypt_data($data, $send_time)
    {
        if($data != md5($this->str_key . '_' . $send_time)) {
            return ['code' => 201, 'msg' => '失败', 'data' => null];
        }
        return ['code' => 200, 'msg' => '成功', 'data' => null];
    }
    //发送群内自定义消息
    public function send_group_msg_custom($message_data, $rid, $from_account, $to_account = []){
        
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = model('api/Tencent')->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        $random = time().rand(111,999);
        $from_account = strval($from_account);
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/send_group_msg?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        if($from_account) {
            $curlPost = array(
                'GroupId'   => $rid,
                'Random'    => $random,
                'From_Account' => $from_account,
                'MsgBody'   => array(
                    array(
                        'MsgType'       => 'TIMCustomElem',
                        'MsgContent'    => array(
                            'Data' => json_encode($message_data),
                        ),
                    ),
                ),
            );
        } else {
            $curlPost = array(
                'GroupId'   => $rid,
                'Random'    => $random,
                'MsgBody'   => array(
                    array(
                        'MsgType'       => 'TIMCustomElem',
                        'MsgContent'    => array(
                            'Data' => json_encode($message_data),
                        ),
                    ),
                ),
            );
        }
        if(!empty($to_account)) {
            $curlPost['To_Account'] = $to_account;
        }
        $curlPost = json_encode($curlPost);
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        return $reslut;
    }
    //发送群内自定义消息
    public function send_group_system_notification($group_id, $content){
        
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = model('api/Tencent')->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        $random = time().rand(111,999);
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/send_group_system_notification?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $curlPost = array(
            'GroupId'   => $group_id,
            'Content' => $content,
        );
        
        $curlPost = json_encode($curlPost);
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        // dump($reslut);die;
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
    
    //发送群消息前置验证
    public function check_send_group_msg($data)
    {
        $rid = $data['GroupId'];
        $from_account = $data['From_Account'];
        $operator_account = $data['Operator_Account'];
        if($rid != $this->server_rid) {
            $content = $data['MsgBody'][0]['MsgContent']['Data'];
            $content_arr = json_decode($content, true);
            if(isset($content_arr['push_type'])) {
                $send_time = $content_arr['send_time'] ?? 0;
                $md5_key = $content_arr['md5_key'] ?? '';
                if(abs($send_time - time()) > 2) {
                    return ['ActionStatus' => 'OK', 'ErrorInfo' => '拦截', 'ErrorCode' => 120001];
                }
                if(empty($md5_key)) {
                    return ['ActionStatus' => 'OK', 'ErrorInfo' => '拦截', 'ErrorCode' => 120001];
                }
                $result = $this->decrypt_data($md5_key, $send_time);
                if($result['code'] != 200) {
                    return ['ActionStatus' => 'OK', 'ErrorInfo' => '拦截', 'ErrorCode' => 120001];
                }
            }
        } else {
            if($from_account != 'administrator' || $operator_account != 'administrator') {
                return ['ActionStatus' => 'OK', 'ErrorInfo' => '拦截', 'ErrorCode' => 120001];
            }
        }
        return ['ActionStatus' => 'OK', 'ErrorInfo' => '', 'ErrorCode' => 0];
    }

    //指定用户发送自定义消息给注册用户
    public function user_custom_sendmsg($receive_uid, $message, $machine_type = 2){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = model('api/Tencent')->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        $postUrl = 'https://console.tim.qq.com/v4/openim/sendmsg?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';

        $curlPost = array(
            'SyncOtherMachine'      => 2,
            'OnlineOnlyFlag' => 1,
            'To_Account'            => strval($receive_uid),
            'MsgRandom'             => time(),
            'MsgBody'               => array(
                array(
                    'MsgType'       => 'TIMCustomElem',
                    'MsgContent'    => array(
                        // 'Text' => $message,
                        'Data'  => json_encode($message),
                        // 'Desc'  => 'notification',
                    ),
                ),
            ),
        );
        $curlPost = json_encode($curlPost);
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        return $reslut;
    }
}
