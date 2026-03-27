#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const platform = process.platform;
const arch = process.arch;

const platformMap = {
  'darwin': 'darwin',
  'linux': 'linux',
  'win32': 'windows'
};

const archMap = {
  'x64': 'amd64',
  'arm64': 'arm64'
};

const npmPlatform = platformMap[platform];
const npmArch = archMap[arch];

if (!npmPlatform || !npmArch) {
  console.error(`Unsupported platform/architecture: ${platform}/${arch}`);
  process.exit(1);
}

const binaryName = `mcp2cli-${npmPlatform}-${npmArch}`;
const binaryPath = path.join(__dirname, 'dist', binaryName);
const targetPath = path.join(__dirname, platform === 'win32' ? 'mcp2cli.exe' : 'mcp2cli');

if (!fs.existsSync(binaryPath)) {
  console.error(`Binary not found: ${binaryPath}`);
  console.error('Please report this issue at: https://github.com/weibaohui/mcp2cli/issues');
  process.exit(1);
}

fs.copyFileSync(binaryPath, targetPath);

if (platform !== 'win32') {
  fs.chmodSync(targetPath, 0o755);
}
