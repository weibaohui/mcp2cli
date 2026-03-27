#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const platform = process.platform;
const arch = process.arch;

// Map platform names
const platformMap = {
  'darwin': 'darwin',
  'linux': 'linux',
  'win32': 'windows'
};

// Map arch names - npm uses x64, Go uses amd64
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
const targetPath = path.join(__dirname, 'mcp2cli');

// Add .exe on Windows
const exeTargetPath = platform === 'win32' ? targetPath + '.exe' : targetPath;

if (!fs.existsSync(binaryPath)) {
  console.error(`Binary not found: ${binaryPath}`);
  console.error('Please report this issue at: https://github.com/weibaohui/mcp2cli/issues');
  process.exit(1);
}

// Copy binary to target location
fs.copyFileSync(binaryPath, exeTargetPath);

// Make executable on Unix systems
if (platform !== 'win32') {
  fs.chmodSync(exeTargetPath, 0o755);
}

console.log(`Installed mcp2cli ${npmPlatform}/${npmArch} successfully`);
