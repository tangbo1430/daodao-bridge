<?php

namespace app\api\model;

use think\Db;
use think\Model;

class GroupV extends Model
{
    public function create_group($uid, $group_name, $memberIds = [], $requestId = '')
    {
        $group_name = trim((string) $group_name);
        if ($group_name === '') {
            return ['code' => 201, 'msg' => '请输入群组名称', 'data' => null];
        }
        if (mb_strlen($group_name, 'UTF-8') > 30) {
            return ['code' => 201, 'msg' => '群组名称不能超过30个字符', 'data' => null];
        }
        $requestId = trim((string) $requestId);
        if ($requestId !== '' && !preg_match('/^[A-Za-z0-9_-]{8,64}$/', $requestId)) {
            return ['code' => 201, 'msg' => '建群请求标识无效', 'data' => null];
        }
        if ($requestId === '') {
            $requestId = 'legacy_' . bin2hex(random_bytes(16));
        }

        $normalized = $this->normalizeInitialMembers($uid, $memberIds);
        if (isset($normalized['error'])) {
            return ['code' => 201, 'msg' => $normalized['error'], 'data' => null];
        }
        $memberIds = $normalized['members'];
        foreach ($memberIds as $memberId) {
            if ($memberId === 'u_10000' || preg_match('/^[0-9a-f]{32}$/i', $memberId)) {
                // This legacy APK endpoint must use the same native-first flow
                // as /api/bridge/group_create. Never create a Tencent-only group
                // when a native contact was selected. Bridge resolves identities.
                try {
                    $response = (new \app\imaccess\service\BridgeService())->createGroupForUser($uid, $group_name, $memberIds, $requestId);
                    $body = $response['body'];
                    if ((int) $response['http_status'] !== 200 || (int) ($body['code'] ?? 502) !== 200) {
                        return ['code' => (int) ($body['code'] ?? 502), 'msg' => (string) ($body['msg'] ?? '创建群聊失败，请稍后重试'), 'data' => null];
                    }
                    $id = (string) ($body['data']['group_id'] ?? '');
                    if (strpos($id, 'dd_') !== 0 || strlen($id) <= 3) {
                        return ['code' => 502, 'msg' => '群创建结果确认中，请重试', 'data' => null];
                    }
                    return ['code' => 200, 'msg' => '创建成功', 'data' => ['group_id' => $id, 'group_type' => 'Public']];
                } catch (\Throwable $e) {
                    return ['code' => 502, 'msg' => '创建群聊失败，请稍后重试', 'data' => null];
                }
            }
        }

        $existing = Db::name('group_v')->where([
            'uid' => (int) $uid,
            'client_request_id' => $requestId,
            'is_delete' => 1,
        ])->find();
        if ($existing) {
            return ['code' => 200, 'msg' => '创建成功', 'data' => [
                'group_id' => (string) $existing['im_g_id'],
                'group_type' => 'Public',
            ]];
        }

        //未使用邀请码注册用户：限制功能，只能拉一个群，只能加一个好友，第二个群第二个好友出现报错 

        //根据uid 对user_id字段进行find_in_set查询
        $user_info = Db::name('invitecode')
            ->whereRaw('FIND_IN_SET(:uid, user_id)', ['uid' => (int) $uid])
            ->find();
        if(empty($user_info)) {
            $group_count = $this->countActiveGroupsByOwner($uid);
            if($group_count > 0) {
                return ['code' => 201, 'msg' => '只能创建一个群', 'data' => null];
            }
        }
        
        $group_id = $this->get_available_group_id();
        $insert_data = [
            'id' => $group_id,
            'group_name' => $group_name,
            'add_time' => time(),
            'update_time' => time(),
            'im_g_id' => $group_id,
            'uid' => $uid,
            'system_type' => config('app.system_type'),
            'client_request_id' => $requestId,
        ];
        $tencentGroupCreated = false;
        Db::startTrans();
        try{
            Db::name('group_v')->insert($insert_data);
            $result = model('api/Tencent')->create_family($group_id, $uid, $group_name, $memberIds);
            if($result['code'] == 201) {
                Db::rollback();
                try {
                    // A transport timeout may happen after Tencent created the group.
                    // Destroying a non-existent group is harmless and avoids orphans.
                    model('api/Tencent')->destroy_group((string) $group_id);
                } catch (\Throwable $ignored) {
                }
                return ['code' => 201, 'msg' => '创建失败：' . $result['msg'], 'data' => null];
            }
            $tencentGroupCreated = true;
            $bootstrapResult = model('api/Tencent')->send_group_created_message($group_id, $uid);
            if ((int) ($bootstrapResult['code'] ?? 201) !== 200) {
                Db::rollback();
                try {
                    model('api/Tencent')->destroy_group((string) $group_id);
                } catch (\Throwable $ignored) {
                }
                return ['code' => 201, 'msg' => '创建失败：' . ($bootstrapResult['msg'] ?? '群初始化消息发送失败'), 'data' => null];
            }
            Db::commit();
            return ['code' => 200, 'msg' => '创建成功', 'data' => [
                'group_id' => strval($group_id),
                'group_type' => 'Public',
            ]];
        } catch(\Throwable $e) {
            Db::rollback();
            if ($tencentGroupCreated) {
                try {
                    model('api/Tencent')->destroy_group((string) $group_id);
                } catch (\Throwable $ignored) {
                }
            }
            $duplicate = Db::name('group_v')->where([
                'uid' => (int) $uid,
                'client_request_id' => $requestId,
                'is_delete' => 1,
            ])->find();
            if ($duplicate) {
                return ['code' => 200, 'msg' => '创建成功', 'data' => [
                    'group_id' => (string) $duplicate['im_g_id'],
                    'group_type' => 'Public',
                ]];
            }
            return ['code' => 201, 'msg' => '创建失败', 'data' => null];
        }
    }

