<?php
namespace app\imaccess\service {
    class AccountMappingService { public function resolve($uid, $app) { return ['im_user_id' => 'fixture-mapped-owner']; } }
    function config($key) { return $key === 'im_callback.' ? ['sdk_app_id' => 'fixture-app', 'token' => 'fixture-secret'] : 'fixture-app'; }
}
namespace app\imcallback\service {
    class CallbackService {
        public static function createTraceId() { return 'fixture-trace'; }
        public function receive($raw, $token, $trace, $query) {
            $GLOBALS['deletion_event'] = json_decode($raw, true);
            if ($token !== 'fixture-secret' || !hash_equals(hash('sha256', $token . $query['RequestTime']), $query['Sign'])) throw new \RuntimeException('Invalid internal callback authentication');
            return ['http_status' => !empty($GLOBALS['fail_queue']) ? 503 : 200, 'body' => ['ErrorCode' => 0]];
        }
    }
}
namespace {
    require (getenv('BRIDGE_SERVICE_FILE') ?: '/var/www/app/application/imaccess/service/BridgeService.php');
    $service = new \app\imaccess\service\BridgeService(['enabled' => true]);
    $service->queueAccountDeleted(700001);
    if ($GLOBALS['deletion_event']['From_Account'] !== 'fixture-mapped-owner' || $GLOBALS['deletion_event']['CallbackCommand'] !== 'Bridge.AccountDeleted') throw new \RuntimeException('Wrong actor or command');
    $GLOBALS['fail_queue'] = true;
    try { $service->queueAccountDeleted(700001); throw new \RuntimeException('Queue failure accepted'); }
    catch (\RuntimeException $e) { if ($e->getMessage() !== 'Bridge account deletion could not be queued') throw $e; }
    echo "ACCOUNT_DELETION_QUEUE_OK\n";
}
