<?php

namespace app\admin\model;

use think\Db;
use think\Model;

class Admin extends Model
{
    protected $pk = 'aid';
    protected $auto = ['update_time'];
    protected $insert = [
        'add_time',
        'system_menu_id_list' => 0,
        'is_delete' => 1,
        'delete_time' => 0,
    ];
    protected $update = ['update_time'];

    protected function setPasswordAttr($value)
    {
        return md5($value);
    }

    protected function setAddTimeAttr()
    {
        return time();
    }

    protected function setUpdateTimeAttr()
    {
        return time();
    }

    public function check_login_token($login_token)
    {
        if (empty($login_token)) {
            return ['code' => 201, 'msg' => '登录失效', 'data' => ''];
        }
        $map = [];
        $map[] = ['login_token', '=', $login_token];
        $map[] = ['is_delete', '=', 1];
        $map[] = ['system_type', '=', 2];
        $user_info = db::name('admin')->where($map)->find();
        if (empty($user_info)) {
            return ['code' => 201, 'msg' => '登录失效', 'data' => ''];
        } else {
            if(time() > $user_info['token_validity_time']){
                Db::name('admin')->where('aid', $user_info['aid'])->update(['login_token' => '', 'update_time' => time()]);
                return ['code' => 201, 'msg' => '登录失效', 'data' => ''];
            } else {
                if($user_info['token_validity_time'] <= (time() + 3600)) {
                    Db::name('admin')->where('aid', $user_info['aid'])->update(['token_validity_time' => time() + 7200, 'update_time' => time()]);
                }
            }
            return ['code' => 200, 'msg' => '登录成功', 'data' => $user_info['aid']];
        }
    }

