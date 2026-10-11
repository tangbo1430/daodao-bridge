<?php

namespace app\common\service;

use think\Db;

class UserProfileViewService
{
    public function enrich(array $users)
    {
        if (empty($users)) {
            return [];
        }

        $uids = [];
        foreach ($users as $user) {
            if (isset($user['uid'])) {
                $uids[] = (int) $user['uid'];
            }
        }

        $mappingByUid = [];
        $profileByAccount = [];
        try {
            $sdkAppId = (string) (getenv('IM_SDK_APP_ID') ?: '');
            $mappingQuery = Db::name('im_account_mapping')->whereIn('uid', array_values(array_unique($uids)));
            if ($sdkAppId !== '') {
                $mappingQuery->where('sdk_app_id', $sdkAppId);
            }
            $mappings = $mappingQuery->select();
            $accounts = array_map('strval', $uids);
            foreach ($mappings as $mapping) {
                $uid = (int) $mapping['uid'];
                $account = (string) $mapping['im_user_id'];
                $mappingByUid[$uid] = $account;
                $accounts[] = $account;
            }

            $accounts = array_values(array_unique(array_filter($accounts, 'strlen')));
            if (!empty($accounts)) {
                $profiles = Db::name('im_user_profile_snapshot')
                    ->whereIn('account', $accounts)
                    ->where('status', 'active')
                    ->select();
                foreach ($profiles as $profile) {
                    $profileByAccount[(string) $profile['account']] = $profile;
                }
            }
        } catch (\Throwable $ignored) {
            // Profile projection is an optional display overlay. Legacy user data remains available.
        }

        foreach ($users as &$user) {
            $uid = isset($user['uid']) ? (int) $user['uid'] : 0;
            $account = isset($mappingByUid[$uid]) ? $mappingByUid[$uid] : (string) $uid;
            $profile = isset($profileByAccount[$account]) ? $profileByAccount[$account] : null;
            $user = $this->merge($user, $profile, $account);
        }
        unset($user);

        return $users;
    }

    public function merge(array $user, array $profile = null, $account = '')
    {
        $businessNick = '';
        if (!empty($user['base64_nick_name'])) {
            $decoded = base64_decode($user['base64_nick_name'], true);
            $businessNick = is_string($decoded) ? mb_convert_encoding($decoded, 'UTF-8', 'UTF-8') : '';
        }
        if ($businessNick === '' && isset($user['nick_name'])) {
            $businessNick = (string) $user['nick_name'];
        }

        $businessAvatar = localpath_to_netpath(isset($user['head_pic']) ? $user['head_pic'] : '');
        $profileNick = $profile && !empty($profile['nick_name']) ? (string) $profile['nick_name'] : '';
        $profileAvatar = $profile && !empty($profile['avatar_url']) ? (string) $profile['avatar_url'] : '';

        $user['nick_name'] = $profileNick !== '' ? $profileNick : $businessNick;
        $user['head_pic'] = $profileAvatar !== '' ? $profileAvatar : $businessAvatar;
        $user['profile_source'] = ($profileNick !== '' || $profileAvatar !== '') ? 'tencent' : 'business';
        $user['im_user_id'] = (string) $account;
        return $user;
    }
}
