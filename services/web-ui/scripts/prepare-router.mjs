import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '../../..');
const mach1Dir = path.join(repoRoot, 'services', 'mach1');
const isWin = process.platform === 'win32';
const routerName = isWin ? 'mach1.exe' : 'mach1';
const ctlName = isWin ? 'mach1ctl.exe' : 'mach1ctl';
const sourceRouter = path.join(repoRoot, 'bin', routerName);
const sourceCtl = path.join(repoRoot, 'bin', ctlName);
const destDir = path.resolve(__dirname, '../src-tauri/resources');
const destRouter = path.join(destDir, routerName);
const destCtl = path.join(destDir, ctlName);

fs.mkdirSync(destDir, { recursive: true });

function buildBinary(cmd, targetPath) {
  const version = process.env.npm_package_version || 'dev';
  const ldflags = `-s -w -X github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/version.Version=v${version}`;
  execFileSync(
    'go',
    ['build', '-trimpath', '-ldflags', ldflags, '-o', targetPath, `./cmd/${cmd}`],
    {
      cwd: mach1Dir,
      stdio: 'inherit',
      env: { ...process.env, CGO_ENABLED: '0' }
    }
  );
}

function prepareBinary(name, sourcePath, destPath, cmd) {
  if (fs.existsSync(destPath)) {
    console.log(`Using existing bundled ${name}: ${destPath}`);
  } else if (fs.existsSync(sourcePath)) {
    fs.copyFileSync(sourcePath, destPath);
    console.log(`Prepared bundled ${name}: ${destPath}`);
  } else {
    try {
      buildBinary(cmd, destPath);
      console.log(`Built bundled ${name}: ${destPath}`);
    } catch (error) {
      console.warn(`Skipping ${name} bundling; source binary not found at ${sourcePath} and local go build failed.`);
      if (error instanceof Error) {
        console.warn(error.message);
      }
    }
  }
}

prepareBinary('router', sourceRouter, destRouter, 'mach1');
prepareBinary('cli', sourceCtl, destCtl, 'mach1ctl');