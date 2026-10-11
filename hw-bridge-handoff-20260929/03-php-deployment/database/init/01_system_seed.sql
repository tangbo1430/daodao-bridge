DELETE FROM `yy_config` WHERE `key_title` IN (
  'is_maintenance', 'file_upload_type', 'loacl_resource_url',
  'web_site', 'invite_down_url', 'tencentyun_im_appid', 'tencentyun_im_key'
);

INSERT INTO `yy_config`
  (`type`, `key_title`, `key_name`, `key_value`, `key_desc`, `sort`, `update_time`, `is_delete`)
VALUES
  (1, 'is_maintenance', '系统维护', '1', '1正常，2维护', 100, UNIX_TIMESTAMP(), 1),
  (1, 'file_upload_type', '文件存储类型', '3', '1本地，2七牛，3腾讯COS；环境变量优先', 90, UNIX_TIMESTAMP(), 1),
  (1, 'loacl_resource_url', '本地资源地址', 'https://admin.awhaha.com', '', 80, UNIX_TIMESTAMP(), 1),
  (1, 'web_site', 'Web地址', 'https://im.awhaha.com', '', 70, UNIX_TIMESTAMP(), 1),
  (1, 'invite_down_url', '邀请下载地址', 'https://im.awhaha.com', '', 60, UNIX_TIMESTAMP(), 1),
  (1, 'tencentyun_im_appid', '腾讯IM AppID', '20044723', '实际值优先读取环境变量', 50, UNIX_TIMESTAMP(), 1),
  (1, 'tencentyun_im_key', '腾讯IM Key', '', '密钥只允许通过环境变量配置', 40, UNIX_TIMESTAMP(), 1);

INSERT INTO `yy_admin`
  (`aid`, `user_name`, `phone`, `password`, `login_token`, `system_menu_id_list`,
   `is_delete`, `delete_time`, `add_time`, `update_time`, `token_validity_time`,
   `error_num`, `status`, `system_type`)
VALUES
  (32, 'admin', NULL, 'bedc0577cd7201d760c51577cabf66b0', '', '0',
   1, 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP(), 0, 0, 1, 3)
ON DUPLICATE KEY UPDATE
  `user_name` = VALUES(`user_name`), `password` = VALUES(`password`),
  `is_delete` = 1, `status` = 1, `system_type` = 3, `error_num` = 0;

INSERT INTO `yy_invitecode`
  (`name`, `status`, `createtime`, `user_id`, `system_type`, `invite_num`)
SELECT 'PROD2026', 0, UNIX_TIMESTAMP(), '', 3, 0
WHERE NOT EXISTS (
  SELECT 1 FROM `yy_invitecode` WHERE `name` = 'PROD2026' AND `system_type` = 3
);
