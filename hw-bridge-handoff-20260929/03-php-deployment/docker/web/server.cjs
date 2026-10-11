const http = require('http');
const fs = require('fs');
const path = require('path');
const net = require('net');

const port = Number(process.env.PORT || 8080);
const staticRoot = process.env.WEB_ROOT || '/app/public';
const apiBase = process.env.API_BASE || 'http://gateway:8080';
const mainAsset = process.env.MAIN_ASSET || '/assets/index-eab254cd.js';
const legacyAppId = process.env.LEGACY_IM_SDK_APP_ID || '1600088222';
const appId = process.env.IM_SDK_APP_ID || '';
const mime = {
  '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8', '.json': 'application/json; charset=utf-8',
  '.png': 'image/png', '.jpg': 'image/jpeg', '.jpeg': 'image/jpeg', '.svg': 'image/svg+xml',
  '.woff': 'font/woff', '.woff2': 'font/woff2', '.ttf': 'font/ttf', '.ico': 'image/x-icon',
};

const mainAssetExposeMarker = 'const fH=iw(t2e);fH.use(qL);fH.mount("#app");const kSt=';
const mainAssetExposeReplacement =
  'const fH=iw(t2e);fH.use(qL);fH.mount("#app");' +
  'globalThis.__YCH_TUI__={TUIStore:qt,StoreName:gt,TUILogin:SR,TUIUserService:Dd,TYPES:an,getChatEvents:()=>SR.EVENT,getChat:()=>SR.getContext()?.chat};' +
  'const kSt=';
const offlineLoginMarker =
  '}).catch(i=>{Fxe({message:At.t("IM登录失败"),grouping:!0,type:"error"})}):e.push({name:"login"})';
const offlineLoginReplacement =
  '}).catch(i=>{globalThis.__YCH_WEB_IM_OFFLINE__&&globalThis.__YCH_WEB_IM_OFFLINE__.canOpen()' +
  '?(globalThis.__YCH_WEB_IM_OFFLINE__.markLocalOnly(!0),e.push({path:"home"}))' +
  ':(globalThis.__YCH_WEB_IM_OFFLINE__?.clearCredentials?.(),' +
  'Fxe({message:At.t("IM登录失败"),grouping:!0,type:"error"}),e.push({name:"login"}))})' +
  ':e.push({name:"login"})';
const localFirstLoginMarker = 'localStorage.getItem("user_sig")?SR.login({';
const localFirstLoginReplacement =
  'localStorage.getItem("user_sig")?globalThis.__YCH_WEB_IM_OFFLINE__.waitForCacheReady().then(()=>SR.login({';
const localFirstLoginCloseMarker = 'framework:"vue3"}).then(()=>{e.push({path:"home"})';
const localFirstLoginCloseReplacement = 'framework:"vue3"})).then(()=>{e.push({path:"home"})';
const cachedResendMarker = 'resendMessage(){return this.getEngine().TUIChat.resendMessage(this._message)}';
const cachedResendReplacement =
  'resendMessage(){return this._message&&this._message.__ychCachedMessageId&&globalThis.YCHWebCache' +
  '?globalThis.YCHWebCache.retryTextMessage(this._message.__ychCachedConversationId,this._message.__ychCachedMessageId)' +
  ':this.getEngine().TUIChat.resendMessage(this._message)}';
const cachedDeleteMarker = 'deleteMessage(){return this.getEngine().TUIChat.deleteMessage([this._message])}';
const cachedDeleteReplacement =
  'deleteMessage(){return this._message&&this._message.__ychCachedMessageId&&globalThis.YCHWebCache' +
  '?globalThis.YCHWebCache.deleteCachedMessage(this._message.__ychCachedConversationId,this._message.__ychCachedMessageId)' +
  ':this.getEngine().TUIChat.deleteMessage([this._message])}';
const offlineMessageListMarker =
  'if(!t.chat.isReady())return Promise.reject({code:pE.GET_MSG_LIST_ERROR,message:AE.GET_MSG_LIST_ERROR});';
const offlineMessageListReplacement =
  'if(!t.chat.isReady())return globalThis.__YCH_WEB_IM_OFFLINE__&&globalThis.__YCH_WEB_IM_OFFLINE__.canOpen()' +
  '?Promise.resolve({data:{messageList:[],nextReqMessageID:"",isCompleted:!0}})' +
  ':Promise.reject({code:pE.GET_MSG_LIST_ERROR,message:AE.GET_MSG_LIST_ERROR});';
const offlineReadMarker = 'setMessageRead(e){return this.getEngine().chat.setMessageRead({conversationID:e})}';
const offlineReadReplacement =
  'setMessageRead(e){const t=this.getEngine();return!t.chat.isReady()&&globalThis.__YCH_WEB_IM_OFFLINE__&&' +
  'globalThis.__YCH_WEB_IM_OFFLINE__.canOpen()?Promise.resolve({}):t.chat.setMessageRead({conversationID:e})}';

