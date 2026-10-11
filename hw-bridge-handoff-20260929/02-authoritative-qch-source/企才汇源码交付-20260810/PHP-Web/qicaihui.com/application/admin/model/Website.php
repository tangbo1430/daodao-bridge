<?php

namespace app\admin\model;

use think\Db;
use think\Model;

class Website extends Model
{
    

    //获取 单页信息 列表
    public function page_list($order, $sort, $page = 1, $limit = 20)
    {
        $map = [];
        $map[] = ['is_delete', '=', 1];
        $map[] = ['system_type', '=', 2];
        $list = db::name('page')->where($map)
            ->order($order, $sort)->page($page, $limit)->select();

        $data = [];
        $data['count'] = db::name('page')->where($map)->count();
        $data['list'] = $list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    //添加 单页信息
    public function add_page($data)
    {
        $add_data = [];
        $add_data['title'] = $data['title'];
        $add_data['url'] = $data['url'];
        $add_data['content'] = $data['content'];
        $add_data['update_time'] = time();
        $add_data['add_time'] = time();
        $add_data['system_type'] = 2;
        $reslut = db::name('page')->insert($add_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '添加失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '添加成功', 'data' => null];
        }
    }

    //编辑  单页信息
    public function edit_page($data)
    {
        if (empty($data['aid'])) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $info = db::name('page')->find($data['aid']);
        if (empty($info)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $update_data = [];
        $update_data['title'] = $data['title'];
        $update_data['url'] = $data['url'];
        $update_data['content'] = html_entity_decode($data['content']);
        $update_data['update_time'] = time();
        $reslut = db::name('page')->where(['aid' => $data['aid']])->update($update_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        }
    }

    //获取 单页信息 详情
    public function get_page($aid)
    {
        if (empty($aid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $info = db::name('page')->where(['aid' => $aid])->find();
        return ['code' => 200, 'msg' => '获取成功', 'data' => $info];
    }

    //删除 单页信息
    public function page_del($aid)
    {
        if (empty($aid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        if ($aid <= 10) {
            return ['code' => 201, 'msg' => '前10个单页禁止删除', 'data' => null];
        }
        $data_del = db::name('page')->where(['aid' => $aid])->update(['is_delete' => 2, 'delete_time' => time()]);
        if ($data_del) {
            return ['code' => 200, 'msg' => '删除成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '删除失败', 'data' => null];
        }
    }
}
