<?php

namespace app\api\model;

use think\Db;
use think\Model;

class UserReport extends Model
{
    public function user_report($uid, $to_uid, $type_id, $image, $content)
    {
        $map = [];
        $map[] = ['id', '=', $type_id];
        $report_type_info = db::name('report_type')->where($map)->find();
        if (empty($report_type_info)) {
            return ['code' => 201, 'msg' => '举报类型异常', 'data' => null];
        }
        // $user_info = db::name('user')->find($to_uid);
        // if (empty($user_info)) {
        //     return ['code' => 201, 'msg' => '举报用户不存在', 'data' => null];
        // }
        $data = [];
        $data['uid'] = $uid;
        // $data['to_uid'] = $to_uid;
        $data['type_id'] = $type_id;
        $data['image'] = $image;
        $data['content'] = $content;
        $data['status'] = 1;
        $data['deal_time'] = 0;
        $data['add_time'] = time();
        $data['update_time'] = time();
        $reslut = db::name('user_report')->insert($data);
        if ($reslut) {
            return ['code' => 200, 'msg' => '提交成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '提交失败', 'data' => null];
        }
    }
    
    public function report_list(){
        $map = [];
        $map[] = ['is_delete', '=', 1];
        $reslut = db::name('report_type')->field('id,type_name')->where($map)->select();
        return ['code' => 200, 'msg' => '获取成功', 'data' => $reslut];
    }
    
}
