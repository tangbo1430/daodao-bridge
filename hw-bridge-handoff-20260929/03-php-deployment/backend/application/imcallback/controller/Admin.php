<?php

namespace app\imcallback\controller;

use app\imcallback\service\AdminAuthService;
use app\imcallback\service\AdminAuditService;
use app\imcallback\service\CallbackService;
use app\imcallback\service\EventCatalog;
use app\imcallback\service\OverviewService;
use app\imcallback\service\QueueService;
use app\imaccess\service\AccountMappingService;
use think\Controller;
use think\Db;

class Admin extends Controller
{
    public function events()
    {
        if (!$this->authorized()) return $this->unauthorized();
        $page = max(1, (int) input('page', 1));
        $limit = min(100, max(1, (int) input('limit', 20)));
        $query = Db::name('im_callback_event');
        $eventId = (int) input('event_id', 0); if ($eventId > 0) $query->where('id', $eventId);
        foreach (['event_category', 'callback_command', 'handler_status', 'queue_status', 'group_id', 'trace_id'] as $field) {
            $value = trim((string) input($field, ''));
            if ($value !== '') $query = $query->where($field, $value);
        }
        $this->applyTimeRange($query);
        $uid = (int) input('business_uid', 0);
        if ($uid > 0) {
            $accounts = Db::name('im_account_mapping')->where('uid', $uid)->column('im_user_id');
            $accounts = array_values(array_unique(array_merge($accounts, [(string) $uid])));
            $query = $query->where(function ($q) use ($accounts) {
                $q->where('from_account', 'in', $accounts)->whereOr('to_account', 'in', $accounts);
            });
        }
        $account = trim((string) input('account', ''));
        if ($account !== '') {
            $query = $query->where(function ($q) use ($account) {
                $q->where('from_account', $account)->whereOr('to_account', $account);
            });
        }
        $count = (clone $query)->count();
        $list = $query->field('id,trace_id,request_id,callback_command,event_category,sdk_app_id,from_account,to_account,group_id,payload_json,summary,handler_status,queue_status,retry_count,error_message,duplicate_count,received_at,processed_at,duration_ms')
            ->order('id', 'desc')->page($page, $limit)->select();
        $this->attachBusinessUids($list);
        foreach ($list as &$item) {
            $item['event_name'] = EventCatalog::name($item['callback_command']);
            if ($item['event_category'] === 'message') $item['summary'] = $this->messageTypeLabel($item['payload_json']);
            unset($item['payload_json']);
        }
        return json(['code' => 0, 'msg' => 'success', 'count' => $count, 'data' => $list]);
    }

    public function detail()
    {
        if (!$this->authorized()) return $this->unauthorized();
        $event = Db::name('im_callback_event')->where('id', (int) input('id', 0))->find();
        if (!$event) return json(['code' => 404, 'msg' => 'event not found', 'data' => null]);
        $includeRaw = (int) input('include_raw', 0) === 1;
        $event['event_name'] = EventCatalog::name($event['callback_command']);
        $event['response'] = $event['response_json'] ? json_decode($event['response_json'], true) : null;
        if ($includeRaw) {
            $event['payload'] = json_decode($event['payload_json'], true);
            (new AdminAuditService())->record($this->adminId(), $event['id'], 'view_raw', '', '', 'success', '', $event['trace_id']);
        }
        $items = [$event];
        $this->attachBusinessUids($items);
        $event = $items[0];
        unset($event['payload_json'], $event['response_json']);
        $event['timeline'] = [
            ['label' => '腾讯回调到达', 'time' => (int) $event['received_at']],
            ['label' => '最后一次接收', 'time' => (int) $event['last_received_at']],
            ['label' => '处理完成', 'time' => (int) $event['processed_at']],
        ];
        $event['action_history'] = Db::name('im_callback_admin_action')->alias('a')
            ->leftJoin('admin d', 'a.admin_id=d.aid')->where('a.event_id', $event['id'])
            ->field('a.action,a.before_status,a.after_status,a.result,a.error_message,a.trace_id,a.created_at,d.user_name')
            ->order('a.id', 'desc')->select();
        return json(['code' => 0, 'msg' => 'success', 'data' => $event]);
    }

    public function overview()
    {
        if (!$this->authorized()) return $this->unauthorized();
        return json(['code' => 0, 'msg' => 'success', 'data' => (new OverviewService())->get((int) input('days', 1))]);
    }