    private function countActiveGroupsByOwner($uid)
    {
        return (int) Db::name('group_v')->where([
            'uid' => (int) $uid,
            'is_delete' => 1,
        ])->count();
    }

    private function normalizeInitialMembers($ownerUid, $memberIds)
    {
        if (!is_array($memberIds)) {
            return ['error' => '群成员参数无效'];
        }
        $members = [];
        foreach ($memberIds as $memberId) {
            $memberId = trim((string) $memberId);
            if ($memberId === '' || $memberId === (string) $ownerUid) {
                continue;
            }
            if (strlen($memberId) > 32 || !preg_match('/^[A-Za-z0-9_-]+$/', $memberId)) {
                return ['error' => '群成员账号无效'];
            }
            $members[$memberId] = $memberId;
        }
        if (count($members) > 99) {
            return ['error' => '初始群成员不能超过99人'];
        }
        foreach ($members as $memberId) {
            $mappingUid = Db::name('im_account_mapping')->where('im_user_id', $memberId)
                ->order('id', 'desc')->value('uid');
            $user = null;
            if ($mappingUid) {
                $user = Db::name('user')->where('uid', (int) $mappingUid)->find();
            } elseif (ctype_digit($memberId)) {
                $user = Db::name('user')->where('uid', (int) $memberId)->find();
            } else {
                // Compatibility for accounts imported by the legacy username flow.
                $user = Db::name('user')->where('user_name', $memberId)->find();
            }
            if ($user && (int) $user['login_status'] !== 1) {
                return ['error' => '部分群成员已被封禁'];
            }
        }
        return ['members' => array_values($members)];
    }
    //过滤靓号
    public function get_available_group_id()
    {
        $group_id = mt_rand(10000, 99999);
        $group_info = db::name('group_v')->field('id')->where('id', $group_id)->find();
        if (!empty($group_info)) {
            return $this->get_available_group_id();
        } else {
            return $group_id;
        }
    }
    