    //获取管理员信息
    public function admin_info($login_token)
    {
        if (empty($login_token)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => ''];
        }
        $info = db::name('admin')->where(['login_token' => $login_token])->field('user_name')->find();
        return ['code' => 200, 'msg' => '获取成功', 'data' => $info];
    }
    public function add_admin($user_name, $password, $re_password)
    {
        $map = [];
        $map[] = ['user_name', '=', $user_name];
        $admin_info = db::name('admin')->where($map)->find();
        if (!empty($admin_info)) {
            return ['code' => 201, 'msg' => '用户名已存在', 'data' => ''];
        }
        $res = check_password_format($password);
        if($res['code'] == 201) {
            return $res;
        }
        
        if ($password != $re_password) {
            return ['code' => 201, 'msg' => '两次密码不一致', 'data' => ''];
        }

        $data = [];
        $data['user_name'] = $user_name;
        $data['password'] = $password;
        $data['system_type'] = 2;
        $validate = validate('admin/admin');
        $reslut = $validate->scene('adminAdd')->check($data);
        if ($reslut !== true) {
            return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
        }
        $Admin = model('admin/admin');
        $reslut = $Admin->save($data);
        if ($reslut) {
            return ['code' => 200, 'msg' => '添加成功', 'data' => ''];
        } else {
            return ['code' => 201, 'msg' => '修改失败', 'data' => ''];
        }
    }

    //修改管理员密码
    public function edit_admin_password($aid, $old_password, $password, $re_password, $phone)
    {
        $admin_info = db::name('admin')->where(['aid' => $aid])->find();
        if (empty($admin_info)) {
            return ['code' => 201, 'msg' => '信息不存在', 'data' => ''];
        }
        if (md5($old_password) != $admin_info['password']) {
            return ['code' => 201, 'msg' => '原始密码错误', 'data' => ''];
        }
        
        $res = check_password_format($password);
        if($res['code'] == 201) {
            return $res;
        }

        if ($password != $re_password) {
            return ['code' => 201, 'msg' => '两次密码不一致', 'data' => ''];
        }
        $data = [];

        $data['password'] = $password;
        $validate = validate('admin/admin');
        $reslut = $validate->scene('adminEditPassword')->check($data);
        if ($reslut !== true) {
            return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
        }
        $Admin = model('admin/admin');
        if($phone != $admin_info['phone']){
            $data['phone'] = $phone;
        }
        $reslut = $Admin->save($data, ['aid' =>  $admin_info['aid']]);
        if ($reslut) {
            return ['code' => 200, 'msg' => '修改成功', 'data' => ''];
        } else {
            return ['code' => 201, 'msg' => '修改失败', 'data' => ''];
        }
    }
    //修改管理员权限
    public function edit_admin_auth($aid, $auth)
    {
        $system_menu_id_list = implode(',', $auth);
        $data = [];
        $data['system_menu_id_list'] = $system_menu_id_list;
        $data['update_time'] = time();
        $res = db::name('admin')->where(['aid' => $aid])->update($data);
        if ($res) {
            return ['code' => 200, 'msg' => '修改成功', 'data' => ''];
        } else {
            return ['code' => 201, 'msg' => '修改失败', 'data' => ''];
        }
    }


    //获取管理员列表
    public function get_admin_list($user_name, $page, $page_limit)
    {
        $map = [];
        $map[] = ['is_delete', '=', 1];
        if (!empty($user_name)) {
            $map[] = ['user_name', 'like', '%' . $user_name . '%'];
        }
        $map[] = ['system_type', '=', 2];
        $list = db::name('admin')->where($map)->order('aid', 'asc')->page($page, $page_limit)->select();
        $data = [];
        $data['count'] = db::name('admin')->where($map)->count();
        $data['list'] = $list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    public function get_admin_info($aid)
    {
        $admin_info = db::name('admin')->find($aid);
        return ['code' => 200, 'msg' => '获取成功', 'data' => $admin_info];
    }
    public function delete_admin($aid)
    {
        if ($aid == 1) {
            return ['code' => 201, 'msg' => '总管理员禁止删除', 'data' => null];
        }
        $data = [];
        $data['is_delete'] = 2;
        $data['delete_time'] = time();
        $data['update_time'] = time();
        $res = db::name('admin')->where(['aid' => $aid])->update($data);
        if ($res) {
            return ['code' => 200, 'msg' => '删除成功', 'data' => ''];
        } else {
            return ['code' => 201, 'msg' => '删除失败', 'data' => ''];
        }
    }
    
    //管理员日志
    public function get_admin_log_list($page, $page_limit){
        $map = [];
        $map[] = ['type', '=', 1];
        $list = db::name('operation')->where($map)->order('op_id','desc')->page($page, $page_limit)->select();
        foreach ($list as $k => &$v) {
            $v['user_name'] = db::name('admin')->where('aid', $v['id'])->value('user_name');
            if(strpos($v['url'],'admin/box/get_box_type_list')){
                $v['operate_name'] = '宝箱礼物列表';
            }else if(strpos($v['url'],'admin/box/get_box_log_list')){
                $v['operate_name'] = '每期奖池列表';
            }else if(strpos($v['url'],'admin/box/edit_box_config')){
                $v['operate_name'] = '修改宝箱信息';
            }else if(strpos($v['url'],'admin/box/delete_box_config')){
                $v['operate_name'] = '删除宝箱信息';
            }else if(strpos($v['url'],'admin/box/add_give_gift')){
                $v['operate_name'] = '添加礼物补发';
            }else if(strpos($v['url'],'admin/box/cancel_give_gift')){
                $v['operate_name'] = '取消礼物补发';
            }else if(strpos($v['url'],'admin/box/get_box_give_gift_list')){
                $v['operate_name'] = '礼物补发列表';
            }else if(strpos($v['url'],'admin/user/get_user_list')){
                $v['operate_name'] = '用户列表';
            }else if(strpos($v['url'],'admin/user/edit_user_info')){
                $v['operate_name'] = '修改用户信息';
            }else if(strpos($v['url'],'admin/user/edit_user_money')){
                $v['operate_name'] = '修改用户资金';
            }else if(strpos($v['url'],'admin/user/edit_user_password')){
                $v['operate_name'] = '修改用户密码';
            }else if(strpos($v['url'],'admin/user/gold_consume_del')){
                $v['operate_name'] = '清除地阶累消';
            }else if(strpos($v['url'],'admin/user/drill_consume_del')){
                $v['operate_name'] = '清除天阶累消';
            }else if(strpos($v['url'],'admin/user/get_user_gift_pack')){
                $v['operate_name'] = '用户背包列表';
            }else if(strpos($v['url'],'admin/user/del_user_gift_pack')){
                $v['operate_name'] = '删除用户背包礼物';
            }else if(strpos($v['url'],'admin/config/config_list')){
                $v['operate_name'] = '系统配置列表';
            }else if(strpos($v['url'],'admin/config/edit_config')){
                $v['operate_name'] = '修改配置信息';
            }else if(strpos($v['url'],'admin/config/del_config')){
                $v['operate_name'] = '删除配置信息';
            }else if(strpos($v['url'],'admin/config/add_config')){
                $v['operate_name'] = '添加配置信息';
            }else if(strpos($v['url'],'admin/Admin/get_admin_log_list')){
                $v['operate_name'] = '管理员日志';
            }else if(strpos($v['url'],'admin/Admin/get_admin_list')){
                $v['operate_name'] = '管理员列表';
            }else if(strpos($v['url'],'admin/Admin/add_admin')){
                $v['operate_name'] = '增加管理员';
            }else if(strpos($v['url'],'admin/Admin/edit_admin_password')){
                $v['operate_name'] = '修改管理员密码';
            }else if(strpos($v['url'],'admin/Admin/edit_admin_auth')){
                $v['operate_name'] = '修改管理员权限';
            }else if(strpos($v['url'],'admin/Admin/delete_admin')){
                $v['operate_name'] = '删除管理员';
            }
        }
        $data = [];
        $data['count'] = db::name('operation')->where($map)->count();
        $data['list'] = $list;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
        
    }
    
    public function quit_admin_login($aid){
        if(empty($aid)){
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        
        $map = [];
        $map[] = ['aid','=', $aid];
        $admin_info = db::name('admin')->where($map)->find();
        if(!$admin_info){
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        
        $map = [];
        $map[] = ['aid', '=', $aid];
        $update_data = [];
        $update_data['login_token'] = '';
        $reslut = db::name('admin')->where($map)->update($update_data);
        if(!$reslut){
            return ['code' => 201, 'msg' => '退出失败', 'data' => null];
        }
        
        return ['code' => 200, 'msg' => '退出成功', 'data' => null];
        
    }
    
    //二级密码校验
    public function check_secondary_password($pass){
        if(empty($pass)){
            return ['code' => 201, 'msg' => '二级密码不能为空', 'data' => null];
        }
        $pass_word = secondary_password();
        if(md5($pass) != $pass_word){
            return ['code' => 201, 'msg' => '二级密码错误', 'data' => null];
        }else{
            return ['code' => 200, 'msg' => '成功', 'data' => null];
        }
    }
    //清除登录状态
    public function clear_admin_token($super_aid, $aid)
    {
        if($super_aid != 1) {
            return ['code' => 201, 'msg' => '无权限操作', 'data' => null];
        }
        $result = Db::name('admin')->where('aid', $aid)->update(['login_token' => '', 'error_num' => 0, 'status' => 1, 'update_time' => time()]);
        if($result) {
            return ['code' => 200, 'msg' => '处理成功', 'data' => null];
        }
        return ['code' => 201, 'msg' => '处理失败', 'data' => null];
    }
    
}
