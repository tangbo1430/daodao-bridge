<?php

namespace app\api\model;


use think\Db;
use think\Model;
use think\facade\Env;
// use Tencent\TLSSigAPI;
use Tencent\TLSSigAPIv2;

class Tencent extends Model
{
    
    //获取腾讯IM UserSig信息
    public function tencent_user_sig_info($uid){
        // $config = model('admin/Config')->get_system_config();
        // $tencentyun_im_appid = $config['tencentyun_im_appid'];
        // $tencentyun_im_public_key = $config['tencentyun_im_public_key'];
        // $tencentyun_im_private_key = $config['tencentyun_im_private_key'];
        // $tencent = new \Tencent\TLSSigAPI();
        // $tencent->SetAppid($tencentyun_im_appid);
        // $tencent->SetPrivateKey($tencentyun_im_private_key);
        // $user_sig = $tencent->genSig($uid);
        // return $user_sig;
        $config = model('admin/Config')->get_uncache_system_config();
        $tencentyun_im_appid = getenv('IM_SDK_APP_ID') ?: ($config['tencentyun_im_appid'] ?? '');
        $tencentyun_im_key = getenv('IM_SDK_SECRET_KEY') ?: ($config['tencentyun_im_key'] ?? '');
        // dump($tencentyun_im_appid);die;
        $tencent = new \Tencent\TLSSigAPIv2($tencentyun_im_appid, $tencentyun_im_key);
        $user_sig = $tencent->genUserSig($uid);
        return $user_sig;
    }
    
    //查询账号
    public function account_check($uid){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        
        $postUrl = 'https://console.tim.qq.com/v4/im_open_login_svc/account_check?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        
        $curlPost = array(
            'CheckItem'   => array(
                array(
                    'UserID'  => $uid,  //用户id
                    ),
                ),
            );
            
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        dump($reslut);exit;
    }
    
    //拉入黑名单
    public function black_list_add($uid, $receive_uid){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        
        $postUrl = 'https://console.tim.qq.com/v4/sns/black_list_add?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        
        $curlPost = array(
            'From_Account'  => $uid,
            'To_Account'     => $receive_uid,
            );
            
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        
        if($reslut['ActionStatus'] == 'OK'){
            return ['code' => 200, 'msg' => '添加成功', 'data' => null];
        }else{
            return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
        }
    }
    
    //移除黑名单
    public function black_list_delete($uid, $receive_uid){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        
        $postUrl = 'https://console.tim.qq.com/v4/sns/black_list_delete?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        
        $curlPost = array(
            'From_Account'  => $uid,
            'To_Account'     => $receive_uid,
            );
            
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        // TODO(local-dev): 腾讯 IM 联调完成后确认是否保留该失败保护。
        if (!is_array($reslut)) {
            return ['code' => 201, 'msg' => '腾讯 IM 接口请求失败', 'data' => null];
        }
        if($reslut['ActionStatus'] == 'OK'){
            return ['code' => 200, 'msg' => '移除成功', 'data' => null];
        }else{
            return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
        }
    }
    
    //发送群内系统消息
    public function send_group_system_notification($rid, $message_data){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/send_group_system_notification?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        
        $curlPost = array(
            'GroupId'   => $rid,
            'Content'   => $message_data,
            );
            
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
    }
    
    //发送群内普通消息
    public function send_group_msg($rid, $message_data){
        
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        $random = time().rand(111,999);
        
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/send_group_msg?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        
        $curlPost = array(
            'GroupId'   => $rid,
            'Random'    => $random,
            'MsgBody'   => array(
                array(
                    'MsgType'       => 'TIMTextElem',
                    'MsgContent'    => array(
                        'Text' => $message_data,
                        ),
                    ),
                ),
            );
        
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        // $txt = date('Y-m-d H:i:s').'测试';
        // error_log($txt, 3, 'a.txt');
        return $reslut;
        
    }

