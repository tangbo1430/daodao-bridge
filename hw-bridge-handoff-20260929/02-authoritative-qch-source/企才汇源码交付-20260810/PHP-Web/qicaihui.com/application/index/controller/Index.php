<?php

namespace app\index\controller;

use think\Controller;
use think\Db;
use think\captcha\Captcha;
class Index extends Controller
{

    public function index($name = '-')
    {
        return 'hello,' . $name;
    }
    
    
    //单页
    public function page_show()
    {
        $aid = input('id', 0);
        $map = [];
        $map[] = ['aid', '=', $aid];
        $map[] = ['is_delete', '=', 1];
        $page_info = db::name('page')->field('title,content')->where($map)->find();
        if (empty($page_info)) {
            // return ['code' => 201, 'msg' => '信息不存在', 'data' => null];
            return json(['code' => 201, 'msg' => '信息不存在', 'data' => null]);
        }
        $this->assign('data', $page_info);
        return $this->view->fetch();
    }
 //注册页
    public function register()
    {
        $config = Db::name('config')->whereIn('key_title', ['web_site', 'invite_down_url'])
            ->column('key_value', 'key_title');
        $this->assign('api_url', $config['web_site']);
        $this->assign('invite_down_url', $config['invite_down_url']);
        return $this->fetch();
    }
    
    //获取图片验证码
    public function verify()
    {
        $config = [
            'codeSet' => '0123456789',
            // 验证码字体大小
            'fontSize' => 30,
            // 验证码位数
            'length' => 4,
            // 关闭验证码杂点
            'useNoise' => false,
        ];
        $captcha = new Captcha($config);
        return $captcha->entry();
    }
}
