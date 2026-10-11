<?php
namespace shengwang;
include("src/RtcTokenBuilder.php");
include("src/RtmTokenBuilder.php");
class Shengwang{
    public function RtcTokenBuilder($uid, $channelName, $role_type=0, $appID, $appCertificate){
       
        $uid = (string)$uid;
        $channelName = (string)$channelName;
        // $appID = "3653ea090a98432a8f8c0beb0f92c963";
        // $appCertificate = "ec2bf365f7014683a32b3dc41141e945";
        $appID = $appID;
        $appCertificate = $appCertificate;
        // $uid = 0;
        // // $uidStr = "2882341273";
        $role = \RtcTokenBuilder::RoleAttendee;
        if($role_type==0){
            $role = \RtcTokenBuilder::RoleAttendee;
        }elseif($role_type==1){
            $role = \RtcTokenBuilder::RolePublisher;
        }elseif($role_type==2){
            $role = \RtcTokenBuilder::RoleSubscriber;
        }else{
             $role = \RtcTokenBuilder::RoleRtmUser;
        }
       
        $expireTimeInSeconds = 86400;
        $currentTimestamp = (new \DateTime("now", new \DateTimeZone('UTC')))->getTimestamp();
        $privilegeExpiredTs = $currentTimestamp + $expireTimeInSeconds;
        
        $token = \RtcTokenBuilder::buildTokenWithUid($appID, $appCertificate, $channelName, $uid, $role, $privilegeExpiredTs);
        // dump($token);
        // $token = substr($token,0,50).chr(rand(97,122)).chr(rand(65,90)).chr(rand(97,122)).chr(rand(65,90)).substr($token,50,999);
        // $token=substr($token,0,50)."####".substr($token,50,999);
        return $token;
    }
    
    public function RtmTokenBuilder($user_id, $appID, $appCertificate){
        /*
        $appID = "3653ea090a98432a8f8c0beb0f92c963";
        $appCertificate = "ec2bf365f7014683a32b3dc41141e945";
        // $uid = 0;
        // // $uidStr = "2882341273";
        $role = \RtmTokenBuilder::RoleRtmUser;
        $expireTimeInSeconds = 3600;
        $currentTimestamp = (new \DateTime("now", new \DateTimeZone('UTC')))->getTimestamp();
        $privilegeExpiredTs = $currentTimestamp + $expireTimeInSeconds;
        //dump($appID);dump($appCertificate);dump($user_id);dump($role);dump($privilegeExpiredTs);
        $token = \RtmTokenBuilder::buildToken($appID, $appCertificate, $user_id, $role, $privilegeExpiredTs);
        // echo $token;exit;
        return $token;
        */
        
        // $appID = "3653ea090a98432a8f8c0beb0f92c963";
        // $appCertificate = "ec2bf365f7014683a32b3dc41141e945";
        $appID = $appID;
        $appCertificate = $appCertificate;
        $user = (string)$user_id;

        $role = \RtcTokenBuilder::RoleAttendee;

        $expireTimeInSeconds = 86400;
        $currentTimestamp = (new \DateTime("now", new \DateTimeZone('UTC')))->getTimestamp();
        $privilegeExpiredTs = $currentTimestamp + $expireTimeInSeconds;
        
        $token = \RtmTokenBuilder::buildToken($appID, $appCertificate, $user, $role, $privilegeExpiredTs);
        
        // $token = substr($token,0,50).chr(rand(97,122)).chr(rand(65,90)).chr(rand(97,122)).chr(rand(65,90)).substr($token,50,999);
         //$token=substr($token,0,50)."####".substr($token,50,999);
        return $token;
    }
}