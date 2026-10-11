<?php

namespace app\admin\model;

use think\Db;
use think\Model;

class Report extends Model
{
    //获取举报列表
    public function report_list($uid, $to_uid, $nick_name, $to_nick_name, $order, $sort, $page = 1, $limit = 20)
    {
        $map = [];
        if (!empty($uid)) {
            $map[] = ['a.uid', '=', $uid];
        }
        if (!empty($to_uid)) {
            $map[] = ['a.to_uid', '=', $to_uid];
        }
        if (!empty($nick_name)) {
            $map[] = ['b.nick_name', 'like', '%' . $nick_name . '%'];
        }
        $list = db::name('user_report')
            ->alias('a')->join('yy_user b', 'a.uid = b.uid')
            ->field('a.*,b.nick_name')->where($map)->order($order, $sort)->page($page, $limit)->select();
        foreach ($list as $k => $v) {
            $list[$k]['user_nick_name'] = $v['uid'] . '-' . $v['nick_name'];
            // $to_nick_name = db::name('user')->where(['uid' => $v['to_uid']])->value('nick_name');
            // $list[$k]['to_nick_name'] = $v['to_uid'] . '-' . $to_nick_name;
            $list[$k]['type_name'] = db::name('report_type')->where(['id' => $v['type_id']])->value('type_name');
        }
        $data = [];
        $data['count'] = db::name('user_report')->alias('a')->join('yy_user b', 'a.uid = b.uid')->where($map)->count();
        $data['list'] = $list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    //获取举报详情
    public function report_info($rid)
    {
        if (empty($rid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $room_type_info = db::name('user_report')->find($rid);
        $room_type_info['uid_name'] = db::name('user')->where(['uid' => $room_type_info['uid']])->value('nick_name');
        $room_type_info['to_uid_name'] = db::name('user')->where(['uid' => $room_type_info['to_uid']])->value('nick_name');
        return ['code' => 200, 'msg' => '获取成功', 'data' => $room_type_info];
    }

    //修改举报信息
    public function edit_user_info($rid, $status)
    {
        if (empty($rid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $upd = db::name('user_report')->where(['rid' => $rid])->update(['status' => $status, 'update_time' => time(), 'deal_time' => time()]);
        if ($upd) {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        }

    }

    //删除举报信息
    public function del_Report($rid)
    {
        $del = db::name('user_report')->where(['rid' => $rid])->delete();
        if ($del) {
            return ['code' => 200, 'msg' => '删除成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '删除失败', 'data' => null];
        }

    }

    //举报类型 列表
    public function report_type_list($order, $sort, $page = 1, $limit = 20)
    {
        $map[] = ['is_delete', '=', 1];

        $banner_list = db::name('report_type')->where($map)->order($order, $sort)->select();
        $data = [];
        $data['count'] = db::name('report_type')->where($map)->count();
        $data['list'] = $banner_list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    //获取举报类型
    public function report_type_info($id)
    {
        if (empty($id)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $report_type = db::name('report_type')->find($id);
        return ['code' => 200, 'msg' => '获取成功', 'data' => $report_type];
    }

    //添加举报类型
    public function add_report_type($type_name)
    {
        $arr = [];
        $arr['type_name'] = $type_name;
        $arr['add_time'] = time();
        $arr['update_time'] = time();
        $add = db::name('report_type')->insert($arr);
        if ($add) {
            return ['code' => 200, 'msg' => '添加成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '添加失败', 'data' => null];
        }

    }

    //修改举报类型
    public function edit_report_type($id, $type_name)
    {
        if (empty($id)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $upd = db::name('report_type')->where(['id' => $id])->update(['type_name' => $type_name, 'update_time' => time()]);
        if ($upd) {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        }

    }

    //删除举报类型
    public function del_report_type($id)
    {
        $del = db::name('report_type')->where(['id' => $id])->update(['is_delete' => 2, 'delete_time' => time()]);
        if ($del) {
            return ['code' => 200, 'msg' => '删除成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '删除失败', 'data' => null];
        }

    }
}