    // Send a persisted first message so initial members receive a recent conversation immediately.
    public function send_group_created_message($groupId, $ownerAccount)
    {
        $config = model('admin/Config')->get_system_config();
        $appId = getenv('IM_SDK_APP_ID') ?: ($config['tencentyun_im_appid'] ?? '');
        $adminAccount = getenv('IM_ADMIN_ACCOUNT') ?: 'administrator';
        $adminSig = $this->tencent_user_sig_info($adminAccount);
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/send_group_msg?sdkappid=' . $appId
            . '&identifier=' . urlencode($adminAccount)
            . '&usersig=' . urlencode($adminSig)
            . '&random=' . random_int(10000000, 99999999)
            . '&contenttype=json';
        $payload = $this->buildGroupCreatedMessagePayload($groupId, $ownerAccount);

        try {
            $result = $this->tencent_post_url($postUrl, json_encode($payload, JSON_UNESCAPED_UNICODE));
            if (is_array($result) && ($result['ActionStatus'] ?? '') === 'OK') {
                return ['code' => 200, 'msg' => '发送成功', 'data' => $result];
            }
            $message = is_array($result)
                ? (($result['ErrorInfo'] ?? '') !== '' ? $result['ErrorInfo'] : ($result['ErrorCode'] ?? '腾讯IM群初始化消息发送失败'))
                : '腾讯IM网络请求失败';
            return ['code' => 201, 'msg' => $message, 'data' => null];
        } catch (\Throwable $e) {
            return ['code' => 201, 'msg' => $e->getMessage(), 'data' => null];
        }
    }

    private function buildGroupCreatedMessagePayload($groupId, $ownerAccount)
    {
        return [
            'GroupId' => (string) $groupId,
            'From_Account' => (string) $ownerAccount,
            'Random' => random_int(100000, 999999999),
            'MsgBody' => [[
                'MsgType' => 'TIMTextElem',
                'MsgContent' => ['Text' => '群聊已创建'],
            ]],
        ];
    }
    
    //指定用户发送消息给注册用户
    public function user_sendmsg($uid, $receive_uid, $message, $machine_type = 2){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        
        $postUrl = 'https://console.tim.qq.com/v4/openim/sendmsg?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        
        $curlPost = array(
            'SyncOtherMachine'      => $machine_type,
            'From_Account'          => strval($uid),
            'To_Account'            => strval($receive_uid),
            'MsgRandom'             => time(),
            'MsgBody'               => array(
                array(
                    'MsgType'       => 'TIMTextElem',
                    'MsgContent'    => array(
                        'Text' => $message,
                        ),
                    ),
                ),
            );
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        return $reslut;
    }
    
    //获取APP中的所有群组
    public function get_appid_group_list(){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/get_appid_group_list?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        
        $curlPost = array(
            'Limit' => 20,
            'Next'  => 0,
            );
        
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
    }
    
    public function txt(){
        $postUrl = 'https://app.yayinyy.com/api/Agora/get_sstoken';
        
        $curlPost = array(
            'code' => '44863d01cd628e583efeb3b9eda510fd',
            );
        
        $curlPost = json_encode($curlPost);
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        return $reslut;
    }
    
