#!/usr/bin/env node
'use strict';

const https = require('https');
const fs = require('fs');
const path = require('path');

const REPO = 'seanseannery/opsfile';
const BIN_DIR = path.join(__dirname, '..', 'bin');
const BIN_PATH = path.join(BIN_DIR, 'ops');

function platformAssetPrefix() {
  switch (process.platform) {
    case 'darwin': return 'ops_darwin_v';
    case 'linux':  return 'ops_unix_v';
    default:
      console.error(`Unsupported platform: ${process.platform}`);
      process.exit(1);
  }
}

function get(url) {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { 'User-Agent': 'opsfile-npm-installer' } }, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) {
        return get(res.headers.location).then(resolve).catch(reject);
      }
      const chunks = [];
      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => resolve(Buffer.concat(chunks)));
      res.on('error', reject);
    }).on('error', reject);
  });
}

async function main() {
  const prefix = platformAssetPrefix();

  console.log(`Fetching latest release from github.com/${REPO} ...`);
  const apiData = await get(`https://api.github.com/repos/${REPO}/releases/latest`);
  const release = JSON.parse(apiData.toString());

  const asset = release.assets.find(a => a.name.startsWith(prefix));
  if (!asset) {
    console.error(`No release asset found matching '${prefix}'`);
    process.exit(1);
  }

  console.log(`Downloading ops ${release.tag_name} for ${process.platform} ...`);
  const binary = await get(asset.browser_download_url);

  fs.mkdirSync(BIN_DIR, { recursive: true });
  fs.writeFileSync(BIN_PATH, binary, { mode: 0o755 });
  console.log(`Installed ops ${release.tag_name}`);
}

main().catch(err => {
  console.error('Install failed:', err.message);
  process.exit(1);
});
