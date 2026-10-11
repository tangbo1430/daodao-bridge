UPDATE `yy_user`
SET `head_pic` = 'head_pic/head_pic.png', `update_time` = UNIX_TIMESTAMP()
WHERE `head_pic` = 'head_pic/head_pic1.png';
