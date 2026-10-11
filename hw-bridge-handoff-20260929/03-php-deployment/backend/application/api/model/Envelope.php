<?php

namespace app\api\model;

use think\Db;
use think\Model;

class Envelope extends Model
{
    public function generateRedPacketsNew($totalAmount, $count, $minAmount = 0.01, $maxAmount = null)
    {
        
        // 参数验证
        if ($totalAmount <= 0 || $count <= 0) {
            return ['code' => 0, 'msg' => '参数错误', 'data' => []];
        }
        
        // 确保每个红包至少有0.01元
        $minAmount = 0.01;
        if ($totalAmount < $count * $minAmount) {
            return ['code' => 0, 'msg' => "总金额太小，无法分配", 'data' => []];
        }
        
        // 转换为分为单位进行计算（纯整数运算）
        $totalCents = intval($totalAmount * 100);
        $minCents = intval($minAmount * 100);
        
        // 计算平均金额（分为单位）
        $avgCents = intval($totalCents / $count);
        $remainderCents = $totalCents % $count;
        
        // 创建红包数组（以分为单位）
        $redPacketsCents = [];
        
        // 先给每个红包分配平均金额
        for ($i = 0; $i < $count; $i++) {
            $redPacketsCents[] = $avgCents;
        }
        
        // 将余数随机分配给红包
        for ($i = 0; $i < $remainderCents; $i++) {
            $randomIndex = mt_rand(0, $count - 1);
            $redPacketsCents[$randomIndex]++;
        }
        
        // 验证整数运算的总和
        $totalCentsCalculated = array_sum($redPacketsCents);
        if ($totalCentsCalculated !== $totalCents) {
            return ['code' => 0, 'msg' => '整数运算错误', 'data' => []];
        }
        
        // 验证每个红包都大于等于最小金额
        foreach ($redPacketsCents as $cents) {
            if ($cents < $minCents) {
                return ['code' => 0, 'msg' => '红包金额小于最小金额', 'data' => []];
            }
        }
        
        // 转换为元为单位（只在显示时转换）
        $redPacketsInYuan = [];
        foreach ($redPacketsCents as $cents) {
            $redPacketsInYuan[] = $cents / 100;
        }
        
        // 计算金额差异
        $minAmountInArray = min($redPacketsInYuan);
        $maxAmountInArray = max($redPacketsInYuan);
        $amountDifference = $maxAmountInArray - $minAmountInArray;
        $avgAmount = $totalAmount / $count;
        $differencePercent = ($amountDifference / $avgAmount) * 100;
        
        // 如果差异太小（<20%），增加随机性
        if ($differencePercent < 20) {
            // 重新分配，增加随机性
            return $this->grabRedPacketWithRandomness($totalAmount, $count);
        }
        
        // 打乱红包顺序
        shuffle($redPacketsInYuan);
        return $redPacketsInYuan;
    }
    
