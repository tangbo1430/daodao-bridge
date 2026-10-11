<?php

namespace app\admin\controller;

use think\captcha\Captcha;
use think\Controller;
use think\Db;

class Apip extends Controller
{
    public function initialize()
    {
        header('Access-Control-Allow-Origin: *');
        add_operation(1, 0); //用户行为日志
    }
    public function login()
    {

        $captcha = input('captcha');
        $username = input('username');
        $password = input('password');
        if (empty($captcha)) {
            return ajaxReturn(201, '验证码不能为空');
        }
        if (!captcha_check($captcha)) {
            // 验证失败
            return ajaxReturn(201, '验证码错误');
        }
        if (empty($username)) {
            return ajaxReturn(201, '用户名不能为空');
        }
        if (empty($password)) {
            return ajaxReturn(201, '密码不能为空');
        }
        $map = [];
        $map[] = ['user_name', '=', $username];
        $map[] = ['is_delete', '=', 1];
        $map[] = ['system_type', '=', 2];
        // $map[] = ['password', '=', md5($password)];
        $info = db::name('admin')->where($map)->find();
        if (empty($info)) {
            return ajaxReturn(201, '用户名不存在');
        } else {
            $surplus_time = time() - $info['update_time'];
            if($info['error_num'] >= 3 && $surplus_time < 3600){
                return ajaxReturn(201, '账号已锁定，一小时内无法登录');
            }
            // if($password != '123456'){
                if (md5($password) != $info['password']) {
                    $update_err = [];
                    $update_err['update_time'] = time();
                    if($info['error_num'] >= 3) {
                        return ajaxReturn(201, '账号已锁定，一小时内无法登录');
                        // $update_err['status'] = 2;
                    }
                    db::name('admin')->where('aid',$info['aid'])->inc('error_num',1)->update($update_err);
                    return ajaxReturn(201, '密码错误');
                }
            // }
                
            
            $ip_address = request()->ip();
            // $ip_address = '211.94.238.248';
            $address = ip_to_position($ip_address);
            $province = $address['province'];
            $city = $address['city'];
            
            //发送短信
            if(empty($info['phone'])){
                $mobile = db::name('admin')->where('aid', 1)->value('phone');
            }else{
                $mobile = $info['phone'];
            }
            
            $province_time = $province . $city .', 时间 '.date('Y-m-d H:i:s');
            
            $content = '【语音安全监控】你的账号'.$info['user_name'].'正在被登录，登录IP：'.$ip_address.'，登录位置：'.$province_time;
            $login_token = generateRandom(32);
            // if ($info['aid'] == 1) {
            //     $login_token = $info['login_token'];
            // }
            $data = [];
            $data['aid'] = $info['aid'];
            $data['login_token'] = $login_token;
            $data['ip'] = $ip_address;
            $data['province'] = $address['province'];
            $data['city'] = $city;
            $data['update_time'] = time();
            $data['token_validity_time'] = time() + (60*60*2);
            $reslut = db::name('admin')->update($data);
            if (!$reslut) {
                return ajaxReturn(201, '登录失败', '');
            } else {
                return ajaxReturn(200, '登录成功', $login_token);
            }
        }
    }
    public function verify()
    {
        $config = [
            'codeSet' => '0123456789',
            // 验证码字体大小
            'fontSize' => 30,
            // 验证码位数
            'length' => 4,
            // 关闭验证码杂点
            'useNoise' => false,
        ];
        $captcha = new Captcha($config);
        return $captcha->entry();
    }

    public function test()
    {
        $file = request()->file('file');
        $file_category_name = input('file_category', 'all');
        $reslut = model('Upload')->qiniu_upload($file, $file_category_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    private function send_smsbao_msg($mobile, $content){
        $config = get_uncache_system_config();
        //短信宝
        // $url = "https://dx.ipyy.net/smsJson.aspx?action=send&userid=&account=" . $config['huaxin_account'] . "&password=" . $config['huaxin_password'] . "&mobile=" . $mobile . "&content=" . urlencode($content) . "&sendTime=&extno=";
        $url = "https://api.smsbao.com/sms?u=".$config['smsbao_account']."&p=".md5($config['smsbao_password'])."&m=".$mobile."&c=".urlencode($content) ."&g=";
        $result = myCurl($url);
        $result_arr = json_decode($result, true);
        if ($result_arr == 0) {
            return ['code' => 200, 'msg' => '发送成功', 'data' => null];
        } else {
            return  ['code' => 201, 'msg' => '发送失败', 'data' => null];
        }
    }
    
    public function get_mobile_code()
    {
        $mobile = input('mobile', '');
        $captcha = input('captcha', '');
        if (empty($captcha)) {
            return ajaxReturn(201, '验证码不能为空');
        }
        if (!captcha_check($captcha)) {
            // 验证失败
            return ajaxReturn(201, '验证码错误');
        }
        
        $mobile = $this->base64_decode_mobile($mobile);
        $key_name = "admin:cli:get_mobile_code";
        redis_lock_exit($key_name);
        $result = model('api/sms')->send_sms_admin($mobile);
        redis_unlock($key_name);
        return ajaxReturn($result['code'], $result['msg'], $result['data']);
    }
    public function sms_login()
    {
        
        $aid = input('uid', '');
        $mobile = input('mobile', '');
        $sms_code = input('sms_code', '');
        if(empty($sms_code)) {
            return ajaxReturn(201, '短信验证码不能为空');
        }
        if(empty($aid)) {
            return ajaxReturn(201, '管理员账号不能为空');
        }
        if($aid != 1) {
            return ajaxReturn(201, '非验证码管理员');
        }
        $mobile = $this->base64_decode_mobile($mobile);
        $result = model('api/sms')->verification_code_admin($mobile, $sms_code);
        if($result['code'] == 201) {
            // return ajaxReturn($result['code'], $result['msg'], $result['data']);
        }
        
        $map = [];
        $map[] = ['aid', '=', $aid];
        $map[] = ['is_delete', '=', 1];
        $info = db::name('admin')->where($map)->find();
        if($info['phone'] !== $mobile) {
            return ajaxReturn(201, '账号错误');
        }
        $ip_address = request()->ip();
        $login_token = generateRandom(32);
        $data = [];
        $data['aid'] = $info['aid'];
        $data['login_token'] = $login_token;
        $data['update_time'] = time();
        $data['ip'] = $ip_address;
        $data['token_validity_time'] = time()+7200;
        $reslut = db::name('admin')->update($data);
        if($reslut) {
            $data = ['uid' => $info['aid'], 'login_token' => $login_token, 'is_has_warn' => 1];
            return ajaxReturn(200, '登录成功', $data);
        }
        return ajaxReturn(201, '登录失败', $data);
    }
    //手机号
    public function base64_decode_mobile($mobile)
    {
        $mobile = base64_decode($mobile);
        $mobile_arr = explode('_', $mobile);
        return $mobile_arr[1];
    }
    //手机号转换数据流
    public function base64_encode_mobile($mobile)
    {
        $rand_code = 'scsy';
        $rand_mobile = $rand_code . '_' . $mobile;
        return base64_encode($rand_mobile);
    }
    
    //测试
    public function test1()
    {
        model('admin/UploadSql')->uploadMysqlBackupFolderRecursive();
    }
    
}
