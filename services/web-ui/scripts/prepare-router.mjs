import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '../../..');
const mach1Dir = path.join(repoRoot, 'services', 'mach1');
const routerName = process.platform === 'win32' ? 'mach1.exe' : 'mach1';
const sourcePath = path.join(repoRoot, 'bin', routerName);
const destDir = path.resolve(__dirname, '../src-tauri/resources');
const destPath = path.join(destDir, routerName);

fs.mkdirSync(destDir, { recursive: true });

function buildRouter(targetPath) {
  execFileSync(
    'go',
    ['build', '-trimpath', '-ldflags', '-s -w', '-o', targetPath, './cmd/mach1'],
    {
      cwd: mach1Dir,
      stdio: 'inherit'
    }
  );
}

if (fs.existsSync(destPath)) {
  console.log(`Using existing bundled router: ${destPath}`);
} else if (fs.existsSync(sourcePath)) {
  fs.copyFileSync(sourcePath, destPath);
  console.log(`Prepared bundled router: ${destPath}`);
} else {
  try {
    buildRouter(destPath);
    console.log(`Built bundled router: ${destPath}`);
  } catch (error) {
    console.warn(`Skipping router bundling; source binary not found at ${sourcePath} and local go build failed.`);
    if (error instanceof Error) {
      console.warn(error.message);
    }
  }
}