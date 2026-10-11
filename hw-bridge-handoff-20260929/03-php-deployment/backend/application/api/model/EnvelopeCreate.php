<?php

namespace app\api\model;

use think\Db;
use think\Model;

class EnvelopeCreate extends Model
{
    public function generateRedPacketsNew($totalAmount, $count, $minAmount = 0.01, $maxAmount = null)
    {
        // 参数验证
        if ($totalAmount <= 0 || $count <= 0) {
            throw new \InvalidArgumentException('总金额和红包个数必须大于0');
        }
        
        if ($count == 1) {
            return [$this->optimizeAmount($totalAmount)];
        }
        
        // 设置默认最大金额
        if ($maxAmount === null) {
            $maxAmount = min($totalAmount * 0.8, $totalAmount - ($count - 1) * $minAmount);
        }
        
        // 确保最小金额和最大金额合理
        if ($minAmount * $count > $totalAmount) {
            throw new \InvalidArgumentException('最小金额设置过大，无法分配');
        }
        
        if ($maxAmount < $minAmount) {
            throw new \InvalidArgumentException('最大金额不能小于最小金额');
        }
        
        // 使用改进的二倍均值算法
        return $this->generateByDoubleMean($totalAmount, $count, $minAmount, $maxAmount);
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
        $maxRetries = 10; // 最大重试次数
        $retryCount = 0;
        
        while ($retryCount < $maxRetries) {
            $packets = [];
            $remainingAmount = $totalAmount;
            $remainingCount = $count;
            
            // 计算每个红包的最小合理金额，避免出现0.01的情况
            $reasonableMinAmount = max($minAmount, $totalAmount / ($count * 10));
            
            // 计算平均值和40%浮动范围限制
            $averageAmount = $totalAmount / $count;
            $minPacketLimit = $averageAmount * 0.6; // 最小红包金额为平均值的60%
            $maxPacketLimit = min($maxAmount, $averageAmount * 1.4); // 最大红包金额为平均值的140%
            
            $success = true; // 标记是否成功生成
            
            for ($i = 0; $i < $count; $i++) {
                if ($remainingCount == 1) {
                    // 最后一个红包，分配剩余金额，确保精确匹配
                    $amount = $remainingAmount;
                } else {
                    // 计算当前红包的最大可能金额，考虑40%浮动限制
                    $currentMaxAmount = min($maxPacketLimit, $remainingAmount - ($remainingCount - 1) * $minPacketLimit);
                    
                    // 使用二倍均值算法，但限制最大值
                    $average = $remainingAmount / $remainingCount;
                    $maxPossible = min($currentMaxAmount, $average * 1.1); // 限制在1.1倍平均值
                    
                    // 确保最小金额合理，考虑40%浮动限制
                    $currentMinAmount = max($minPacketLimit, $reasonableMinAmount, $minAmount);
                    
                    // 生成随机金额
                    $amount = $this->generateRandomAmount($currentMinAmount, $maxPossible);
                }
                
                // 优化金额，确保两位小数且最后一位不为0
                $amount = $this->optimizeAmount($amount);
                
                // 严格限制每个红包都在40%浮动范围内
                if ($amount < $minPacketLimit) {
                    $amount = $minPacketLimit;
                }
                if ($amount > $maxPacketLimit) {
                    $amount = $maxPacketLimit;
                }
                
                $packets[] = $amount;
                $remainingAmount -= $amount;
                $remainingCount--;
            }
            
            // 检查所有红包是否都在40%浮动范围内
            foreach ($packets as $a) {
                if ($a < $minPacketLimit || $a > $maxPacketLimit) {
                    $success = false;
                    break;
                }
            }
            
            if (!$success) {
                $retryCount++;
                continue; // 重新尝试
            }
            
            // 强制确保总金额精确匹配（最高优先级）
            $actualTotal = array_sum($packets);
            $diff = $totalAmount - $actualTotal;
            if (abs($diff) > 0.001) {
                // 调整最后一个红包，确保总金额精确匹配，且在40%浮动范围内
                $lastIndex = count($packets) - 1;
                $newAmount = $packets[$lastIndex] + $diff;
                if ($newAmount >= $minPacketLimit && $newAmount <= $maxPacketLimit) {
                    $packets[$lastIndex] = $this->optimizeAmount($newAmount);
                } else {
                    // 如果调整后金额超出范围，重新尝试
                    $success = false;
                    $retryCount++;
                    continue;
                }
            }
            
            // 最终强制验证：确保总金额完全相等
            $finalTotal = array_sum($packets);
            if (abs($finalTotal - $totalAmount) > 0.001) {
                // 如果仍然不匹配，强制调整最后一个红包，但在40%浮动范围内
                $finalDiff = $totalAmount - $finalTotal;
                $newAmount = $packets[count($packets) - 1] + $finalDiff;
                if ($newAmount >= $minPacketLimit && $newAmount <= $maxPacketLimit) {
                    $packets[count($packets) - 1] = $this->optimizeAmount($newAmount);
                } else {
                    // 如果调整后超出范围，重新尝试
                    $success = false;
                    $retryCount++;
                    continue;
                }
            }
            
            // 最终强制验证：确保总金额完全相等（最高优先级）
            $finalTotal = array_sum($packets);
            if (abs($finalTotal - $totalAmount) > 0.001) {
                // 如果仍然不匹配，强制调整最后一个红包，但在40%浮动范围内
                $finalDiff = $totalAmount - $finalTotal;
                $newAmount = $packets[count($packets) - 1] + $finalDiff;
                if ($newAmount >= $minPacketLimit && $newAmount <= $maxPacketLimit) {
                    // 强制保留两位小数
                    $packets[count($packets) - 1] = round($newAmount, 2);
                } else {
                    // 如果调整后超出范围，重新尝试
                    $success = false;
                    $retryCount++;
                    continue;
                }
            }
            
            // 将最大金额的红包移到第一个位置
            $maxIndex = array_keys($packets, max($packets))[0];
            if ($maxIndex > 0) {
                $maxPacketAmount = $packets[$maxIndex];
                array_splice($packets, $maxIndex, 1);
                array_unshift($packets, $maxPacketAmount);
            }
            
            // 最终验证：确保总金额完全相等
            $finalTotal = array_sum($packets);
            if (abs($finalTotal - $totalAmount) > 0.001) {
                // 如果仍然不匹配，重新尝试
                $success = false;
                $retryCount++;
                continue;
            }
            
            // 最终检查：确保所有红包都在40%浮动范围内
            foreach ($packets as $packet) {
                if ($packet < $minPacketLimit || $packet > $maxPacketLimit) {
                    // 如果有红包超出范围，重新尝试
                    $success = false;
                    $retryCount++;
                    continue 2; // 跳出两层循环
                }
            }
            
            // 如果所有检查都通过，返回结果
            if ($success) {
                // 最终强制确保总金额精确匹配
                $packets = $this->ensureExactTotalMatch($packets, $totalAmount, $maxPacketLimit);
                return $packets;
            }
        }
        
        // 如果重试次数用完，使用备用方案
        return $this->generateSimilarRedPackets($totalAmount, $count, $minAmount);
    }
    
