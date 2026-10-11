SET NAMES utf8mb4;

INSERT INTO `yy_config`
  (`type`,`key_title`,`key_name`,`key_value`,`key_desc`,`sort`,`update_time`,`is_delete`)
SELECT 1,'sms_every_day_send_limit','用户每天限制发送短信条数','500','用户每天限制发送短信条数',0,UNIX_TIMESTAMP(),2
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_config` WHERE `key_title`='sms_every_day_send_limit'
);

INSERT INTO `yy_config`
  (`type`,`key_title`,`key_name`,`key_value`,`key_desc`,`sort`,`update_time`,`is_delete`)
SELECT 1,'sms_send_model','1正式2模拟','1','短信是否真实发送1正式2模拟',0,UNIX_TIMESTAMP(),2
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_config` WHERE `key_title`='sms_send_model'
);

UPDATE `yy_config`
SET `key_value`='1', `update_time`=UNIX_TIMESTAMP()
WHERE `key_title`='sms_send_model';

INSERT INTO `yy_config`
  (`type`,`key_title`,`key_name`,`key_value`,`key_desc`,`sort`,`update_time`,`is_delete`)
SELECT 1,'smsbao_account','短信宝短信账号','','短信宝短信账号',0,UNIX_TIMESTAMP(),2
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_config` WHERE `key_title`='smsbao_account'
);

INSERT INTO `yy_config`
  (`type`,`key_title`,`key_name`,`key_value`,`key_desc`,`sort`,`update_time`,`is_delete`)
SELECT 1,'smsbao_password','短信宝短信秘钥','','短信宝短信秘钥',0,UNIX_TIMESTAMP(),2
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_config` WHERE `key_title`='smsbao_password'
);
