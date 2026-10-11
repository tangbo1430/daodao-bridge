const { chromium } = require('playwright');
const path = require('path');
const os = require('os');
const token = process.env.IM_CALLBACK_TEST_ADMIN_TOKEN || '';
const base = process.env.IM_CALLBACK_TEST_WEB_URL || 'http://127.0.0.1:8001/admin/page/imcallback';
if (!token) { process.stderr.write('IM_CALLBACK_TEST_ADMIN_TOKEN is required.\n'); process.exit(2); }
const pages = ['overview.html','anomalies.html','callback_log.html','im_state.html'];
(async()=>{
  const browser = await chromium.launch({headless:true,executablePath:process.env.IM_ACCESS_BROWSER_PATH});
  const output = process.env.IM_CALLBACK_SCREENSHOT_DIR || path.join(os.tmpdir(),'imcallback-ui');
  const fs = require('fs'); fs.mkdirSync(output,{recursive:true});
  let failures=0;
  for (const viewport of [{name:'desktop',width:1440,height:900},{name:'mobile',width:390,height:844}]) {
    const context = await browser.newContext({viewport:{width:viewport.width,height:viewport.height}});
    await context.addInitScript(value=>localStorage.setItem('admin_login_token',value),token);
    for (const name of pages) {
      const page = await context.newPage(); const errors=[];
      page.on('pageerror',e=>errors.push(e.message));
      page.on('console',m=>{if(m.type()==='error'&&!m.text().includes('404')) errors.push(m.text())});
      page.on('response',r=>{if(r.status()===404&&!r.url().endsWith('/favicon.ico')) errors.push('404 '+r.url())});
      await page.goto(base+'/'+name,{waitUntil:'networkidle'});
      await page.screenshot({path:path.join(output,viewport.name+'-'+name+'.png'),fullPage:true});
      const layout=await page.evaluate(()=>({width:document.documentElement.scrollWidth,client:document.documentElement.clientWidth,title:document.title}));
      const ok=errors.length===0 && layout.width<=layout.client+4;
      process.stdout.write((ok?'PASS ':'FAIL ')+viewport.name+' '+name+' title='+layout.title+' width='+layout.width+'/'+layout.client+(errors.length?' errors='+errors.join(' | '):'')+'\n');
      if(!ok) failures++;
      await page.close();
    }
    await context.close();
  }
  await browser.close(); process.stdout.write('Screenshots: '+output+'\n'); process.exit(failures?1:0);
})().catch(e=>{process.stderr.write('ADMIN UI E2E FAIL '+e.message+'\n');process.exit(1)});
