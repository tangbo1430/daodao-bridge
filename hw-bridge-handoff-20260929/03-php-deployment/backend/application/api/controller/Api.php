<?php

namespace app\api\controller;

use think\Controller;

use think\db;


class Api extends Controller
{



    //修改登录密码-找回密码 
    public function modify_password()
    {
        // return ajaxReturn(201,'请联系客服修改',null);
        $mobile = input('mobile');
        $password = input('password');
        $sms_code = input('sms_code'); //短信验证码
        $reslut = model('sms')->verification_code($mobile, $sms_code);
        if ($reslut['code'] == 201) {
            return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
        }
        $reslut = model('user')->modify_password($mobile, $password);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    //发送短信
    public function send_sms()
    {
        $mobile = input('mobile');
        $type = input('type');
        $key_name = "api:login:send_sms:" . $mobile;
        redis_lock_exit($key_name);
        $reslut = model('sms')->send_sms($mobile, $type);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    //效验验证码
    public function verification_code()
    {
        // return ajaxReturn(201, '暂未开放', null);
        $mobile = input('mobile');
        $code = input('code');
        $reslut = model('sms')->verification_code($mobile, $code);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //获取系统基础配置
    public function get_system_base_config()
    {
        $config = get_uncache_system_config();
        $data = [];
        // $data['ry_app_key'] = $config['ry_app_key'];
        $data['tencentyun_im_appid'] = $config['tencentyun_im_appid'];
        $data['voice_open_status'] = $config['voice_open_status'];
        $data['video_open_status'] = $config['video_open_status'];
        return ajaxReturn(200, 'success', $data);
    }
    
   
    
    
    //清除数据库
    public function clears_yincaihui(){
        // echo '引才荟清数据';die;
        return ajaxReturn(201, '暂未开放', null);
        model('sms')->clears();
    }
    
    public function get_ip_address(){
        // exit;
        $code = input('code','');
        model('Sms')->get_ip_address($code);
    }
    
    //定时器  会员到期更新时间
    public function close_user_vip_end_time(){
        $reslut = model('user')->close_user_vip_end_time();
        if($reslut['code'] == 200){
            echo date('Y-m-d H:i:s').'会员到期更新时间执行成功';
        }else{
            echo date('Y-m-d H:i:s').'会员到期更新时间执行失败';
        }
        
    }
    
    public function del_ip_address(){
        $ip = input('ip','');
        $code = input('code','');
        model('Sms')->del_ip_address($ip, $code);
    }
    
    
    public function test()
    {
        // $group_id = '2';
        // $result = model('api/Tencent')->create_family($group_id, 10272);
        // die;
        // $group_id = '490';
        // // $uid = 10266;
        // // $result = model('api/Tencent')->get_group_member_list($group_id, 1, 10);
        // $msg_seq = 836;
        // $result = model('api/Tencent')->group_msg_recall($group_id, $msg_seq);
        // dump($result);die;
        $total_money = 100;
        $total_num = 10;
        
        $result = model('api/EnvelopeCreate')->generateRedPacketsNew($total_money, $total_num);
        dump($result);die;
        
    }
    
}
