<?php

namespace app\api\controller;

use think\App;
use think\Controller;


class Common extends Controller
{
    public $uid;

    protected function initialize()
    {
        header('Access-Control-Allow-Origin: *');
        // ajaxReturn(301, '系统维护');
        $config = get_uncache_system_config();
        if($config['is_maintenance'] == 2){
            return ajaxReturn(301, '正在维护');
        }
        $login_token = input('login_token', 0);
        $reslut = model('User')->check_login_token($login_token);
        if ($reslut['code'] == 201) {
            return ajaxReturn(301, '登录失效');
        } elseif ($reslut['code'] == 202) {
            return ajaxReturn(301, $reslut['msg']);
        } else {
            // $this->uid = 1;
            $this->uid = $reslut['data'];
        }
        // $reslut = validate_param_sign(input(), $login_token); //验证签名是否合法
        // if ($reslut['code'] == 201) {
        //     return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
        // }
        // add_operation(2, $this->uid); //用户行为日志
    }


    public function check_login_status()
    {
        ajaxReturn(1, '登录成功', '');
    }
}
