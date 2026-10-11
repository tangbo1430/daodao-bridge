<?php
namespace app\admin\controller;
use think\Controller;
// 网站管理
class Website extends Common
{



    //单页信息 列表
    public function page_list()
    {
        $order = input('order', 'aid');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Website')->page_list($order,$sort,$page,$limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }

    //添加 单页信息
    public function add_page()
    {
        $data = input('post.');
        $data = model('Website')->add_page($data);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

    //获取 单页信息 详情
    public function get_page()
    {
        $aid = input('aid', 0);
        $data = model('Website')->get_page($aid);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

     //修改 单页信息
    public function edit_page()
    {
        $data = input('post.');
        $data = model('Website')->edit_page($data);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

    //删除 单页信息
    public function page_del()
    {
        $aid = input('aid', 0);
        $data = model('Website')->page_del($aid);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }
}
