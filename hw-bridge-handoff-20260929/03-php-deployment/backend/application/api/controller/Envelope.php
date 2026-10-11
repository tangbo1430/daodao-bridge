<?php

namespace app\api\controller;

use think\Controller;

class Envelope extends Common
{
    //发红包
    public function give_red_envelope(){
        $uid = $this->uid;
        $family_id = input('group_id', 0);
        $money = input('money', 0);
        $num = input('num', 0);
        $title = input('title', '');
        $interval = input('interval', 0);
        $trade_password = input('trade_password', '');
        $is_special = input('is_special', 2);
        $receive_uid = input('receive_uid', 0);
        $key_name = "api:Envelope:give_red_envelope:uid:".$uid;
        // redis_lock_exit($key_name);
        $reslut = model('api/Envelope')->give_red_envelope($uid, $family_id, $money, $num, $title, $interval, $trade_password, $is_special, $receive_uid);
        // redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //抢红包
    public function snatch_red_envelope(){
        $uid = $this->uid;
        $group_id = input('group_id', 0);
        $envelope_id = input('envelope_id', 0);
        $key_name = "api:FamilyEnvelope:snatch_red_envelope:uid:".$uid;
        redis_lock_exit($key_name);
        $reslut = model('envelope')->snatch_red_envelope($uid, $group_id, $envelope_id);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //红包信息
    public function red_envelope_info(){
        $uid = $this->uid;
        $id = input('envelope_id', 0);
        
        $reslut = model('envelope')->red_envelope_info($uid, $id);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //抢完红包详情:中将记录
    public function snatch_red_envelope_info(){
        $uid = $this->uid;
        $id = input('envelope_id', 0);
        $reslut = model('envelope')->snatch_red_envelope_info($uid, $id);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    
}
