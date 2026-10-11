SET @system_root = (
  SELECT `id` FROM `yy_system_menu`
  WHERE `pid`=0 AND `type`=1 AND `title`=CONVERT(0xE7B3BBE7BB9FE7AEA1E79086 USING utf8mb4)
  ORDER BY `id` ASC LIMIT 1
);

INSERT INTO `yy_system_menu` (`pid`,`type`,`limit_sign`,`title`,`icon`,`href`,`target`,`status`,`sort`)
SELECT @system_root,1,'im_operations_group',CONVERT(0x494DE8BF90E7BBB4 USING utf8mb4),'layui-icon layui-icon-console','','_self',1,85
WHERE @system_root IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM `yy_system_menu` WHERE `limit_sign`='im_operations_group');

SET @im_operations_group = (
  SELECT `id` FROM `yy_system_menu` WHERE `limit_sign`='im_operations_group' LIMIT 1
);

UPDATE `yy_system_menu`
SET `pid`=@im_operations_group
WHERE `limit_sign` IN ('im_callback_overview','im_callback_anomaly','im_callback_log','im_callback_state')
  AND @im_operations_group IS NOT NULL;

UPDATE `yy_admin`
SET `system_menu_id_list`=CONCAT_WS(',',NULLIF(`system_menu_id_list`,''),@im_operations_group)
WHERE `aid`=32
  AND @im_operations_group IS NOT NULL
  AND FIND_IN_SET(@im_operations_group,`system_menu_id_list`)=0;
