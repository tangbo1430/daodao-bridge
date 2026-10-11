<?php

namespace app\imaccess\service;

use think\Db;

class BlocklistService
{
    const RELATION_TYPE = 'app_blocklist';

    public function status($loginToken, $peerAccount)
    {
        $owner = $this->authenticate($loginToken);
        $peer = $this->validatePeer($owner, $peerAccount);
        $blocked = $this->hasActiveRelation($owner['account'], $peer)
            || $this->hasLegacyRelation($owner['uid'], $peer);

        return $this->success(['blocked' => $blocked]);
    }

    public function change($loginToken, $peerAccount, $blocked)
    {
        $owner = $this->authenticate($loginToken);
        $peer = $this->validatePeer($owner, $peerAccount);
        if (preg_match('/^[0-9a-f]{32}$/i', $peer)) {
            $result = (new BridgeService())->friendAction($loginToken, $peer, $blocked ? 'block' : 'unblock');
            if ($result['http_status'] !== 200 || (int) ($result['body']['code'] ?? 502) !== 200) {
                throw new BlocklistException((int) ($result['body']['code'] ?? 502), (string) ($result['body']['msg'] ?? '黑名单更新失败'));
            }
        }
        $where = [
            'owner_account' => $owner['account'],
            'peer_account' => $peer,
            'relation_type' => self::RELATION_TYPE,
        ];
        $now = time();
        $data = [
            'status' => $blocked ? 'active' : 'deleted',
            'raw_json' => json_encode([
                'source' => 'app',
                'owner_account' => $owner['account'],
                'peer_account' => $peer,
                'blocked' => (bool) $blocked,
            ], JSON_UNESCAPED_UNICODE),
            'updated_at' => $now,
        ];
        if ($blocked) {
            $this->upsertRelation($where, $data);
        } else {
            $hasNativeRelation = Db::name('im_relation_snapshot')->where([
                'owner_account' => $owner['account'],
                'peer_account' => $peer,
                'relation_type' => 'blacklist',
                'status' => 'active',
            ])->count() > 0;
            if ($hasNativeRelation) {
                $nativeResult = model('api/Tencent')->black_list_delete($owner['account'], [$peer]);
                if (!is_array($nativeResult) || !isset($nativeResult['code']) || (int) $nativeResult['code'] !== 200) {
                    throw new BlocklistException(502, '腾讯历史黑名单解除失败，请稍后重试');
                }
            }

            Db::startTrans();
            try {
                Db::name('im_relation_snapshot')->where([
                    'owner_account' => $owner['account'],
                    'peer_account' => $peer,
                ])->where('relation_type', 'in', [self::RELATION_TYPE, 'blacklist'])->update($data);

                $peerUid = $this->resolveBusinessUid($peer);
                if ($peerUid > 0) {
                    Db::name('user_black')->where([
                        'uid' => $owner['uid'],
                        'receive_uid' => $peerUid,
                    ])->delete();
                }
                Db::commit();
            } catch (\Throwable $e) {
                Db::rollback();
                throw $e;
            }
        }

        return $this->success(['blocked' => (bool) $blocked]);
    }

    public function listing($loginToken)
    {
        $owner = $this->authenticate($loginToken);
        $rows = Db::name('im_relation_snapshot')->where([
            'owner_account' => $owner['account'],
            'status' => 'active',
        ])->where('relation_type', 'in', [self::RELATION_TYPE, 'blacklist'])
            ->order('updated_at', 'desc')->limit(1000)->select();

        $rowByAccount = [];
        foreach ($rows as $row) {
            $account = (string) $row['peer_account'];
            if (!isset($rowByAccount[$account]) || (int) $row['updated_at'] > (int) $rowByAccount[$account]['updated_at']) {
                $rowByAccount[$account] = $row;
            }
        }
        $legacyRows = Db::name('user_black')->where('uid', $owner['uid'])->limit(1000)->select();
        foreach ($legacyRows as $legacy) {
            $account = $this->resolveImAccount((int) $legacy['receive_uid']);
            if (!isset($rowByAccount[$account])) {
                $rowByAccount[$account] = [
                    'peer_account' => $account,
                    'updated_at' => isset($legacy['add_time']) ? (int) $legacy['add_time'] : 0,
                ];
            }
        }
        $rows = array_values($rowByAccount);
        usort($rows, function ($left, $right) {
            return (int) $right['updated_at'] <=> (int) $left['updated_at'];
        });
        $accounts = array_values(array_unique(array_column($rows, 'peer_account')));
        if (!$accounts) {
            return $this->success([]);
        }

        $profiles = Db::name('im_user_profile_snapshot')->where('account', 'in', $accounts)
            ->column('*', 'account');
        $mappingRows = Db::name('im_account_mapping')->where('im_user_id', 'in', $accounts)->select();
        $uidByAccount = [];
        foreach ($mappingRows as $mapping) {
            $uidByAccount[(string) $mapping['im_user_id']] = (int) $mapping['uid'];
        }
        foreach ($accounts as $account) {
            if (!isset($uidByAccount[$account]) && ctype_digit((string) $account)) {
                $uidByAccount[$account] = (int) $account;
            }
        }
        $users = $uidByAccount
            ? Db::name('user')->where('uid', 'in', array_values($uidByAccount))->column('*', 'uid')
            : [];

        $result = [];
        foreach ($rows as $row) {
            $account = (string) $row['peer_account'];
            $profile = isset($profiles[$account]) ? $profiles[$account] : [];
            $uid = isset($uidByAccount[$account]) ? $uidByAccount[$account] : 0;
            $user = $uid && isset($users[$uid]) ? $users[$uid] : [];
            $nick = isset($profile['nick_name']) ? (string) $profile['nick_name'] : '';
            if ($nick === '' && !empty($user['base64_nick_name'])) {
                $decoded = base64_decode($user['base64_nick_name'], true);
                $nick = is_string($decoded) ? $decoded : '';
            }
            if ($nick === '' && isset($user['nick_name'])) {
                $nick = (string) $user['nick_name'];
            }
            $avatar = isset($profile['avatar_url']) ? (string) $profile['avatar_url'] : '';
            if ($avatar === '' && !empty($user['head_pic'])) {
                $avatar = localpath_to_netpath($user['head_pic']);
            }
            $result[] = [
                'peer_account' => $account,
                'nick_name' => $nick,
                'avatar_url' => $avatar,
                'updated_at' => (int) $row['updated_at'],
            ];
        }

        return $this->success($result);
    }