   /**
     * 带随机性的红包分配方法
     * @param float $totalAmount 总金额
     * @param int $count 红包个数
     * @return array 红包金额数组
     */
    /**
     * 带随机性的红包分配方法
     * @param float $totalAmount 总金额
     * @param int $count 红包个数
     * @return array 红包金额数组
     */
    private function grabRedPacketWithRandomness($totalAmount, $count)
    {
        // 转换为分为单位
        $totalCents = intval($totalAmount * 100);
        $minCents = 1; // 最小1分
        
        // 计算基础平均金额
        $baseAvgCents = intval($totalCents / $count);
        
        // 创建红包数组
        $redPacketsCents = [];
        $remainingCents = $totalCents;
        
        // 随机分配红包
        for ($i = 0; $i < $count - 1; $i++) {
            // 确保剩余红包都能分到最小金额
            $maxPossible = $remainingCents - ($count - $i - 1) * $minCents;
            
            // 在最小金额和最大可能金额之间随机分配
            $minForThis = max($minCents, $baseAvgCents * 0.6); // 不低于平均值的60%
            $maxForThis = min($maxPossible, $baseAvgCents * 1.4); // 不超过平均值的140%
            
            $amount = mt_rand(intval($minForThis), intval($maxForThis));
            $redPacketsCents[] = $amount;
            $remainingCents -= $amount;
        }
        
        // 最后一个红包分配剩余金额
        $redPacketsCents[] = $remainingCents;
        
        // 验证总和
        if (array_sum($redPacketsCents) !== $totalCents) {
            return $this->grabRedPacketWithRandomness($totalAmount, $count);
        }
        
        // 转换为元
        $redPacketsInYuan = [];
        foreach ($redPacketsCents as $cents) {
            $redPacketsInYuan[] = $cents / 100;
        }
        
        // 计算差异
        $minAmount = min($redPacketsInYuan);
        $maxAmount = max($redPacketsInYuan);
        $amountDifference = $maxAmount - $minAmount;
        $avgAmount = $totalAmount / $count;
        $differencePercent = ($amountDifference / $avgAmount) * 100;
        // 如果差异不在20%-50%范围内，重新分配
        if ($differencePercent < 30 || $differencePercent > 60) {
            return $this->grabRedPacketWithRandomness($totalAmount, $count);
        }
        // 打乱顺序
        shuffle($redPacketsInYuan);
        
        return $redPacketsInYuan;
    }
    /**
     * 生成指定范围内的随机浮点数
     * 
     * @param float $min 最小值
     * @param float $max 最大值
     * @return float 随机浮点数
     */
    public function generateByDoubleMean($totalAmount, $count, $minAmount, $maxAmount)
    {
        $packets = [];
        $remainingAmount = $totalAmount;
        $remainingCount = $count;
        
        for ($i = 0; $i < $count; $i++) {
            if ($remainingCount == 1) {
                // 最后一个红包，分配剩余金额
                $amount = $remainingAmount;
            } else {
                // 计算当前红包的最大可能金额
                $currentMaxAmount = min($maxAmount, $remainingAmount - ($remainingCount - 1) * $minAmount);
                
                // 使用二倍均值算法
                $average = $remainingAmount / $remainingCount;
                $maxPossible = min($currentMaxAmount, $average * 2);
                
                // 生成随机金额
                $amount = $this->generateRandomAmount($minAmount, $maxPossible);
            }
            
            // 优化金额，减少小数位数
            $amount = $this->optimizeAmount($amount);
            
            $packets[] = $amount;
            $remainingAmount -= $amount;
            $remainingCount--;
        }
        
        // 验证总金额
        $actualTotal = array_sum($packets);
        if (abs($actualTotal - $totalAmount) > 0.01) {
            // 如果总金额不匹配，调整最后一个红包
            $diff = $totalAmount - $actualTotal;
            $lastIndex = count($packets) - 1;
            $newAmount = $packets[$lastIndex] + $diff;
            
            // 确保调整后的金额不小于0.01
            if ($newAmount < 0.01) {
                // 如果调整后金额太小，需要重新分配
                $packets = $this->redistributeAmounts($packets, $totalAmount);
            } else {
                $packets[$lastIndex] = $this->optimizeAmount($newAmount);
            }
        }
        // 将最大金额的红包移到第一个位置
        $maxIndex = array_keys($packets, max($packets))[0];
        if ($maxIndex > 0) {
            $maxAmount = $packets[$maxIndex];
            array_splice($packets, $maxIndex, 1);
            array_unshift($packets, $maxAmount);
        }
        return $packets;
    }
    
    /**
     * 生成指定范围内的随机金额
     * 优化算法，优先生成整数或简单小数
     * 
     * @param float $min 最小金额
     * @param float $max 最大金额
     * @return float
     */
    private function generateRandomAmount($min, $max)
    {
        // 确保最小金额不小于0.01
        $min = max(0.01, $min);
        
        // 优先生成整数金额的概率
        $integerProbability = 0.6;
        
        if (mt_rand(1, 100) <= $integerProbability * 100 && $min <= floor($max) && floor($max) >= 1) {
            // 生成整数金额
            $intMin = max(1, ceil($min));
            $intMax = floor($max);
            if ($intMin <= $intMax) {
                return mt_rand($intMin, $intMax);
            }
        }
        
        // 生成简单小数（0.5的倍数）
        $simpleDecimalProbability = 0.3;
        if (mt_rand(1, 100) <= $simpleDecimalProbability * 100) {
            $halfMin = max($min, 0.5);
            $halfMax = min($max, floor($max) + 0.5);
            if ($halfMin <= $halfMax) {
                $base = mt_rand(ceil($halfMin * 2), floor($halfMax * 2));
                $amount = $base / 2;
                if ($amount >= $min && $amount <= $max) {
                    return $amount;
                }
            }
        }
        
        // 使用简化的随机算法，减少复杂计算
        $random = mt_rand() / mt_getrandmax();
        $amount = $min + $random * ($max - $min);
        
        // 确保在范围内且不小于0.01
        return max(0.01, min($max, $amount));
    }
    
