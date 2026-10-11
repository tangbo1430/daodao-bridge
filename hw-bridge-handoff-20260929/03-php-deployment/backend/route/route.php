<?php
\think\facade\Route::post('internal/bridge/registration_lookup', 'imaccess/bridge_internal/registrationLookup');
\think\facade\Route::post('internal/bridge/account_status', 'imaccess/bridge_internal/accountStatus');
// +----------------------------------------------------------------------
// | ThinkPHP [ WE CAN DO IT JUST THINK ]
// +----------------------------------------------------------------------
// | Copyright (c) 2006~2018 http://thinkphp.cn All rights reserved.
// +----------------------------------------------------------------------
// | Licensed ( http://www.apache.org/licenses/LICENSE-2.0 )
// +----------------------------------------------------------------------
// | Author: liu21st <liu21st@gmail.com>
// +----------------------------------------------------------------------

Route::get('think', function () {
    return 'hello,ThinkPHP5!';
});

Route::get('hello/:name', 'index/hello');

Route::get('api/v1/im/bridge/config', 'imaccess/bridge/config');
Route::post('api/v1/im/bridge/provision', 'imaccess/bridge/provision');
Route::post('api/v1/im/bridge/disable', 'imaccess/bridge/disable');
Route::get('api/bridge/search_user', 'imaccess/bridge/searchUser');
Route::post('api/bridge/search_user', 'imaccess/bridge/searchUser');
Route::post('api/bridge/friend_apply', 'imaccess/bridge/friendApply');
Route::post('api/bridge/friend_accept', 'imaccess/bridge/friendAccept');
Route::post('api/bridge/friend_refuse', 'imaccess/bridge/friendRefuse');
Route::post('api/bridge/friend_delete', 'imaccess/bridge/friendDelete');
Route::get('api/bridge/friend_requests', 'imaccess/bridge/friendRequests');
Route::post('api/bridge/friend_requests', 'imaccess/bridge/friendRequests');
Route::rule('api/bridge/media', 'imaccess/bridge_media/download', 'GET|HEAD');
Route::post('api/bridge/group_create', 'imaccess/bridge/groupCreate');
Route::post('api/bridge/group_invite', 'imaccess/bridge/groupInvite');
Route::post('api/bridge/group_remove', 'imaccess/bridge/groupRemove');
Route::post('api/bridge/group_exit', 'imaccess/bridge/groupExit');
Route::post('api/bridge/group_disband', 'imaccess/bridge/groupDisband');
Route::post('api/bridge/group_update', 'imaccess/bridge/groupUpdate');
Route::post('api/bridge/group_sync', 'imaccess/bridge/groupSync');

return [

];
