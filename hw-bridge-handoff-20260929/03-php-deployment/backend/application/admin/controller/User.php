<?php

namespace app\admin\controller;

use think\Controller;
use think\facade\Request;

class User extends Common
{

    //获取用户列表
    public function get_user_list()
    {
        $uid = input('uid', 0);
        $user_name = input('user_name', '');
        $nick_name = input('nick_name', '');
        $order = input('order', 'uid');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('User')->get_user_list($uid, $user_name, $nick_name, $order, $sort, $page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }
    //获取用户资料
    public function get_user_info()
    {
        $uid = input('uid', 0);
        $reslut = model('User')->get_user_info($uid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
        
    }
    //修改用户资料
    public function edit_user_info()
    {
        $uid = input('uid', 0);
        $nick_name = input('nick_name', '');
        $sex = input('sex', 0);
        $login_status = input('login_status', 0);
        $data = model('User')->edit_user_info($uid, $nick_name, $sex, $login_status);

        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }
    //修改用户资料
    public function add_user_info()
    {
        $user_name = input('user_name', 0);
        $password = input('password', '');
        $data = model('User')->add_user_info($user_name, $password);

        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }
    
    //修改用户资金
    public function edit_user_money()
    {
        $uid = input('uid', 0);
        $change_value = input('change_value', 0);
        $secondary_password = input('secondary_password','');
        $remarks = input('remarks', '');
        //二级密码
        // $check_pass = model('admin/admin')->check_secondary_password($secondary_password);
        // if($check_pass['code'] == 201){
        //     // return ajaxReturn($check_pass['code'], $check_pass['msg'], $check_pass['data']);
        // }
        $reslut = model('User')->edit_user_money($uid, $change_value, $secondary_password, $remarks, $this->aid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    //修改用户密码
    public function edit_user_password()
    {
        $data = Request::only(['uid', 'password']);
        $reslut = model('User')->edit_user_password($data);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //修改用户VIP时间
    public function edit_user_vip_time()
    {
        $data = Request::only(['uid', 'day']);
        $reslut = model('User')->edit_user_vip_time($data);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //修改用户VIP时间
    public function edit_users_vip_time()
    {
        $data = Request::only(['uid', 'vip_end_time']);
        $reslut = model('User')->edit_users_vip_time($data);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //用户实名审核列表
    public function user_real_name_list()
    {
        $uid = input('uid', 0);
        $user_name = input('user_name', '');
        $nick_name = input('nick_name', '');
        $order = input('order', 'uid');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('User')->user_real_name_list($uid, $user_name, $nick_name, $order, $sort, $page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }
    //获取实名信息
    public function get_user_real_name(){
        $nid = input('nid',0);
        $reslut = model('User')->get_user_real_name($nid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //编辑用户实名信息
    public function edit_user_real_name(){
        $nid = input('nid',0);
        $status = input('status',0);
        $remarke = input('remarke','');
        $reslut = model('User')->edit_user_real_name($nid,$status,$remarke);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
      //送审处理
    public function add_no_check_user()
    {
        $filepath = input('filepath', '');
        $type = input('type', 1);
        $reslut = model('User')->add_no_check_user($filepath, $type);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //封禁文件
    public function add_banned_user_file()
    {
        $filepath = input('filepath', '');
        $type = input('type', 1);
        $reslut = model('User')->add_banned_user_file($filepath, $type);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
          //封禁文件
    public function add_banned_user_file_ip()
    {
        $filepath = input('filepath', '');
        $type = input('type', 1);
        $reslut = model('User')->add_banned_user_file_ip($filepath, $type);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
}
