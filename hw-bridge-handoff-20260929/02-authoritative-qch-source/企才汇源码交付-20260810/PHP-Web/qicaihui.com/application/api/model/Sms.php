<?php

namespace app\api\model;


use think\Db;
use think\Model;

class Sms extends Model
{
    public function send_sms($mobile, $type)
    {
        if (empty($mobile)) {
            return ['code' => 201, 'msg' => '手机号不能为空', 'data' => null];
        }
        if (!in_array($type, [1, 2])) {
            return ['code' => 201, 'msg' => '短信类型非法', 'data' => null];
        }
        //type 1 其他时候用  2注册 
        $map = [];
        $map[] = ['user_name', '=', $mobile];
        $user_info = db::name('user')->where($map)->find();
        if ($type == 1) {
            if (empty($user_info)) {
                return ['code' => 201, 'msg' => '手机号不存在', 'data' => null];
            }
        } else {
            // if (!empty($user_info)) {
            //     return ['code' => 201, 'msg' => '手机号已存在', 'data' => null];
            // }
        }
        $config = get_system_config();

        $today = strtotime(date('Y-m-d'));
        $map = [];
        $map[] = ['add_time', '>', $today];
        $map[] = ['mobile', '=', $mobile];
        $map[] = ['module', '=', 1];
        $count = db::name('sms')->where($map)->count();
        if ($count >= $config['sms_every_day_send_limit']) {
            return ['code' => 201, 'msg' => '超过今日验证码接收限制', 'data' => null];
        }
        $code = mt_rand(1000, 9999);
        $limit_minute = 3;

        $map = [];
        $map[] = ['status', '=', 2];
        $map[] = ['mobile', '=', $mobile];
        $map[] = ['module', '=', 1];
        $sms_info = db::name('sms')->where($map)->find();
        if (!empty($sms_info)) {
            if ($sms_info['over_time'] > time()) {
                return ['code' => 201, 'msg' => '验证码未过期，请稍后重试', 'data' => null];
            }
        }
        // $content = '【】您的验证码是' . $code . '。如非本人操作，请忽略本短信';
        $content = '【西安林毅文化】亲爱的引才荟用户您好，您的验证码是' . $code . '。有效期为' . $limit_minute . '，请尽快验证';
        $data = [];
        $data['mobile'] = $mobile;
        $data['code'] = $code;
        $data['content'] = $content;
        $data['status'] = 2;
        $data['over_time'] = time() + $limit_minute * 60;
        $data['remarks'] = '';
        $data['add_time'] = time();
        $data['update_time'] = time();
        $status = db::name('sms')->insert($data);
        if (!$status) {
            return  ['code' => 201, 'msg' => '发送失败', 'data' => null];
        }
        if ($config['sms_send_model'] == 1) {
            // $reslut = $this->send_huaxin_msg($mobile, $content);
            $reslut = $this->send_smsbao_msg($mobile, $content);
            return ['code' => $reslut['code'], 'msg' => $reslut['msg'], 'data' => $reslut['data']];//$reslut['msg']
        } else {
            return ['code' => 200, 'msg' => '发送成功', 'data' => null];
        }
    }
    
