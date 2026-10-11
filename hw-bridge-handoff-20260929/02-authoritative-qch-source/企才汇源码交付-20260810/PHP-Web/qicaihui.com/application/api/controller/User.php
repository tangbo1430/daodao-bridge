<?php

namespace app\api\controller;


class User extends Common
{


    //获取用户详情
    public function get_user_info()
    {
        $reslut = model('user')->get_user_info($this->uid);
        // dump($reslut);die;
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //修改用户 信息
    public function modify_user_info()
    {
        $key_name = "api:user:follow_user:" . $this->uid;
        redis_lock_exit($key_name);
        $nick_name = input('nick_name');
        $birthday = input('birthday');
        $sex = input('sex', 1);
        $head_pic = input('head_pic', '');
        $city = input('city', 0);
        $autograph = input('autograph', '');
        $hobby = input('hobby','');
        $reslut = model('user')->modify_user_info($this->uid, $nick_name, $birthday, $sex, $head_pic, $city, $autograph, $hobby);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    //根据日志获取用户星座
    public function get_user_constellation()
    {
        $birthday = input('birthday');
        $reslut = model('user')->get_user_constellation($birthday);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    
    //退出登录
    public function user_log_out(){
        $reslut = model('user')->log_out($this->uid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //提现
    public function user_withdrawal()
    {
        $address = input('address','');
        $money = input('money', 0);
        $code = input('code', 0);
        // if($code != 9999){
        //     $reslut = model('sms')->verification_code_by_uid($this->uid, $code);
        //     if ($reslut['code'] == 201) {
        //         return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
        //     }
        // }
        $trade_password = input('trade_password', '');
        $key_name = "api:user:user_withdrawal:" . $this->uid;
        redis_lock_exit($key_name,3);
        $reslut = model('user')->user_withdrawal($this->uid,$address, $money, $trade_password);
        // if ($reslut['code'] != 200) {
        //     ajaxReturn($reslut['code'], $reslut['msg']);
        // }
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //实名认证
    public function real_name_authentication(){
        $real_name = input('real_name','');
        $card_id = input('card_id','');
        $identity1 = input('identity1','');
        $identity2 = input('identity2','');
        $reslut = model('user')->real_name_authentication($this->uid,$real_name, $card_id,$identity1,$identity2);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    
    //获取提现记录
    public function get_user_money_list()
    {
        $page = input('page', 1);
        $page_limit = input('page_limit', 10);
        $reslut = model('user')->get_user_money_list($this->uid, $page, $page_limit);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //修改密码
    public function modify_password()
    {
        $password = input('password', '');
        $sms_code = input('sms_code', ''); //短信验证码
        if($sms_code != 9999){
            $reslut = model('sms')->verification_code_by_uid($this->uid, $sms_code);
            if ($reslut['code'] == 201) {
                return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
            }
        }
        $user_name = input('mobile', '');
        $reslut = model('user')->modify_password($this->uid, $user_name, $password);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //修改密码
    public function modify_password_account()
    {
        $password = input('password', '');
        $old_password = input('old_password', ''); //短信验证码
        $reslut = model('user')->modify_password_account($this->uid, $password, $old_password);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //修改交易密码
    public function modify_trade_password()
    {
        $trade_password = input('trade_password');
        $sms_code = input('sms_code', ''); //短信验证码
        if($sms_code != 9999) {
            $reslut = model('sms')->verification_code_by_uid($this->uid, $sms_code);
            if ($reslut['code'] == 201) {
                return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
            }
        }
        
        $reslut = model('user')->modify_trade_password($this->uid, $trade_password);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //获取提现记录
    public function get_user_withdrawal_list()
    {
        $page = input('page', 1);
        $page_limit = input('page_limit', 10);
        $reslut = model('user')->get_user_withdrawal_list($this->uid, $page, $page_limit);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //注销账号
    public function cancel_account(){
        $reslut  = model('User')->cancel_account($this->uid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //实名认证
    public function add_system_account(){
        $reslut = model('user')->add_system_account($this->uid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
}
