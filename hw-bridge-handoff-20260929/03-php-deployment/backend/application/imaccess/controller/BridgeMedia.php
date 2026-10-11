<?php
namespace app\imaccess\controller;

use think\Controller;
use app\imaccess\service\MediaSourceService;

/** Signed file capability. No arbitrary URL, user token or TangSeng credential is accepted. */
class BridgeMedia extends Controller
{
    public function download()
    {
        $method = strtoupper(request()->method());
        if (!in_array($method, ['GET', 'HEAD'], true)) return response('Method Not Allowed', 405);
        $id = (string) input('id', '');
        $signature = (string) input('sig', '');
        $config = config('im_bridge.');
        $secret = (string) ($config['secret'] ?? '');
        if ($secret === '' || empty($config['enabled']) || empty($config['url'])) return response('Media unavailable', 503);
        if (!preg_match('/^[a-f0-9]{64}$/D', $id) || !preg_match('/^[a-f0-9]{64}$/D', $signature)
            || !hash_equals(hash_hmac('sha256', "hw-bridge-media-v1\n" . $id, $secret), $signature)) {
            return response('Invalid media signature', 403);
        }
        $range = (string) request()->header('range', '');
        if ($range !== '' && !preg_match('/^bytes=[0-9]*-[0-9]*$/D', $range)) return response('Invalid range', 416);
        $source = (new MediaSourceService())->resolve($config, $id, $signature, $method === 'HEAD');
        $headers = $source['headers'];
        if ($range !== '') $headers[] = 'Range: ' . $range;
        $url = $source['url'];
        // ThinkPHP debug/output buffers otherwise hold the entire video until
        // curl finishes. Release them only after the signed capability is checked.
        while (ob_get_level() > 0) {
            if (!ob_end_clean()) break;
        }
        if (session_status() === PHP_SESSION_ACTIVE) session_write_close();
        @ini_set('zlib.output_compression', '0');
        @set_time_limit(620);
        header('X-Accel-Buffering: no');
        header('X-Content-Type-Options: nosniff');
        header('Cache-Control: private, max-age=3600');
        if (!empty($source['name'])) {
            $inline = preg_match('/^(image|audio|video)\//', $source['mime'] ?? '');
            header('Content-Disposition: ' . ($inline ? 'inline' : 'attachment') . "; filename*=UTF-8''" . rawurlencode($source['name']));
        }
        $curl = curl_init($url);
        $size = 0;
        curl_setopt_array($curl, [
            CURLOPT_HTTPHEADER => $headers, CURLOPT_FOLLOWLOCATION => false,
            CURLOPT_RESOLVE => $source['resolve'],
            CURLOPT_CONNECTTIMEOUT => 8, CURLOPT_TIMEOUT => 600,
            CURLOPT_LOW_SPEED_LIMIT => 1024, CURLOPT_LOW_SPEED_TIME => 60,
            CURLOPT_NOBODY => $method === 'HEAD',
            CURLOPT_HEADERFUNCTION => function ($ch, $line) use ($source) {
                $value = trim($line);
                if (preg_match('/^HTTP\/\S+\s+(\d+)/', $value, $match)) {
                    http_response_code((int) $match[1]);
                } elseif (strpos($value, ':') !== false) {
                    list($name) = explode(':', $value, 2);
                    if (strtolower($name) === 'cache-control' || (strtolower($name) === 'content-disposition' && !empty($source['name']))) return strlen($line);
                    if (in_array(strtolower($name), ['content-type','content-length','content-range','content-disposition',
                            'accept-ranges','etag','last-modified','cache-control','x-content-type-options'], true)) header($value);
                }
                return strlen($line);
            },
            CURLOPT_WRITEFUNCTION => function ($ch, $chunk) use (&$size) {
                $size += strlen($chunk);
                if ($size > 104857600) return 0;
                echo $chunk;
                flush();
                return strlen($chunk);
            },
        ]);
        $ok = curl_exec($curl);
        $errno = curl_errno($curl);
        $status = curl_getinfo($curl, CURLINFO_HTTP_CODE);
        curl_close($curl);
        if ($ok === false) {
            // Log identifiers and byte counts only; URLs contain signed capabilities.
            error_log('bridge_media_download_failed id=' . $id . ' curl_errno=' . $errno
                . ' upstream_status=' . $status . ' bytes=' . $size);
            if (!headers_sent()) {
                header_remove('Content-Length');
                header_remove('Content-Range');
                header('Content-Type: text/plain; charset=utf-8');
                http_response_code(502);
                echo 'Media unavailable';
            }
        }
        exit;
    }
}