    /**
     * 生成指定范围内的随机金额
     * 确保保留两位小数且最后一位不能是0
     * 
     * @param float $min 最小金额
     * @param float $max 最大金额
     * @return float
     */
    private function generateRandomAmount($min, $max)
    {
        // 确保最小金额不小于0.01，并且尽量不小于0.1
        $min = max(0.1, $min);
        
        // 如果最大值太小，调整最小值
        if ($max < 0.1) {
            $min = max(0.01, $min);
        }
        
        // 生成随机金额，确保最后一位不是0
        do {
            $random = mt_rand() / mt_getrandmax();
            $amount = $min + $random * ($max - $min);
            
            // 保留两位小数
            $amount = round($amount, 2);
            
            // 确保在范围内且不小于0.01
            $amount = max(0.01, min($max, $amount));
            
            // 检查最后一位是否为0，如果是则重新生成
            $lastDigit = intval(($amount * 100) % 10);
        } while ($lastDigit == 0);
        
        return $amount;
    }
    
    /**
     * 优化金额，确保保留两位小数且最后一位不能是0
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
        
        // 保留两位小数
        $amount = round($amount, 2);
        
        // 检查最后一位是否为0，如果是则调整
        $lastDigit = intval(($amount * 100) % 10);
        if ($lastDigit == 0) {
            // 随机增加0.01到0.09之间的值
            $adjustment = mt_rand(1, 9) / 100;
            $amount += $adjustment;
            $amount = round($amount, 2);
        }
        
        return max(0.01, $amount);
    }
    
    /**
     * 重新分配红包金额，确保每个红包都不小于最小金额，精确匹配优先
     * 
     * @param array $packets 当前红包数组
     * @param float $totalAmount 总金额
     * @return array 重新分配后的红包数组
     */
    private function redistributeAmounts($packets, $totalAmount)
    {
        $count = count($packets);
        $minAmount = 0.01;
        
        // 计算合理的最小金额，避免出现0.01的情况
        $reasonableMinAmount = max($minAmount, $totalAmount / ($count * 10));
        
        // 确保每个红包至少有合理的最小金额
        $reservedAmount = $count * $reasonableMinAmount;
        $remainingAmount = $totalAmount - $reservedAmount;
        
        if ($remainingAmount < 0) {
            // 如果总金额不够分配最小金额，使用原始最小金额
            $reservedAmount = $count * $minAmount;
            $remainingAmount = $totalAmount - $reservedAmount;
            $reasonableMinAmount = $minAmount;
        }
        
        // 重新分配剩余金额
        $newPackets = [];
        for ($i = 0; $i < $count; $i++) {
            if ($i == $count - 1) {
                // 最后一个红包，分配所有剩余金额，确保精确匹配
                $amount = $reasonableMinAmount + $remainingAmount;
            } else {
                // 随机分配剩余金额的一部分
                $maxPossible = $remainingAmount / ($count - $i);
                $randomAmount = $this->generateRandomAmount(0.01, $maxPossible);
                $amount = $reasonableMinAmount + $randomAmount;
                $remainingAmount -= $randomAmount;
            }
            
            $newPackets[] = $this->optimizeAmount($amount);
        }
        
        // 确保总金额精确匹配
        $actualTotal = array_sum($newPackets);
        $diff = $totalAmount - $actualTotal;
        if (abs($diff) > 0.001) {
            // 调整最后一个红包，确保精确匹配
            $newPackets[$count - 1] += $diff;
            $newPackets[$count - 1] = $this->optimizeAmount($newPackets[$count - 1]);
        }
        
        // 最终强制验证：确保总金额完全相等
        $finalTotal = array_sum($newPackets);
        if (abs($finalTotal - $totalAmount) > 0.001) {
            // 如果仍然不匹配，强制调整最后一个红包，不考虑优化
            $finalDiff = $totalAmount - $finalTotal;
            $newPackets[$count - 1] += $finalDiff;
            $newPackets[$count - 1] = round($newPackets[$count - 1], 2);
        }
        
        // 最终验证：确保总金额完全相等
        $finalTotal = array_sum($newPackets);
        if (abs($finalTotal - $totalAmount) > 0.001) {
            // 如果仍然不匹配，抛出异常
            throw new \Exception("重新分配后红包金额总和 {$finalTotal} 与总金额 {$totalAmount} 不匹配，差值: " . ($totalAmount - $finalTotal));
        }
        
        return $newPackets;
    }

