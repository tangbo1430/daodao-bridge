<?php
// Run with PHP CLI and curl enabled. Fixtures never use production credentials.
namespace think { class Controller {} }
namespace app\imaccess\service { class MediaSourceService {
    public function resolve($config, $id, $sig) { return ['url' => 'https://fixture.invalid', 'headers' => [], 'resolve' => []]; }
} }
namespace app\imaccess\controller {
    function request() { return new class {
        public function method() { return 'GET'; }
        public function header($name, $default) { return $default; }
    }; }
    function input($name, $default) {
        $id = str_repeat('a', 64);
        return $name === 'id' ? $id : hash_hmac('sha256', "hw-bridge-media-v1\n" . $id, 'fixture-only');
    }
    function config($name) { return ['enabled' => true, 'secret' => 'fixture-only', 'url' => 'https://fixture.invalid']; }
    function curl_init($url) { return 1; }
    function curl_setopt_array($handle, $options) { $GLOBALS['media_options'] = $options; }
    function curl_exec($handle) {
        $o = $GLOBALS['media_options'];
        if (ob_get_level() !== 0) throw new \RuntimeException('Framework buffer still active');
        if ($o[CURLOPT_TIMEOUT] < 600 || $o[CURLOPT_LOW_SPEED_TIME] > 60) throw new \RuntimeException('Incorrect stream timeout');
        $header = $o[CURLOPT_HEADERFUNCTION];
        $header($handle, "HTTP/1.1 200 OK\r\n");
        $header($handle, "Content-Length: 8\r\n");
        $write = $o[CURLOPT_WRITEFUNCTION];
        if ($write($handle, 'fixture!') !== 8) throw new \RuntimeException('Incorrect byte count');
        return true;
    }
    function curl_errno($handle) { return 0; }
    function curl_getinfo($handle, $key) { return 200; }
    function curl_close($handle) {}
}
namespace {
    require (getenv('BRIDGE_MEDIA_FILE') ?: '/var/www/app/application/imaccess/controller/BridgeMedia.php');
    ob_start();
    echo 'framework-buffer-must-not-leak';
    (new \app\imaccess\controller\BridgeMedia())->download();
}