    public function anomalies()
    {
        if (!$this->authorized()) return $this->unauthorized();
        $page = max(1, (int) input('page', 1));
        $limit = min(100, max(1, (int) input('limit', 20)));
        $type = trim((string) input('type', ''));
        $query = Db::name('im_callback_event');
        if ($type === 'pending') $query->where('queue_status', 'pending');
        elseif ($type === 'processing') $query->where('queue_status', 'processing')->where('received_at', '<', time() - 60);
        elseif ($type === 'failed') $query->where('queue_status', 'failed');
        elseif ($type === 'dead') $query->where('queue_status', 'dead');
        else $query->where(function ($q) {
            $q->where('queue_status', 'in', ['pending', 'failed', 'dead'])
                ->whereOr(function ($nested) { $nested->where('queue_status', 'processing')->where('received_at', '<', time() - 60); });
        });
        $count = (clone $query)->count();
        $list = $query->field('id,trace_id,callback_command,event_category,from_account,to_account,group_id,handler_status,queue_status,retry_count,manual_retry_count,next_retry_at,error_message,received_at,processed_at,duration_ms')
            ->order('id', 'desc')->page($page, $limit)->select();
        foreach ($list as &$item) {
            $item['event_name'] = EventCatalog::name($item['callback_command']);
            $item['wait_seconds'] = max(0, time() - (int) $item['received_at']);
        }
        return json(['code' => 0, 'msg' => 'success', 'count' => $count, 'data' => $list]);
    }

    public function retry()
    {
        $adminId = $this->adminId();
        if (!$adminId) return $this->unauthorized();
        $eventId = (int) input('id', 0);
        $event = Db::name('im_callback_event')->where('id', $eventId)->find();
        if (!$event) return json(['code' => 404, 'msg' => '事件不存在', 'data' => null], 404);
        $eligible = in_array($event['queue_status'], ['failed', 'dead'], true) ||
            ($event['queue_status'] === 'processing' && (int) $event['received_at'] < time() - 60);
        if (!$eligible) return json(['code' => 409, 'msg' => '当前状态不允许人工重试', 'data' => null], 409);

        $before = $event['handler_status'] . '/' . $event['queue_status'];
        $traceId = CallbackService::createTraceId();
        $claimed = Db::name('im_callback_event')->where('id', $eventId)
            ->where('queue_status', $event['queue_status'])->update([
                'handler_status' => 'received', 'queue_status' => 'pending', 'retry_count' => 0,
                'manual_retry_count' => Db::raw('manual_retry_count + 1'), 'next_retry_at' => 0,
                'error_message' => '', 'processed_at' => 0,
            ]);
        if (!$claimed) return json(['code' => 409, 'msg' => '事件状态已变化，请刷新', 'data' => null], 409);
        $queued = true; $error = '';
        try { (new QueueService())->enqueue($eventId); }
        catch (\Throwable $e) { $queued = false; $error = 'Redis 不可用，已转数据库补偿'; }
        (new AdminAuditService())->record($adminId, $eventId, 'manual_retry', $before, 'received/pending', 'success', $error, $traceId);
        return json(['code' => 0, 'msg' => $queued ? '已重新入队' : $error, 'data' => ['trace_id' => $traceId, 'redis_queued' => $queued]]);
    }

    public function actionLogs()
    {
        if (!$this->authorized()) return $this->unauthorized();
        $page = max(1, (int) input('page', 1)); $limit = min(100, max(1, (int) input('limit', 20)));
        $query = Db::name('im_callback_admin_action')->alias('a')->leftJoin('admin d', 'a.admin_id=d.aid');
        $eventId = (int) input('event_id', 0); if ($eventId) $query->where('a.event_id', $eventId);
        $count = (clone $query)->count();
        $list = $query->field('a.*,d.user_name')->order('a.id', 'desc')->page($page, $limit)->select();
        return json(['code' => 0, 'msg' => 'success', 'count' => $count, 'data' => $list]);
    }

    public function catalog()
    {
        if (!$this->authorized()) return $this->unauthorized();
        return json(['code' => 0, 'msg' => 'success', 'data' => EventCatalog::all()]);
    }

    public function groups()
    {
        return $this->table('im_group_snapshot', 'updated_at');
    }

    public function members()
    {
        $where = [];
        $groupId = trim((string) input('group_id', ''));
        if ($groupId !== '') $where['group_id'] = $groupId;
        return $this->table('im_group_member_snapshot', 'updated_at', $where);
    }

