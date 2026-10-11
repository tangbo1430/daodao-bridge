<?php


namespace app\admin\model;


use think\Db;
use think\Model;
class SignConfig extends Model
{
    //分类列表
    public function get_list($page, $page_limit)
    {
        $map = [];
        $list = Db::name('sign_config')->where($map)
            ->page($page, $page_limit)->select();
        $data = [];
        $data['count'] = db::name('sign_config')->where($map)->count();
        $data['data'] = $list;
        $data['code'] = 0;
        $data['msg'] = '获取数据成功';
        return json($data);
    }
    //添加
    public function add($data)
    {
        return ['code' => 200, 'msg' => '添加成功', 'data' => null];
        $cate_name = $data['cate_name'] ?? '';
        $sort = $data['sort'] ?? 0;
        if (empty($cate_name)) {
            return ['code' => 201, 'msg' => '分类名称不能为空', 'data' => null];
        }
        //是否存在
        $map = [];
        $map[] = ['cate_name', '=', $cate_name];
        $tag_count = Db::name('live_room_cate')->where($map)->count();
        if ($tag_count) {
            return ['code' => 201, 'msg' => '分类名称已存在', 'data' => null];
        }
        $add_data = [];
        $add_data['cate_name'] = $cate_name;
        $add_data['sort'] = $sort;
        $add_data['update_time'] = time();
        $add_data['add_time'] = time();
        $reslut = db::name('live_room_cate')->insert($add_data);
        if (!$reslut) {

            return ['code' => 201, 'msg' => '添加失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '添加成功', 'data' => null];
        }
    }
    //获取信息
    public function get_info($id)
    {
        if (empty($id)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $info = db::name('sign_config')->where(['id' => $id])->find();
        return ['code' => 200, 'msg' => '获取成功', 'data' => $info];
    }
    //编辑
    public function edit($data)
    {

        $integral = $data['integral'] ?? 0;
        $id = $data['id'] ?? '';
        $info = db::name('sign_config')->find($id);
        if (empty($info)) {
            return ['code' => 201, 'msg' => '参数异常3', 'data' => null];
        }

        $update_data = [];
        $update_data['integral'] = $integral;
        $update_data['update_time'] = time();
        $reslut = db::name('sign_config')->where(['id' => $id])->update($update_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '添加失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '添加成功', 'data' => null];
        }
    }
    //删除
    public function del($id)
    {
        return ['code' => 201, 'msg' => '数据不存在', 'data' => null];
        $info = Db::name('live_room_cate')->find( $id);
        if (empty($info)) {
            return ['code' => 201, 'msg' => '数据不存在', 'data' => null];
        }

        Db::startTrans();
        try{
            Db::name('live_room_cate')->where('id', $id)
                ->delete();
            Db::commit();
            return ['code' => 200, 'msg' => '删除成功', 'data' => null];
        } catch (\Exception $e) {
            Db::rollback();
            return ['code' => 201, 'msg' => '删除失败', 'data' => null];
        }
    }

    //获取用户签到记录
    public function get_user_sign_list($page, $page_limit)
    {
        $uid = input('uid', 0);
        $map = [];
        if ($uid) {
            $map[] = ['a.uid', '=', $uid];
        }

        $list = Db::name('sign_user')->alias('a')
            ->join('user b', 'a.uid = b.uid')
            ->field('a.*,b.base64_nick_name')
            ->where($map)
            ->page($page, $page_limit)->select();
        foreach ($list as &$val) {
            $val['nick_name'] = decode_base64($val['base64_nick_name']);
        }
        $data = [];
        $data['count'] = db::name('sign_user')->alias('a')->where($map)->count();
        $data['data'] = $list;
        $data['code'] = 0;
        $data['msg'] = '获取数据成功';
        return json($data);
    }
}