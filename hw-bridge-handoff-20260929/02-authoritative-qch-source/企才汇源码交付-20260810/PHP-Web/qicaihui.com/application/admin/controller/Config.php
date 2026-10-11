<?php

namespace app\admin\controller;

use think\Controller;

class Config extends Common
{
   
   
    //获取系统配置
    public function config_list()
    {
        $cid = input('cid', 0);
        $key_name = input('key_name', 0);
        $order = input('order', 'cid');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('config')->config_list($cid,$key_name,$order, $sort, $page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }

    //获取 配置 详情
    public function config_info()
    {
        $cid = input('cid', 0);
        $data = model('config')->config_info($cid);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }
    //编辑 配置
    public function edit_config()
    {
        $data = input('post.');
        $data = model('config')->edit_config($data);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

    //添加 配置
    public function add_config()
    {
        $data = input('post.');
        $data = model('config')->add_config($data);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

    //删除 配置
    public function del_config()
    {
        $cid = input('cid', 0);
        $data = model('config')->del_config($cid);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

    
    
}