    /**
     * 优化金额，减少小数位数
     * 
     * @param float $amount 原始金额
     * @return float 优化后的金额
     */
    private function optimizeAmount($amount)
    {
        // 确保金额不小于0.01
        if ($amount < 0.01) {
            return 0.01;
        }
        
        // 如果金额接近整数（差值小于0.1），调整为整数
        $intAmount = round($amount);
        if ($intAmount >= 0.01 && abs($amount - $intAmount) < 0.1) {
            return $intAmount;
        }
        
        // 如果金额接近0.5的倍数，调整为0.5的倍数
        $halfAmount = round($amount * 2) / 2;
        if ($halfAmount >= 0.01 && abs($amount - $halfAmount) < 0.05) {
            return $halfAmount;
        }
        
        // 如果金额接近0.25的倍数，调整为0.25的倍数
        $quarterAmount = round($amount * 4) / 4;
        if ($quarterAmount >= 0.01 && abs($amount - $quarterAmount) < 0.03) {
            return $quarterAmount;
        }
        
        // 如果金额接近0.1的倍数，调整为0.1的倍数
        $tenthAmount = round($amount * 10) / 10;
        if ($tenthAmount >= 0.01 && abs($amount - $tenthAmount) < 0.02) {
            return $tenthAmount;
        }
        
        // 最后才保留两位小数，确保不小于0.01
        $roundedAmount = round($amount, 2);
        return max(0.01, $roundedAmount);
    }
    
    /**
     * 重新分配红包金额，确保每个红包都不小于0.01
     * 
     * @param array $packets 当前红包数组
     * @param float $totalAmount 总金额
     * @return array 重新分配后的红包数组
     */
    private function redistributeAmounts($packets, $totalAmount)
    {
        $count = count($packets);
        $minAmount = 0.01;
        
        // 确保每个红包至少有最小金额
        $reservedAmount = $count * $minAmount;
        $remainingAmount = $totalAmount - $reservedAmount;
        
        if ($remainingAmount < 0) {
            // 如果总金额不够分配最小金额，抛出异常
            throw new \InvalidArgumentException('总金额太小，无法为每个红包分配最小金额');
        }
        
        // 重新分配剩余金额
        $newPackets = [];
        for ($i = 0; $i < $count; $i++) {
            if ($i == $count - 1) {
                // 最后一个红包，分配所有剩余金额
                $amount = $minAmount + $remainingAmount;
            } else {
                // 随机分配剩余金额的一部分
                $maxPossible = $remainingAmount / ($count - $i);
                $randomAmount = $this->generateRandomAmount(0.01, $maxPossible);
                $amount = $minAmount + $randomAmount;
                $remainingAmount -= $randomAmount;
            }
            
            $newPackets[] = $this->optimizeAmount($amount);
        }
        
        return $newPackets;
    }

    /**
     * 生成金额相近的红包
     * @param float $totalAmount 总金额
     * @param int $count 红包个数
     * @param float $minAmount 最小金额
     * @return array
     */
    public function generateSimilarRedPackets($totalAmount, $count, $minAmount = 0.01)
    {
        if ($totalAmount <= 0 || $count <= 0) {
            throw new \InvalidArgumentException('总金额和红包个数必须大于0');
        }
        if ($minAmount * $count > $totalAmount) {
            throw new \InvalidArgumentException('最小金额设置过大，无法分配');
        }
        
        $base = floor(($totalAmount / $count) * 100) / 100; // 保留两位小数
        $packets = array_fill(0, $count, $base);
        $remaining = round($totalAmount - $base * $count, 2);
        
        // 在每个红包基础上做微小随机调整，幅度不超过±10%
        for ($i = 0; $i < $count - 1; $i++) {
            $maxDelta = min($base * 0.1, $packets[$i] - $minAmount); // 最大可减少幅度
            $minDelta = max(-$maxDelta, -$packets[$i] + $minAmount); // 最大可增加幅度
            $delta = 0;
            if ($maxDelta > 0) {
                $delta = round((mt_rand(-100, 100) / 100) * $maxDelta, 2);
            }
            $packets[$i] += $delta;
            $packets[$i] = max($minAmount, round($packets[$i], 2));
            $remaining -= $delta;
        }
        // 最后一个红包补齐差额
        $packets[$count - 1] += round($remaining, 2);
        $packets[$count - 1] = max($minAmount, round($packets[$count - 1], 2));
        return $packets;
    }
    
