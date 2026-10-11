SET @system_root = (
  SELECT `id` FROM `yy_system_menu`
  WHERE `pid`=0 AND `type`=1 AND `title`=CONVERT(0xE7B3BBE7BB9FE7AEA1E79086 USING utf8mb4)
  ORDER BY `id` ASC LIMIT 1
);

UPDATE `yy_system_menu`
SET `pid`=@system_root
WHERE `limit_sign` IN ('im_callback_overview','im_callback_anomaly','im_callback_log','im_callback_state')
  AND @system_root IS NOT NULL;

DELETE FROM `yy_system_menu`
WHERE `limit_sign`='im_callback_root'
  AND @system_root IS NOT NULL;