    private function hasActiveRelation($ownerAccount, $peerAccount)
    {
        return Db::name('im_relation_snapshot')->where([
            'owner_account' => $ownerAccount,
            'peer_account' => $peerAccount,
            'status' => 'active',
        ])->where('relation_type', 'in', [self::RELATION_TYPE, 'blacklist'])->count() > 0;
    }

    private function hasLegacyRelation($ownerUid, $peerAccount)
    {
        $peerUid = $this->resolveBusinessUid($peerAccount);
        if ($peerUid <= 0) {
            return false;
        }
        return Db::name('user_black')->where([
            'uid' => (int) $ownerUid,
            'receive_uid' => $peerUid,
        ])->count() > 0;
    }

    private function upsertRelation(array $where, array $data)
    {
        $existing = Db::name('im_relation_snapshot')->where($where)->find();
        if ($existing) {
            Db::name('im_relation_snapshot')->where('id', $existing['id'])->update($data);
            return;
        }
        try {
            Db::name('im_relation_snapshot')->insert(array_merge($where, $data));
        } catch (\Throwable $e) {
            // A concurrent identical request may win the unique key race.
            Db::name('im_relation_snapshot')->where($where)->update($data);
        }
    }

    private function resolveBusinessUid($account)
    {
        $uid = Db::name('im_account_mapping')->where('im_user_id', (string) $account)
            ->order('id', 'desc')->value('uid');
        if ($uid) {
            return (int) $uid;
        }
        return ctype_digit((string) $account) ? (int) $account : 0;
    }

    private function resolveImAccount($uid)
    {
        $account = Db::name('im_account_mapping')->where('uid', (int) $uid)
            ->order('id', 'desc')->value('im_user_id');
        return $account !== null && $account !== '' ? (string) $account : (string) $uid;
    }

    private function authenticate($loginToken)
    {
        $token = trim((string) $loginToken);
        if ($token === '') {
            throw new BlocklistException(401, '登录已失效');
        }
        $user = Db::name('user')->where('login_token', $token)->find();
        if (!$user) {
            throw new BlocklistException(401, '登录已失效');
        }
        if ((int) $user['login_status'] !== 1) {
            throw new BlocklistException(403, '用户已被封禁');
        }
        $mapping = Db::name('im_account_mapping')->where('uid', (int) $user['uid'])
            ->order('id', 'desc')->find();
        return [
            'uid' => (int) $user['uid'],
            'account' => $mapping ? (string) $mapping['im_user_id'] : (string) $user['uid'],
        ];
    }

    private function validatePeer(array $owner, $peerAccount)
    {
        $peer = trim((string) $peerAccount);
        if ($peer === '' || strlen($peer) > 255) {
            throw new BlocklistException(400, '用户账号无效');
        }
        if ($peer === $owner['account']) {
            throw new BlocklistException(400, '不能屏蔽自己');
        }
        return $peer;
    }

    private function success($data)
    {
        return ['http_status' => 200, 'body' => ['code' => 200, 'msg' => '操作成功', 'data' => $data]];
    }
}

class BlocklistException extends \RuntimeException
{
    private $httpStatus;

    public function __construct($httpStatus, $message)
    {
        parent::__construct($message);
        $this->httpStatus = (int) $httpStatus;
    }

    public function httpStatus()
    {
        return $this->httpStatus;
    }
}
