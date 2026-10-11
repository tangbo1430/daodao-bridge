<?php

namespace app\admin\controller;

use think\Controller;

class Admin extends Common
{

    //获取管理员列表
    public function get_admin_list()
    {
        $user_name = input('user_name', '');

        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('admin')->get_admin_list($user_name, $page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }
    //获取 管理员 详情
    public function admin_info()
    {
        $login_token = input('login_token', 0);
        $reslut = model('Admin')->admin_info($login_token);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    //获取 管理员 详情
    public function add_admin()
    {
        $user_name = input('user_name', '');
        $password = input('password', '');
        $re_password = input('re_password', '');
        $reslut = model('Admin')->add_admin($user_name, $password, $re_password);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }


    //修改管理员密码
    public function edit_admin_password()
    {

        $aid = input('aid', 0);
        $old_password = input('old_password', '');
        $password = input('new_password', '');
        $re_password = input('re_password', '');
        $phone = input('phone', '');
        $reslut = model('Admin')->edit_admin_password($aid, $old_password, $password, $re_password, $phone);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //修改管理员密码
    public function edit_admin_auth()
    {
        $aid = input('aid', 0);
        $auth = input('auth', '');
        $reslut = model('Admin')->edit_admin_auth($aid, $auth);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }

    //获取管理员详情
    public function get_admin_info()
    {
        $aid = input('aid', 0);
        $reslut = model('Admin')->get_admin_info($aid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //获取管理员详情
    public function delete_admin()
    {
        $aid = input('aid', 0);
        $reslut = model('Admin')->delete_admin($aid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    public function get_all_system_menu_list()
    {
        $aid = input('aid', 0);
        $reslut = model('SystemMenu')->get_all_system_menu_list($aid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    
    //获取管理员日志
    public function get_admin_log_list(){
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Admin')->get_admin_log_list($page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }
    
    //退出登录
    public function quit_admin_login(){
        $aid = $this->aid;
        $reslut = model('Admin')->quit_admin_login($aid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //清除登录token
    public function clear_admin_token()
    {
        $super_aid = $this->aid;
        $aid = input('aid', 0);
        $reslut = model('Admin')->clear_admin_token($super_aid, $aid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    public function export_operate_log()
    {
        model('Admin')->export_operate_log();
    }
}
