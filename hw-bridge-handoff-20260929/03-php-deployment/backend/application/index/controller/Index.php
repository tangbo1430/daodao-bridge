<?php

namespace app\index\controller;

use think\Controller;
use think\Db;
use think\captcha\Captcha;
class Index extends Controller
{
    private $protocols = [
        'user_agreement' => [
            'title' => '企才汇用户协议',
            'url' => 'app_user_agreement',
            'default_content' => '<h2>企才汇用户协议</h2><p>请在后台「网站管理 - 协议与隐私」中编辑用户协议内容。</p>',
        ],
        'privacy_policy' => [
            'title' => '企才汇隐私政策',
            'url' => 'app_privacy_policy',
            'default_content' => '<h2>企才汇隐私政策</h2><p>请在后台「网站管理 - 协议与隐私」中编辑隐私政策内容。</p>',
        ],
    ];

    public function index($name = '-')
    {
        return 'hello,' . $name;
    }
    
    
    //单页
    public function page_show()
    {
        $type = input('type', '');
        if (!empty($type)) {
            if (!isset($this->protocols[$type])) {
                return json(['code' => 201, 'msg' => '协议类型错误', 'data' => null]);
            }
            $page_info = $this->getProtocolPageInfo($type, 'title,content');
            if (empty($page_info)) {
                return json(['code' => 201, 'msg' => '信息不存在', 'data' => null]);
            }
            $this->assign('data', $page_info);
            return $this->view->fetch();
        }

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

    // APP协议内容
    public function protocol()
    {
        $type = input('type', '');
        if (!isset($this->protocols[$type])) {
            return json(['code' => 201, 'msg' => '协议类型错误', 'data' => null]);
        }
        $page_info = $this->getProtocolPageInfo($type, 'title,content,update_time');
        if (empty($page_info)) {
            return json(['code' => 201, 'msg' => '信息不存在', 'data' => null]);
        }
        return json(['code' => 200, 'msg' => '获取成功', 'data' => $page_info]);
    }

    private function getProtocolPageInfo($type, $field)
    {
        $config = $this->protocols[$type];
        $map = [
            ['url', '=', $config['url']],
            ['system_type', '=', config('app.system_type')],
        ];
        $pageInfo = Db::name('page')->field($field . ',aid,is_delete')->where($map)->order('aid', 'asc')->find();
        if ($pageInfo) {
            if ((int) $pageInfo['is_delete'] !== 1) {
                Db::name('page')->where(['aid' => $pageInfo['aid']])->update([
                    'is_delete' => 1,
                    'delete_time' => 0,
                    'update_time' => time(),
                ]);
            }
            unset($pageInfo['aid'], $pageInfo['is_delete']);
            return $pageInfo;
        }

        $now = time();
        $aid = Db::name('page')->insertGetId([
            'title' => $config['title'],
            'url' => $config['url'],
            'content' => $config['default_content'],
            'is_delete' => 1,
            'delete_time' => 0,
            'add_time' => $now,
            'update_time' => $now,
            'system_type' => config('app.system_type'),
        ]);
        if (!$aid) {
            return null;
        }
        return Db::name('page')->field($field)->where(['aid' => $aid])->find();
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
