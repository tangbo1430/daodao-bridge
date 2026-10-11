<?php

namespace app\admin\model;

use think\Db;
use think\Model;

class Website extends Model
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


    //获取 单页信息 列表
    public function page_list($order, $sort, $page = 1, $limit = 20)
    {
        $map = [];
        $map[] = ['is_delete', '=', 1];
        $map[] = ['system_type', '=', config('app.system_type')];
        $list = db::name('page')->where($map)
            ->order($order, $sort)->page($page, $limit)->select();

        $data = [];
        $data['count'] = db::name('page')->where($map)->count();
        $data['list'] = $list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    //添加 单页信息
    public function add_page($data)
    {
        $add_data = [];
        $add_data['title'] = $data['title'];
        $add_data['url'] = $data['url'];
        $add_data['content'] = $data['content'];
        $add_data['update_time'] = time();
        $add_data['add_time'] = time();
        $add_data['system_type'] = config('app.system_type');
        $reslut = db::name('page')->insert($add_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '添加失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '添加成功', 'data' => null];
        }
    }

    //编辑  单页信息
    public function edit_page($data)
    {
        if (empty($data['aid'])) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $info = db::name('page')->find($data['aid']);
        if (empty($info)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $update_data = [];
        $update_data['title'] = $data['title'];
        $update_data['url'] = $data['url'];
        $update_data['content'] = html_entity_decode($data['content']);
        $update_data['update_time'] = time();
        $reslut = db::name('page')->where(['aid' => $data['aid']])->update($update_data);
        if (!$reslut) {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        }
    }

    //获取 单页信息 详情
    public function get_page($aid)
    {
        if (empty($aid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $info = db::name('page')->where(['aid' => $aid])->find();
        return ['code' => 200, 'msg' => '获取成功', 'data' => $info];
    }

    //删除 单页信息
    public function page_del($aid)
    {
        if (empty($aid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        if ($aid <= 10) {
            return ['code' => 201, 'msg' => '前10个单页禁止删除', 'data' => null];
        }
        $data_del = db::name('page')->where(['aid' => $aid])->update(['is_delete' => 2, 'delete_time' => time()]);
        if ($data_del) {
            return ['code' => 200, 'msg' => '删除成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '删除失败', 'data' => null];
        }
    }

    public function protocol_list()
    {
        $list = [];
        foreach ($this->protocols as $type => $config) {
            $info = $this->ensureProtocol($type);
            if ($info) {
                $info['type'] = $type;
                $info['show_url'] = '/index/index/page_show?type=' . $type;
                $list[] = $info;
            }
        }
        return ['code' => 200, 'msg' => '获取成功', 'data' => $list];
    }

    public function get_protocol($type)
    {
        $type = trim((string) $type);
        if (!$this->isProtocolType($type)) {
            return ['code' => 201, 'msg' => '协议类型错误', 'data' => null];
        }
        $info = $this->ensureProtocol($type);
        $info['type'] = $type;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $info];
    }

    public function save_protocol($data)
    {
        $type = isset($data['type']) ? trim((string) $data['type']) : '';
        if (!$this->isProtocolType($type)) {
            return ['code' => 201, 'msg' => '协议类型错误', 'data' => null];
        }
        if (!isset($data['content']) || trim((string) $data['content']) === '') {
            return ['code' => 201, 'msg' => '协议内容不能为空', 'data' => null];
        }
        $config = $this->protocols[$type];
        $info = $this->ensureProtocol($type);
        if (!$info) {
            return ['code' => 201, 'msg' => '协议初始化失败', 'data' => null];
        }
        $title = isset($data['title']) && trim((string) $data['title']) !== ''
            ? trim((string) $data['title'])
            : $config['title'];
        $updateData = [
            'title' => $title,
            'url' => $config['url'],
            'content' => html_entity_decode((string) $data['content']),
            'update_time' => time(),
            'system_type' => config('app.system_type'),
            'is_delete' => 1,
        ];
        db::name('page')->where(['aid' => $info['aid']])->update($updateData);
        return ['code' => 200, 'msg' => '保存成功', 'data' => null];
    }

    private function ensureProtocol($type)
    {
        if (!$this->isProtocolType($type)) {
            return null;
        }
        $config = $this->protocols[$type];
        $map = [
            ['url', '=', $config['url']],
            ['system_type', '=', config('app.system_type')],
        ];
        $info = db::name('page')->where($map)->order('aid', 'asc')->find();
        if ($info) {
            if ((int) $info['is_delete'] !== 1) {
                db::name('page')->where(['aid' => $info['aid']])->update([
                    'is_delete' => 1,
                    'delete_time' => 0,
                    'update_time' => time(),
                ]);
                $info['is_delete'] = 1;
                $info['delete_time'] = 0;
            }
            return $info;
        }
        $now = time();
        $aid = db::name('page')->insertGetId([
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
        return db::name('page')->where(['aid' => $aid])->find();
    }

    private function isProtocolType($type)
    {
        return isset($this->protocols[$type]);
    }
}
