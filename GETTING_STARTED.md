# OneMCP Getting Started Guide

OneMCP is a lightweight, high-performance router that lets you manage multiple Model Context Protocol (MCP) servers behind a single entry point.

## 1. Installation & Environment
The binaries are located in the `bin/` directory. For the best experience, add this directory to your system `PATH`.

```powershell
# Temporary PATH for current session
$env:Path += ";C:\projects\work\OneMcp\bin"
```

Verify the installation with the "doctor" command:
```powershell
onemcpctl doctor
```

## 2. Navigating the "Market" (Catalog)
OneMCP comes with a built-in catalog of popular MCP servers.

**List available servers:**
```powershell
onemcpctl catalog list
```

**Install a server (e.g., Knowledge Graph Memory):**
```powershell
onemcpctl install memory
```

**Configure required environment variables:**
If a server requires configuration (like `filesystem` needing a root path), the CLI will notify you.
```powershell
onemcpctl env set filesystem ONEMCP_FS_ROOT="C:\my-allowed-folder"
```

## 3. Adding OneMCP to VS Code & Other Clients
OneMCP can automatically configure your favorite AI tools to use the central router.

### For VS Code (GitHub Copilot / Roo Code / Continue)
```powershell
onemcpctl connect vscode
```
This updates `%APPDATA%\Code\User\mcp.json`. OneMCP will now appear as a single server with access to all your installed MCPs.

### For Cursor or Claude Desktop
```powershell
onemcpctl connect cursor
onemcpctl connect claude
```

## 4. Managing Installed Servers
**List your active router configuration:**
```powershell
onemcpctl list
```

**Enable/Disable/Uninstall:**
```powershell
onemcpctl disable memory
onemcpctl enable memory
onemcpctl uninstall memory
```

## 5. Running the Backend Manually (Advanced)
The CLI handles the background process for you, but you can run the daemon directly for debugging:
```powershell
centralmcpd --log debug
```

## 6. Testing Your Setup
Run the end-to-end smoke test suite to ensure everything is connected:
```powershell
onemcpe2e --bin bin\centralmcpd.exe
```
