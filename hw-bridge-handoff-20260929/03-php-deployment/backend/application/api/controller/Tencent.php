<?php

namespace app\api\controller;

use think\Controller;

class Tencent extends Controller
{
    //腾讯IM回调
    public function call_back(){
        $data = input('post.');
        $sign_data = input('get.');
        $reslut = model('UserTencent')->tencent_call_back($data,$sign_data);
        return json($reslut);
    }
    
    // public function tec()
    // {
    //     $reslut = model('Tencent')->tencent_user_sig_info();
    //     return json($reslut);
    // }
    
}