    public function relations()
    {
        return $this->table('im_relation_snapshot', 'updated_at');
    }

    public function profiles()
    {
        return $this->table('im_user_profile_snapshot', 'updated_at');
    }

    public function states()
    {
        return $this->table('im_user_state_snapshot', 'updated_at');
    }

    private function table($table, $order, array $where = [])
    {
        if (!$this->authorized()) return $this->unauthorized();
        $page = max(1, (int) input('page', 1));
        $limit = min(100, max(1, (int) input('limit', 20)));
        $query = Db::name($table)->where($where);
        $status = trim((string) input('status', ''));
        if ($status !== '') $query->where($table === 'im_user_state_snapshot' ? 'state' : 'status', $status);
        $groupId = trim((string) input('group_id', '')); if ($groupId !== '' && in_array($table, ['im_group_snapshot','im_group_member_snapshot'], true)) $query->where('group_id', $groupId);
        $account = trim((string) input('account', ''));
        if ($account !== '') {
            $field = ['im_group_snapshot'=>'owner_account','im_group_member_snapshot'=>'account','im_relation_snapshot'=>'owner_account','im_user_profile_snapshot'=>'account','im_user_state_snapshot'=>'account'];
            if (isset($field[$table])) $query->where($field[$table], $account);
        }
        $uid = (int) input('business_uid', 0);
        if ($uid > 0) {
            $accounts = Db::name('im_account_mapping')->where('uid', $uid)->column('im_user_id');
            $accounts = array_values(array_unique(array_merge($accounts, [(string) $uid])));
            $field = ['im_group_snapshot'=>'owner_account','im_group_member_snapshot'=>'account','im_relation_snapshot'=>'owner_account','im_user_profile_snapshot'=>'account','im_user_state_snapshot'=>'account'];
            if (isset($field[$table])) $query->where($field[$table], 'in', $accounts);
        }
        $count = (clone $query)->count();
        $list = $query->order($order, 'desc')->page($page, $limit)->select();
        foreach ($list as &$item) {
            unset($item['raw_json'], $item['profile_json']);
            $accountValue = isset($item['account']) ? $item['account'] : (isset($item['owner_account']) ? $item['owner_account'] : '');
            $item['business_uid'] = $accountValue === '' ? null : (new AccountMappingService())->businessUid($accountValue, config('im_callback.sdk_app_id'));
            $item['source_event_name'] = !empty($item['source_event_id']) ? EventCatalog::name((string) Db::name('im_callback_event')->where('id', $item['source_event_id'])->value('callback_command')) : '';
        }
        return json(['code' => 0, 'msg' => 'success', 'count' => $count, 'data' => $list]);
    }

    private function authorized()
    {
        return (new AdminAuthService())->authorize((string) input('login_token', '')) > 0;
    }

    private function adminId()
    {
        return (new AdminAuthService())->authorize((string) input('login_token', ''));
    }

    private function unauthorized()
    {
        return json(['code' => 403, 'msg' => '无权访问 IM 回调数据', 'data' => null], 403);
    }

    private function attachBusinessUids(array &$items)
    {
        $resolver = new AccountMappingService();
        foreach ($items as &$item) {
            $appId = isset($item['sdk_app_id']) ? $item['sdk_app_id'] : config('im_callback.sdk_app_id');
            $item['from_uid'] = $resolver->businessUid(isset($item['from_account']) ? $item['from_account'] : '', $appId);
            $item['to_uid'] = $resolver->businessUid(isset($item['to_account']) ? $item['to_account'] : '', $appId);
        }
    }

    private function applyTimeRange($query)
    {
        $start = (int) input('start_time', 0); $end = (int) input('end_time', 0);
        if ($start > 0) $query->where('received_at', '>=', $start);
        if ($end > 0) $query->where('received_at', '<=', $end);
    }

    private function messageTypeLabel($raw)
    {
        $payload = json_decode((string) $raw, true); $types = [];
        if (is_array($payload) && !empty($payload['MsgBody']) && is_array($payload['MsgBody'])) {
            foreach ($payload['MsgBody'] as $body) if (!empty($body['MsgType'])) $types[] = $body['MsgType'];
        }
        return $types ? '[' . implode(', ', array_unique($types)) . ']' : '[消息内容已隐藏]';
    }
}