    /**
     * 生成红包金额
     * @param float $total 红包总金额
     * @param int $num 红包数量
     * @param float $min 最小红包金额
     * @return array 红包金额数组
     * @throws \InvalidArgumentException
     */
    public function generateRedPackets(float $total, int $num, float $min = 0.01): array
    {
        

        // 如果只有一个红包，直接返回总金额
        if ($num === 1) {
            return [round($total, 2)];
        }

        $redPackets = [];
        $remainAmount = $total;
        $remainNum = $num;

        // 生成n-1个红包
        for ($i = 0; $i < $num - 1; $i++) {
            // 剩余每个人至少要分到的金额
            $minRemain = $min;
            // 剩余待分配的平均值
            $avgRemain = round(($remainAmount - $minRemain * $remainNum) / $remainNum, 2);

            // 计算本次红包的最大可能金额
            $maxPossible = $remainAmount - ($remainNum - 1) * $minRemain;

            // 保证本次红包金额不会过大，不超过剩余平均值的2倍
            $maxPossible = min($maxPossible, $avgRemain * 2);

            // 在最小值和最大可能金额之间随机生成红包金额
            $amount = round(mt_rand($minRemain * 100, $maxPossible * 100) / 100, 2);

            $redPackets[] = $amount;
            $remainAmount = round($remainAmount - $amount, 2);
            $remainNum--;
        }

        // 最后一个红包直接使用余额
        $redPackets[] = round($remainAmount, 2);

        // 打乱红包顺序
        shuffle($redPackets);

        return $redPackets;
    }
    
