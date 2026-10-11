<?php

namespace app\api\model;

use think\Db;
use think\Model;
use think\facade\Env;
use RongCloud;

class User extends Model
{
    // 提现记录
    public function get_user_withdrawal_list($uid, $page, $page_limit)
    {
        $page = intval($page);
        $page_limit = $page_limit < 30 ? $page_limit : 30;
        $list = Db::name('user_withdrawal')->field('order_sn, money,general_money,remarke,status,add_time')->where('uid', $uid)->order('wid', 'desc')->page($page, $page_limit)->select();
        foreach ($list as $key => &$v) {
            if ($v['status'] == 1) {
                $v['status_desc'] = '审核中';
            } elseif ($v['status'] == 2) {
                $v['status_desc'] = '已提现';
            } elseif ($v['status'] == 3) {
                $v['status_desc'] = '已拒绝' . ':' . $v['remarke'];
            }
        }
        return ['code' => 200, 'msg' => '获取成功', 'data' => $list];
    }
    public function user_login($user_name, $password, $last_login_device, $system)
    {
        if (empty($user_name)) {
            return ['code' => 201, 'msg' => '用户名不能为空', 'data' => null];
        }
        if (empty($password)) {
            return ['code' => 201, 'msg' => '密码不能为空', 'data' => null];
        }
        $map = [];
        $map[] = ['user_name', '=', $user_name];
        $user_info = db::name('user')->where($map)->find();
        if (empty($user_info)) {
            return ['code' => 201, 'msg' => '用户名不存在', 'data' => null];
        }
        if (md5($password) != $user_info['password']) {
            return ['code' => 201, 'msg' => '密码错误', 'data' => null];
        }
        if ($user_info['login_status'] != 1) {
            return ['code' => 201, 'msg' => '用户已被封禁', 'data' => null];
        }
        
        $data = [];
        $data['uid'] = $user_info['uid'];
        $login_token_string = md5($user_info['uid'] . date('YmdHis') . generateRandom(32));
        // if (!empty($user_info['login_token'])) {
        //     $login_token_string = $user_info['login_token'];
        // }
        // $user_sigs = $user_info['user_sig'];
        // if(empty($user_info['user_sig'])){
            $user_sig = model('Tencent')->tencent_user_sig_info($user_info['uid']);
            if(empty($user_sig)){
                return ['code' => 201, 'msg' => '登录失败', 'data' => null];
            }
            $data['user_sig'] = $user_sig;
            $user_sigs = $user_sig;
        // }
        $data['system'] = $system;
        $data['login_token'] = $login_token_string;
        $data['last_login_time'] = time();
        $data['last_login_device'] = $last_login_device;
        $data['login_ip'] = request()->ip();
        $data['update_time'] = time();
        if($user_info['is_first_login'] == 1) {
            $data['is_first_login'] = 2;
        }
        $reslut = db::name('user')->update($data);

        $return_res = [];
        $return_res['uid'] = $user_info['uid'];
        $return_res['head_pic'] = localpath_to_netpath($user_info['head_pic']);
        $return_res['user_name'] = $user_name;
        $return_res['login_token'] = $login_token_string;
        $return_res['user_sig'] = $user_sigs;
        $return_res['is_real'] = $user_info['is_real'];
        $return_res['nick_name'] = mb_convert_encoding(base64_decode($user_info['base64_nick_name']), 'UTF-8', 'UTF-8');
        $return_res['system_type'] = $user_info['system_type'];
        if (!$reslut) {
            return ['code' => 201, 'msg' => '登录失败', 'data' => null];
        } else {
            return  ['code' => 200, 'msg' => '登录成功', 'data' => $return_res];
        }
    }
    
