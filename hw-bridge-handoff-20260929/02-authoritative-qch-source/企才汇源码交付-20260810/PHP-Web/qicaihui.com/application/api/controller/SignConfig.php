<?php


namespace app\api\controller;


class SignConfig extends Common
{
    public function get_sign_config()
    {
        $list = model('api/SignConfig')->get_sign_config($this->uid);
        ajaxReturn($list['code'], $list['msg'], $list['data']);
    }

    //签到
    public function sign_in()
    {
        $day_num = input('day_num/d', 0);
        $key_name = "api:SignConfig:sign_in:uid:".$this->uid;
        redis_lock_exit($key_name);
        $data = model('api/SignConfig')->sign_in($this->uid, $day_num);
        redis_unlock($key_name);
        ajaxReturn($data['code'], $data['msg'], $data['data']);
    }
}