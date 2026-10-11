<?php

namespace app\api\model;

use think\Db;
use think\Model;

class UserZone extends Model
{
    //获取社区动态
    public function get_zone_list($uid, $is_follow, $page, $page_limit,$status)
    {
        
        $page = intval($page);
        $page_limit = $page_limit < 10 ? $page_limit : 10;
        $map = [];
        // $map[] = ['a.show_status', '=', 2];
        $map[] = ['a.is_delete','=',1];

        $order_string = 'a.zid desc'; //排序规则
        
        $model = Db::name('UserZone')->alias('a')->join('yy_user b', 'a.uid = b.uid');
        $model = $model->where($map);
        if (!empty($is_follow)) {
            $model = $model->where("a.uid in (select follow_uid from yy_user_follow where uid = :uid)", ['uid' => $uid]);
        }
        $list = $model->field('a.zid,a.uid,a.images,a.content,a.base64_content,a.praise_num,a.read_num,a.share_num,a.comment_num,a.add_time,b.base64_nick_name,b.sex,b.head_pic,a.show_status')->order($order_string)->page($page, $page_limit)->select();
        
        foreach ($list as $k => &$v) {
            
            // if (!empty($v['images'])) {
            //     $images_data = json_decode($v['images'], true);
            //     $images_data = explode(',', $images_data);

            //     $images = [];
            //     foreach ($images_data as $m => $n) {
            //         $images[localpath_to_netpath($n)] = localpath_to_netpath($n);
            //     }
            //     $images = array_values($images);
            //     $v['images'] = implode(',', $images);
            // } else {
            //     $v['images'] = '';
            // }
            if(!empty($v['images'])){
                $images_data = explode(',', $v['images']);
                foreach ($images_data as $m => &$n) {
                    $n = localpath_to_netpath($n);
                }
                $v['images']=$images_data;
            }else{
                $v['images']=[];
            }
            $v['content'] = mb_convert_encoding(base64_decode($v['base64_content']), 'UTF-8', 'UTF-8');
            $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']), 'UTF-8', 'UTF-8');
            $v['head_pic'] = localpath_to_netpath($v['head_pic']);
            $v['is_praise'] = 0; //是否点赞
            $praise = db::name('user_zone_praise')->where(['zid' => $v['zid'], 'uid' => $uid])->find();
            if (!empty($praise)) {
                $v['is_praise'] = 1;
            }
            $v['is_follow'] = 0; //是否关注
            $follow = db::name('user_follow')->where(['follow_uid' => $v['uid'], 'uid' => $uid])->find();
            if (!empty($follow)) {
                $v['is_follow'] = 1;
            }
            $v['add_time'] = date('Y-m-d H:i:s', $v['add_time']);
        }
        if($status == 2){
            $map = [];
            $map[] = ['a.uid','=',$uid];
            // $map[] = ['is_delete','=',1];
            // $map[] = ['show_status','=',2];
            
            $my_zone = db::name('user_zone')->alias('a')->join('yy_user b','a.uid = b.uid')->where($map)->field('a.zid,a.uid,a.images,a.content,a.base64_content,a.praise_num,a.read_num,a.share_num,a.comment_num,a.add_time,b.base64_nick_name,b.sex,b.head_pic')->order($order_string)->page($page, $page_limit)->select();
            
            foreach ($my_zone as $k=>&$v){
                if(!empty($v['images'])){
                    $images_data = explode(',',$v['images']);
                    foreach($images_data as $m => $n){
                        $n = localpath_to_netpath($n);
                    }
                    $v['images'] = $images_data;
                }else{
                    $v['images'] = [];
                }
                $v['content'] = mb_convert_encoding(base64_decode($v['base64_content']),'UTF-8','UTF-8');
                $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']),'UTF-8','UTF-8');
                $v['head_pic'] = localpath_to_netpath($v['head_pic']);
                $v['add_time'] = date('Y-m-d H:i:s',$v['add_time']);
            }
            return ['code' => 200, 'msg' => '获取成功', 'data' => $my_zone];
        }
        
        
        return ['code' => 200, 'msg' => '获取成功', 'data' => $list];
    }
    

    
    //发布动态
    public function publish_zone($uid, $images, $content)
    {

        $images_data = explode(',', $images);
        foreach ($images_data as $k => $v) {
            if(!empty($v)){
                //过滤网络文件地址
                if (!preg_match("/user_upload\/\d{8}\/\S{32}\.(png|jpg|gif|jpeg)/", $v)) {
                    return ['code' => 201, 'msg' => '上传图片参数格式非法', 'data' => null];
                }
            }
        }
        // if(!empty($sound)){
        //     if (!preg_match("/user_upload\/\d{8}\/\S{32}\.(mp3)/", $sound)) {
        //         return ['code' => 201, 'msg' => '上传音频参数格式非法', 'data' => null];
        //     }
        // }
       
        // if(!empty($video)){
        //     if (!preg_match("/user_upload\/\d{8}\/\S{32}\.(mp4)/", $video)) {
        //         return ['code' => 201, 'msg' => '上传视频参数格式非法', 'data' => null];
        //     }
        // }
        
        if (empty($images_data) ) {
            return ['code' => 201, 'msg' => '发布内容不能为空', 'data' => null];
        }

        $data = [];
        $data['uid'] = $uid;
        $data['images'] = $images;
        $data['content'] = $content;
        $data['base64_content'] = base64_encode($content);
        $validate = validate('admin/UserZone');
        $reslut = $validate->scene('apiAdd')->check($data);
        if ($reslut !== true) {
            return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
        }
        $data['add_time'] = time();
        $data['update_time'] = time();
        $reslut = model('user_zone')->save($data);
        if ($reslut) {
            return ['code' => 200, 'msg' => '发布成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '发布失败', 'data' => null];
        }
    }


    public function get_zone_info($uid, $zid)
    {

        $map = [];
        $map[] = ['a.zid', '=', $zid];
        // $map[] = ['a.show_status', '=', 2];
        $map[] = ['a.is_delete', '=', 1];
        $model = Db::name('UserZone')->alias('a')->join('yy_user b', 'a.uid = b.uid');
        $model = $model->where($map);
        $info = $model->field('a.zid,a.uid,a.images,a.content,a.base64_content,a.praise_num,a.read_num,a.share_num,a.comment_num,a.add_time,b.base64_nick_name,b.sex,b.head_pic,a.show_status')->find();

        if (empty($info)) {
            return ['code' => 201, 'msg' => '信息不存在', 'data' => null];
        }
        if (!empty($info['images'])) {
            $images_data = explode(',', $info['images']);
            foreach ($images_data as $m => &$n) {
                $n = localpath_to_netpath($n);
            }
            $info['images'] = $images_data;
        } else {
            $info['images'] = [];
        }
        $info['content'] = mb_convert_encoding(base64_decode($info['base64_content']), 'UTF-8', 'UTF-8');
        
        $info['nick_name'] = mb_convert_encoding(base64_decode($info['base64_nick_name']), 'UTF-8', 'UTF-8');
        $info['head_pic'] = localpath_to_netpath($info['head_pic']);
        $info['is_praise'] = 0; //是否点赞
        $praise = db::name('user_zone_praise')->where(['zid' => $info['zid'], 'uid' => $uid])->find();
        if (!empty($praise)) {
            $info['is_praise'] = 1;
        }
        $info['is_follow'] = 0; //是否关注
        $follow = db::name('user_follow')->where(['follow_uid' => $info['uid'], 'uid' => $uid])->find();
        if (!empty($follow)) {
            $info['is_follow'] = 1;
        }



        return ['code' => 200, 'msg' => '获取成功', 'data' => $info];
    }

    //点赞  取消点赞
    public function praise_zone($uid, $zid)
    {
        $map = [];
        $map[] = ['uid', '=', $uid];
        $map[] = ['zid', '=', $zid];
        $info = db::name('user_zone_praise')->where($map)->find();
        if (!empty($info)) {
            $where = [];
            $where[] = ['zid', '=', $zid];
            db::name('user_zone')->where($where)->setDec('praise_num', 1);
            $del = db::name('user_zone_praise')->where($map)->delete();
            if ($del) {
                return ['code' => 200, 'msg' => '取消点赞成功', 'data' => null];
            }
        }
        Db::startTrans();
        try {
            $data = [];
            $data['uid'] = $uid;
            $data['zid'] = $zid;
            $data['add_time'] = time();
            $data['update_time'] = time();
            $reslut = db::name('user_zone_praise')->insertGetId($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '请重试', 'data' => null];
            }
            //增加点赞数量
            $map = [];
            $map[] = ['zid', '=', $zid];
            $reslut = db::name('user_zone')->where($map)->setInc('praise_num', 1);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '请重试', 'data' => null];
            }
            Db::commit();
            return ['code' => 200, 'msg' => "点赞成功", 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            Db::rollback();
            return ['code' => 201, 'msg' => "请重试", 'data' => null];
        }
    }
    public function comment_zone($uid, $zid, $content)
    {
        $info = db::name('user_zone')->find($zid);
        if (empty($info)) {
            return ['code' => 201, 'msg' => "评论动态不存在", 'data' => null];
        }
        Db::startTrans();
        try {
            $data = [];
            $data['zid'] = $zid;
            $data['uid'] = $uid;
            $data['receive_uid'] = $info['uid'];
            $data['content'] = $content;
            $data['base64_content'] = base64_encode($content);
            $data['is_show'] = 1;
            $data['add_time'] = time();
            $data['update_time'] = time();
            $validate = validate('admin/UserZoneComment');
            $reslut = $validate->scene('apiAdd')->check($data);
            if ($reslut !== true) {
                return ['code' => 201, 'msg' => $validate->getError(), 'data' => null];
            }
            $reslut = db::name('user_zone_comment')->insert($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '请重试', 'data' => null];
            }
            //增加评论数量
            $map = [];
            $map[] = ['zid', '=', $zid];
            $reslut = db::name('user_zone')->where($map)->setInc('comment_num', 1);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '请重试1', 'data' => null];
            }
            //提醒说说动态发布人有人评论
            // $map=[];
            // $map[]=['uid','=',$uid];
            // $user_info=db::name('user')->where($map)->find();
            // $content =  $user_info['nick_name'] . "评论了您的动态";
            // model('api/user_message')->send_message($info['uid'], 4,$info['zid'], "我的动态", $content);
            
            Db::commit();
            return ['code' => 200, 'msg' => "评论成功", 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            Db::rollback();
            // dump($e);
            return ['code' => 201, 'msg' => "请重试2", 'data' => null];
        }
    }


    public function get_comment_list($zid, $page, $page_limit)
    {

        $page = intval($page);
        $page_limit = $page_limit < 10 ? $page_limit : 10;
        $map = [];
        $map[] = ['a.is_show', '=', 1];
        $map[] = ['a.zid', '=', $zid];
        $order_string = 'a.add_time desc'; //排序规则
        $model = Db::name('user_zone_comment')->alias('a')->join('yy_user b', 'a.uid = b.uid');
        $model = $model->where($map);
        $list = $model->field('a.zid,a.content,a.base64_content,a.add_time,a.uid,b.base64_nick_name,b.sex,b.head_pic')->order($order_string)->page($page, $page_limit)->select();
        foreach ($list as $k => &$v) {
            $v['content'] = mb_convert_encoding(base64_decode($v['base64_content']), 'UTF-8', 'UTF-8');
            $v['nick_name'] = mb_convert_encoding(base64_decode($v['base64_nick_name']), 'UTF-8', 'UTF-8');
            $v['head_pic'] = localpath_to_netpath($v['head_pic']);
            unset($v['base64_nick_name']);
        }
        return ['code' => 200, 'msg' => '获取成功', 'data' => $list];
    }
    
    //动态 关注  取消关注 
    public function follow_zone($uid, $fid)
    {
        $map = [];
        $map[] = ['uid', '=', $uid];
        $map[] = ['follow_uid', '=', $fid];
        $info = db::name('user_follow')->where($map)->find();
        if (!empty($info)) {
            $del = db::name('user_follow')->where($map)->delete();
            if ($del) {
                return ['code' => 200, 'msg' => '取消关注成功', 'data' => null];
            }
        }
        Db::startTrans();
        try {
            $data = [];
            $data['uid'] = $uid;
            $data['follow_uid'] = $fid;
            $data['add_time'] = time();
            $data['update_time'] = time();
            $reslut = db::name('user_follow')->insert($data);
            if (!$reslut) {
                Db::rollback();
                return ['code' => 201, 'msg' => '请重试', 'data' => null];
            }

            Db::commit();
            return ['code' => 200, 'msg' => "关注成功", 'data' => null];
        } catch (\Exception $e) {
            // 回滚事务
            Db::rollback();
            return ['code' => 201, 'msg' => "请重试", 'data' => null];
        }
    }
    
    //删除动态
    public function delete_zone($uid, $zid)
    {

        $map = [];
        $map[] = ['uid', '=', $uid];
        $map[] = ['zid', '=', $zid];
        $map[] = ['is_delete', '=', 1];
        $zone_info = db::name('user_zone')->where($map)->find();
        if (empty($zone_info)) {
            return ['code' => 201, 'msg' => '信息不存在', 'data' => null];
        }
        $map = [];
        $map[] = ['zid', '=', $zone_info['zid']];

        $data = [];
        $data['is_delete'] = 2;
        $data['delete_time'] = time();
        $reslut = db::name('user_zone')->where($map)->update($data);
        if ($reslut) {
            return ['code' => 200, 'msg' => '删除成功', 'data' => null];
        } else {
            return ['code' => 201, 'msg' => '删除失败', 'data' => null];
        }
    }
    
}
