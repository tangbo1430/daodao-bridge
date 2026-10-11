<?php

namespace app\imaccess\service;

use think\Db;

class AccountMappingService
{
    public function resolve($uid, $sdkAppId)
    {
        $uid = (int) $uid;
        $sdkAppId = (string) $sdkAppId;
        $mapping = Db::name('im_account_mapping')->where([
            'uid' => $uid,
            'sdk_app_id' => $sdkAppId,
        ])->find();
        if ($mapping) return $mapping;

        $now = time();
        try {
            Db::name('im_account_mapping')->insert([
                'uid' => $uid,
                'im_user_id' => (string) $uid,
                'sdk_app_id' => $sdkAppId,
                'import_status' => 'pending',
                'import_error' => '',
                'imported_at' => 0,
                'created_at' => $now,
                'updated_at' => $now,
            ]);
        } catch (\Throwable $e) {
            $mapping = Db::name('im_account_mapping')->where([
                'uid' => $uid,
                'sdk_app_id' => $sdkAppId,
            ])->find();
            if (!$mapping) throw $e;
            return $mapping;
        }

        return Db::name('im_account_mapping')->where([
            'uid' => $uid,
            'sdk_app_id' => $sdkAppId,
        ])->find();
    }

    public function businessUid($imUserId, $sdkAppId)
    {
        if ((string) $imUserId === '') return null;
        $uid = Db::name('im_account_mapping')->where([
            'im_user_id' => (string) $imUserId,
            'sdk_app_id' => (string) $sdkAppId,
        ])->value('uid');
        return $uid === null ? null : (int) $uid;
    }
}
