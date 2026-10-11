<?php

namespace app\admin\controller;

use think\Controller;
use think\Db;

class Group extends Common
{
   
   
 public function get_group_list()
{
    $uid = input('uid', 0);
    $group_name = input('group_name', '');
    $order = input('order', 'id');
    $sort = input('sort', 'desc');
    $page = input('page', 1);
    $limit = input('limit', 20);
    $reslut = model('Group')->get_group_list($uid, $group_name, $order, $sort, $page, $limit);
    
    // 新增：为每个群组添加成员信息
    if (!empty($reslut['data']['list'])) {
        foreach ($reslut['data']['list'] as $key => &$group) {
            // 使用现有的Group模型方法获取群成员
            $memberResult = model('Group')->get_group_user_list(0, $group['id'], '', 'uid', 'desc', 1, 999);
            
            $group['members'] = $memberResult['data']['list'] ?: [];
            $group['member_count'] = $memberResult['data']['count'] ?: 0;
        }
    }
    
    $data = [];
    $data['code'] = 0;
    $data['msg'] = '获取成功';
    $data['count'] = $reslut['data']['count'];
    $data['data'] = $reslut['data']['list'];
    return json($data);
}
    //获取举报详情
    public function add_group()
    {
        $group_name = input('group_name', '');
        $uid = input('uid', 0);
        $data = model('Group')->add_group($group_name, $uid);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }

    }
    //获取举报类型详情
    public function get_group_info()
    {
        $id = input('id', 0);
        $data = model('Group')->get_group_info($id);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }

    }
    //处理举报信息
    public function edit_group()
    {
        $id = input('id', 0);
        $group_name = input('group_name', '');
        $uid = input('uid', 0);
        $data = model('Group')->edit_group($id, $group_name, $uid);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

    //删除举报信息
    public function del_group()
    {
        $rid = input('id', 0);
        $data = model('Group')->del_group($rid);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }
    //获取用户列表
    public function get_user_list()
    {
        $uid = input('uid', 0);
        
        $group_id = input('group_id', '');
        $user_name = input('user_name', '');

        $order = input('order', 'uid');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Group')->get_user_list($uid, $group_id, $user_name, $order, $sort, $page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }
    //群成员 列表
    public function get_group_user_list()
    {
        
        $order = input('order', 'uid');
        $sort = input('sort', 'desc');
        $group_id = input('group_id', 0);
        $uid = input('uid', 0);
        $user_name = input('user_name', 0);
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Group')->get_group_user_list($uid, $group_id, $user_name, $order,$sort,$page,$limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }
    //添加 群成员
    public function add_group_user()
    {
        $group_id = input('group_id', 0);
        $uid_arr = input('info', '');
        $data = model('Group')->add_group_user($group_id, $uid_arr);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }
    
    //群文件 列表
    public function get_group_file_list()
    {
     
        $group_id = input('group_id', 0);
        $file_name = input('file_name', '');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Group')->get_group_file_list( $group_id, $file_name, $page,$limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }
    //添加 群文件
    public function add_group_file()
    {
        $group_id = input('group_id', 0);
        $file_name = input('file_name', '');
        $file_path = input('file_url', '');
        $data = Db::name('group_file')->insert([
            'group_id' => $group_id,
            'file_name' => $file_name,
            'file_url' => $file_path,
            'createtime' => time(),
            'im_g_id' => $group_id,
        ]);
        if ($data['code'] == 201) {
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
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

     //删除举报类型
    public function del_group_user()
    {
        $uid = input('id', 0);
        $group_id = input('group_id', 0);
        $data = model('Report')->del_group_user($uid, $group_id);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }
     //删除群文件
    public function del_group_file()
    {
        $id = input('id', 0);
        $data = Db::name('group_file')->delete($id);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }
}