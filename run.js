#!/usr/bin/env node

const { execFileSync } = require('child_process');
const path = require('path');

const binary = process.platform === 'win32'
  ? path.join(__dirname, 'mcp2cli.exe')
  : path.join(__dirname, 'mcp2cli');

try {
  execFileSync(binary, process.argv.slice(2), { stdio: 'inherit' });
} catch (e) {
  process.exit(e.status || 1);
}
