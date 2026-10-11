<?php

$host = getenv('DB_HOST') ?: '127.0.0.1';
$port = getenv('DB_PORT') ?: '3306';
$database = getenv('DB_NAME');
$username = getenv('DB_USER');
$password = getenv('DB_PASSWORD');
$prefix = getenv('DB_PREFIX') ?: 'yy_';
$retentionDays = (int) (getenv('IM_CALLBACK_RETENTION_DAYS') ?: 30);

if (!$database || !$username || $retentionDays < 1) {
    fwrite(STDERR, "DB_NAME, DB_USER and a positive retention period are required.\n");
    exit(1);
}

$pdo = new PDO(
    "mysql:host={$host};port={$port};dbname={$database};charset=utf8mb4",
    $username,
    $password,
    [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
);

$cutoff = time() - ($retentionDays * 86400);
$statement = $pdo->prepare("DELETE FROM `{$prefix}im_callback_event` WHERE received_at < :cutoff");
$statement->execute(['cutoff' => $cutoff]);

fwrite(STDOUT, 'Deleted rows: ' . $statement->rowCount() . PHP_EOL);
