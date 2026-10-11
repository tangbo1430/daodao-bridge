<?php

namespace app\imcallback\handler;

class HandlerRegistry
{
    private static $beforeEvents = [
        'C2C.CallbackBeforeSendMsg' => 'message',
        'Group.CallbackBeforeSendMsg' => 'message',
        'Group.CallbackBeforeCreateGroup' => 'group',
        'Group.CallbackBeforeApplyJoinGroup' => 'group',
        'Group.CallbackBeforeInviteJoinGroup' => 'group',
        'Sns.CallbackPrevFriendAdd' => 'relation',
        'Sns.CallbackPrevFriendResponse' => 'relation',
    ];

    private static $asyncEvents = [
        'Bridge.AccountDeleted' => 'state',
        'C2C.CallbackAfterSendMsg' => 'message',
        'C2C.CallbackAfterMsgReport' => 'message',
        'C2C.CallbackAfterMsgWithDraw' => 'message',
        'C2C.CallbackAfterSendMsgException' => 'message',
        'Group.CallbackAfterSendMsg' => 'message',
        'Group.CallbackAfterMsgWithDraw' => 'message',
        'Group.CallbackAfterRecallMsg' => 'message',
        'Group.CallbackAfterSendMsgException' => 'message',

        'Group.CallbackAfterCreateGroup' => 'group',
        'Group.CallbackAfterNewMemberJoin' => 'group',
        'Group.CallbackAfterMemberExit' => 'group',
        'Group.CallbackAfterGroupFull' => 'group',
        'Group.CallbackAfterGroupDestroyed' => 'group',
        'Group.CallbackAfterGroupInfoChanged' => 'group',
        'Group.CallbackAfterMemberInfoChanged' => 'group',
        'Group.CallbackAfterMemberFieldChanged' => 'group',
        'Group.CallbackAfterChangeGroupOwner' => 'group',

        'Sns.CallbackFriendAdd' => 'relation',
        'Sns.CallbackFriendDelete' => 'relation',
        'Sns.CallbackBlackListAdd' => 'relation',
        'Sns.CallbackBlackListDelete' => 'relation',

        'Profile.CallbackPortraitSet' => 'profile',
        'State.StateChange' => 'state',
    ];

    public static function resolve($command)
    {
        if ($command === 'C2C.CallbackAfterSendMsg') {
            return new C2CAfterSendMsgHandler();
        }

        if (isset(self::$beforeEvents[$command])) {
            return new AllowBeforeEventHandler(self::$beforeEvents[$command]);
        }

        if (isset(self::$asyncEvents[$command])) {
            return new RecordOnlyEventHandler(self::$asyncEvents[$command]);
        }

        return null;
    }

    public static function supportedCommands()
    {
        return array_merge(array_keys(self::$beforeEvents), array_keys(self::$asyncEvents));
    }
}
