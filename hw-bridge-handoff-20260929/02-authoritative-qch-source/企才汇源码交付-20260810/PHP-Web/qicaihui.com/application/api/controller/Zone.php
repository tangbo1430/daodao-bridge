<?php

namespace app\api\controller;

use think\Controller;

class Zone extends Common
{
    //获取社区列表
    public function get_zone_list()
    {
        $is_follow = input('is_follow', 0);
        $page = input('page', 1);
        $page_limit = input('page_limit', 10);
        $status = input('status',1);
        $reslut = model('UserZone')->get_zone_list($this->uid, $is_follow, $page, $page_limit,$status);

        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    
    //获取社区列表
    public function get_zone_info()
    {
        $zid = input('zid', 0);
        $reslut = model('UserZone')->get_zone_info($this->uid, $zid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //发布社区信息
    public function publish_zone()
    {
        $images = input('images', ''); //多图上传json 类型
        $content = input('content', '');
        $reslut = model('UserZone')->publish_zone($this->uid, $images, $content);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    //动态点赞
    public function praise_zone()
    {
        $key_name = "api:usezone:praise_zone:" . $this->uid;
        redis_lock_exit($key_name);
        $zid = input('zid', '');
        $reslut = model('UserZone')->praise_zone($this->uid, $zid);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
    //动态关注
    public function follow_zone()
    {
        $key_name = "api:usezone:follow_zone:" . $this->uid;
        redis_lock_exit($key_name);
        $fid = input('follow_uid', '');
        $reslut = model('UserZone')->follow_zone($this->uid, $fid);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    

    //动态评论
    public function comment_zone()
    {
        $key_name = "api:usezone:comment_zone:" . $this->uid;
        redis_lock_exit($key_name);
        $zid = input('zid', 0);
        $content = input('content', '');
        $reslut = model('UserZone')->comment_zone($this->uid, $zid, $content);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
   
    //获取指定动态评论列表
    public function get_comment_list()
    {
        $key_name = "api:usezone:get_comment_list:" . $this->uid;
        redis_lock_exit($key_name);
        $zid = input('zid', 0);
        $page = input('page', 1);
        $page_limit = input('page_limit', 10);
        $reslut = model('UserZone')->get_comment_list($zid, $page, $page_limit);
        redis_unlock($key_name);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
     //删除社区信息
    public function delete_zone()
    {
        $zid = input('zid', 0);
        $reslut = model('UserZone')->delete_zone($this->uid, $zid);
        return ajaxReturn($reslut['code'], $reslut['msg'], $reslut['data']);
    }
    
   

}