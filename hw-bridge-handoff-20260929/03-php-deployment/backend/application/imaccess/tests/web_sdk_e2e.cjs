const { chromium } = require('playwright');

const baseUrl = process.env.IM_ACCESS_TEST_URL || 'http://127.0.0.1:8001';
const loginToken = process.env.IM_ACCESS_TEST_TOKEN || '';
if (!loginToken) {
  process.stderr.write('IM_ACCESS_TEST_TOKEN is required.\n');
  process.exit(2);
}

(async () => {
  const launchOptions = { headless: true };
  if (process.env.IM_ACCESS_BROWSER_PATH) launchOptions.executablePath = process.env.IM_ACCESS_BROWSER_PATH;
  const browser = await chromium.launch(launchOptions);
  const page = await browser.newPage();
  const errors = [];
  page.on('console', message => {
    if (message.type() === 'error') errors.push(message.text());
  });
  page.on('pageerror', error => errors.push(error.message));

  try {
    await page.goto(baseUrl + '/imaccess-sdk-test.html', { waitUntil: 'networkidle' });
    await page.locator('#token').fill(loginToken);
    await page.locator('#login').click();
    await page.locator('#output').filter({ hasText: 'login success' }).waitFor({ timeout: 20000 });
    const loginOutput = await page.locator('#output').textContent();
    process.stdout.write('WEB SDK LOGIN PASS ' + loginOutput.replace(/\s+/g, ' ') + '\n');

    await page.locator('#logout').click();
    await page.locator('#output').filter({ hasText: 'logout success' }).waitFor({ timeout: 10000 });
    process.stdout.write('WEB SDK LOGOUT PASS\n');
    if (errors.length) process.stdout.write('BROWSER WARNINGS ' + errors.join(' | ') + '\n');
  } finally {
    await browser.close();
  }
})().catch(error => {
  process.stderr.write('WEB SDK E2E FAIL ' + error.message + '\n');
  process.exit(1);
});