    public function add_system_account($uid)
    {
        $user_info = Db::name('user')->find($uid);
        // if($user_info['is_first_login'] != 1) {
        //     return ['code' => 200, 'msg' => '', 'data' => null];
        // }
        $system_type = $user_info['system_type'];
        if($system_type == 4) {
            $redis = connectionRedis();
            $key_name = 'im:customer:list:system_type:' . $system_type; 
            $redis_len = $redis->llen($key_name);
            //人数不足自动添加
            if($redis_len < 4) {
                $uid_arr = Db::name('user')->where('system_type', $system_type)->where('is_customer', 1)->column('uid');
                array_unshift($uid_arr, $key_name);
                call_user_func_array([$redis, 'rPush'], $uid_arr);
            }
            $customer_id = $redis->lpop($key_name);
            if($customer_id > 0) {
                model('api/Tencent')->friend_add($uid, $customer_id);
            }
        }
       
        // dump($result);
        return ['code' => 200, 'msg' => '', 'data' => null];
    }
    public function user_reg($user_name, $password, $reg_code)
    {
        $pid = 0;
        $path = '0';
        $system_type = 2;
        $invite_code_open_status = Db::name('config')->where('key_title', 'invite_code_open_status')->value('key_value');
        // dump($reg_code);die;
        $invite_code = '';
        if (!empty($reg_code)) {
            if($invite_code_open_status == 1) {
                $invite_code = Db::name('config')->where('key_title', 'system_invite_code')->value('key_value');
                if($invite_code != $reg_code) {
                    $invitecode = Db::name('invitecode')->where(['name' => $reg_code])->find();
                    // dump($invitecode);die;
                    if(empty($invitecode)){
                        
                        return ['code' => 201, 'msg' => '邀请码不存在', 'data' => null];
                    } else {
                        $system_type = $invitecode['system_type'];
                        $invite_code = $reg_code;
                    }
                }
            } else {
                // dump($reg_code);
                $invitecode = Db::name('invitecode')->where(['name' => $reg_code])->find();
                // dump($invitecode);die;
                if(empty($invitecode)){
                    
                    
                } else {
                    $system_type = $invitecode['system_type'];
                    $invite_code = $reg_code;
                }
            }
             
            // $map = [];
            // $map[] = ['reg_code', '=', $reg_code];
            // $p_user_info = Db::name('user')->where($map)->find();
            // if (empty($p_user_info)) {
            //     return ['code' => 201, 'msg' => '邀请码不存在', 'data' => null];
            // }
            // $pid = $p_user_info['uid'];
            // $path = $p_user_info['path'];
           
        } else {
            if($invite_code_open_status == 1) {
                return ['code' => 201, 'msg' => '请输入邀请码', 'data' => null];
            }
        }
        $map = [];
        $map[] = ['user_name', '=', $user_name];
        $user_info = db::name('user')->where($map)->find();
        if (!empty($user_info)) {
            return ['code' => 201, 'msg' => '用户名已存在', 'data' => null];
        }

        $data = [];
        $uid = $this->get_available_uid(); //获取自增用户id 过滤靓号
        $data['uid'] = $uid;
        $data['user_name'] = $user_name;
        $nick_name = model('admin/User')->get_rand_nick_name();
        // $nick_name = '游客'.$uid;
        $data['nick_name'] = $nick_name;
        $data['base64_nick_name'] = base64_encode($nick_name);
        $data['password'] = $password;
        $data['pid'] = $pid;
        $reg_ip = request()->ip();
        $data['reg_ip'] = $reg_ip;
        $position = ip_to_position($reg_ip);
        $data['country'] = $position['country'];
        $data['province'] = $position['province'];
        $data['city'] = $position['city'];
        $birthday = date('Y-m-d');
        $data['birthday'] = $birthday;
        $constellation = $this->get_user_constellation($birthday);
        $data['constellation'] = $constellation['data'];
        $data['head_pic'] = 'head_pic/head_pic1.png';
        $data['autograph'] = '这个人很懒，什么都没写';
        $data['hobby'] = '暂无';
        $data['is_vip'] = "1";
        $data['vip_end_time'] = time()+60*60*24*5;
        $data['reg_type'] = 1;
        $data['system_type'] = $system_type;
        $data['invite_code'] = $invite_code;
        $validate = validate('admin/user');
        $reslut = $validate->scene('apiAdd')->check($data);
        if ($reslut !== true) {
            return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
        }

        Db::startTrans();
        try {
            $User = model('admin/user');
            $reslut = $User->save($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '注册失败', 'data' => null];
            };
            /*********** 腾讯云IM  ********/
            $user_sig = model('Tencent')->tencent_user_sig_info($uid);
            if(empty($user_sig)){
                Db::rollback();
                return ['code' => 201, 'msg' => '登录失败', 'data' => null];
            }
            $data = [];
            $data['uid'] =  $uid;
            $data['path'] = $path . ',' . $uid;
            $data['user_sig'] = $user_sig;
            $data['update_time'] = time();
            $reslut = db::name('user')->where(['uid' => $uid])->update($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '注册失败', 'data' => null];
            }
            if (isset($invitecode)) {
                $invitecode = Db::name('invitecode')->where('id',$invitecode['id'])
                ->inc('invite_num', 1)
                ->update(['status'=>1,'user_id'=>$uid]);      
                //自动添加客服为好友
                
            } 
            Db::commit();
            return ['code' => 200, 'msg' => '注册成功', 'data' => ['nick_name' => $nick_name]];
        } catch (\Exception $e) {
            // 回滚事务
            dump($e);
            Db::rollback();
            return ['code' => 201, 'msg' => '注册失败', 'data' => null];
        }
    }
    
    public function user_reg_account($user_name, $password, $reg_code, $repassword)
    {
        if($password != $repassword) {
            return ['code' => 201, 'msg' => '密码和确认密码不一致', 'data' => null];
        }
        // dump(validate_user_name($user_name));
        // dump($user_name);die;
        if(empty(validate_user_name($user_name))) {
            return ['code' => 201, 'msg' => '账号格式不是字母或数字', 'data' => null];
        }
        if(strlen($user_name) > 12) {
            return ['code' => 201, 'msg' => '账号不能超过12个字符', 'data' => null];
        }
        if(strlen($user_name) < 6) {
            return ['code' => 201, 'msg' => '账号不能低于6个字符', 'data' => null];
        }
        $pid = 0;
        $path = '0';
        $system_type = 2;
        $invite_code_open_status = Db::name('config')->where('key_title', 'invite_code_open_status')->value('key_value');
        // dump($reg_code);die;
        $invite_code = '';
        if (!empty($reg_code)) {
            if($invite_code_open_status == 1) {
                $invite_code = Db::name('config')->where('key_title', 'system_invite_code')->value('key_value');
                if($invite_code != $reg_code) {
                    $invitecode = Db::name('invitecode')->where(['name' => $reg_code])->find();
                    // dump($invitecode);die;
                    if(empty($invitecode)){
                        
                        return ['code' => 201, 'msg' => '邀请码不存在', 'data' => null];
                    } else {

                        if($invitecode['invite_num'] > 0){
                            if(strpos($invitecode['user_id'],',') !== false){
                                $count = count(explode(',',$invitecode['user_id']));
                                if($count >= $invitecode['invite_num']){
                                    return ['code' => 201, 'msg' => '邀请码已被使用', 'data' => null];
                                }   
                            }
                        }

                        $system_type = $invitecode['system_type'];
                        $invite_code = $reg_code;
                    }
                }
            } else {
                // dump($reg_code);
                $invitecode = Db::name('invitecode')->where(['name' => $reg_code])->find();
                // dump($invitecode);die;
                if(empty($invitecode)){
                    
                    
                } else {

                    if($invitecode['invite_num'] > 0){
                        if(strpos($invitecode['user_id'],',') !== false){
                            $count = count(explode(',',$invitecode['user_id']));
                            if($count >= $invitecode['invite_num']){
                                return ['code' => 201, 'msg' => '邀请码已被使用', 'data' => null];
                            }   
                        }
                    }
                    $system_type = $invitecode['system_type'];
                    $invite_code = $reg_code;
                }
            }
            
	    
            // $map = [];
            // $map[] = ['reg_code', '=', $reg_code];
            // $p_user_info = Db::name('user')->where($map)->find();
            // if (empty($p_user_info)) {
            //     return ['code' => 201, 'msg' => '邀请码不存在', 'data' => null];
            // }
            // $pid = $p_user_info['uid'];
            // $path = $p_user_info['path'];
           
        } else {
            if($invite_code_open_status == 1) {
                return ['code' => 201, 'msg' => '请输入邀请码', 'data' => null];
            }
        }
        // dump($system_type);die;
        $map = [];
        $map[] = ['user_name', '=', $user_name];
        $user_info = db::name('user')->where($map)->find();
        if (!empty($user_info)) {
            return ['code' => 201, 'msg' => '用户名已存在', 'data' => null];
        }

        $data = [];
        $uid = $this->get_available_uid(); //获取自增用户id 过滤靓号
        $data['uid'] = $uid;
        $data['user_name'] = $user_name;
        $nick_name = $user_name;
        // $nick_name = '游客'.$uid;
        $data['nick_name'] = $nick_name;
        $data['base64_nick_name'] = base64_encode($nick_name);
        $data['password'] = $password;
        $data['pid'] = $pid;
        $reg_ip = request()->ip();
        $data['reg_ip'] = $reg_ip;
        $position = ip_to_position($reg_ip);
        $data['country'] = $position['country'];
        $data['province'] = $position['province'];
        $data['city'] = $position['city'];
        $birthday = date('Y-m-d');
        $data['birthday'] = $birthday;
        $constellation = $this->get_user_constellation($birthday);
        $data['constellation'] = $constellation['data'];
        $data['head_pic'] = 'head_pic/head_pic1.png';
        $data['autograph'] = '这个人很懒，什么都没写';
        $data['hobby'] = '暂无';
        $data['is_vip'] = "1";
        $data['reg_type'] = 2;
        $data['vip_end_time'] = time()+60*60*24*5;
        // $data['system_type'] = 2;
        $data['system_type'] = $system_type;//来自新后台
        $data['invite_code'] = $invite_code;
        // $validate = validate('admin/user');
        // $reslut = $validate->scene('apiAdd')->check($data);
        // if ($reslut !== true) {
        //     return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
        // }

        Db::startTrans();
        try {
            $User = model('admin/user');
            $reslut = $User->save($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '注册失败11', 'data' => null];
            };
            /*********** 腾讯云IM  ********/
            $user_sig = model('Tencent')->tencent_user_sig_info($uid);
            if(empty($user_sig)){
                Db::rollback();
                return ['code' => 201, 'msg' => '登录失败', 'data' => null];
            }
            $data = [];
            $data['path'] = $path . ',' . $uid;
            $data['user_sig'] = $user_sig;
            $data['update_time'] = time();
            $reslut = db::name('user')->where(['uid' => $uid])->update($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '注册失败22', 'data' => null];
            }

            if (isset($invitecode)) {
                $invitecode = Db::name('invitecode')->where('id',$invitecode['id'])
                ->inc('invite_num', 1)
                ->update(['status'=>1,'user_id'=>$uid]);      
                //自动添加客服为好友
                
            } 
            Db::commit();
            return ['code' => 200, 'msg' => '注册成功', 'data' => ['nick_name' => $nick_name]];
        } catch (\Exception $e) {
            // 回滚事务
            
            Db::rollback();
            dump($e);
            return ['code' => 201, 'msg' => '注册失败33', 'data' => null];
        }
    }
    
    public function user_login_account($user_name, $password, $last_login_device, $system)
    {
        if (empty($user_name)) {
            return ['code' => 201, 'msg' => '用户名不能为空', 'data' => null];
        }
        if (empty($password)) {
            return ['code' => 201, 'msg' => '密码不能为空', 'data' => null];
        }
        $map = [];
        $map[] = ['user_name', '=', $user_name];
        $user_info = db::name('user')->where($map)->find();
        if (empty($user_info)) {
            return ['code' => 201, 'msg' => '用户名不存在', 'data' => null];
        }
        if (md5($password) != $user_info['password']) {
            return ['code' => 201, 'msg' => '密码错误', 'data' => null];
        }
        if ($user_info['login_status'] != 1) {
            return ['code' => 201, 'msg' => '用户已被封禁', 'data' => null];
        }
        
        $data = [];
        $data['uid'] = $user_info['uid'];
        $login_token_string = md5($user_info['uid'] . date('YmdHis') . generateRandom(32));
        // if (!empty($user_info['login_token'])) {
        //     $login_token_string = $user_info['login_token'];
        // }
        // $user_sigs = $user_info['user_sig'];
        // if(empty($user_info['user_sig'])){
            $user_sig = model('Tencent')->tencent_user_sig_info($user_info['uid']);
            if(empty($user_sig)){
                return ['code' => 201, 'msg' => '登录失败', 'data' => null];
            }
            $data['user_sig'] = $user_sig;
            $user_sigs = $user_sig;
        // }
        $data['system'] = $system;
        $data['login_token'] = $login_token_string;
        $data['last_login_time'] = time();
        $data['last_login_device'] = $last_login_device;
        $data['login_ip'] = request()->ip();
        $data['update_time'] = time();
        if($user_info['is_first_login'] == 1) {
            $data['is_first_login'] = 2;
        }
        $reslut = db::name('user')->update($data);

        $return_res = [];
        $return_res['uid'] = $user_info['uid'];
        $return_res['head_pic'] = localpath_to_netpath($user_info['head_pic']);
        $return_res['user_name'] = $user_name;
        $return_res['login_token'] = $login_token_string;
        $return_res['user_sig'] = $user_sigs;
        $return_res['is_real'] = $user_info['is_real'];
        $return_res['system_type'] = $user_info['system_type'];
        $return_res['nick_name'] = mb_convert_encoding(base64_decode($user_info['base64_nick_name']), 'UTF-8', 'UTF-8');
        if (!$reslut) {
            return ['code' => 201, 'msg' => '登录失败', 'data' => null];
        } else {
            return  ['code' => 200, 'msg' => '登录成功', 'data' => $return_res];
        }
    }
    //过滤靓号
    public function get_available_uid($uid = 0)
    {
        // if (empty($uid)) {
        //     $uid = db::name('user')->order('uid desc')->value('uid');
        //     if(empty($uid)){
        //         $uid = 10001;
        //     }
        // }
        $uid = mt_rand(10001, 99999);
        // $uid = $uid + mt_rand(1, 10);
        $config = get_system_config();
        $filter_uid_arr = explode(",", $config['filter_uid']);
        if (in_array($uid, $filter_uid_arr)) {
            $this->get_available_uid();
        }
        $user_info = db::name('user')->field('uid')->where('uid', $uid)->find();
        if (!empty($user_info)) {
            $this->get_available_uid($user_info['uid']);
        } else {
            return $uid;
        }
    }

    //获取用户信息
    public function get_user_info($uid)
    {
        // echo 333;die;
        $config = model('admin/Config')->get_system_config();
        $map = [];
        $map[] = ['uid', '=', $uid];
        $user_info = db::name('user')->field('uid,user_name,base64_nick_name,head_pic,sex,birthday,money,frozen_money,integral,autograph,hobby,country,province,city,constellation,user_sig,is_vip,vip_end_time,reg_code,is_real,is_customer,real_name,card_id')->where($map)->find();
        if (empty($user_info)) {
            return ['code' => 201, 'msg' => '用户信息不存在', 'data' => null];
        }
        $user_info['head_pic'] = localpath_to_netpath($user_info['head_pic']);
        $user_info['nick_name'] = mb_convert_encoding(base64_decode($user_info['base64_nick_name']), 'UTF-8', 'UTF-8');
        $web_site = Db::name('config')->where('key_title', 'web_site')->value('key_value');
        //邀请链接
        $user_info['invite_url'] = $web_site . '/index/index/register?reg_code=' . $user_info['reg_code'];
        $user_info['user_withdraw_rate'] = $config['user_withdraw_rate'];
        $user_info['customer_id'] = $config['customer_uid'];
        // if(strlen($user_info['card_id']) >= 14) {
        //     $user_info['card_id'] = substr_replace($user_info['card_id'], '***********', 4, 13);
            
            
        // }
        $user_info['card_id'] = '';
        $user_info['identity1'] = '';
        $user_info['identity2'] = '';
        $user_real_name_info = Db::name('user_real_name')->where('uid', $uid)->find();
        if($user_real_name_info) {
            if($user_real_name_info['status'] == 1) {
                $user_info['is_real'] = 3;
            }
            $user_info['card_id'] = substr_replace($user_real_name_info['card_id'], '***********', 4, 13);
            $user_info['identity1'] = $user_real_name_info['identity1'];
            $user_info['identity2'] = $user_real_name_info['identity2'];
        }
        // dump($user_info);die;
        return ['code' => 200, 'msg' => '获取成功', 'data' => $user_info];
    }


    //获取星座
    public function get_user_constellation($birthday)
    {
        $birthday = date('Y-m-d', strtotime($birthday));
        if (empty($birthday)) {
            return ['code' => 201, 'msg' => '生日日期格式非法', 'data' => null];
        }
        $birthday_arr = explode('-', $birthday);
        $month = $birthday_arr[1];
        $day = $birthday_arr[2];
        $xingzuo = '';
        // 检查参数有效性
        if ($month < 1 || $month > 12 || $day < 1 || $day > 31) {
            return $xingzuo;
        }

        if (($month == 1 && $day >= 20) || ($month == 2 && $day <= 18)) {
            $xingzuo = "水瓶座";
        } else if (($month == 2 && $day >= 19) || ($month == 3 && $day <= 20)) {
            $xingzuo = "双鱼座";
        } else if (($month == 3 && $day >= 21) || ($month == 4 && $day <= 19)) {
            $xingzuo = "白羊座";
        } else if (($month == 4 && $day >= 20) || ($month == 5 && $day <= 20)) {
            $xingzuo = "金牛座";
        } else if (($month == 5 && $day >= 21) || ($month == 6 && $day <= 21)) {
            $xingzuo = "双子座";
        } else if (($month == 6 && $day >= 22) || ($month == 7 && $day <= 22)) {
            $xingzuo = "巨蟹座";
        } else if (($month == 7 && $day >= 23) || ($month == 8 && $day <= 22)) {
            $xingzuo = "狮子座";
        } else if (($month == 8 && $day >= 23) || ($month == 9 && $day <= 22)) {
            $xingzuo = "处女座";
        } else if (($month == 9 && $day >= 23) || ($month == 10 && $day <= 23)) {
            $xingzuo = "天秤座";
        } else if (($month == 10 && $day >= 24) || ($month == 11 && $day <= 22)) {
            $xingzuo = "天蝎座";
        } else if (($month == 11 && $day >= 23) || ($month == 12 && $day <= 21)) {
            $xingzuo = "射手座";
        } else if (($month == 12 && $day >= 22) || ($month == 1 && $day <= 19)) {
            $xingzuo = "摩羯座";
        }
        return ['code' => 200, 'msg' => '获取成功', 'data' => $xingzuo];
    }
    
    public function modify_user_info($uid, $nick_name, $birthday, $sex, $head_pic, $city, $autograph, $hobby)
    {
        $user_info = Db::name('user')->find($uid);
        
        $data = [];
        $data['uid'] = $uid;
        if (!empty($nick_name)) {
            $nick_name = $this->get_available_nick_name($uid,$nick_name);
            $data['nick_name'] = $nick_name;
            $data['base64_nick_name'] = base64_encode($nick_name);
            if(in_array($user_info['system_type'], [3,4])) {
                if($data['base64_nick_name'] != $user_info['base64_nick_name']) {
                    return ['code' => 201, 'msg' => '修改失败', 'data' => null];
                }
                
            }
        }
        if (!empty($birthday)) {
            $data['birthday'] = $birthday;
            $constellation = $this->get_user_constellation($birthday);
            $data['constellation'] = $constellation['data'];
        }
        if (!empty($head_pic)) {
            $data['head_pic'] = $head_pic;
        }
        if (!empty($city)) {
            $data['city'] = $city;
        }

        if (!empty($autograph)) {
            $data['autograph'] = $autograph;
        }
        if(!empty($hobby)){
            $data['hobby'] = $hobby;
        }
        $validate = validate('admin/User');
        $reslut = $validate->scene('apiEditInfo')->check($data);
        if ($reslut !== true) {
            return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
        }
        $data['sex'] = $sex;
        $data['update_time'] = time();
        // $reslut = model('admin/user')->save($data);
        $reslut = db::name('user')->update($data);
        if ($reslut) {
            $user_info = db::name('user')->field('uid,user_name,base64_nick_name,head_pic,sex,birthday,money,frozen_money,integral,autograph,hobby,country,province,city,constellation')->find($uid);
            $user_info['nick_name'] = mb_convert_encoding(base64_decode($user_info['base64_nick_name']), 'UTF-8', 'UTF-8');
            $user_info['head_pic'] = localpath_to_netpath($user_info['head_pic']);
            return ['code' => 200, 'msg' => '修改成功', 'data' => $user_info];
        } else {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        }
    }
    
    //昵称重复变更
    public function get_available_nick_name($uid, $nick_name, $number = 0){
        if(!empty($number)){
            $nick_names = $nick_name.$number;
        }else{
            $nick_names = $nick_name;
        }
        
        $base64_nick_name = base64_encode($nick_names);
        
        //是否有重复
        $map = [];
        $map[] = ['uid', 'neq', $uid];
        $map[] = ['base64_nick_name', '=', $base64_nick_name];
        $user_info = db::name('user')->where($map)->find();
        if(!empty($user_info)){
            $number += 1;
            $nick_name = $this->get_available_nick_name($uid,$nick_name, $number);
            return $nick_name;
        }else{
            if(empty($number)){
                return $nick_name;
            }else{
                return $nick_name.$number;
            }
            
        }
        
    }
    
    //退出登录
    public function log_out($uid){
        $user_info = db::name('user')->find($uid);
        if($user_info){
            $map = [];
            $map[] = ['uid', '=', $uid];
            $data = [];
            $data['login_token'] = '';
          db::name('user')->where($map)->update($data); 
        }
        $this->quit_rongyun_group($uid);
        return ['code' => 200, 'msg' => '退出成功', 'data' => null];
    }
    
    //
    public function close_user_vip_end_time(){
        $now_time = time();
        $map = [];
        $map[] = ['is_vip', '=', 1];
        $map[] = ['vip_end_time', '<', $now_time];
        $user_list = db::name('user')->where($map)->select();
        if(!empty($user_list)){
            $update = [];
            $update['is_vip'] = 2;
            $update['vip_end_time'] = 0;
            $update['update_time'] = time();
            $reslut = db::name('user')->where($map)->update($update);
            if(!$reslut){
                return ['code' => 201, 'msg' => '失败', 'data' => null];
            }
        }
        
        return ['code' => 200, 'msg' => '成功', 'data' => null];
    }
    
    //检查用户登录状态
    public function check_login_token($login_token)
    {
        if (empty($login_token)) {
            return ['code' => 201, 'msg' => '登录失效', 'data' => null];
        }
        $map = [];
        $map[] = ['login_token', '=', $login_token];
        $user_info = db::name('user')->where($map)->find();
        if (empty($user_info)) {
            return ['code' => 201, 'msg' => '登录失效', 'data' => null];
        } elseif ($user_info['login_status'] != 1) {
            return ['code' => 202, 'msg' => '用户已被封禁', 'data' => null];
        } else {
            return ['code' => 200, 'msg' => '登录成功', 'data' => $user_info['uid']];
        }
    }
    
    
    //用户提现
    public function user_withdrawal($uid, $address, $money, $trade_password)
    {
        $user_info = Db::name('user')->find($uid);
    
        if(empty($user_info['trade_password'])) {
            return ['code' => 602, 'msg' => '请设置交易密码', 'data' => null];
        }
        if(empty($trade_password)) {
            return ['code' => 201, 'msg' => '请输入支付密码', 'data' => null];
        }
        if(md5($trade_password) != $user_info['trade_password']) {
            return ['code' => 201, 'msg' => '支付密码错误', 'data' => null];
        }
        $config = model('admin/Config')->get_system_config();
        if ($money < $config['min_withdraw_amount']) {
            return ['code' => 201, 'msg' => '提现金额不能小于' . $config['min_withdraw_amount'], 'data' => null];
        }
        // $map = [];
        // $map[] = ['uid', '=', $uid];
        // $map[] = ['status', '=', 2];
        // $user_player_info = Db::name('user_player')->where($map)->find();
        // if (empty($user_player_info)) {
        //     return ['code' => 201, 'msg' => '请先通过陪玩资质认证，即可提现'];
        // }
        $user_withdraw_info = Db::name('user_withdrawal')->where(['uid' => $uid, 'status' => 1])->find();
        if (!empty($user_withdraw_info)) {
            return ['code' => 201, 'msg' => '您已有待处理提现记录', 'data' => null];
        }

        // $withdraw_week = get_system_config('withdraw_week');
        // $withdraw_week = explode(',', $withdraw_week);
        // if (!in_array(date('w'), $withdraw_week)) {
        //     $weekarray = ["日", "一", "二", "三", "四", "五", "六"];
        //     $msg = "请于";
        //     foreach ($withdraw_week as $k => $v) {
        //         $msg .= "周" . $weekarray[$v];
        //     }
        //     $msg .= "进行提现";
        //     return ['code' => 201, 'msg' => $msg, 'data' => null];
        // }

        if ($money > $user_info['money']) {
            return ['code' => 201, 'msg' => '余额不足', 'data' => null];
        }

        Db::startTrans();
        try {
            $order_sn = $this->create_user_withdrawal_order_sn();
            $data = [];
            $data['order_sn'] = $order_sn;
            $data['uid'] = $uid;
            $data['address'] = $address;
            $data['money'] = $money;
            $data['general_money'] =  $money * (1 - $config['user_withdraw_rate']); //到账金额
            $data['service_money'] = $money - $data['general_money'];
            $data['remarke'] =  '';
            $data['status'] = 1;
            $data['add_time'] = time();

            $user_withdrawal_wid = DB::name('user_withdrawal')->insertGetId($data);
            if (!$user_withdrawal_wid) {
                Db::rollback();
                return ['code' => 201, 'msg' => "请重试", 'data' => null];
            }
            //扣除账户余额
            $reslut = model('admin/User')->change_user_money_by_uid($uid, -$money, 5, "余额提现" , $uid, $user_withdrawal_wid);
            if ($reslut['code'] != 200) {
                Db::rollback();
                return ['code' => 201, 'msg' => $reslut['msg'], 'data' => null];
            }
            //增加冻结余额
            $reslut = Db::name('user')->where('uid', $user_info['uid'])->setInc('frozen_money', $money);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => "请重试", 'data' => null];
            }
            // 提交事务
            Db::commit();
            return ['code' => 200, 'msg' => "提现成功", 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            Db::rollback();
            // dump($e->getMessage());
            // die;
            return ['code' => 201, 'msg' => "提现失败", 'data' => null];
        }
    }
    
    
    //生成订单号
    private function create_user_withdrawal_order_sn()
    {
        $order_sn = 'TX' . date('YmdHis') . mt_rand(10000, 99999);
        $map = [];
        $map[] = ['order_sn', '=', $order_sn];
        $reslut = db::name('user_withdrawal')->where($map)->find();
        if (empty($reslut)) {
            return $order_sn;
        } else {
            $this->create_user_withdrawal_order_sn();
        }
    }
    
    //实名信息提交
    public function real_name_authentication($uid, $real_name, $card_id,$identity1,$identity2)
    {

        if (empty($real_name)) {
            return ['code' => 201, 'msg' => '真实姓名必须', 'data' => null];
        }
        if (empty($card_id)) {
            return ['code' => 201, 'msg' => '身份证号必须', 'data' => null];
        }

        if (empty($identity1)) {
            return ['code' => 201, 'msg' => '身份证正面必须', 'data' => null];
        }

        if (empty($identity2)) {
            return ['code' => 201, 'msg' => '身份证反面必须', 'data' => null];
        }
        $map = [];
        $map[] = ['card_id', '=', $card_id];
        $user_info = db::name('user')->where($map)->find();

        if (!empty($user_info) && $user_info['uid'] != $uid) {
            return ['code' => 201, 'msg' => '该身份证号已被使用', 'data' => null];
        }

        $user_info = db::name('user')->find($uid);
        if ($user_info['is_real'] == 1) {
            return ['code' => 201, 'msg' => '已实名', 'data' => null];
        }

        if ($user_info['is_real'] == 3) {
            return ['code' => 201, 'msg' => '实名审核中,请勿重复提交', 'data' => null];
        }

        $map = [];
        $map[] = ['uid', '=', $uid];
        $map[] = ['is_real', '=', 0];
        $data = [];
        $data['is_real'] = 3;
        $data['real_name'] = $real_name;
        $data['card_id'] = $card_id;
        // $data['identity1'] = $identity1;
        // $data['identity2'] = $identity2;
        $data['update_time'] = time();
        
        // 启动事务
        Db::startTrans();
        try {
            //提交认证
            $reslut = db::name('user')->where($map)->update($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '提交失败', 'data' => null];
            }
            
            //添加审核记录
            unset($data['is_real']);
            unset($data['update_time']);
            
            $data['uid'] = $uid;
            $data['identity1'] = $identity1;
            $data['identity2'] = $identity2;
            $data['add_time'] = time();
            $data['status'] = 1;
            $reslut = db::name('user_real_name')->insert($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '提交失败', 'data' => null];
            }
            // 提交事务
            Db::commit();
            return ['code' => 200, 'msg' => '提交成功', 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            dump($e);
            Db::rollback();
            return ['code' => 201, 'msg' => '提交失败', 'data' => null];
        }

    }
    
    // 提现记录
    public function get_user_money_list($uid, $page, $page_limit)
    {
        $page = intval($page);
        $page_limit = $page_limit < 30 ? $page_limit : 30;
        $map = [];
        $map[] = ['uid', '=', $uid];
        $list = db::name('user_money_log')->field('change_type,change_value,remarks,add_time')->where($map)->order('log_id desc')->page($page, $page_limit)->select();
        foreach ($list as $k => &$v) {
            $v['change_type_desc'] = '余额';
        }
        return ['code' => 200, 'msg' => '获取成功', 'data' => $list];
    }
    
    public function modify_password($uid, $user_name, $password)
    {
        
        $map = [];
        $map[] =  ['uid', '=', $uid];
        $user_info = Db::name('user')->where($map)->find();
        if(empty($user_info)) {
            return ['code' => 201, 'msg' => '获取成功', 'data' => null];
        }
        if($user_name != $user_info['user_name']) {
            return ['code' => 201, 'msg' => '手机号错误', 'data' => null];
        }
        $data = [];
        $data['password'] = md5($password);
        $data['login_token'] = '';
        $reslut = db::name('user')->where('uid', $uid)->update($data);
        if ($reslut) {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        }
    }
    public function modify_password_account($uid, $password, $old_password)
    {
        
        $map = [];
        $map[] =  ['uid', '=', $uid];
        $user_info = Db::name('user')->where($map)->find();
        if(empty($user_info)) {
            return ['code' => 201, 'msg' => '获取成功', 'data' => null];
        }
        if(md5($old_password) != $user_info['password']) {
            return ['code' => 201, 'msg' => '老密码不正确', 'data' => null];
        }
        
        $data = [];
        $data['password'] = md5($password);
        $data['login_token'] = '';
        $reslut = db::name('user')->where('uid', $uid)->update($data);
        if ($reslut) {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        }
    }
    //修改交易密码
    public function modify_trade_password($uid, $trade_password)
    {
        $data = [];
        $data['uid'] = $uid;
        $data['trade_password'] = $trade_password;
        $validate = validate('admin/User');
        $reslut = $validate->scene('apiEditTradePassword')->check($data);
        if ($reslut !== true) {
            return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
        }
        $data['trade_password'] = md5($trade_password);
        $reslut = db::name('user')->update($data);
        if ($reslut) {
            return ['code' => 200, 'msg' => '修改成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '修改失败', 'data' => null];
        }
    }
    
    
    /**
     * 执行账号注销
     **/
    public  function  cancel_account($uid){
        return ['code' => 200, 'msg' => '注销成功', 'data' => null];
        //查找用户账号
        $account = db::name('user')->where(['uid'=>$uid])->find();
        if(empty($account)){
            return ['code' => 201, 'msg' => '账号不存在!', 'data' => null];
        }
        //账号状态
        if(in_array($account['login_status'],[3])){
            return ['code' => 201, 'msg' => '您的账号已经注销过了!', 'data' => null];
        }
        
        //执行账号注销
        $new_user_name = $account['user_name'] . '_zx_' . date('YmdHis');
        db::name('user')->where(['uid'=>$uid])->update(['login_status'=>3,'login_token'=>'','user_name'=>$new_user_name]);
        return ['code' => 200, 'msg' => '注销成功', 'data' => null];
    }
}
