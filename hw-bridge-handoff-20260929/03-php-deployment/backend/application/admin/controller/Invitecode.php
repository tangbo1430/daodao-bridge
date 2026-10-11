<?php

namespace app\admin\controller;

use think\Controller;
use think\Db;

class Invitecode extends Common
{
   
   
    //获取举报列表
    public function get_group_list()
    {

        
        $name = input('name', '');
        

        $order = input('order', 'id');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Invitecode')->get_invitecode_list($name, $order, $sort, $page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        return json($data);
    }
    ////生成8位随机数
    public function get_rand(){
        $rand = mt_rand(10000000, 99999999);
        //判断是否重复
        $count = db::name('invitecode')->where('name', $rand)->count();
        if($count > 0){
            $rand = $this->get_rand();
        }
        return $rand;
    }
    //获取举报详情
    public function add_group()
    {
        $params = $this->request->param();
        $num = isset($params['num']) ? (int)$params['num'] : 1;
        $data = [];
        for ($i = 0; $i < $num; $i++) {
            //生成8位随机数
           
            $data[] = ['name' => $this->get_rand(),'system_type'=>config('app.system_type'),'createtime' => time()];
        }
    
        model('Invitecode')->saveAll($data);
       
        return ajaxReturn(200, '生成成功', $data);
    }

    //处理举报信息
    public function edit_group()
    {
        $id = input('id', 0);
        $invite_num = input('invite_num', '');
        $data = db::name('invitecode')->where('id', $id)->update(['invite_num' => $invite_num]);
        if ($data) {
            return ajaxReturn(200, '修改成功', $data);
        } else {
            return ajaxReturn(201, '修改失败', $data);
        }
    }

    //删除举报信息
    public function del_group()
    {
        $rid = input('id', 0);
        $data = db::name('invitecode')->where('id', $rid)->delete();
        if ($data) {
            return ajaxReturn(200, '删除成功', $data);
        } else {
            return ajaxReturn(201, '删除失败', $data);
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
    //举报类型 列表
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
    //添加 举报类型信息
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
}