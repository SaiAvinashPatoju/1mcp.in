# 1mcp.in Getting Started Guide

1mcp.in is a lightweight, high-performance router that lets you manage multiple Model Context Protocol (MCP) servers behind a single entry point.

## 1. Installation & Environment
The binaries are located in the `bin/` directory. For the best experience, add this directory to your system `PATH`.

```powershell
# Temporary PATH for current session
$env:Path += ";C:\projects\work\Mach1\bin"
```

Verify the installation with the "doctor" command:
```powershell
mach1ctl doctor
```

## 2. Navigating the "Market" (Catalog)
1mcp.in comes with a built-in catalog of popular MCP servers.

**List available servers:**
```powershell
mach1ctl catalog list
```

**Install a server (e.g., Knowledge Graph Memory):**
```powershell
mach1ctl install memory
```

**Configure required environment variables:**
If a server requires configuration (like `filesystem` needing a root path), the CLI will notify you.
```powershell
mach1ctl env set filesystem MACH1_FS_ROOT="C:\my-allowed-folder"
```

## 3. Adding 1mcp.in to VS Code & Other Clients
1mcp.in can automatically configure your favorite AI tools to use the central router.

### For VS Code (GitHub Copilot / Roo Code / Continue)
```powershell
mach1ctl connect vscode
```
This updates `%APPDATA%\Code\User\mcp.json`. 1mcp.in will now appear as a single server with access to all your installed MCPs.

### For Cursor or Claude Desktop
```powershell
mach1ctl connect cursor
mach1ctl connect claude
```

## 4. Managing Installed Servers
**List your active router configuration:**
```powershell
mach1ctl list
```

**Enable/Disable/Uninstall:**
```powershell
mach1ctl disable memory
mach1ctl enable memory
mach1ctl uninstall memory
```

## 5. Running the Backend Manually (Advanced)
The CLI handles the background process for you, but you can run the daemon directly for debugging:
```powershell
mach1 --log debug
```

## 6. Testing Your Setup
Run the end-to-end smoke test suite to ensure everything is connected:
```powershell
mach1e2e --bin bin\mach1.exe
```
