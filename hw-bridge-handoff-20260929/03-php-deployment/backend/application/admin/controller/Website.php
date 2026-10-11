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

    // APP协议列表
    public function protocol_list()
    {
        $data = model('Website')->protocol_list();
        return ajaxReturn($data['code'], $data['msg'], $data['data']);
    }

    // APP协议详情
    public function get_protocol()
    {
        $type = input('type', '');
        $data = model('Website')->get_protocol($type);
        return ajaxReturn($data['code'], $data['msg'], $data['data']);
    }

    // APP协议保存
    public function save_protocol()
    {
        $data = input('post.');
        $result = model('Website')->save_protocol($data);
        return ajaxReturn($result['code'], $result['msg'], $result['data']);
    }
}
