<?php


namespace app\api\model;


use think\Db;
use think\Model;

class SignConfig extends Model
{
    //签到配置
    public function get_sign_config($uid)
    {
        
        $list = Db::name('sign_config')
            ->field('id as sign_id,day_num,money')
            ->order('day_num asc')
            ->select();
        $day_time = date('Ymd');
        $last_sign_info = Db::name('sign_user')->where('uid', $uid)->order('id', 'desc')->find();
        //昨天
        $yesterday = date('Ymd', strtotime('-1 day'));
        if ($last_sign_info) {
            $sign_day_num = $last_sign_info['day_num'];
            if($last_sign_info['day_num'] >= 7) {
                $sign_day_num = 0;
            }
        } else {
            $sign_day_num = 0;
        }
        $today_num = 0;
        $data = [];
        foreach($list as $key => $v) {
            $v['is_today'] = 2;
            if ($sign_day_num == 0) {
                if ($v['day_num'] == 1) {
                    $v['sign_status'] = 2;
                    $v['is_today'] = 1;
                    $today_num++;
                } else {
                    $v['sign_status'] = 3;
                }
                $data[$key] = $v;
            } else {
                if ($sign_day_num >= $v['day_num']) {
                    $v['sign_status'] = 1;
                    $data[$key] = $v;
                    if ($sign_day_num == $v['day_num']) {
                        $sign_day_time = Db::name('sign_user')
                            ->where(['uid' => $uid, 'day_num' => $v['day_num']])
                            ->order('id', 'desc')
                            ->value('day_time');
                        if ($sign_day_time == $day_time) {
                            $v['is_today'] = 1;
                            $today_num++;
                            $data[$key] = $v;
                        } 
                    } 
                } else {
                    if($today_num == 0) {
                        $v['is_today'] = 1;
                        $v['sign_status'] = 2;
                        $today_num++;
                        $data[$key] = $v;
                    } else {
                        $v['sign_status'] = 3;
                        $data[$key] = $v;
                    }
                    
                    
                        
                }
            }
        }
        foreach($data as &$v) {
            
            $v['day_time_format'] = 0;
            if($v['sign_status'] == 1) {
                $day_time_format = Db::name('sign_user')->where(['uid' => $uid, 'day_num' => $v['day_num']])->order('id desc')->value('add_time');
                $v['day_time_format'] = $day_time_format;
            }
            
            // if($day_time_format) {
                
            // }
        }
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    //签到
    public function sign_in($uid, $day_num)
    {
        
        $day_time = date('Ymd');
        $yesterday = date('Ymd', strtotime('-1 day'));
        
        $last_sign_info = Db::name('sign_user')->where('uid', $uid)->order('id', 'desc')->find();
        if ($last_sign_info) {
            if ($last_sign_info['day_time'] == $day_time) {
                return ['code' => 201, 'msg' => '今天已经签到过了', 'data' => null];
            } else {
                if($last_sign_info['day_num'] != 7) {
                    if($day_num > $last_sign_info['day_num'] + 1) {
                        return ['code' => 201, 'msg' => '签到失败，不能超过今天签到', 'data' => null];
                    }
                } else {
                    if($day_num != 1) {
                        return ['code' => 201, 'msg' => '签到失败', 'data' => null];
                    }
                }
                
            }
        } else {
            if ($day_num != 1) {
                return ['code' => 201, 'msg' => '签到失败，不是从头签到', 'data' => null];
            }
        }
        $sign_config = Db::name('sign_config')->where('day_num', $day_num)->find();
        if (empty($sign_config)) {
            return ['code' => 201, 'msg' => '签到失败，签到配置不存在', 'data' => null];
        }
        Db::startTrans();
        try {
            $money = 0;
            // $reslut88 = model('admin/User')->change_user_money_by_uid($uid, $money, 2, "用户每日签到", $uid, 0);
            // if ($reslut88['code'] != 200) {
            //     Db::rollback();
            //     return ['code' => 201, 'msg' => $reslut88['msg'], 'data' => null];
            // }
            Db::name('sign_user')->insert([
                'uid' => $uid,
                'money' => $money,
                'day_num' => $day_num,
                'day_time' => $day_time,
                'add_time' => time(),
                'update_time' => time(),
            ]);
            Db::commit();
            return ['code' => 200, 'msg' => '签到成功', 'data' => null];
        } catch (\Exception $e) {
            Db::rollback();
            dump($e);
            return ['code' => 201, 'msg' => '签到失败', 'data' => null];
        }
    }
}