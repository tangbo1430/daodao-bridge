SET NAMES utf8mb4;

INSERT INTO `yy_config`
  (`type`,`key_title`,`key_name`,`key_value`,`key_desc`,`sort`,`update_time`,`is_delete`)
SELECT defaults.`type`,defaults.`key_title`,defaults.`key_name`,defaults.`key_value`,
       defaults.`key_desc`,defaults.`sort`,UNIX_TIMESTAMP(),defaults.`is_delete`
FROM (
  SELECT 1 `type`,'account_reg_type' `key_title`,'账号注册方式' `key_name`,'2' `key_value`,'1手机号2账号' `key_desc`,0 `sort`,2 `is_delete`
  UNION ALL SELECT 1,'customer_link','客服地址','','客服地址',0,2
  UNION ALL SELECT 1,'customer_open_status','客服开关','2','1开启2关闭',0,2
  UNION ALL SELECT 1,'customer_uid','客服UID','0','新环境默认无客服用户',0,2
  UNION ALL SELECT 1,'file_upload_type','文件存储类型','3','1本地，2七牛，3腾讯COS；环境变量优先',90,1
  UNION ALL SELECT 1,'filter_uid','过滤用户ID','','逗号分隔的保留UID',0,2
  UNION ALL SELECT 1,'group_video_open_status','群视频开关','2','1开启2关闭',0,2
  UNION ALL SELECT 1,'group_voice_open_status','群语音开关','2','1开启2关闭',0,2
  UNION ALL SELECT 1,'huaxin_account','华信短信账号','','未启用',0,2
  UNION ALL SELECT 1,'huaxin_password','华信短信密钥','','未启用',0,2
  UNION ALL SELECT 1,'invite_down_url','邀请下载地址','https://im.awhaha.com','邀请下载地址',60,1
  UNION ALL SELECT 1,'is_maintenance','系统维护','1','1正常2维护',100,1
  UNION ALL SELECT 1,'loacl_resource_url','本地资源地址','https://admin.awhaha.com','本地上传资源地址',80,1
  UNION ALL SELECT 1,'min_withdraw_amount','最低提现金额','30','最低提现金额',0,2
  UNION ALL SELECT 1,'qiniu_access_key','七牛公钥','','外部密钥不得写入部署包',0,2
  UNION ALL SELECT 1,'qiniu_bucket_name','七牛空间名称','','未启用',0,2
  UNION ALL SELECT 1,'qiniu_cdn_url','七牛加速域名','','未启用',0,2
  UNION ALL SELECT 1,'qiniu_secret_key','七牛私钥','','外部密钥不得写入部署包',0,2
  UNION ALL SELECT 1,'show_reg','显示注册','1','1显示2隐藏',0,2
  UNION ALL SELECT 1,'tencent_bucket_name','腾讯COS存储桶','','优先读取COS_BUCKET环境变量',0,2
  UNION ALL SELECT 1,'tencent_cdn_url','腾讯COS地址','','优先读取COS_PUBLIC_URL环境变量',0,2
  UNION ALL SELECT 1,'tencent_secret_id','腾讯COS SecretId','','外部密钥不得写入部署包',0,2
  UNION ALL SELECT 1,'tencent_secret_key','腾讯COS SecretKey','','外部密钥不得写入部署包',0,2
  UNION ALL SELECT 1,'tencent_territory','腾讯COS区域','ap-guangzhou','优先读取COS_REGION环境变量',0,2
  UNION ALL SELECT 1,'tencentyun_im_appid','腾讯IM AppID','20044723','实际值优先读取环境变量',50,1
  UNION ALL SELECT 1,'tencentyun_im_key','腾讯IM Key','','密钥只允许通过环境变量配置',40,1
  UNION ALL SELECT 1,'tencentyun_im_private_key','腾讯IM私钥','','未启用',0,2
  UNION ALL SELECT 1,'tencentyun_im_public_key','腾讯IM公钥','','未启用',0,2
  UNION ALL SELECT 1,'user_withdraw_rate','提现手续费','0','0到1之间的小数',0,2
  UNION ALL SELECT 1,'video_open_status','视频开关','2','1开启2关闭',0,2
  UNION ALL SELECT 1,'voice_open_status','语音开关','1','1开启2关闭',0,2
  UNION ALL SELECT 1,'web_site','Web地址','https://im.awhaha.com','Web地址',70,1
) defaults
LEFT JOIN `yy_config` current_config ON current_config.`key_title`=defaults.`key_title`
WHERE current_config.`cid` IS NULL;

UPDATE `yy_config`
SET `key_value`='https://admin.awhaha.com', `update_time`=UNIX_TIMESTAMP()
WHERE `key_title`='loacl_resource_url' AND `key_value`='https://admin.awhaha.com/uploads/';