function patchMainAsset(source) {
  let patched = source.replaceAll(legacyAppId, appId);
  if (!patched.includes(mainAssetExposeMarker)) {
    throw new Error('Web main asset expose marker not found');
  }
  if (!patched.includes(offlineLoginMarker)) {
    throw new Error('Web main asset offline marker not found');
  }
  if (!patched.includes(localFirstLoginMarker) || !patched.includes(localFirstLoginCloseMarker)) {
    throw new Error('Web main asset local-first marker not found');
  }
  if (!patched.includes(cachedResendMarker) || !patched.includes(cachedDeleteMarker)
      || !patched.includes(offlineMessageListMarker) || !patched.includes(offlineReadMarker)) {
    throw new Error('Web cached message action marker not found');
  }
  patched = patched.replace(mainAssetExposeMarker, mainAssetExposeReplacement);
  patched = patched.replace(offlineLoginMarker, offlineLoginReplacement);
  patched = patched.replace(localFirstLoginMarker, localFirstLoginReplacement);
  patched = patched.replace(localFirstLoginCloseMarker, localFirstLoginCloseReplacement);
  patched = patched.replace(cachedResendMarker, cachedResendReplacement);
  patched = patched.replace(cachedDeleteMarker, cachedDeleteReplacement);
  patched = patched.replace(offlineMessageListMarker, offlineMessageListReplacement);
  patched = patched.replace(offlineReadMarker, offlineReadReplacement);
  return patched;
}

function readBody(request) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    request.on('data', chunk => chunks.push(chunk));
    request.on('end', () => resolve(Buffer.concat(chunks)));
    request.on('error', reject);
  });
}

async function proxyApi(request, response, url) {
  const body = await readBody(request);
  const headers = { 'content-type': request.headers['content-type'] || 'application/json' };
  const clientIp = String(request.headers['x-internal-client-ip'] || '').trim();
  if (net.isIP(clientIp)) headers['x-forwarded-for'] = clientIp;
  const query = url.search ? '&' + url.search.slice(1) : '';
  const upstreamUrl = apiBase + '/index.php?s=/' + url.pathname.replace(/^\//, '') + query;
  const upstream = await fetch(upstreamUrl, {
    method: request.method,
    headers,
    body: ['GET', 'HEAD'].includes(request.method) ? undefined : body,
  });
  let raw = Buffer.from(await upstream.arrayBuffer());
  let responseStatus = upstream.status;

  if (url.pathname === '/api/login/user_login' && upstream.ok) {
    const legacy = JSON.parse(raw.toString('utf8'));
    if (legacy.code === 200 && legacy.data && legacy.data.login_token) {
      const sessionResponse = await fetch(apiBase + '/index.php?s=/imaccess/session', {
        method: 'POST',
        headers: {
          'content-type': 'application/x-www-form-urlencoded',
          ...(net.isIP(clientIp) ? { 'x-forwarded-for': clientIp } : {}),
        },
        body: new URLSearchParams({ login_token: legacy.data.login_token }),
      });
      const session = await sessionResponse.json();
      if (!sessionResponse.ok || session.code !== 200) {
        responseStatus = sessionResponse.status || 502;
        raw = Buffer.from(JSON.stringify({ code: 502, msg: session.msg || 'IM session signing failed', data: null }));
      } else {
        legacy.data.user_sig = session.data.user_sig;
        legacy.data.uid = session.data.im_user_id;
        legacy.data.sdk_app_id = session.data.sdk_app_id;
        legacy.data.im_user_id = session.data.im_user_id;
        legacy.data.expires_in = session.data.expires_in;
        legacy.data.expires_at = session.data.expires_at;
        raw = Buffer.from(JSON.stringify(legacy));
      }
    }
  }

  response.writeHead(responseStatus, {
    'content-type': upstream.headers.get('content-type') || 'application/json; charset=utf-8',
    'cache-control': 'no-store',
  });
  response.end(raw);
}

function serveStatic(response, pathname) {
  const relative = pathname === '/' ? '/index.html' : pathname;
  const decoded = decodeURIComponent(relative).replaceAll('/', path.sep);
  const file = path.resolve(staticRoot, '.' + decoded);
  if (!file.startsWith(path.resolve(staticRoot)) || !fs.existsSync(file) || !fs.statSync(file).isFile()) return false;
  let content = fs.readFileSync(file);
  if (pathname === mainAsset) {
    if (!appId) throw new Error('IM_SDK_APP_ID is required');
    content = Buffer.from(patchMainAsset(content.toString('utf8')));
  }
  const noStore = new Set(['/index.html', '/web-bootstrap.js', '/web-prelude.js', '/web-persistence.js', '/service-worker.js']);
  response.writeHead(200, {
    'content-type': mime[path.extname(file).toLowerCase()] || 'application/octet-stream',
    'cache-control': pathname === mainAsset || noStore.has(pathname) ? 'no-store' : 'public, max-age=3600',
  });
  response.end(content);
  return true;
}

http.createServer(async (request, response) => {
  try {
    const url = new URL(request.url, `http://${request.headers.host}`);
    if (url.pathname.startsWith('/api/') || url.pathname === '/imaccess/session') {
      await proxyApi(request, response, url);
      return;
    }
    if (serveStatic(response, url.pathname)) return;
    response.writeHead(404, { 'content-type': 'text/plain; charset=utf-8' });
    response.end('Not Found');
  } catch (error) {
    response.writeHead(502, { 'content-type': 'application/json; charset=utf-8' });
    response.end(JSON.stringify({ code: 502, msg: 'Web gateway error', data: null }));
  }
}).listen(port, '0.0.0.0');
