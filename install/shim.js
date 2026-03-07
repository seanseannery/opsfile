#!/usr/bin/env node
'use strict';

// ops npm shim — downloads the platform binary on first use and forwards all args to it.

const { execFileSync } = require('child_process');
const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');

const REPO = 'seanseannery/opsfile';
const CACHE_DIR = path.join(os.homedir(), '.opsfile');
const BIN_PATH = path.join(CACHE_DIR, 'ops');

function platformAssetPrefix() {
  switch (process.platform) {
    case 'darwin': return 'ops_darwin_v';
    case 'linux':  return 'ops_unix_v';
    default:
      console.error(`ops: unsupported platform: ${process.platform}`);
      process.exit(1);
  }
}

function get(url) {
  const headers = { 'User-Agent': 'opsfile-npm-shim' };
  if (process.env.GITHUB_TOKEN) {
    headers['Authorization'] = `Bearer ${process.env.GITHUB_TOKEN}`;
  }
  return new Promise((resolve, reject) => {
    https.get(url, { headers }, (res) => {
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

async function download() {
  const prefix = platformAssetPrefix();
  const apiData = await get(`https://api.github.com/repos/${REPO}/releases/latest`);
  const release = JSON.parse(apiData.toString());

  if (release.message) throw new Error(`GitHub API error: ${release.message}`);
  if (!release.assets) throw new Error('GitHub API response missing assets — no published release found');

  const asset = release.assets.find(a => a.name.startsWith(prefix));
  if (!asset) throw new Error(`No release asset found matching '${prefix}'`);

  process.stderr.write(`ops: downloading ${release.tag_name} for ${process.platform}...\n`);
  const binary = await get(asset.browser_download_url);

  fs.mkdirSync(CACHE_DIR, { recursive: true });
  fs.writeFileSync(BIN_PATH, binary, { mode: 0o755 });
}

async function main() {
  if (!fs.existsSync(BIN_PATH)) {
    await download();
  }

  try {
    execFileSync(BIN_PATH, process.argv.slice(2), { stdio: 'inherit' });
  } catch (e) {
    process.exit(e.status ?? 1);
  }
}

main().catch(err => {
  console.error('ops:', err.message);
  process.exit(1);
});
