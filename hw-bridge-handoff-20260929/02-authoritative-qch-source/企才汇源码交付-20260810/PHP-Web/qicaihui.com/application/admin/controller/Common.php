<?php

namespace app\admin\controller;

use think\Controller;
use think\Db;

class Common extends Controller
{
    public $aid;
    public function initialize()
    {
        header('Access-Control-Allow-Origin: *');

        // ajaxReturn(301, '系统维护');
        $login_token = input('login_token', 0);
        $reslut = model('Admin')->check_login_token($login_token);
        if ($reslut['code'] == 201) {
            return ajaxReturn(301, $reslut['msg'], $reslut['data']);
        } else {
            $this->aid = $reslut['data'];
        }
        
        $check_ip = $this->check_ip();
        if($check_ip['code'] == 201){
            db::name('admin')->where('aid', $this->aid)->update(['login_token' => '']);
            return ajaxReturn(301, $check_ip['msg'], $check_ip['data']);
        }
        
        $header = request()->header();
        if(!isset($header['referer'])){
            return ajaxReturn(301, '登录失效', $reslut['data']);
        }
    
        add_operation(1, $this->aid); //用户行为日志
    }
    public function check_login_status()
    {
        $user_name = Db::name("admin")->where(array("aid" => $this->aid))->value("user_name");
        $data['data'] = $user_name;
        $check_ip = $this->check_ip();
        if($check_ip['code'] == 201){
            db::name('admin')->where('aid', $this->aid)->update(['login_token' => '']);
            return ajaxReturn(301, $check_ip['msg'], $check_ip['data']);
        }
        ajaxReturn(1, '登录成功', $data);
    }
 public function get_menu_list()
{
    $data = model('SystemMenu')->getSystemInit($this->aid);

    // 根据返回的数据字段风格调整
    if (isset($data['logoInfo'])) {
        $data['logoInfo']['title'] = '后台管理A';
    } elseif (isset($data['logo_info'])) {
        $data['logo_info']['title'] = '后台管理A';
    }

    return json($data);
}
    
    //限制ip
    public function check_ip(){
        $check_ip = request()->ip();
        if($check_ip){
            // $check_ip = explode('.',$check_ip);
            // $check_ip = $check_ip[0].'.'.$check_ip[1];
            // $ip = array(
            //     '61.52',
            //     '219.157',
            //     '113.137',
            //     '117.37',
            //     '123.149',
            //     '125.41',
            //     '115.60',
            //     '113.141',
            //     '221.15',
            //     '36.46',
            //     '113.137',
            //     '123.53',
            //     '1.198',
            //     '111.18',
            //     '1.193',
            //     '119.137',
            //     '111.19'
            //     );
            // if(!in_array($check_ip,$ip)){
            //     return ['code' => 201, 'msg' => 'ip不匹配无法登陆！', 'data'=>null];
            // }
            $check_ips = has_ip();
            if(!$check_ips){
                // return ['code' => 201, 'msg' => 'ip不匹配无法登陆！', 'data'=>null];
            }
        }else{
            return ['code' => 201, 'msg' => 'ip不匹配无法登陆！', 'data'=>null];
        }
        return ['code' => 200, 'msg' => 'ip可登陆', 'data'=>null];
    }
    
    
}
