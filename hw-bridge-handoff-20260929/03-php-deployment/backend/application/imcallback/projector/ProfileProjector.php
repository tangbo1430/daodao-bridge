<?php

namespace app\imcallback\projector;

use app\imcallback\service\Payload;
use app\imcallback\service\ProjectionOrder;
use think\Db;

class ProfileProjector implements ProjectorInterface
{
    public function project(array $event, array $payload)
    {
        $account = Payload::first($payload, ['From_Account', 'To_Account', 'UserID']);
        if ($account === '') {
            throw new \InvalidArgumentException('Profile account is required');
        }

        $profile = [];
        if (!empty($payload['ProfileItem']) && is_array($payload['ProfileItem'])) {
            foreach ($payload['ProfileItem'] as $item) {
                if (isset($item['Tag'])) {
                    $profile[$item['Tag']] = isset($item['Value']) ? $item['Value'] : '';
                }
            }
        }
        $source = ProjectionOrder::values($event, $payload);
        $data = [
            'nick_name' => isset($profile['Tag_Profile_IM_Nick']) ? (string) $profile['Tag_Profile_IM_Nick'] : '',
            'avatar_url' => isset($profile['Tag_Profile_IM_Image']) ? (string) $profile['Tag_Profile_IM_Image'] : '',
            'status' => 'active',
            'profile_json' => json_encode($payload, JSON_UNESCAPED_UNICODE),
            'updated_at' => time(),
        ] + $source;
        $existing = Db::name('im_user_profile_snapshot')->where('account', $account)->find();
        if ($existing) {
            if (ProjectionOrder::isStale($existing, $source)) return;
            if ($data['nick_name'] === '') unset($data['nick_name']);
            if ($data['avatar_url'] === '') unset($data['avatar_url']);
            Db::name('im_user_profile_snapshot')->where('account', $account)->update($data);
        } else {
            $data['account'] = $account;
            Db::name('im_user_profile_snapshot')->insert($data);
        }
    }
}
