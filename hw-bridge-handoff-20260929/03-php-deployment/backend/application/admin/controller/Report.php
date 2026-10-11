<?php

namespace app\admin\controller;

use think\Controller;

class Report extends Common
{
   
   
    //获取举报列表
    public function report_list()
    {
        $uid = input('uid', 0);
        $to_uid = input('to_uid', 0);
        $nick_name = input('nick_name', '');
        $to_nick_name = input('to_nickname', '');

        $order = input('order', 'rid');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Report')->report_list($uid, $to_uid,$nick_name, $to_nick_name, $order, $sort, $page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }
    //获取举报详情
    public function report_info()
    {
        $rid = input('rid', 0);
        $data = model('Report')->report_info($rid);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }

    }
    //处理举报信息
    public function edit_user_info()
    {
        $rid = input('rid', 0);
        $status = input('status', '');
        $data = model('Report')->edit_user_info($rid,$status);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

    //删除举报信息
    public function del_Report()
    {
        $rid = input('rid', 0);
        $data = model('Report')->del_Report($rid);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

     //举报类型 列表
    public function report_type_list()
    {
        $order = input('order', 'id');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Report')->report_type_list($order,$sort,$page,$limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }

     //获取举报类型详情
    public function report_type_info()
    {
        $id = input('id', 0);
        $data = model('Report')->report_type_info($id);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }

    }

    //添加 举报类型信息
    public function add_report_type()
    {
        $type_name = input('type_name', '');
        $data = model('Report')->add_report_type($type_name);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

    //修改 举报类型信息
    public function edit_report_type()
    {
        $id = input('id', 0);
        $type_name = input('type_name', '');
        $data = model('Report')->edit_report_type($id,$type_name);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

     //删除举报类型
    public function del_report_type()
    {
        $id = input('id', 0);
        $data = model('Report')->del_report_type($id);
        if ($data['code'] == 0) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }
}