    //发红包
    public function give_red_envelope($uid, $group_id, $money, $num, $title, $interval = 5, $trade_password, $is_special = 2, $receive_uid = 0){
        $user_info = db::name('user')->field('uid, money, nick_name, base64_nick_name, head_pic, trade_password')->find($uid);
        if(!$user_info){
            return ['code' => 201, 'msg' => '参数错误', 'data' => null];
        }
        if(empty($user_info['trade_password'])) {
            return ['code' => 602, 'msg' => '请设置交易密码', 'data' => null];
        }
        if(empty($trade_password)) {
            return ['code' => 201, 'msg' => '请输入支付密码', 'data' => null];
        }
        if(md5($trade_password) != $user_info['trade_password']) {
            return ['code' => 201, 'msg' => '支付密码错误', 'data' => null];
        }
        $group_v_id = $group_id;
        $nick_name = mb_convert_encoding(base64_decode($user_info['base64_nick_name']), 'UTF-8', 'UTF-8');
        // if(strpos($group_id, 'G_') !== true) {
        //     $group_id_arr = explode('_', $group_id);
        //     $group_id = $group_id_arr[1];
        // }
        //家族是否存在
        $map = [];
        $map[] = ['id', '=', $group_id];
        $map[] = ['is_delete', '=', 1];
        $group_info = db::name('group_v')->where($map)->find();
        if(!$group_info){
            return ['code' => 201, 'msg' => '群不存在', 'data' => null];
        }
        
        $envelope_type_info = db::name('envelope_type')->where('id', 3)->find();
        if(!$envelope_type_info){
            return ['code' => 201, 'msg' => '红包类型错误', 'data' => null];
        }
        if($money <= 0) {
            return ['code' => 201, 'msg' => '红包金额不能小于0', 'data' => null];
        }
        if($money < 0.1) {
            return ['code' => 201, 'msg' => '红包金额不能小于0.1', 'data' => null];
        }
        if($num <= 0) {
            return ['code' => 201, 'msg' => '抢红包人数不能小于0', 'data' => null];
        }
        $one_money = bcdiv($money, $num, 2);
        if($one_money < 0.01) {
            return ['code' => 201, 'msg' => '单个红包不能小于0.01', 'data' => null];
        }
        // $money = $envelope_type_info['max_price'];
        // $num = $envelope_type_info['max_num'];
        
        // if($money > $envelope_type_info['max_price']){
        //     return ['code' => 201, 'msg' => '红包金额不能大于'.$envelope_type_info['max_price'], 'data' => null];
        // }
        $uid_str = strval($uid);
        $result = model('api/Tencent')->get_group_member_info($group_v_id, [$uid_str], ['Owner', 'Admin']);
        
        if($result['code'] == 201) {
            return ['code' => 201, 'msg' => '查找数据错误', 'data' => null];
        } else {
            if(empty($result['data'])) {
                return ['code' => 201, 'msg' => '不是管理或群主', 'data' => null];
            }
        }
        
        
        //是否是专属红包
        $receive_nick_name = '';
        if($is_special == 1) {
            if(empty($receive_uid)) {
                return ['code' => 201, 'msg' => '请输入专属红包接受人', 'data' => null];
            }
            if($num != 1) {
                return ['code' => 201, 'msg' => '专属红包只能发放一个', 'data' => null];
            }
            $receive_user_info = Db::name('user')->where('uid', $receive_uid)->field('uid,base64_nick_name')->find();
            if(empty($receive_user_info)) {
                return ['code' => 201, 'msg' => '收红包用户不存在', 'data' => null];
            }
            $num = 1;
            $receive_nick_name = mb_convert_encoding(base64_decode($receive_user_info['base64_nick_name']), 'UTF-8', 'UTF-8');
        }
        
        if($user_info['money'] < $money){
            return ['code' => 201, 'msg' => '金币不足', 'data' => null];
        }
        $now_time = time();
        Db::startTrans();
        try {
            //生成红包
            $insert_data = [];
            $insert_data['type'] = 3;
            $insert_data['uid'] = $uid;
            $insert_data['group_id'] = $group_id;
            $insert_data['give_type'] = 2;
            $insert_data['title'] = $title;
            $insert_data['base64_title'] = base64_encode($title);
            $insert_data['money'] = $money;
            $insert_data['price'] = $money;
            $insert_data['num'] = $num;
            $insert_data['open_num'] = 0;
            $insert_data['interval'] = $interval;
            $insert_data['start_time'] = $now_time;
            $insert_data['add_time'] = $now_time;
            $insert_data['end_time'] = $now_time + (60*60*24 * 300);
            $insert_data['is_special'] = $is_special;
            $insert_data['receive_uid'] = $receive_uid;
            $sid = db::name('red_envelope')->insertGetId($insert_data);
            if(!$sid){
                Db::rollback();
                return ['code' => 201, 'msg' => '请重试', 'data' => null];
            }
            
            //扣除蓝豆
            $resluts = model('admin/User')->change_user_money_by_uid($uid, -$money, 3, "发红包支出", $uid, $sid);
            if($resluts['code'] != 200){
                Db::rollback();
                return ['code' => 201, 'msg' => $resluts['msg'], 'data' => null];
            }
            
            //生成redis
            $redis = connectionRedis();
            $key_name = "api:FamilyEnvelope:envelope_id:".$sid;
            
            // $max_price = bcdiv($money, 2, 3);
            // $min_price = bcdiv($money, ($num + 2), 3);
            $envelope_price_array = $this->generateRedPackets($money, $num);
            // dump($envelope_price_array);die;
            $envelope_insert_data = $envelope_price_array;
            array_unshift($envelope_insert_data, $key_name);
            call_user_func_array([$redis, 'rPush'], $envelope_insert_data);
            $push_data = [
                
                    'send_uid' => $uid_str,
                    'head_pic' => localpath_to_netpath($user_info['head_pic']),
                    'envelope_id' => $sid,
                    'send_nick_name' => $nick_name,
                    'businessID' => 'send_envelope',
                    'is_special' => $is_special,
                    'receive_uid' => $receive_uid,
                    'receive_nick_name' => $receive_nick_name
                 
                ];
            
            model('api/WebSocketPushIm')->send_to_group($group_v_id, $push_data, $uid_str, [strval($receive_uid), $uid_str]);
            Db::commit();
            
            return ['code' => 200, 'msg' => '成功', 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            Db::rollback();
            dump($e);
            return ['code' => 201, 'msg' => '失败', 'data' => null];
        }
    }
    
    //抢红包
    public function snatch_red_envelope($uid, $group_id, $envelope_id){
        $user_info = db::name('user')->field('uid,money,is_real,base64_nick_name')->find($uid);
        if(!$user_info){
            return ['code' => 201, 'msg' => '参数错误', 'data' => null];
        }
        $group_v_id = $group_id;
        // if(strpos($group_id, 'G_') !== false) {
        //     $group_id_arr = explode('_', $group_id);
        //     $group_id = $group_id_arr[1];
        // }
        if($user_info['is_real'] != 1){
            return ['code' => 602, 'msg' => '未实名认证的无法参与', 'data' => null];
        }
        $receive_nick_name = mb_convert_encoding(base64_decode($user_info['base64_nick_name']), 'UTF-8', 'UTF-8');
        //该家族是否存在
        
        $map = [];
        $map[] = ['id', '=', $group_id];
        $map[] = ['is_delete', '=', 1];
        $group_info = db::name('group_v')->where($map)->find();
        if(!$group_info){
            return ['code' => 201, 'msg' => '该群组不存在', 'data' => null];
        }
        
        //是否有进行中的
        $map = [];
        $map[] = ['id', '=', $envelope_id];
        $map[] = ['is_finish', '=', 2];
        $map[] = ['is_delete', '=', 1];
        $map[] = ['is_stop', '=', 2];
        $red_envelope_info = db::name('red_envelope')->where($map)->find();
        if(!$red_envelope_info){
            return ['code' => 201, 'msg' => '该红包已结束', 'data' => null];
        }
        if($red_envelope_info['is_special'] == 1) {
            if($uid != $red_envelope_info['receive_uid']) {
                return ['code' => 201, 'msg' => '您不是专属红包的抢红包人', 'data' => null];
            }
        }
        $send_user_info =db::name('user')->field('uid,money,is_real,base64_nick_name')->find($red_envelope_info['uid']);
        if(empty($send_user_info)) {
            return ['code' => 201, 'msg' => '发红包人不存在', 'data' => null];
        }
        $send_nick_name = mb_convert_encoding(base64_decode($send_user_info['base64_nick_name']), 'UTF-8', 'UTF-8');
        //是否到达可抢时间
        if($red_envelope_info['start_time'] > time()){
            return ['code' => 201, 'msg' => '该红包尚未到达开启时间', 'data' => null];
        }
        
        //是否抢过该红包
        $map = [];
        $map[] = ['uid', '=', $uid];
        $map[] = ['eid', '=', $red_envelope_info['id']];
        $user_snatch_red_envelope = db::name('user_red_envelope_log')->where($map)->find();
        if($user_snatch_red_envelope){
            return ['code' => 201, 'msg' => '你已抢过该红包', 'data' => null];
        }
        
        //该红包数量是否剩余
        $map = [];
        $map[] = ['eid', '=', $red_envelope_info['id']];
        $user_snatch_red_envelope_sum = db::name('user_red_envelope_log')->where($map)->count();
        if($user_snatch_red_envelope_sum == $red_envelope_info['num']){
            return ['code' => 201, 'msg' => '红包已被抢完2', 'data' => null];
        }
        
        
        $redis = connectionRedis();
        $user_red_envelope_id_list = [];
        $keyname = "api:FamilyEnvelope:envelope_id:".$red_envelope_info['id'];
        //抢红包记录
        $list_len = $redis->llen($keyname);
        if($list_len == 0){
            return ['code' => 201, 'msg' => '该红包已被抢完1', 'data' => null];
        }
        $snatch_price_data = $redis->blpop($keyname, 2);
        if(empty($snatch_price_data)){
            return ['code' => 201, 'msg' => '红包已被抢完3', 'data' => null];
        }
        $is_can_redis_callback = 1;
        Db::startTrans();
        try {
            $snatch_price = $snatch_price_data[1];
            
            $data = [];
            $data['price'] = $snatch_price;
            $user_red_envelope_id_list[] = $data;
            
            //记录
            $insert_data_log = [];
            $insert_data_log['uid'] = $uid;
            $insert_data_log['group_id'] = $group_id;
            $insert_data_log['type'] = $red_envelope_info['type'];
            $insert_data_log['eid'] = $red_envelope_info['id'];
            $insert_data_log['money'] = $red_envelope_info['money'];
            $insert_data_log['price'] = $red_envelope_info['price'];
            $insert_data_log['snatch_price'] = $snatch_price;
            $insert_data_log['num'] = $red_envelope_info['num'];
            $insert_data_log['add_time'] = time();
            $sid = db::name('user_red_envelope_log')->insertGetId($insert_data_log);
            if(!$sid){
                Db::rollback();
                $this->redis_envelope_data_rollback($redis, $keyname, $user_red_envelope_id_list);
                return ['code' => 201, 'msg' => '请重试', 'data' => null];
            }
            
            //修改红包已开次数
            $map = [];
            $map[] = ['id', '=', $red_envelope_info['id']];
            $map[] = ['open_num', '<', $red_envelope_info['num']];
            $update = [];
            $update['update_time'] = time();
            $reslut = db::name('red_envelope')->where($map)->inc('open_num', 1)->update($update);
            if(!$reslut){
                Db::rollback();
                $this->redis_envelope_data_rollback($redis, $keyname, $user_red_envelope_id_list);
                return ['code' => 201, 'msg' => '请重试', 'data' => null];
            }
            
            //抢红包收入
            $resluts = model('admin/User')->change_user_money_by_user_info($user_info, $snatch_price, 4, "抢红包收入", $uid, $sid);
            if(!$resluts){
                Db::rollback();
                $this->redis_envelope_data_rollback($redis, $keyname, $user_red_envelope_id_list);
                return ['code' => 201, 'msg' => '请重试', 'data' => null];
            }
            
            //是否最后一个红包
            $where = [];
            $where[] = ['eid', '=', $red_envelope_info['id']];
            $red_envelope_list = db::name('user_red_envelope_log')->where($where)->order('snatch_price desc,id asc')->select();
            if(count($red_envelope_list) == $red_envelope_info['num']){
                //修改红包状态
                $update_data = [];
                $update_data['is_finish'] = 1;
                $update_data['is_stop'] = 1;
                $update_data['stop_time'] = time();
                $reslut = db::name('red_envelope')->where('id', $red_envelope_info['id'])->update($update_data);
                if(!$reslut){
                    Db::rollback();
                    $this->redis_envelope_data_rollback($redis, $keyname, $user_red_envelope_id_list);
                    return ['code' => 201, 'msg' => '请重试', 'data' => null];
                }
                
                //找到手气最佳新生成一轮红包
                foreach ($red_envelope_list as $k => $v){
                    if($k == 0){
                        //修改手气最佳
                        $update_data = [];
                        $update_data['is_lucky'] = 1;
                        $reslut = db::name('user_red_envelope_log')->where('id', $v['id'])->update($update_data);
                        if(!$reslut){
                            Db::rollback();
                            $this->redis_envelope_data_rollback($redis, $keyname, $user_red_envelope_id_list);
                            return ['code' => 201, 'msg' => '请重试', 'data' => null];
                        }
                    }
                }

            }
            // $push_data = [
            //     'receive_uid' => $uid,
            //     'send_uid' => $send_user_info['uid'],
            //     'send_nick_name' => $send_nick_name,
            //     'receive_nick_name' => $receive_nick_name,
            //     'msg_type' => 'receive_envelope',
            // ];
            // $result = model('api/WebSocketPushIm')->send_group_system_notification($group_v_id, json_encode($push_data));
            $uid_str = strval($uid);
            $push_data = [
                'receive_uid' => $uid,
                'send_uid' => $send_user_info['uid'],
                'send_nick_name' => $send_nick_name,
                'receive_nick_name' => $receive_nick_name,
                'businessID' => 'receive_envelope',
            ];
            // $to_account = [$uid_str, strval($send_user_info['uid'])];
            $to_account = [];
            // $result = model('api/WebSocketPushIm')->send_to_group($group_v_id, $push_data, 0, $to_account);
            // dump($result);die;
            Db::commit();
            $is_can_redis_callback = 0;
            
            $data = [];
            $data['snatch_price'] = $snatch_price;
            
            return ['code' => 200, 'msg' => '成功', 'data' => $data];
        } catch (\Exception $e) {
            // 回滚事务
            
            if($is_can_redis_callback == 1) {
                $this->redis_envelope_data_rollback($redis, $keyname, $user_red_envelope_id_list);
            }
            Db::rollback();
            dump($e);
            return ['code' => 201, 'msg' => '失败', 'data' => null];
        }
        
    }
    
    //返还用户抽中红包金额到队列
    private function redis_envelope_data_rollback($redis, $keyname, $user_red_envelope_id_list)
    {
        $insert_data = [];
        foreach ($user_red_envelope_id_list as $k => $v) {
            $insert_data[] = $v['price'];
        }
        if (!empty($insert_data)) {
            array_unshift($insert_data, $keyname);
            //右侧取 右侧插入
            call_user_func_array([$redis, 'lPush'], $insert_data);
        }
        return ['code' => 200, 'msg' => '返还成功', 'data' => null];
    }
    
    //红包记录信息
    public function red_envelope_info($uid, $envelope_id){
        $user_info = db::name('user')->find($uid);
        if(!$user_info){
            return ['code' => 201, 'msg' => '参数错误', 'data' => null];
        }
        
        $data = [];
        
        //红包信息
        $map = [];
        $map[] = ['id', '=', $envelope_id];
        $red_envelope_info = db::name('red_envelope')->where($map)->find();
        $data['is_open'] = 2;
        if($red_envelope_info) {
            if($red_envelope_info['num'] > $red_envelope_info['open_num']) {
                $data['is_open'] = 1;
                // if($red_envelope_info['is_special'] == 1) {
                //     if($red_envelope_info['receive_uid'] == $uid) {
                //         $data['is_open'] = 1;
                //     } else {
                //         $data['is_open'] = 2;
                //     }
                // } else {
                //     $data['is_open'] = 1;
                // }
                
            }
            
            
        }
        //当前用户是否已开
        $map = [];
        $map[] = ['uid', '=', $uid];
        $map[] = ['eid', '=', $envelope_id];
        // dump($map);die;
        $user_red_envelope_info = db::name('user_red_envelope_log')->where($map)->find();
        // dump($user_red_envelope_info);die;
        if($user_red_envelope_info){
            $data['is_open'] = 3;
            $data['snatch_price'] = $user_red_envelope_info['snatch_price'];
        }else{
            $data['snatch_price'] = 0;
        }
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
    }
    
    //抢完红包记录信息
    public function snatch_red_envelope_info($uid, $id){
        $user_info = db::name('user')->find($uid);
        if(!$user_info){
            return ['code' => 201, 'msg' => '参数错误', 'data' => null];
        }
        
        $data = [];
        
        //红包信息
        $map = [];
        $map[] = ['id', '=', $id];
        $red_envelope_info = db::name('red_envelope')->alias('a')->join('yy_user b', 'a.uid = b.uid')->where($map)->field('a.id as envelope_id,a.type,a.uid,a.title,a.base64_title,a.money,a.num,a.open_num,a.add_time,a.end_time,a.is_finish,a.is_stop, b.nick_name, b.base64_nick_name, b.head_pic,a.is_special')->find();
        if(!$red_envelope_info){
            return ['code' => 201, 'msg' => '该红包信息不存在', 'data' => null];
        }
        $red_envelope_info['nick_name'] = mb_convert_encoding(base64_decode($red_envelope_info['base64_nick_name']), 'UTF-8', 'UTF-8');
        $red_envelope_info['head_pic'] = localpath_to_netpath($red_envelope_info['head_pic']);
        if($red_envelope_info['is_finish'] == 1 && $red_envelope_info['is_stop'] == 1){
            $red_envelope_info['surplus_time'] = 0;
        }else{
            $red_envelope_info['surplus_time'] = $red_envelope_info['end_time'] - time();
            if($red_envelope_info['surplus_time'] < 0){
                $red_envelope_info['surplus_time'] = 0;
            }
        }
        
        $data['red_envelope_info'] = $red_envelope_info;
        
        //已抢用户信息
        $user_red_envelope_list = db::name('user_red_envelope_log')->alias('a')->join('yy_user b', 'a.uid = b.uid')->field('a.id as envelope_id, a.uid, a.snatch_price, a.add_time, a.is_lucky, b.nick_name, b.base64_nick_name, b.head_pic')->where('a.eid', $id)->order('a.add_time desc')->select();
        $my_red_envelope_money = 0;
        $total_snatch_red_envelope_money = 0;
        foreach ($user_red_envelope_list as $k => &$v){
            $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']), 'UTF-8', 'UTF-8');
            $v['head_pic'] = localpath_to_netpath($v['head_pic']);
            if($v['uid'] == $uid) {
                $my_red_envelope_money = $v['snatch_price'];
            }
            if($red_envelope_info['is_special'] == 1) {
                $v['is_lucky'] = 2;
            }
            $total_snatch_red_envelope_money += $v['snatch_price'];
        }
        $data['user_red_envelope_list'] = $user_red_envelope_list;
        $data['user_red_envelope_count'] = count($user_red_envelope_list);//抢红包人数
        $data['my_red_envelope_money'] = $my_red_envelope_money;
        $data['total_snatch_red_envelope_money'] = bcmul($total_snatch_red_envelope_money, 1, 2);//已抢金额
        $data['red_envelope_info']['money'] = bcmul($red_envelope_info['money'], 1, 2);
        return ['code' => 200, 'msg' => '获取成功', 'data' => $data];
        
    }
    
    
    //家族红包限时结束
    public function red_envelope_time_limit(){
        Db::startTrans();
        try {
            $map = [];
            $map[] = ['is_stop', '=', 2];
            $map[] = ['is_delete', '=', 1];
            $map[] = ['type', '=', 3];
            $red_envelope_list = db::name('red_envelope')->where($map)->select();
            if($red_envelope_list){
                foreach ($red_envelope_list as $k => $v){
                    //是否超时
                    if($v['end_time'] < time()){
                        $red_envelope_log_count = db::name('user_red_envelope_log')->where('eid', $v['id'])->count();
                        if(!empty($red_envelope_log_count)){
                            $user_total_snatch_price = db::name('user_red_envelope_log')->where('eid', $v['id'])->sum('snatch_price');
                            $open_num = $red_envelope_log_count;
                            if($v['num'] != $open_num){
                                $surplus_price = $v['money'] - $user_total_snatch_price;
                            }else{
                                $surplus_price = 0;
                            }
                        }else{
                            $surplus_price = $v['money'];
                        }
                        //结束
                        $update_data = [];
                        $update_data['is_stop'] = 1;
                        $update_data['stop_time'] = time();
                        $update_data['surplus_price'] = $surplus_price;
                        $reslut = db::name('red_envelope')->where('id', $v['id'])->update($update_data);
                        if(!$reslut){
                            Db::rollback();
                            return ['code' => 201, 'msg' => '失败', 'data' => null];
                        }
                        //回退金额
                        if($surplus_price > 0){
                            $resluts = model('admin/User')->change_user_money_by_uid($v['uid'], $surplus_price, 7, "发红包过期回退", $v['uid'], $v['id']);
                            if(!$resluts){
                                Db::rollback();
                                return ['code' => 201, 'msg' => '失败', 'data' => null];
                            }
                        }
                    }
                }
            }
            
            Db::commit();
            return ['code' => 200, 'msg' => '成功', 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            Db::rollback();
            return ['code' => 201, 'msg' => '失败', 'data' => null];
        }
    }
    
}
