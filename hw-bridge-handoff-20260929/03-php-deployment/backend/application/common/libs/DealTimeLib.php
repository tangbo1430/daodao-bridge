<?php


namespace app\common\libs;


class DealTimeLib
{
    static $weekDayNum = 7;
    /**
     * 本周开始结束时间 1 表示星期一 7 表示周日
     * @param $time
     * @param bool $isNow
     * @return false|array
     */
    static function getWeek($time, $isNow = true)
    {
        if ($time == 0) {
            return false;
        }
        $num = date('N', $time);
        $startNum = $num - 1;
        $startTimeFormat = date('Y-m-d', strtotime("-$startNum day", $time));
        $startTime = strtotime($startTimeFormat);
        $endNum = self::$weekDayNum - $startNum;
        $endTimeFormat = date('Y-m-d', strtotime("+$endNum day", $time));
        $endTime = strtotime($endTimeFormat);
        if ($isNow) $endTime = $_SERVER['REQUEST_TIME'];
        return ['start_time' => $startTime, 'end_time' => $endTime];
    }

    static function getMonth($time, $isNow = true)
    {
        if ($time == 0) {
            return false;
        }
        $timeFormat = date('Y-m-01', $time);
        return ['start_time' => strtotime($timeFormat), 'end_time' => $time];
    }

    static function getDay($time) {
        $start_time = strtotime(date('Y-m-d', $time));
        return ['start_time' => $start_time, 'end_time' => $time];
    }
    public function get_follow_tab_type()
    {
        $tab_type = input('tab_type', 1);
        $time_type = input('time_type', 'day');
    }
}