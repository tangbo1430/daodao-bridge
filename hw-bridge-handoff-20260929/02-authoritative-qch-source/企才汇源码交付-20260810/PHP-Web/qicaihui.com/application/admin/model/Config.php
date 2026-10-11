<?php

namespace app\admin\model;

use think\Model;
use think\Db;

class Config extends Model
{
    //获取配置参数
    public function get_system_config($key = "")
    {
        $config = Db::name('config')->cache(10)->order('sort desc')->select();
        $config_data = [];
        foreach ($config as $k => $v) {
            $config_data[$v['key_title']] = $v['key_value'];
        }
        if (empty($key)) {
            return $config_data;
        } else {
            if (!isset($config_data[$key]) || $config_data[$key] == "") {
                return ajaxReturn(201, '配置参数:' . $key . '不存在', null);
            }
            return $config_data[$key];
        }
    }
    
    //无缓存获取配置参数
    public function get_uncache_system_config($key = "")
    {
        $config = Db::name('config')->order('sort desc')->select();
        $config_data = [];
        foreach ($config as $k => $v) {
            $config_data[$v['key_title']] = $v['key_value'];
        }
        if (empty($key)) {
            return $config_data;
        } else {
            if (!isset($config_data[$key]) || $config_data[$key] == "") {
                return ajaxReturn(201, '配置参数:' . $key . '不存在', null);
            }
            return $config_data[$key];
        }
    }

    //配置列表
    public function config_list($cid, $key_name, $order, $sort, $page = 1, $limit = 20)
    {
        $map = [];
        if (!empty($cid)) {
            $map[] = ['cid', '=', $cid];
        }
        if (!empty($key_name)) {
            $map[] = ['key_name|key_value', 'like', '%' . $key_name . '%'];
        }
        $map[] = ['is_delete', '=', 1];
        $list = db::name('config')->where($map)->order($order, $sort)->page($page, $limit)->select();

        $data = [];
        $data['count'] = db::name('config')->where($map)->count();
        $data['list'] = $list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }
    //编辑配置
    public function edit_config($data)
    {
        if (empty($data)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $info = db::name('config')->find($data['cid']);
        if (empty($info)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $update_data = [];
        $update_data['key_title'] = $data['key_title'];
        $update_data['key_name'] = $data['key_name'];
        $update_data['key_value'] = $data['key_value'];
        $update_data['key_desc'] = $data['key_desc'];
        $update_data['update_time'] = time();
        $reslut = db::name('config')->where(['cid' => $data['cid']])->update($update_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        }
    }
    //添加配置
    public function add_config($data)
    {
        $add_data = [];
        $add_data['key_title'] = $data['key_title'];
        $add_data['key_name'] = $data['key_name'];
        $add_data['key_value'] = $data['key_value'];
        $add_data['key_desc'] = $data['key_desc'];
        $add_data['update_time'] = time();
        $reslut = db::name('config')->insert($add_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '添加失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '添加成功', 'data' => null];
        }
    }

    //获取配置信息
    public function config_info($cid)
    {
        if (empty($cid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $gift_info = db::name('config')->where(['cid' => $cid])->find();
        return ['code' => 200, 'msg' => '获取成功', 'data' => $gift_info];
    }

    //删除配置
    public function del_config($cid)
    {
        return ['code' => 201, 'msg' => '操作已禁止', 'data' => null];
        if (empty($cid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $reslut = db::name('config')->where(['cid' => $cid])->delete();
        if ($reslut) {
            return ['code' => 200, 'msg' => '删除成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '删除失败', 'data' => null];
        }
    }
}