    //撤回群消息
    public function group_msg_recall($uid, $group_id, $msg_seq)
    {
        if (strpos((string) $group_id, 'dd_') === 0) {
            try {
                $bridge = (new \app\imaccess\service\BridgeService())->groupRecall($uid, $group_id, $msg_seq);
                return (int) ($bridge['code'] ?? 0) === 200
                    ? ['code' => 200, 'msg' => '撤回成功', 'data' => null]
                    : ['code' => 201, 'msg' => '桥接撤回尚未确认', 'data' => null];
            } catch (\Throwable $e) {
                \think\facade\Log::warning('[IM bridge] group recall failed group=' . $group_id . ' seq=' . $msg_seq);
                return ['code' => 201, 'msg' => '桥接撤回未完成，请稍后重试', 'data' => null];
            }
        }
        $uid_str = strval($uid);
        $result = model('api/Tencent')->get_group_member_info($group_id, [$uid_str], ['Owner', 'Admin']);
        
        if($result['code'] == 201) {
            return ['code' => 201, 'msg' => '查找数据错误', 'data' => null];
        } else {
            if(empty($result['data'])) {
                return ['code' => 201, 'msg' => '不是管理或群主,不能撤回', 'data' => null];
            }
        }
        $result = model('api/Tencent')->group_msg_recall($group_id, $msg_seq);
        return $result;
    }
    //删除举报信息
    public function del_group($uid, $id)
    {
        $group_info = Db::name('group_v')->find($id);
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        if($group_info['is_delete'] == 2) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        if($group_info['uid'] != $uid) {
            return ['code' => 201, 'msg' => '不是群主，不能解散群', 'data' => null];
        }
        Db::startTrans();
        try{
            db::name('group_v')->where('id', $id)->update(['is_delete' => 2]);
            $result = model('api/Tencent')->destroy_group($group_info['im_g_id']);
            if($result['code'] == 201) {
                Db::rollback();
                return ['code' => 201, 'msg' => '解散失败', 'data' => null];
            }
            
            Db::commit();
            return ['code' => 200, 'msg' => '解散成功', 'data' => null];
        } catch(\Exception $e) {
            Db::rollback();
            return ['code' => 201, 'msg' => '解散失败', 'data' => null];
        }
    }
    //设置群保护
    public function open_protect($uid, $group_id)
    {
        $group_info = Db::name('group_v')->find($group_id);
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        if($group_info['uid'] != $uid) {
            return ['code' => 201, 'msg' => '不是群主，不能设置群保护', 'data' => null];
        }
        if($group_info['open_protect'] == 1) {
            $open_protect = 2;
        } else {
            $open_protect = 1;
        }
        $result = db::name('group_v')->where('id', $group_id)->update(['open_protect' => $open_protect, 'update_time' => time()]);
        if($result) {
            return ['code' => 200, 'msg' => '设置成功', 'data' => null];
        }
        return ['code' => 201, 'msg' => '设置失败', 'data' => null];
    }
    //获取群保护状态
    public function get_open_protect($group_id)
    {
        $group_info = Db::name('group_v')->find($group_id);
        if(empty($group_info)) {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        return ['code' => 200, 'msg' => '获取数据成功', 'data' => ['open_protect' => $group_info['open_protect']]];
    }
    //获取群文件列表
    public function get_group_file_list($group_id)
    {
        $group_id = trim((string) $group_id);
        if ($group_id === '') {
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }

        // Legacy groups live in group_v; groups created by the Web SDK are
        // tracked independently by the Tencent callback projection.
        $group_info = Db::name('group_v')
            ->where(['im_g_id' => $group_id, 'is_delete' => 1])
            ->find();
        if (empty($group_info) && ctype_digit($group_id)) {
            $group_info = Db::name('group_v')
                ->where(['id' => (int) $group_id, 'is_delete' => 1])
                ->find();
        }

        $im_group_id = empty($group_info) ? $group_id : (string) $group_info['im_g_id'];
        if (empty($group_info)) {
            $snapshot = Db::name('im_group_snapshot')
                ->where('group_id', $group_id)
                ->where('status', '<>', 'destroyed')
                ->find();
            if (empty($snapshot)) {
                return ['code' => 201, 'msg' => '群不存在或群信息尚未同步', 'data' => null];
            }
        }

        $file_list = Db::name('group_file')
            ->where(['im_g_id' => $im_group_id, 'is_delete' => 1])
            ->select();
        return ['code' => 200, 'msg' => '获取数据成功', 'data' => ['file_list' => $file_list]];
    }
}
