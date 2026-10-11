<?php

namespace app\admin\controller;


class Capital extends Common
{
    //获取资金改变类型
    public function get_change_type()
    {
        $data = model('UserMoneyLog')->ChangeTypeLable();
        return ajaxReturn(200, '', $data);
    }

    //获取资金日志列表
    public function user_money_log()
    {
        $uid = input('uid', 0);
        $nick_name = input('nick_name', 0);
        $change_type = input('change_type', 0);
        $money_type = input('money_type', '');
        $change_value = input('change_value', '');
        $from_id = input('from_id', 0);
        $from_uid = input('from_uid', 0);
        $order = input('order', 'a.log_id');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Capital')->user_money_log($uid, $nick_name, $change_type, $money_type, $change_value, $from_id, $from_uid, $order, $sort, $page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        $data['totalRow'] = $reslut['data']['totalRow'];
        return json($data);
    }
    
    //获取提现订单 列表
    public function user_withdrawal()
    {
        $wid = input('wid', 0);
        $order_sn = input('order_sn', 0);
        $uid = input('uid', 0);
        $nick_name = input('nick_name', '');
        $money = input('money', '');
        $alipay_name = input('alipay_name', 0);
        $status = input('status', 0);
        $alipay_account = input('alipay_account', 0);
        $order = input('order', 'a.wid');
        $sort = input('sort', 'desc');
        $page = input('page', 1);
        $limit = input('limit', 20);
        $reslut = model('Capital')->user_withdrawal($wid, $order_sn, $uid, $nick_name, $money, $alipay_name, $status, $alipay_account, $order, $sort, $page, $limit);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['count'] = $reslut['data']['count'];
        $data['data'] = $reslut['data']['list'];
        $data['totalRow'] = $reslut['data']['totalRow'];
        return json($data);
    }

    //修改提现订单
    public function user_withdrawal_edit()
    {
        $wid = input('wid', 0);
        $status = input('status');
        $remarke = input('remarke');
        $data = model('Capital')->user_withdrawal_edit($wid, $status, $remarke);
        if ($data['code'] == 201) {
            return ajaxReturn(201, $data['msg'], $data['data']);
        } else {
            return ajaxReturn(200, $data['msg'], $data['data']);
        }
    }

    //获取提现订单详情
    public function user_withdrawal_info()
    {
        $wid = input('wid', 0);
        $reslut = model('Capital')->user_withdrawal_info($wid);
        $data = [];
        $data['code'] = 0;
        $data['msg'] = '获取成功';
        $data['data'] = $reslut['data'];
        return json($data);
    }
}
