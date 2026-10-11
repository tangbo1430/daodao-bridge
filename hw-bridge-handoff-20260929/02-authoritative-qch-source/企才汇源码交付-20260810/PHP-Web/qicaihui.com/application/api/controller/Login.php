<?php

namespace app\api\controller;


use think\Controller;


class Login extends Controller
{
    protected function initialize()
    {
        $config = get_uncache_system_config();
        if($config['is_maintenance'] == 2){
            ajaxReturn(301, '系统维护中');
        }
        // ajaxReturn(301, '系统维护');
        // add_operation(0, 0); //用户行为日志
    }
    
    
    public function user_login()
    {

        $user_name = input('user_login', '');
        $password = input('password', '');
        $system = input('system',0);
        $last_login_device = input('last_login_device', '');
        $reslut = model('user')->user_login($user_name, $password, $last_login_device, $system);
        ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    //注册
    public function user_reg()
    {
        header("Access-Control-Allow-Origin: *"); // 允许任意域名发起的跨域请求
        $user_name = input('user_login');
        $password = input('password');
        $reg_code = input('reg_code', '');
        $sms_code = input('sms_code', ''); //短信验证码
        $key_name = "api:login:user_reg:" . $user_name;
        
        
        if($sms_code != 9999){
            $reslut = model('sms')->verification_code($user_name, $sms_code);
            if ($reslut['code'] == 201) {
                return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
            }
        }
        redis_lock_exit($key_name);
        $reslut = model('user')->user_reg($user_name, $password, $reg_code);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    public function user_login_account()
    {

        $user_name = input('user_login', '');
        $password = input('password', '');
        $system = input('system',0);
        $last_login_device = input('last_login_device', '');
        $reslut = model('user')->user_login_account($user_name, $password, $last_login_device, $system);
        ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    //注册
    public function user_reg_account()
    {
        header("Access-Control-Allow-Origin: *"); // 允许任意域名发起的跨域请求
        $user_name = input('user_login');
        $password = input('password');
        $reg_code = input('invite_code', '');
        $repassword = input('repassword', '');

        
        $key_name = "api:login:send_sms:" . $user_name;
        $captcha_code = input('captcha_code', '');
        $captcha_key = input('captcha_key', '');
        $result = model('api/ImgCaptcha')->check_captcha($captcha_code, $captcha_key);
        if ($result['code'] != 200) {
            // ajaxReturn($result['code'], '验证码错误', $result['data']);
        }
        redis_lock_exit($key_name);
        $reslut = model('user')->user_reg_account($user_name, $password, $reg_code, $repassword);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //发送短信
    public function send_sms()
    {
        header("Access-Control-Allow-Origin: *"); // 允许任意域名发起的跨域请求
        $mobile = input('mobile');
        $type = input('type');
        $key_name = "api:login:send_sms:" . $mobile;
        $key_name = "api:login:send_sms:" . $mobile;
        $captcha_code = input('captcha_code', '');
        $captcha_key = input('captcha_key', '');
        $result = model('api/ImgCaptcha')->check_captcha($captcha_code, $captcha_key);
        if ($result['code'] != 200) {
            ajaxReturn($result['code'], $result['msg'], $result['data']);
        }
        redis_lock_exit($key_name);
        $reslut = model('sms')->send_sms($mobile, $type);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //获取图片验证码
    public function create_captcha()
    {
        $reslut = model('api/ImgCaptcha')->create_captcha();
        ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //获取是否显示注册
    public function show_reg()
    {
        $config = get_uncache_system_config();
        $data = ['show_reg' => $config['show_reg']];
        $data['voice_open_status'] = $config['voice_open_status'];
        $data['video_open_status'] = $config['video_open_status'];
        $data['group_voice_open_status'] = $config['group_voice_open_status'];
        $data['group_video_open_status'] = $config['group_video_open_status'];
        $data['account_reg_type'] = $config['account_reg_type'];
        $data['customer_link'] = $config['customer_link'];
        $data['customer_open_status'] = $config['customer_open_status'];
      return ajaxReturn(200, 'success', $data);
    }
}