    //腾讯IM请求封装方法
    public function tencent_post_url($postUrl, $curlPost){
        // Overseas SDKAppIDs must use their regional REST domain. Keep legacy
        // callers unchanged and normalize their hard-coded China endpoint here.
        $restBaseUrl = rtrim(getenv('IM_REST_BASE_URL') ?: 'https://adminapisgp.im.qcloud.com', '/');
        $postUrl = preg_replace('#^https://console\.tim\.qq\.com#', $restBaseUrl, $postUrl);

        $headerArray =array(
            "Content-type:application/json",
            "Accept:application/json",
            );
        
        $ch = curl_init();//初始化curl
        curl_setopt($ch, CURLOPT_URL, $postUrl);//抓取指定网页
        // curl_setopt($ch, CURLOPT_HEADER, 0);//设置header 
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1);//要求结果为字符串且输出到屏幕上
        curl_setopt($ch, CURLOPT_POST, 1);//post提交方式
        curl_setopt($ch, CURLOPT_POSTFIELDS, $curlPost);
        curl_setopt($ch, CURLOPT_HTTPHEADER, $headerArray);
        curl_setopt($ch, CURLOPT_TIMEOUT, 20);
        $data = curl_exec($ch);//运行curl
        curl_close($ch);
        if ($data) {
            $data = json_decode($data, true);
        }
        // dump($data);exit;
        return $data;
    }
    //创建家族
    public function  create_family($group_id,$uid, $group_name, $memberIds = []){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);

        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/create_group?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $group_id = strval($group_id);
       
        $uid = strval($uid);
        $curlPost = array(
            'Owner_Account'     => $uid,
            'Type'              => 'Public',
            'GroupId'           => $group_id,
            'Name'              => $group_name,
            'InviteJoinOption' => 'FreeAccess'
        );
        if (is_array($memberIds) && !empty($memberIds)) {
            $curlPost['MemberList'] = [];
            foreach ($memberIds as $memberId) {
                $curlPost['MemberList'][] = ['Member_Account' => (string) $memberId];
            }
        }
       
        try{
            $curlPost = json_encode($curlPost);

            $reslut = $this->tencent_post_url($postUrl, $curlPost);
        // dump($reslut);die;
            if(is_array($reslut) && isset($reslut['ActionStatus']) && $reslut['ActionStatus'] == 'OK'){
                
                return ['code' => 200, 'msg' => '添加成功', 'data' => null];
            }else{
                
                $message = is_array($reslut)
                    ? (isset($reslut['ErrorInfo']) && $reslut['ErrorInfo'] !== '' ? $reslut['ErrorInfo'] : ($reslut['ErrorCode'] ?? '腾讯IM创建群失败'))
                    : '腾讯IM网络请求失败';
                return ['code' => 201, 'msg' => $message, 'data' => null];
            }
        }catch(\Exception $e){
            
            return ['code' => 201, 'msg' => $e->getMessage(), 'data' => null];
        }


    }
    //增加群成员
    public function  create_group_member($family_id,$uid){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);

        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/add_group_member?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $family_id = strval($family_id);
        $member_uid = strval($uid);
        
        $curlPost = [
            'GroupId' => $family_id,
            'Silence' => 0,
            'MemberList' => [
                ['Member_Account' => $member_uid]
            ]
        ];
        
        try{
            $curlPost = json_encode($curlPost);

            $reslut = $this->tencent_post_url($postUrl, $curlPost);
//         dump($reslut);die;
            if($reslut['ActionStatus'] == 'OK'){
                
                return ['code' => 200, 'msg' => '添加成功', 'data' => null];
            }else{
                
                return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
            }
        }catch (\Exception $e){
            
            return ['code' => 201, 'msg' => $e->getMessage(), 'data' => null];
        }

    }
    
    //获取群成员
    public function get_group_member_list($group_id, $page, $page_limit)
    {
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        $page = $page - 1;
        $limit = $page * $page_limit;
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/get_group_member_info?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $group_id = strval($group_id);
        if($page_limit == 0) {
            $curlPost = [
                'GroupId' => $group_id,
            ];
        } else {
            $curlPost = [
                'GroupId' => $group_id,
                'Limit' => $limit,
                'Offset' => $page,
            ];
        }
        
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        // TODO(local-dev): 腾讯 IM 联调完成后确认是否保留该失败保护。
        if (!is_array($reslut)) {
            return ['code' => 201, 'msg' => '腾讯 IM 接口请求失败', 'data' => null];
        }
        if($reslut['ActionStatus'] == 'OK'){
            return ['code' => 200, 'msg' => '添加成功', 'data' => $reslut['MemberList']];
        }else{
            return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
        }
    }
    //获取群成员详细信息
    public function get_group_member_info($group_id, $uid_arr, $role_filter)
    {
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
     
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/get_specified_group_member_info?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $group_id = strval($group_id);
        
        $curlPost = [
            'GroupId' => $group_id,
            "Member_List_Account" => $uid_arr,
            "MemberRoleFilter" => $role_filter
        ];
        
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        // dump($reslut);die;
        if($reslut['ActionStatus'] == 'OK'){
            if(isset($reslut['MemberList'])) {
                return ['code' => 200, 'msg' => '添加成功', 'data' => $reslut['MemberList']];
            } else {
                return ['code' => 200, 'msg' => '添加成功', 'data' => []];
            }
            
        }else{
            return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
        }
    }
    //修改群基础资料
    public function modify_group_info($group_id, $group_name)
    {
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
     
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/modify_group_base_info?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $group_id = strval($group_id);
        
        $curlPost = [
            'GroupId' => $group_id,
            "Name" => $group_name,
        ];
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        if($reslut['ActionStatus'] == 'OK'){
            return ['code' => 200, 'msg' => '修改成功', 'data' => ''];
        }else{
            return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
        }
    }
    //修改群主
    public function change_group_owner($group_id, $uid)
    {
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
     
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/change_group_owner?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $group_id = strval($group_id);
        
        $curlPost = [
            'GroupId' => $group_id,
            "NewOwner_Account" => $uid,
        ];
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        if($reslut['ActionStatus'] == 'OK'){
            return ['code' => 200, 'msg' => '修改成功', 'data' => ''];
        }else{
            return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
        }
    }
    
    //解散家族
    public function destroy_group($group_id){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        $group_id = strval($group_id);
      
        //解散群主
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/destroy_group?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $curlPost = array(
            'GroupId'           => $group_id,
        );
        $curlPost = json_encode($curlPost);
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        if($reslut['ActionStatus'] == 'OK'){
            db::commit();
            return ['code' => 200, 'msg' => '操作成功', 'data' => null];
        }else{
            db::rollback();
            return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
        }
    }
    //批量增加群成员
    public function  batch_create_group_member($family_id, $uid_arr){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);

        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/add_group_member?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $family_id = strval($family_id);
        
        $member_list = [];
        foreach($uid_arr as $v) {
            $member_list[] = ['Member_Account' => strval($v)];
        }
        $curlPost = [
            'GroupId' => $family_id,
            'Silence' => 0,
            'MemberList' => $member_list
        ];
        
        try{
            $curlPost = json_encode($curlPost);

            $reslut = $this->tencent_post_url($postUrl, $curlPost);
//         dump($reslut);die;
            if($reslut['ActionStatus'] == 'OK'){
                
                return ['code' => 200, 'msg' => '添加成功', 'data' => null];
            }else{
                
                return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
            }
        }catch (\Exception $e){
            
            return ['code' => 201, 'msg' => $e->getMessage(), 'data' => null];
        }

    }
    
    //删除群成员
    public function  delete_group_member($group_id,$uid){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);

        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/delete_group_member?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $family_id = strval($family_id);
        $member_uid = strval($uid);
        $curlPost = [
            'GroupId' => $family,
            'Silence' => 0,
            "MemberToDel_Account" => $member_uid
        ];
        
        try{
            $curlPost = json_encode($curlPost);
            $reslut = $this->tencent_post_url($postUrl, $curlPost);
            if($reslut['ActionStatus'] == 'OK'){
                return ['code' => 200, 'msg' => '删除成功', 'data' => null];
            }else{
                return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
            }
        }catch (\Exception $e){
            return ['code' => 201, 'msg' => $e->getMessage(), 'data' => null];
        }

    }
    //发送群内自定义消息
    public function send_group_msg_txt($message_data, $rid){
        
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = model('api/Tencent')->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        $random = time().rand(111,999);
        $from_account = strval($from_account);
        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/send_group_msg?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $curlPost = array(
            'GroupId'   => $rid,
            'Random'    => $random,
            'MsgBody'   => array(
                array(
                    'MsgType'       => 'TIMTextElem',
                    'MsgContent'    => array(
                        'Text' => $message_data,
                    ),
                ),
            ),
        );
        
        $curlPost = json_encode($curlPost);
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        return $reslut;
    }
    
    //增加群成员
    public function  group_msg_recall($family_id,$msg_seq){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);

        $postUrl = 'https://console.tim.qq.com/v4/group_open_http_svc/group_msg_recall?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        $family_id = strval($family_id);
        // $member_uid = strval($uid);
        
        $curlPost = [
            'GroupId' => $family_id,
            'MsgSeqList' => [
                ['MsgSeq' => $msg_seq]
            ]
        ];
        
        try{
            $curlPost = json_encode($curlPost);

            $reslut = $this->tencent_post_url($postUrl, $curlPost);
//         dump($reslut);die;
            if($reslut['ActionStatus'] == 'OK'){
                
                return ['code' => 200, 'msg' => '撤回成功', 'data' => null];
            }else{
                
                return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
            }
        }catch (\Exception $e){
            
            return ['code' => 201, 'msg' => $e->getMessage(), 'data' => null];
        }

    }
    
    //添加好友
    public function friend_check($uid, $friend_uid){
        $config = model('admin/Config')->get_system_config();
        $tencentyun_im_appid = $config['tencentyun_im_appid'];
        $im_admin = 'administrator';
        $admin_sig = $this->tencent_user_sig_info($im_admin);
        $rand = rand(111111111,9999999999);
        
        $postUrl = 'https://console.tim.qq.com/v4/sns/friend_check?sdkappid='.$tencentyun_im_appid.'&identifier='.$im_admin.'&usersig='.$admin_sig.'&random='.$rand.'&contenttype=json';
        // dump($friend_uid);
        $curlPost = array(
            'From_Account'  => strval($uid),
            'To_Account'     => $friend_uid,
            'CheckType' => 'CheckResult_Type_Single'
            );
        // dump($curlPost);die;    
        $curlPost = json_encode($curlPost);
        
        $reslut = $this->tencent_post_url($postUrl, $curlPost);
        
        if($reslut['ActionStatus'] == 'OK'){
            $item_info = $reslut['InfoItem'];
            if(isset($item_info[0])) {
                if($item_info[0]['Relation'] == 'CheckResult_Type_NoRelation') {
                    return ['code' => 200, 'msg' => '可以添加', 'data' => null];
                } else {
                    return ['code' => 201, 'msg' => '已添加好友', 'data' => null];
                }
            } else {
                return ['code' => 201, 'msg' => '腾讯im校验错误', 'data' => null];
            }
            
        }else{
            return ['code' => 201, 'msg' => $reslut['ErrorCode'], 'data' => null];
        }
    }
}