    /**
     * 生成金额相近的红包，精确匹配优先
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
        
        // 计算合理的最小金额，避免出现0.01的情况
        $reasonableMinAmount = max($minAmount, $totalAmount / ($count * 10));
        
        $base = floor(($totalAmount / $count) * 100) / 100; // 生成两位小数
        $packets = array_fill(0, $count, $base);
        $remaining = round($totalAmount - $base * $count, 2);
        
        // 在每个红包基础上做微小随机调整，幅度不超过±10%
        for ($i = 0; $i < $count - 1; $i++) {
            $maxDelta = min($base * 0.1, $packets[$i] - $reasonableMinAmount); // 最大可调整幅度
            $delta = 0;
            if ($maxDelta > 0) {
                $delta = round((mt_rand(-100, 100) / 100) * $maxDelta, 2); // 调整为两位小数
            }
            $packets[$i] += $delta;
            $packets[$i] = $this->optimizeAmount(max($reasonableMinAmount, $packets[$i])); // 使用优化方法
            $remaining -= $delta;
        }
        
        // 最后一个红包补齐差额，确保总金额精确匹配
        $packets[$count - 1] += round($remaining, 2);
        $packets[$count - 1] = $this->optimizeAmount(max($reasonableMinAmount, $packets[$count - 1]));
        
        // 强制确保总金额精确匹配（最高优先级）
        $actualTotal = array_sum($packets);
        $diff = $totalAmount - $actualTotal;
        if (abs($diff) > 0.001) {
            $packets[$count - 1] += $diff; // 直接调整，不进行优化
            $packets[$count - 1] = $this->optimizeAmount($packets[$count - 1]);
        }
        
        // 最终强制验证：确保总金额完全相等
        $finalTotal = array_sum($packets);
        if (abs($finalTotal - $totalAmount) > 0.001) {
            // 如果仍然不匹配，强制调整最后一个红包，不考虑优化
            $finalDiff = $totalAmount - $finalTotal;
            $packets[$count - 1] += $finalDiff;
            $packets[$count - 1] = $this->optimizeAmount($packets[$count - 1]);
        }
        
        // 最终强制验证：确保总金额完全相等（最高优先级）
        $finalTotal = array_sum($packets);
        if (abs($finalTotal - $totalAmount) > 0.001) {
            // 如果仍然不匹配，强制调整最后一个红包，不考虑任何限制
            $finalDiff = $totalAmount - $finalTotal;
            $packets[$count - 1] += $finalDiff;
            // 强制保留两位小数
            $packets[$count - 1] = round($packets[$count - 1], 2);
        }
        
        // 将最大金额的红包移到第一个位置
        $maxIndex = array_keys($packets, max($packets))[0];
        if ($maxIndex > 0) {
            $maxAmount = $packets[$maxIndex];
            array_splice($packets, $maxIndex, 1);
            array_unshift($packets, $maxAmount);
        }
        
        // 最终验证：确保总金额完全相等
        $finalTotal = array_sum($packets);
        if (abs($finalTotal - $totalAmount) > 0.001) {
            // 如果仍然不匹配，抛出异常
            throw new \Exception("相似红包金额总和 {$finalTotal} 与总金额 {$totalAmount} 不匹配，差值: " . ($totalAmount - $finalTotal));
        }
        
        return $packets;
    }

    /**
     * 测试红包生成算法
     * @param float $totalAmount 总金额
     * @param int $count 红包个数
     * @return array 测试结果
     */
    public function testRedPacketGeneration($totalAmount = 100, $count = 10)
    {
        $result = [
            'total_amount' => $totalAmount,
            'count' => $count,
            'packets' => [],
            'validation' => []
        ];
        
        try {
            // 生成红包
            $packets = $this->generateRedPacketsNew($totalAmount, $count);
            $result['packets'] = $packets;
            
            // 使用专门的验证方法
            $totalValidation = $this->validateTotalAmount($packets, $totalAmount);
            $result['validation']['total_validation'] = $totalValidation;
            
            // 验证结果
            $actualTotal = array_sum($packets);
            $result['validation']['total_match'] = $totalValidation['is_acceptable_match'];
            $result['validation']['actual_total'] = $actualTotal;
            $result['validation']['difference'] = $totalAmount - $actualTotal;
            
            // 严格验证总金额是否相等
            if (!$totalValidation['is_acceptable_match']) {
                $result['validation']['error'] = "红包金额总和 {$actualTotal} 与总金额 {$totalAmount} 不匹配！差值: " . ($totalAmount - $actualTotal);
                $result['validation']['success'] = false;
                return $result;
            }
            
            // 额外验证：确保总金额完全相等（容差0.0001）
            if (abs($actualTotal - $totalAmount) > 0.0001) {
                $result['validation']['warning'] = "红包金额总和与总金额存在微小差异，差值: " . ($totalAmount - $actualTotal);
            }
            
            // 最终验证：确保总金额绝对精确
            $result['validation']['exact_match'] = abs($actualTotal - $totalAmount) == 0;
            $result['validation']['precision'] = number_format($actualTotal, 2) == number_format($totalAmount, 2);
            
            // 验证最大红包金额限制
            $maxPacket = max($packets);
            $averageAmount = $totalAmount / $count;
            $maxLimit = min($totalAmount * 0.08, $averageAmount * 1.2);
            $maxPacketValid = $maxPacket <= $maxLimit;
            $result['validation']['max_packet_valid'] = $maxPacketValid;
            $result['validation']['max_packet'] = $maxPacket;
            $result['validation']['max_limit'] = $maxLimit;
            
            // 验证40%浮动范围
            $minPacket = min($packets);
            $minLimit = $averageAmount * 0.6;
            $maxLimit40Percent = $averageAmount * 1.4;
            $rangeValid = $minPacket >= $minLimit && $maxPacket <= $maxLimit40Percent;
            $result['validation']['range_valid'] = $rangeValid;
            $result['validation']['min_packet'] = $minPacket;
            $result['validation']['min_limit'] = $minLimit;
            $result['validation']['max_limit_40_percent'] = $maxLimit40Percent;
            $result['validation']['average_amount'] = $averageAmount;
            $result['validation']['range_percentage'] = round((($maxPacket - $minPacket) / $averageAmount) * 100, 2) . '%';
            
            // 验证每个红包的格式
            $validFormat = true;
            $firstIsMax = true;
            $firstAmount = $packets[0];
            $minAmountCount = 0; // 统计0.01的数量
            
            foreach ($packets as $index => $amount) {
                // 检查是否保留两位小数
                $decimalPlaces = strlen(substr(strrchr($amount, "."), 1));
                if ($decimalPlaces != 2) {
                    $validFormat = false;
                    $result['validation']['format_error'] = "红包 {$index} 不是两位小数: {$amount}";
                    break;
                }
                
                // 检查最后一位是否为0
                $lastDigit = intval(($amount * 100) % 10);
                if ($lastDigit == 0) {
                    $validFormat = false;
                    $result['validation']['zero_error'] = "红包 {$index} 最后一位是0: {$amount}";
                    break;
                }
                
                // 检查第一个是否最大
                if ($amount > $firstAmount) {
                    $firstIsMax = false;
                }
                
                // 统计0.01的数量
                if ($amount == 0.01) {
                    $minAmountCount++;
                }
            }
            
            $result['validation']['valid_format'] = $validFormat;
            $result['validation']['first_is_max'] = $firstIsMax;
            $result['validation']['min_amount_count'] = $minAmountCount;
            $result['validation']['min_amount_percentage'] = round(($minAmountCount / $count) * 100, 2) . '%';
            $result['validation']['success'] = $validFormat && $maxPacketValid && $rangeValid;
            
        } catch (\Exception $e) {
            $result['validation']['error'] = $e->getMessage();
            $result['validation']['success'] = false;
        }
        
        return $result;
    }
    
