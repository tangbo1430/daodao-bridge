<?php

namespace app\admin\model;

use think\Db;
use think\Model;

class Capital extends Model
{
    //用户资金日志
    public function user_money_log($uid, $nick_name, $change_type, $money_type, $change_value, $from_id, $from_uid, $order, $sort, $page = 1, $limit = 20)
    {
        $map = [];
        if (!empty($uid)) {
            $map[] = ['a.uid', '=', $uid];
        }
        if (!empty($nick_name)) {
            $map[] = ['b.nick_name', 'like', '%' . $nick_name . '%'];
        }
        if (!empty($change_type)) {
            $map[] = ['a.change_type', '=', $change_type];
        }
        if (!empty($change_value)) {
            $map[] = ['a.change_value', '=', $change_value];
        }
        if (!empty($from_id)) {
            $map[] = ['a.from_id', '=', $change_value];
        }
        if (!empty($from_uid)) {
            $map[] = ['a.from_uid', '=', $from_uid];
        }
        $change_type = model('UserMoneyLog')->ChangeTypeLable();
        $list = db::name('user_money_log')->alias('a')->join('yy_user b', 'a.uid = b.uid')->field('a.*,b.nick_name')->where($map)->order($order, $sort)->page($page, $limit)->select();
        foreach ($list as $k => $v) {
            $list[$k]['user_nick_name'] = $v['uid'] . '-' . $v['nick_name'];
            $list[$k]['change_type'] = $change_type[$v['change_type']];
        }
        $data = [];
        $data['count'] = db::name('user_money_log')->alias('a')->join('yy_user b', 'a.uid = b.uid')->where($map)->count();
        $data['list'] = $list;
        $totalRowData = db::name('user_money_log')->alias('a')->join('yy_user b', 'a.uid = b.uid')->field('count(a.log_id) as count,SUM(a.change_value) as change_value')->where($map)->find();
        unset($totalRowData['count']);
        //dump($totalRowData);
        $data['totalRow'] = $totalRowData;

        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }
    
    //用户提现订单
    public function user_withdrawal($wid, $order_sn, $uid, $nick_name, $money, $alipay_name, $status, $alipay_account, $order, $sort, $page = 1, $limit = 20)
    {
        $map = [];
        if (!empty($wid)) {
            $map[] = ['a.wid', '=', $wid];
        }
        if (!empty($order_sn)) {
            $map[] = ['a.order_sn', 'like', '%' . $order_sn . '%'];
        }
        if (!empty($nick_name)) {
            $map[] = ['b.nick_name', 'like', '%' . $nick_name . '%'];
        }
        if (!empty($uid)) {
            $map[] = ['a.uid', '=', $uid];
        }
        if (!empty($status)) {
            $map[] = ['a.status', '=', $status];
        }

        if (!empty($real_name)) {
            $map[] = ['a.real_name', '=', $alipay_name];
        }
        if (!empty($alipay_account)) {
            $map[] = ['a.alipay_account', '=', $alipay_account];
        }

        $list = db::name('user_withdrawal')->alias('a')->join('yy_user b', 'a.uid = b.uid')->field('a.*,b.nick_name')->where($map)->order($order, $sort)->page($page, $limit)->select();
        foreach ($list as $k => $v) {
            $list[$k]['user_nick_name'] = $v['uid'] . '-' . $v['nick_name'];
        }
        $data = [];

        $data['list'] = $list;
        $totalRowData = db::name('user_withdrawal')->alias('a')->join('yy_user b', 'a.uid = b.uid')->field('count(a.wid) as count,SUM(a.money) as money,SUM(a.general_money) as general_money')->where($map)->find();
        $data['count'] = $totalRowData['count'];
        unset($totalRowData['count']);
        //dump($totalRowData);
        $data['totalRow'] = $totalRowData;

        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }

    //获取提现申请详情
    public function user_withdrawal_info($wid)
    {
        if (empty($wid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $user_withdrawal = db::name('user_withdrawal')->where(['wid' => $wid])->find();
        return ['code' => 200, 'msg' => '获取成功', 'data' => $user_withdrawal];
    }

    //修改提现申请
    public function user_withdrawal_edit($wid, $status, $remarke)
    {
        if (empty($wid)) {
            return ['code' => 201, 'msg' => '参数异常', 'data' => null];
        }
        $info  = db::name('user_withdrawal')->where(['wid' => $wid])->find();
        if (empty($info)) {
            return ['code' => 201, 'msg' => '信息不存在', 'data' => null];
        }
        if ($info['status'] != 1) {
            return ['code' => 201, 'msg' => '该提现已处理', 'data' => null];
        }
        Db::startTrans();
        try {
            $arr = [];
            $arr['status'] = $status;
            $arr['remarke'] = $remarke;
            $arr['update_time'] = time();
            $reslut = db::name('user_withdrawal')->where(['wid' => $wid])->update($arr);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => "处理失败", 'data' => null];
            }
            if ($status == 3) {
                // //提现驳回
                // $reslut = model('api/UserMessage')->send_message($info['uid'], 1, 0, '提现审核', '您的提现申请已被驳回,原因如下：' . $remarke);
                // if ($reslut['code'] != 200) {
                //     Db::rollback();
                //     return ['code' => 201, 'msg' => "处理失败", 'data' => null];
                // }
                $reslut = model('admin/User')->change_user_money_by_uid($info['uid'], $info['money'], 6, "余额提现驳回", $info['uid'], $info['wid']);
                if ($reslut['code'] != 200) {
                    Db::rollback();
                    return ['code' => 201, 'msg' => "处理失败", 'data' => null];
                }
            }
            // 提交事务
            Db::commit();
            return ['code' => 200, 'msg' => '处理成功', 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            Db::rollback();
            return ['code' => 201, 'msg' => '处理失败', 'data' => null];
        }
    }
}