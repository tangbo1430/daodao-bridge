<?php

namespace app\admin\model;

use think\Model;
use think\Db;

class Statistics extends Model
{
    //获取首页基础统计数据
    public function welcome_data()
    {
        $data = [];
        //获取系统会员总人数
        $data['user_all_count'] = db::name('user')->count();
        //今日会员新增总数
        $data['user_today_count'] = db::name('user')->whereTime('add_time', 'today')->count();
        //本周会员新增总数
        $data['user_week_count'] = db::name('user')->whereTime('add_time', 'week')->count();
        
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }
    
}
