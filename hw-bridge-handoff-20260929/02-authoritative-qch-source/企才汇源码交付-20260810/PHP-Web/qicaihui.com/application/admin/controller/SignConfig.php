<?php


namespace app\admin\controller;


class SignConfig extends Common
{
//获取列表
    public function get_list()
    {
        $page = input('page', 1);
        $page_limit = input('limit', 15);
        return model('admin/SignConfig')->get_list($page, $page_limit);
    }
    //添加
    public function add()
    {
        $data = input();
        $result = model('admin/SignConfig')->add($data);
        ajaxReturn($result['code'], $result['msg'], $result['data']);
    }
    //获取信息
    public function get_info()
    {
        $id = input('id', 0);
        $result = model('admin/SignConfig')->get_info($id);
        ajaxReturn($result['code'], $result['msg'], $result['data']);
    }
    //编辑
    public function edit()
    {
        $data = input();
        $result = model('admin/SignConfig')->edit($data);
        ajaxReturn($result['code'], $result['msg'], $result['data']);
    }
    //删除
    public function del()
    {
        $id = input('id', 0);
        $result = model('admin/SignConfig')->del($id);
        ajaxReturn($result['code'], $result['msg'], $result['data']);
    }

    public function get_user_sign_list()
    {
        $page = input('page', 1);
        $page_limit = input('limit', 15);
        return model('admin/SignConfig')->get_user_sign_list($page, $page_limit);
    }
}