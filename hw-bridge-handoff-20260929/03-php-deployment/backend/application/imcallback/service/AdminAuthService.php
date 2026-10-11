<?php

namespace app\imcallback\service;

use think\Db;

class AdminAuthService
{
    public function authorize($token)
    {
        if (!$token) return 0;
        $admin = Db::name('admin')->where([
            'login_token' => $token,
            'is_delete' => 1,
            'status' => 1,
        ])->find();
        if (!$admin || time() > (int) $admin['token_validity_time']) return 0;

        $allowed = array_filter(array_map('intval', explode(',', config('im_callback.admin_ids'))));
        return in_array((int) $admin['aid'], $allowed, true) ? (int) $admin['aid'] : 0;
    }
}
