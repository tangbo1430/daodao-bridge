<?php

namespace app\imcallback\projector;

use app\imcallback\service\Payload;
use app\imcallback\service\ProjectionOrder;
use think\Db;

class GroupProjector implements ProjectorInterface
{
    public function project(array $event, array $payload)
    {
        $command = $event['callback_command'];
        $groupId = Payload::first($payload, ['GroupId']);
        if ($groupId === '') {
            throw new \InvalidArgumentException('GroupId is required');
        }

        $now = time();
        $source = ProjectionOrder::values($event, $payload);
        $raw = json_encode($payload, JSON_UNESCAPED_UNICODE);
        $group = [
            'group_name' => Payload::first($payload, ['Name', 'GroupName']),
            'owner_account' => Payload::first($payload, ['Owner_Account', 'NewOwner_Account']),
            'group_type' => Payload::first($payload, ['Type', 'GroupType']),
            'raw_json' => $raw,
            'updated_at' => $now,
        ] + $source;

        $existing = Db::name('im_group_snapshot')->where('group_id', $groupId)->find();
        $groupStale = $existing && ProjectionOrder::isStale($existing, $source);
        if (!$existing) {
            $group['group_id'] = $groupId;
            $group['created_at'] = $now;
            $group['status'] = 'active';
            Db::name('im_group_snapshot')->insert($group);
        } else {
            if (!$groupStale) {
                foreach (['group_name', 'owner_account', 'group_type'] as $field) {
                    if ($group[$field] === '') {
                        unset($group[$field]);
                    }
                }
                Db::name('im_group_snapshot')->where('group_id', $groupId)->update($group);
            }
        }

        if ($command === 'Group.CallbackAfterGroupDestroyed') {
            if ($groupStale) return;
            Db::name('im_group_snapshot')->where('group_id', $groupId)->update([
                'status' => 'destroyed',
                'updated_at' => $now,
            ] + $source);
            Db::name('im_group_member_snapshot')->where('group_id', $groupId)->update([
                'status' => 'inactive',
                'updated_at' => $now,
            ] + $source);
            return;
        }

        if ($command === 'Group.CallbackAfterGroupFull') {
            if ($groupStale) return;
            Db::name('im_group_snapshot')->where('group_id', $groupId)->update([
                'status' => 'full',
                'updated_at' => $now,
            ] + $source);
        }

        if ($command === 'Group.CallbackAfterChangeGroupOwner') {
            if ($groupStale) return;
            Db::name('im_group_snapshot')->where('group_id', $groupId)->update([
                'owner_account' => Payload::first($payload, ['NewOwner_Account', 'Owner_Account']),
                'updated_at' => $now,
            ] + $source);
        }

        $memberLists = [
            'Group.CallbackAfterNewMemberJoin' => ['NewMemberList', 'MemberList'],
            'Group.CallbackAfterMemberExit' => ['ExitMemberList', 'MemberList'],
            'Group.CallbackAfterMemberInfoChanged' => ['MemberList'],
        ];
        if (!isset($memberLists[$command])) {
            return;
        }

        $members = [];
        foreach ($memberLists[$command] as $key) {
            if (!empty($payload[$key]) && is_array($payload[$key])) {
                $members = $payload[$key];
                break;
            }
        }

        foreach ($members as $member) {
            $account = is_array($member)
                ? Payload::first($member, ['Member_Account', 'UserID', 'To_Account'])
                : (string) $member;
            if ($account === '') {
                continue;
            }
            $status = $command === 'Group.CallbackAfterMemberExit' ? 'exited' : 'active';
            $data = [
                'role' => is_array($member) ? Payload::first($member, ['Role', 'MemberRole']) : '',
                'status' => $status,
                'join_time' => is_array($member) ? (int) Payload::first($member, ['JoinTime'], 0) : 0,
                'raw_json' => json_encode($member, JSON_UNESCAPED_UNICODE),
                'updated_at' => $now,
            ] + $source;
            $exists = Db::name('im_group_member_snapshot')
                ->where(['group_id' => $groupId, 'account' => $account])->find();
            if ($exists) {
                if (ProjectionOrder::isStale($exists, $source)) continue;
                Db::name('im_group_member_snapshot')->where('id', $exists['id'])->update($data);
            } else {
                $data['group_id'] = $groupId;
                $data['account'] = $account;
                Db::name('im_group_member_snapshot')->insert($data);
            }
        }

        $count = Db::name('im_group_member_snapshot')
            ->where(['group_id' => $groupId, 'status' => 'active'])->count();
        $countUpdate = [
            'member_count' => $count,
            'updated_at' => $now,
        ];
        if (!$groupStale) $countUpdate += $source;
        Db::name('im_group_snapshot')->where('group_id', $groupId)->update($countUpdate);
    }
}