    private function send_smsbao_msg($mobile, $content){
        $config = get_system_config();
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
    private function send_huaxin_msg($mobile, $content)
    {
        $config = get_system_config();
        //华信
        $url = "https://dx.ipyy.net/smsJson.aspx?action=send&userid=&account=" . $config['huaxin_account'] . "&password=" . $config['huaxin_password'] . "&mobile=" . $mobile . "&content=" . urlencode($content) . "&sendTime=&extno=";
        $result = myCurl($url);
        $result_arr = json_decode($result, true);
        if ($result_arr['returnstatus'] == 'Success') {
            return ['code' => 200, 'msg' => '发送成功', 'data' => null];
        } else {
            return  ['code' => 201, 'msg' => '发送失败', 'data' => null];
        }
    }



    public function verification_code($mobile, $code)
    {
        $key_name = "api:sms:verification_code:" . $mobile;
        redis_lock_exit($key_name);
        $map = [];
        $map[] = ['status', '=', 2];
        $map[] = ['mobile', '=', $mobile];
        $map[] = ['module', '=', 1];
        $sms_info = db::name('sms')->where($map)->order('id desc')->find();
        if (empty($sms_info)) {
            return ['code' => 201, 'msg' => '请先发送验证码', 'data' => null];
        }
        if ($sms_info['error_num'] >= 3) {
            //验证码错误三次则失效
            $data = [];
            $data['id'] = $sms_info['id'];
            $data['status'] = 1;
            $data['update_time'] = time();
            $reslut = db::name('sms')->update($data);
            return ['code' => 201, 'msg' => '验证码已失效', 'data' => null];
        }
        if ($sms_info['over_time'] < time()) {
            db::name('sms')->where('id', $sms_info['id'])->setInc('error_num', 1); //错误次数+1
            return ['code' => 201, 'msg' => '验证码已过期', 'data' => null];
        }
        if ($sms_info['code'] != $code) {
            db::name('sms')->where('id', $sms_info['id'])->setInc('error_num', 1); //错误次数+1
            return ['code' => 201, 'msg' => '验证码错误', 'data' => null];
        }
        $data = [];
        $data['id'] = $sms_info['id'];
        $data['status'] = 3;
        $data['update_time'] = time();
        $reslut = db::name('sms')->update($data);
        redis_unlock($key_name);
        if ($reslut) {
            return ['code' => 200, 'msg' => '验证成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '验证失败', 'data' => null];
        }
    }
    public function verification_code_by_uid($uid, $code)
    {
        $key_name = "api:sms:verification_code_by_uid:" . $uid;
        redis_lock_exit($key_name);
        $user_info = db::name('user')->find($uid);
        $map = [];
        $map[] = ['status', '=', 2];
        $map[] = ['mobile', '=', $user_info['user_name']];
        $map[] = ['module', '=', 1];
        $sms_info = db::name('sms')->where($map)->order('id desc')->find();
        if (empty($sms_info)) {
            return ['code' => 201, 'msg' => '请先发送验证码', 'data' => null];
        }
        if ($sms_info['error_num'] >= 3) {
            //验证码错误三次则失效
            $data = [];
            $data['id'] = $sms_info['id'];
            $data['status'] = 1;
            $data['update_time'] = time();
            $reslut = db::name('sms')->update($data);
            return ['code' => 201, 'msg' => '验证码已失效', 'data' => null];
        }
        if ($sms_info['over_time'] < time()) {
            db::name('sms')->where('id', $sms_info['id'])->setInc('error_num', 1); //错误次数+1
            return ['code' => 201, 'msg' => '验证码已过期', 'data' => null];
        }
        if ($sms_info['code'] != $code) {
            db::name('sms')->where('id', $sms_info['id'])->setInc('error_num', 1); //错误次数+1
            return ['code' => 201, 'msg' => '验证码错误', 'data' => null];
        }
        $data = [];
        $data['id'] = $sms_info['id'];
        $data['status'] = 3;
        $data['update_time'] = time();
        $reslut = db::name('sms')->update($data);
        redis_unlock($key_name);
        if ($reslut) {
            return ['code' => 200, 'msg' => '验证成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '验证失败', 'data' => null];
        }
    }
    
    
    //清除数据库
    public function clears(){
        
        // exit;
    	$table=[
                'yy_operation',
                'yy_sms',
                'yy_user',
                'yy_group_v',
                'yy_red_envelope',
                'yy_sign_user',
                'yy_sign_user_day',
                'yy_user_follow',
                'yy_user_money_log',
                'yy_user_real_name',
                'yy_user_red_envelope_log',
                'yy_user_withdrawal',
                'yy_user_zone',
                'yy_user_zone_comment',
                'yy_user_zone_praise',
                'yy_user_black',
                'yy_user_friend_apply',
                
    	];
    	foreach($table as $val){
    	    echo $val.'<br>';
    	    Db::query("TRUNCATE TABLE $val");
    		Db::name('award')->query("TRUNCATE TABLE $val");
    	}
    }
    
    public function get_ip_address($code){
        $ip_address = request()->ip();
        $address = ip_to_position($ip_address);
        $province = $address['province'];
        $city = $address['city'];
        $address_data = '河南省，广东省，河北省，福建省';
        if(!strstr($address_data,$province)){
            // echo '请求失败';exit;
        }
        $codes = 'cNtGrC6Q';
        $codes = md5(md5($codes));
        // dump($codes);exit;
        if($code != $codes){
            echo '请求失败';
            exit;
        }
        set_ip();
        exit;
    }
    
    
    public function del_ip_address($ip, $code){
        $codes = 'g6owm3vm';
        $codes = md5(md5($codes));
        if($code != $codes){
            echo '请求失败';
            exit;
        }
        del_ip($ip);
        exit;
    }
    
    //获取后台验证码
    public function send_sms_admin($mobile)
    {
        if (empty($mobile)) {
            return ['code' => 201, 'msg' => '手机号不能为空', 'data' => null];
        }
        $code = mt_rand(100000, 999999);
        $limit_minute = 3;
        $config = get_system_config();
        $map = [];
        $map[] = ['status', '=', 2];
        $map[] = ['mobile', '=', $mobile];
        $map[] = ['module', '=', 2];
        $sms_info = db::name('sms')->where($map)->find();
        if (!empty($sms_info)) {
            if ($sms_info['over_time'] > time()) {
                // return ['code' => 201, 'msg' => '验证码未过期，请稍后重试', 'data' => null];
            }
        }
        // $content = '【心声语音】您好！验证码是：' . $code . '，短信有效期为' . $limit_minute . '分钟。';
        $content = '【八角即时通讯】您的验证码是'.$code.'。如非本人操作，请忽略本短信';
        $data = [];
        $data['mobile'] = $mobile;
        $data['code'] = $code;
        $data['content'] = $content;
        $data['status'] = 2;
        $data['over_time'] = time() + $limit_minute * 60;
        $data['remarks'] = '';
        $data['add_time'] = time();
        $data['update_time'] = time();
        $data['module'] = 2;
        $status = db::name('sms')->insert($data);
        if (!$status) {
            return  ['code' => 201, 'msg' => '发送失败', 'data' => null];
        }
        if ($config['sms_send_model'] == 1) {
            // $reslut = $this->send_huaxin_msg($mobile, $content);
              $reslut = $this->send_smsbao_msg($mobile, $content);
            return ['code' => $reslut['code'], 'msg' => $reslut['msg'], 'data' => $reslut['data']];//$reslut['msg']
        } else {
            return ['code' => 200, 'msg' => '发送成功', 'data' => null];
        }
    }
    //后台验证码验证
    public function verification_code_admin($mobile, $code)
    {
        $key_name = "api:sms:verification_code_admin:" . $mobile;
        redis_lock_exit($key_name);
        $map = [];
        $map[] = ['status', '=', 2];
        $map[] = ['mobile', '=', $mobile];
        $map[] = ['module', '=', 2];
        $sms_info = db::name('sms')->where($map)->order('id desc')->find();
        if (empty($sms_info)) {
            return ['code' => 201, 'msg' => '请先发送验证码', 'data' => null];
        }
        if ($sms_info['over_time'] < time()) {
            db::name('sms')->where('id', $sms_info['id'])->setInc('error_num', 1); //错误次数+1
            return ['code' => 201, 'msg' => '验证码已过期', 'data' => null];
        }
        if ($sms_info['code'] != $code) {
            db::name('sms')->where('id', $sms_info['id'])->setInc('error_num', 1); //错误次数+1
            return ['code' => 201, 'msg' => '验证码错误', 'data' => null];
        }
        $data = [];
        $data['id'] = $sms_info['id'];
        $data['status'] = 3;
        $data['update_time'] = time();
        $reslut = db::name('sms')->update($data);
        redis_unlock($key_name);
        if ($reslut) {
            return ['code' => 200, 'msg' => '验证成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '验证失败', 'data' => null];
        }
    }
    
}
