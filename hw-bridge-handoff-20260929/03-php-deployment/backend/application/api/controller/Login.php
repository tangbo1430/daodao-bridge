<?php

namespace app\api\controller;


use think\Controller;
use think\Db;


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
        return json(['code' => $reslut['code'], 'msg' => $reslut['msg'], 'data' => $reslut['data']]);
    }

    //注册
    public function user_reg()
    {
        header("Access-Control-Allow-Origin: *"); // 允许任意域名发起的跨域请求
        $user_name = trim((string) input('user_login', ''));
        $password = (string) input('password', '');
        $reg_code = trim((string) input('reg_code', ''));
        $sms_code = trim((string) input('sms_code', '')); //短信验证码
        $key_name = "api:login:user_reg:" . strtolower($user_name);
        $result_key = $this->registrationResultKey('sms', $user_name, $sms_code);

        redis_lock_exit($key_name, 1, 30);
        try {
            $cached = $this->getRegistrationResult($result_key);
            if ($cached !== null) {
                return json($cached);
            }

            if ($user_name === '' || $password === '') {
                return json(['code' => 201, 'msg' => '账号和密码不能为空', 'data' => null]);
            }
            if (Db::name('user')->where('user_name', $user_name)->count() > 0) {
                return json(['code' => 201, 'msg' => '用户名已存在', 'data' => null]);
            }

            $verified = $this->registrationVerification($result_key, [$user_name,$password,$reg_code,$sms_code]);
            if (!$verified && !is_sms_test_code_allowed($sms_code)) {
                $result = model('sms')->verification_code($user_name, $sms_code);
                if ($result['code'] != 200) {
                    return json(['code' => $result['code'], 'msg' => $result['msg'], 'data' => $result['data']]);
                }
            }
            $this->registrationVerification($result_key, [$user_name,$password,$reg_code,$sms_code], true);
            $result = model('user')->user_reg($user_name, $password, $reg_code, $result_key);
            $body = ['code' => $result['code'], 'msg' => $result['msg'], 'data' => $result['data']];
            if ((int) $result['code'] === 200) {
                $this->storeRegistrationResult($result_key, $body);
            }
            return json($body);
        } finally {
            redis_unlock($key_name);
        }
    }

    public function user_login_account()
    {

        $user_name = input('user_login', '');
        $password = input('password', '');
        $system = input('system',0);
        $last_login_device = input('last_login_device', '');
        $reslut = model('user')->user_login_account($user_name, $password, $last_login_device, $system);
        return json(['code' => $reslut['code'], 'msg' => $reslut['msg'], 'data' => $reslut['data']]);
    }

    //注册
    public function user_reg_account()
    {
        header("Access-Control-Allow-Origin: *"); // 允许任意域名发起的跨域请求
        $user_name = trim((string) input('user_login', ''));
        $password = (string) input('password', '');
        $reg_code = trim((string) input('invite_code', input('reg_code', '')));
        $repassword = (string) input('repassword', '');
        $captcha_code = trim((string) input('captcha_code', ''));
        $captcha_key = trim((string) input('captcha_key', ''));

        $key_name = "api:login:user_reg_account:" . strtolower($user_name);
        redis_lock_exit($key_name, 1, 30);
        $result_key = $this->registrationResultKey('account', $user_name, $captcha_key);
        $invite_lock_key = '';
        $invite_lock_acquired = false;
        try {
            $cached = $this->getRegistrationResult($result_key);
            if ($cached !== null) {
                return json($cached);
            }

            if ($reg_code !== '') {
                $invite_lock_key = "api:login:invite_code:" . md5($reg_code);
                if (!redis_lock_exits($invite_lock_key, 1, 30)) {
                    return json(['code' => 302, 'msg' => '邀请码正在使用，请稍后重试', 'data' => null]);
                }
                $invite_lock_acquired = true;
            }

            $validation = model('user')->validate_user_reg_account($user_name, $password, $reg_code, $repassword);
            if ((int) $validation['code'] !== 200) {
                return json(['code' => $validation['code'], 'msg' => $validation['msg'], 'data' => null]);
            }

            $verification = [$user_name,$password,$reg_code,$repassword,$captcha_code];
            if (!$this->registrationVerification($result_key,$verification)) {
                $result = model('api/ImgCaptcha')->check_captcha($captcha_code, $captcha_key);
                if ($result['code'] != 200) {
                    return json(['code' => $result['code'], 'msg' => '验证码错误', 'data' => $result['data']]);
                }
                $this->registrationVerification($result_key,$verification,true);
            }
            $result = model('user')->user_reg_account($user_name, $password, $reg_code, $repassword, $result_key);
            $body = ['code' => $result['code'], 'msg' => $result['msg'], 'data' => $result['data']];
            if ((int) $result['code'] === 200) {
                $this->storeRegistrationResult($result_key, $body);
            }
            return json($body);
        } finally {
            if ($invite_lock_acquired) {
                redis_unlock($invite_lock_key);
            }
            redis_unlock($key_name);
        }
    }

    // A consumed CAPTCHA can authorize retries of the exact verified request
    // only. Store an HMAC, never passwords, codes, invites or raw request data.
    private function registrationVerification($key, array $fields, $save=false)
    {
        if (!\app\imaccess\service\RegistrationService::enabled()) return false;
        $secret=(string)config('im_bridge.secret');
        if ($secret==='') return false;
        $fields[]=(string)request()->server('REMOTE_ADDR','');
        $hash=hash_hmac('sha256',json_encode($fields),$secret);
        $key.=':verified';
        if ($save) return connectionRedis()->setex($key,600,$hash);
        $old=connectionRedis()->get($key);
        return is_string($old) && hash_equals($old,$hash);
    }

    private function registrationResultKey($type, $account, $verificationId)
    {
        return 'api:login:registration_result:' . hash('sha256', $type . '|' . strtolower($account) . '|' . $verificationId);
    }

    private function getRegistrationResult($key)
    {
        $raw = connectionRedis()->get($key);
        if (!is_string($raw) || $raw === '') {
            return null;
        }
        $result = json_decode($raw, true);
        return is_array($result) ? $result : null;
    }

    private function storeRegistrationResult($key, array $result)
    {
        connectionRedis()->setex($key, 300, json_encode($result, JSON_UNESCAPED_UNICODE));
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
            return json(['code' => $result['code'], 'msg' => $result['msg'], 'data' => $result['data']]);
        }
        redis_lock_exit($key_name);
        $reslut = model('sms')->send_sms($mobile, $type);
        redis_unlock($key_name);
        return json(['code' => $reslut['code'], 'msg' => $reslut['msg'], 'data' => $reslut['data']]);
    }
    
    //获取图片验证码
    public function create_captcha()
    {
        $reslut = model('api/ImgCaptcha')->create_captcha();
        return json(['code' => $reslut['code'], 'msg' => $reslut['msg'], 'data' => $reslut['data']]);
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
        $data['bridge_strict_registration'] = \app\imaccess\service\RegistrationService::enabled();
        $data['bridge_username_rule'] = '8至22位；可使用纯数字，或以英文字母开头的字母、数字和下划线';
        $data['bridge_invite_required'] = false;
        $data['customer_link'] = $config['customer_link'];
        $data['customer_open_status'] = $config['customer_open_status'];
       return json(['code' => 200, 'msg' => 'success', 'data' => $data]);
    }
}