    /**
     * 确保总金额精确匹配
     * @param array $packets 红包数组
     * @param float $totalAmount 总金额
     * @param float $maxPacketLimit 最大红包金额限制
     * @return array 确保总金额精确匹配后的红包数组
     */
    private function ensureExactTotalMatch($packets, $totalAmount, $maxPacketLimit)
    {
        $count = count($packets);
        $averageAmount = $totalAmount / $count;
        $minPacketLimit = $averageAmount * 0.6; // 最小红包金额为平均值的60%
        $maxPacketLimit = min($maxPacketLimit, $averageAmount * 1.4); // 最大红包金额为平均值的140%
        
        $actualTotal = array_sum($packets);
        $diff = $totalAmount - $actualTotal;
        
        if (abs($diff) > 0.001) {
            // 调整最后一个红包，确保总金额精确匹配，且在40%浮动范围内
            $lastIndex = $count - 1;
            $newAmount = $packets[$lastIndex] + $diff;
            if ($newAmount >= $minPacketLimit && $newAmount <= $maxPacketLimit) {
                $packets[$lastIndex] = $this->optimizeAmount($newAmount);
            } else {
                // 如果调整后金额超出范围，重新尝试
                return $this->generateSimilarRedPackets($totalAmount, $count, 0.01);
            }
        }
        
        // 最终强制验证：确保总金额完全相等
        $finalTotal = array_sum($packets);
        if (abs($finalTotal - $totalAmount) > 0.001) {
            // 如果仍然不匹配，强制调整最后一个红包，但在40%浮动范围内
            $finalDiff = $totalAmount - $finalTotal;
            $newAmount = $packets[$count - 1] + $finalDiff;
            if ($newAmount >= $minPacketLimit && $newAmount <= $maxPacketLimit) {
                $packets[$count - 1] = $this->optimizeAmount($newAmount);
            } else {
                // 如果调整后超出范围，重新尝试
                return $this->generateSimilarRedPackets($totalAmount, $count, 0.01);
            }
        }
        
        // 最终强制验证：确保总金额完全相等（最高优先级）
        $finalTotal = array_sum($packets);
        if (abs($finalTotal - $totalAmount) > 0.001) {
            // 如果仍然不匹配，强制调整最后一个红包，但在40%浮动范围内
            $finalDiff = $totalAmount - $finalTotal;
            $newAmount = $packets[$count - 1] + $finalDiff;
            if ($newAmount >= $minPacketLimit && $newAmount <= $maxPacketLimit) {
                // 强制保留两位小数
                $packets[$count - 1] = round($newAmount, 2);
            } else {
                // 如果调整后超出范围，重新尝试
                return $this->generateSimilarRedPackets($totalAmount, $count, 0.01);
            }
        }
        
        // 将最大金额的红包移到第一个位置
        $maxIndex = array_keys($packets, max($packets))[0];
        if ($maxIndex > 0) {
            $maxAmount = $packets[$maxIndex];
            array_splice($packets, $maxIndex, 1);
            array_unshift($packets, $maxAmount);
        }
        
        // 最终验证：确保总金额完全相等
        $finalTotal = array_sum($packets);
        if (abs($finalTotal - $totalAmount) > 0.001) {
            // 如果仍然不匹配，重新尝试
            return $this->generateSimilarRedPackets($totalAmount, $count, 0.01);
        }
        
        // 最终检查：确保所有红包都在40%浮动范围内
        foreach ($packets as $packet) {
            if ($packet < $minPacketLimit || $packet > $maxPacketLimit) {
                // 如果有红包超出范围，重新尝试
                return $this->generateSimilarRedPackets($totalAmount, $count, 0.01);
            }
        }
        
        return $packets;
    }

    /**
     * 验证红包总金额精确匹配
     * @param array $packets 红包数组
     * @param float $totalAmount 总金额
     * @return array 验证结果
     */
    public function validateTotalAmount($packets, $totalAmount)
    {
        $actualTotal = array_sum($packets);
        $diff = $totalAmount - $actualTotal;
        
        return [
            'total_amount' => $totalAmount,
            'actual_total' => $actualTotal,
            'difference' => $diff,
            'absolute_difference' => abs($diff),
            'is_exact_match' => abs($diff) == 0,
            'is_precise_match' => abs($diff) < 0.0001,
            'is_acceptable_match' => abs($diff) < 0.001,
            'formatted_match' => number_format($actualTotal, 2) == number_format($totalAmount, 2),
            'validation_passed' => abs($diff) < 0.001,
            'packets_count' => count($packets),
            'packets_summary' => [
                'min' => min($packets),
                'max' => max($packets),
                'average' => $actualTotal / count($packets)
            ]
        ];
    }
}
