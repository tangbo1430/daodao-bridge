<?php

namespace app\imcallback\service;

class EventCatalog
{
    private static $names = [
        'C2C.CallbackBeforeSendMsg' => '单聊发送前',
        'C2C.CallbackAfterSendMsg' => '单聊发送后',
        'C2C.CallbackAfterMsgReport' => '单聊已读',
        'C2C.CallbackAfterMsgWithDraw' => '单聊撤回',
        'C2C.CallbackAfterSendMsgException' => '单聊发送异常',
        'Group.CallbackBeforeSendMsg' => '群消息发送前',
        'Group.CallbackAfterSendMsg' => '群消息发送后',
        'Group.CallbackAfterMsgWithDraw' => '群消息撤回',
        'Group.CallbackAfterSendMsgException' => '群消息发送异常',
        'Group.CallbackBeforeCreateGroup' => '创建群组前',
        'Group.CallbackAfterCreateGroup' => '创建群组后',
        'Group.CallbackBeforeApplyJoinGroup' => '申请入群前',
        'Group.CallbackBeforeInviteJoinGroup' => '邀请入群前',
        'Group.CallbackAfterNewMemberJoin' => '成员入群后',
        'Group.CallbackAfterMemberExit' => '成员退群后',
        'Group.CallbackAfterGroupFull' => '群组已满',
        'Group.CallbackAfterGroupDestroyed' => '群组解散',
        'Group.CallbackAfterGroupInfoChanged' => '群资料变更',
        'Group.CallbackAfterMemberInfoChanged' => '群成员资料变更',
        'Group.CallbackAfterChangeGroupOwner' => '群主变更',
        'Sns.CallbackPrevFriendAdd' => '添加好友前',
        'Sns.CallbackPrevFriendResponse' => '回应好友前',
        'Sns.CallbackFriendAdd' => '添加好友后',
        'Sns.CallbackFriendDelete' => '删除好友',
        'Sns.CallbackBlackListAdd' => '加入黑名单',
        'Sns.CallbackBlackListDelete' => '移除黑名单',
        'Profile.CallbackPortraitSet' => '用户资料变更',
        'State.StateChange' => '在线状态变更',
    ];

    public static function name($command)
    {
        return isset(self::$names[$command]) ? self::$names[$command] : $command;
    }

    public static function all()
    {
        return self::$names;
    }
}
