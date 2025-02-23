import puppeteer from 'puppeteer';
import 'dotenv/config'

const SERVER_PORT = process.env.PORT || 3000;

/**
 * @param {string} [profile] - chromium profile to use for starting browser
 * @param {string} [accessToken] - access token to use for verification
 * @param {boolean} [closeBrowser] - closes the browser after screenshot is captured. If false, the server will handle closing
 * @returns {Promise<void>}
 */
async function saveImgSearchScreenshot({
   profile,
   accessToken,
   closeBrowser = false,
}) {
    try {
        const params = new URLSearchParams();
        if (profile) {
            params.set('profile', profile)
        }
        if (accessToken) {
            params.set('accessToken', accessToken)
        }

        // Launch the browser
        const browser = await puppeteer.connect({
            browserWSEndpoint: `ws://localhost:${SERVER_PORT}/connect?${params.toString()}`,
        });

        const screenshotFilePath = `scripts/example.png`;
        const page = await browser.newPage();
        await page.goto('https://news.ycombinator.com', {
            waitUntil: 'networkidle2',
        });
        await page.screenshot({ path:  screenshotFilePath })
        await page.close()

        // use browser.disconnect() in workflows where the same browser config will be reused for multiple executions
        if (closeBrowser) {
            await browser.close()
        } else {
            browser.disconnect();
        }
        console.log(`screenshot saved at ${screenshotFilePath}`);
    } catch (e) {
        console.error(`encountered error for search: ${search}`, e);
    }
}

void saveImgSearchScreenshot({})
