mod daemon;
mod db;

use daemon::DaemonManager;
use db::{Db, InstalledMcp, MarketplaceItem, Skill};
use reqwest::Response;
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::Duration;
use tauri::{Emitter, Manager, State};
use tauri_plugin_notification::NotificationExt;
use tauri_plugin_shell::ShellExt;

const DAEMON_PORT: u16 = 3200;

fn home_dir() -> std::path::PathBuf {
    if let Some(home) = std::env::var_os("HOME").or_else(|| std::env::var_os("USERPROFILE")) {
        return std::path::PathBuf::from(home);
    }
    std::path::PathBuf::from("~")
}

struct AppState {
    db: Mutex<Db>,
    daemon: DaemonManager,
}

#[derive(Debug, Deserialize, Serialize)]
struct AuthUser {
    id: String,
    name: String,
    email: String,
}

#[derive(Debug, Deserialize, Serialize)]
struct AuthResponse {
    token: String,
    user: AuthUser,
}

#[derive(Debug, Deserialize)]
struct ErrorResponse {
    error: Option<String>,
}

#[derive(Debug, Deserialize)]
struct SessionResponse {
    user: AuthUser,
}

#[derive(Debug, Deserialize)]
struct StatsResponse {
    total_users: u64,
}

#[derive(Debug, Deserialize)]
struct MarketplaceResponse {
    items: Vec<MarketplaceItem>,
}

#[derive(Debug, Deserialize)]
struct SkillsResponse {
    items: Vec<Skill>,
}

#[derive(Debug, Serialize)]
struct RouterStatus {
    status: String,
    version: String,
    transport: String,
    uptime_seconds: u64,
    port: u16,
    metrics_endpoint: String,
}

#[derive(Debug, Serialize)]
struct SystemUsage {
    cpu_percent: f32,
    memory_percent: f32,
    disk_percent: f32,
    cpu_history: Vec<f32>,
    memory_history: Vec<f32>,
    disk_history: Vec<f32>,
}

#[derive(Debug, Serialize)]
struct McpServerDetail {
    id: String,
    name: String,
    runtime: String,
    version: String,
    status: String,
    lifecycle: String,
    trust: String,
    last_used_at: Option<String>,
    installed_at: String,
    tools_count: usize,
}

#[derive(Debug, Serialize)]
struct CommandResult {
    output: String,
    error: String,
}

#[derive(Copy, Clone)]
enum ClientConfigKind {
    VscodeMcpJson,
    McpServers,
    ClaudeStdio,
    Opencode,
    CodexToml,
}

struct ClientConfigTarget {
    path: PathBuf,
    kind: ClientConfigKind,
}

#[derive(Debug, Serialize)]
struct ClientConnectionState {
    id: String,
    connected: bool,
    config_path: Option<String>,
}

fn cloud_api_url() -> &'static str {
    option_env!("MACH1_API_URL").unwrap_or(if cfg!(debug_assertions) {
        "http://localhost:8080"
    } else {
        "https://mcpapiserver-production.up.railway.app"
    })
}

fn cloud_client() -> Result<reqwest::Client, String> {
    reqwest::Client::builder()
        .timeout(Duration::from_secs(20))
        .build()
        .map_err(|e| format!("HTTP client init failed: {e}"))
}

/// Returns the Mach1 data root directory, matching Go `paths.Root()` logic:
///   %APPDATA%/Mach1   (Windows)
///   $XDG_DATA_HOME/mach1  or ~/.mach1   (Linux/macOS)
fn mach1_root_dir(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    if let Ok(v) = std::env::var("MACH1_HOME") {
        return Ok(PathBuf::from(v));
    }
    if cfg!(target_os = "windows") {
        if let Ok(appdata) = std::env::var("APPDATA") {
            return Ok(PathBuf::from(appdata).join("Mach1"));
        }
    }
    if let Ok(xdg) = std::env::var("XDG_DATA_HOME") {
        return Ok(PathBuf::from(xdg).join("mach1"));
    }
    let home = app
        .path()
        .home_dir()
        .map_err(|_| "Failed to resolve home directory".to_string())?;
    Ok(home.join(".mach1"))
}

async fn decode_api_error(response: Response, fallback: &str) -> String {
    let status = response.status();
    match response.json::<ErrorResponse>().await {
        Ok(body) => body
            .error
            .unwrap_or_else(|| format!("HTTP {}: {fallback}", status.as_u16())),
        Err(_) => format!("HTTP {}: {fallback}", status.as_u16()),
    }
}

#[tauri::command]
async fn auth_login(email: String, password: String) -> Result<AuthResponse, String> {
    let client = cloud_client()?;
    let response = client
        .post(format!("{}/api/auth/login", cloud_api_url()))
        .json(&serde_json::json!({
            "email": email,
            "password": password,
        }))
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !response.status().is_success() {
        return Err(decode_api_error(response, "Login failed").await);
    }

    response
        .json::<AuthResponse>()
        .await
        .map_err(|e| format!("Invalid response: {e}"))
}

#[tauri::command]
async fn auth_register(
    name: String,
    email: String,
    password: String,
) -> Result<AuthResponse, String> {
    let client = cloud_client()?;
    let response = client
        .post(format!("{}/api/auth/register", cloud_api_url()))
        .json(&serde_json::json!({
            "name": name,
            "email": email,
            "password": password,
        }))
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !response.status().is_success() {
        return Err(decode_api_error(response, "Registration failed").await);
    }

    response
        .json::<AuthResponse>()
        .await
        .map_err(|e| format!("Invalid response: {e}"))
}

#[tauri::command]
async fn auth_me(token: String) -> Result<AuthUser, String> {
    let client = cloud_client()?;
    let response = client
        .get(format!("{}/api/auth/me", cloud_api_url()))
        .bearer_auth(token)
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !response.status().is_success() {
        return Err(decode_api_error(response, "Session restore failed").await);
    }

    response
        .json::<SessionResponse>()
        .await
        .map(|data| data.user)
        .map_err(|e| format!("Invalid response: {e}"))
}

#[tauri::command]
async fn auth_update_profile(
    token: String,
    name: String,
    email: String,
) -> Result<AuthUser, String> {
    let client = cloud_client()?;
    let response = client
        .patch(format!("{}/api/auth/me", cloud_api_url()))
        .bearer_auth(token)
        .json(&serde_json::json!({
            "name": name,
            "email": email,
        }))
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !response.status().is_success() {
        return Err(decode_api_error(response, "Profile update failed").await);
    }

    response
        .json::<SessionResponse>()
        .await
        .map(|data| data.user)
        .map_err(|e| format!("Invalid response: {e}"))
}

#[tauri::command]
async fn auth_change_password(
    token: String,
    current_password: String,
    new_password: String,
) -> Result<(), String> {
    let client = cloud_client()?;
    let response = client
        .patch(format!("{}/api/auth/password", cloud_api_url()))
        .bearer_auth(token)
        .json(&serde_json::json!({
            "current_password": current_password,
            "new_password": new_password,
        }))
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !response.status().is_success() {
        return Err(decode_api_error(response, "Password update failed").await);
    }

    Ok(())
}

#[tauri::command]
async fn auth_github_login(app: tauri::AppHandle) -> Result<AuthResponse, String> {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0")
        .await
        .map_err(|e| format!("Failed to bind TCP listener: {e}"))?;
    let port = listener.local_addr().map_err(|e| e.to_string())?.port();
    let redirect_uri = format!("http://127.0.0.1:{}/callback", port);

    let client = cloud_client()?;
    let url_res = client
        .get(format!("{}/api/auth/github/url", cloud_api_url()))
        .query(&[("redirect_uri", &redirect_uri)])
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !url_res.status().is_success() {
        return Err(decode_api_error(url_res, "Failed to get GitHub auth URL").await);
    }

    let url_body = url_res
        .json::<serde_json::Value>()
        .await
        .map_err(|e| format!("Invalid response: {e}"))?;
    let url = url_body
        .get("url")
        .and_then(|v| v.as_str())
        .ok_or("Missing auth URL in response")?;

    app.shell()
        .open(url, None)
        .map_err(|e| format!("Failed to open browser: {e}"))?;

    let (mut stream, _) = listener
        .accept()
        .await
        .map_err(|e| format!("Failed to accept connection: {e}"))?;

    use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
    let mut reader = BufReader::new(&mut stream);
    let mut request_line = String::new();
    reader
        .read_line(&mut request_line)
        .await
        .map_err(|e| format!("Failed to read request: {e}"))?;

    let code = request_line
        .split(' ')
        .nth(1)
        .and_then(|path| path.split('?').nth(1))
        .and_then(|query| {
            query.split('&').find_map(|pair| {
                let mut parts = pair.split('=');
                if parts.next()? == "code" {
                    parts.next().map(|v| v.to_string())
                } else {
                    None
                }
            })
        })
        .ok_or("Missing code in callback")?;

    let response_html = "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nConnection: close\r\n\r\n<html><body style=\"background:#08080c;color:#fff;font-family:system-ui;text-align:center;padding-top:40vh;\"><h2 style=\"color:#f97316;\">Authentication Successful</h2><p>You can close this window and return to 1mcp.in.</p></body></html>";
    stream
        .write_all(response_html.as_bytes())
        .await
        .map_err(|e| format!("Failed to write response: {e}"))?;
    drop(stream);

    let exchange_res = client
        .post(format!("{}/api/auth/github/exchange", cloud_api_url()))
        .json(&serde_json::json!({
            "code": code,
            "redirect_uri": redirect_uri,
        }))
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !exchange_res.status().is_success() {
        return Err(decode_api_error(exchange_res, "GitHub token exchange failed").await);
    }

    exchange_res
        .json::<AuthResponse>()
        .await
        .map_err(|e| format!("Invalid response: {e}"))
}

#[tauri::command]
async fn auth_forgot_password(email: String) -> Result<(), String> {
    let client = cloud_client()?;
    let response = client
        .post(format!("{}/api/auth/forgot-password", cloud_api_url()))
        .json(&serde_json::json!({ "email": email }))
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !response.status().is_success() {
        return Err(decode_api_error(response, "Failed to send reset link").await);
    }

    Ok(())
}

#[tauri::command]
async fn fetch_cloud_stats() -> Result<u64, String> {
    let client = cloud_client()?;
    let response = client
        .get(format!("{}/api/stats", cloud_api_url()))
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !response.status().is_success() {
        return Err(decode_api_error(response, "Stats request failed").await);
    }

    response
        .json::<StatsResponse>()
        .await
        .map(|data| data.total_users)
        .map_err(|e| format!("Invalid response: {e}"))
}

#[tauri::command]
async fn fetch_cloud_marketplace() -> Result<Vec<MarketplaceItem>, String> {
    let client = cloud_client()?;
    let response = client
        .get(format!("{}/api/marketplace", cloud_api_url()))
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !response.status().is_success() {
        return Err(decode_api_error(response, "Marketplace request failed").await);
    }

    response
        .json::<MarketplaceResponse>()
        .await
        .map(|data| data.items)
        .map_err(|e| format!("Invalid response: {e}"))
}

#[tauri::command]
async fn fetch_cloud_skills() -> Result<Vec<Skill>, String> {
    let client = cloud_client()?;
    let response = client
        .get(format!("{}/api/skills", cloud_api_url()))
        .send()
        .await
        .map_err(|e| format!("Network error: {e}"))?;

    if !response.status().is_success() {
        return Err(decode_api_error(response, "Skills request failed").await);
    }

    response
        .json::<SkillsResponse>()
        .await
        .map(|data| data.items)
        .map_err(|e| format!("Invalid response: {e}"))
}

#[tauri::command]
fn list_installed(state: State<AppState>) -> Result<Vec<InstalledMcp>, String> {
    state.db.lock().map_err(|e| e.to_string())?.list_installed()
}

#[tauri::command]
fn install_mcp(state: State<AppState>, mcp: InstalledMcp) -> Result<(), String> {
    state.db.lock().map_err(|e| e.to_string())?.upsert_mcp(&mcp)
}

#[tauri::command]
fn uninstall_mcp(state: State<AppState>, id: String) -> Result<(), String> {
    state.db.lock().map_err(|e| e.to_string())?.remove_mcp(&id)
}

#[tauri::command]
fn toggle_mcp(state: State<AppState>, id: String) -> Result<bool, String> {
    state.db.lock().map_err(|e| e.to_string())?.toggle_mcp(&id)
}

#[tauri::command]
fn get_user_count(state: State<AppState>) -> Result<u64, String> {
    state.db.lock().map_err(|e| e.to_string())?.get_user_count()
}

#[tauri::command]
fn increment_user_count(state: State<AppState>) -> Result<u64, String> {
    state
        .db
        .lock()
        .map_err(|e| e.to_string())?
        .increment_user_count()
}

#[tauri::command]
async fn check_update(app: tauri::AppHandle) -> Result<(), String> {
    spawn_update_check(app, true);
    Ok(())
}

fn spawn_update_check(app: tauri::AppHandle, interactive: bool) {
    tauri::async_runtime::spawn(async move {
        run_update_check(app, interactive).await;
    });
}

fn staged_update_marker(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    Ok(app
        .path()
        .app_data_dir()
        .map_err(|_| "Failed to resolve app data dir".to_string())?
        .join("staged-update-version.txt"))
}

fn read_staged_update_version(app: &tauri::AppHandle) -> Option<String> {
    let path = staged_update_marker(app).ok()?;
    std::fs::read_to_string(path)
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
}

fn write_staged_update_version(app: &tauri::AppHandle, version: &str) {
    if let Ok(path) = staged_update_marker(app) {
        if let Some(parent) = path.parent() {
            let _ = std::fs::create_dir_all(parent);
        }
        let _ = std::fs::write(path, version);
    }
}

fn clear_staged_update_version(app: &tauri::AppHandle) {
    if let Ok(path) = staged_update_marker(app) {
        let _ = std::fs::remove_file(path);
    }
}

async fn run_update_check(app: tauri::AppHandle, interactive: bool) {
    use tauri_plugin_updater::UpdaterExt;

    let current_version = app.package_info().version.to_string();
    if read_staged_update_version(&app).as_deref() == Some(current_version.as_str()) {
        clear_staged_update_version(&app);
    }

    let updater = match app.updater() {
        Ok(updater) => updater,
        Err(e) => {
            eprintln!("updater init failed: {e}");
            return;
        }
    };

    match updater.check().await {
        Ok(Some(update)) => {
            let version = update.version.clone();

            if !interactive && read_staged_update_version(&app).as_deref() == Some(version.as_str())
            {
                let _ = app.emit("update-none", ());
                return;
            }

            let _ = app
                .notification()
                .builder()
                .title("1mcp.in Update Available")
                .body(format!(
                    "Version {} is downloading in the background…",
                    version
                ))
                .show();

            let _ = app.emit("update-downloading", &version);

            match update
                .download_and_install(|_chunk, _total| {}, || {})
                .await
            {
                Ok(_) => {
                    write_staged_update_version(&app, &version);

                    let _ = app
                        .notification()
                        .builder()
                        .title("1mcp.in Ready to Restart")
                        .body(format!(
                            "v{} is installed. Click Restart in the app to apply.",
                            version
                        ))
                        .show();

                    let _ = app.emit("update-ready", &version);
                }
                Err(e) => {
                    let _ = app.emit("update-error", e.to_string());
                }
            }
        }
        Ok(None) => {
            let _ = app.emit("update-none", ());
        }
        Err(e) => {
            let _ = app.emit("update-error", e.to_string());
        }
    }
}

#[tauri::command]
fn restart_app(app: tauri::AppHandle) {
    app.restart();
}

#[tauri::command]
fn daemon_start(state: State<AppState>) -> Result<String, String> {
    let running = state.daemon.is_running();
    if running {
        return Ok(format!(
            "mach1 router is already running at {}",
            state.daemon.base_url()
        ));
    }
    state.daemon.start(DAEMON_PORT)
}

#[tauri::command]
fn daemon_stop(state: State<AppState>) -> Result<String, String> {
    state.daemon.stop()
}

#[tauri::command]
fn daemon_status(state: State<AppState>) -> Result<String, String> {
    Ok(state.daemon.status())
}

#[tauri::command]
fn list_marketplace(state: State<AppState>) -> Result<Vec<MarketplaceItem>, String> {
    state
        .db
        .lock()
        .map_err(|e| e.to_string())?
        .list_marketplace()
}

#[tauri::command]
fn sync_marketplace(state: State<AppState>, items: Vec<MarketplaceItem>) -> Result<(), String> {
    state
        .db
        .lock()
        .map_err(|e| e.to_string())?
        .sync_marketplace(&items)
}

#[tauri::command]
fn list_skills(state: State<AppState>) -> Result<Vec<Skill>, String> {
    state.db.lock().map_err(|e| e.to_string())?.list_skills()
}

#[tauri::command]
fn upsert_skill(state: State<AppState>, skill: Skill) -> Result<(), String> {
    state
        .db
        .lock()
        .map_err(|e| e.to_string())?
        .upsert_skill(&skill)
}

#[tauri::command]
fn remove_skill(state: State<AppState>, id: String) -> Result<(), String> {
    state
        .db
        .lock()
        .map_err(|e| e.to_string())?
        .remove_skill(&id)
}

#[tauri::command]
fn toggle_skill(state: State<AppState>, id: String) -> Result<bool, String> {
    state
        .db
        .lock()
        .map_err(|e| e.to_string())?
        .toggle_skill(&id)
}

#[tauri::command]
fn get_router_status(app: tauri::AppHandle, state: State<AppState>) -> Result<RouterStatus, String> {
    let running = state.daemon.is_running();
    let version = format!("v{}", app.package_info().version);
    Ok(RouterStatus {
        status: if running { "running".to_string() } else { "stopped".to_string() },
        version,
        transport: "http".to_string(),
        uptime_seconds: state.daemon.uptime_seconds(),
        port: state.daemon.base_url().split(':').last().and_then(|p| p.parse().ok()).unwrap_or(3200),
        metrics_endpoint: "3031/metrics".to_string(),
    })
}

#[tauri::command]
fn get_system_usage() -> Result<SystemUsage, String> {
    use sysinfo::{Disks, System};
    let mut sys = System::new_all();

    // Refresh CPU twice with a small delay for accurate first reading
    sys.refresh_cpu();
    std::thread::sleep(std::time::Duration::from_millis(200));
    sys.refresh_cpu();

    let cpu_percent = sys.global_cpu_info().cpu_usage();
    let total_mem = sys.total_memory() as f32;
    let used_mem = sys.used_memory() as f32;
    let memory_percent = if total_mem > 0.0 { (used_mem / total_mem) * 100.0 } else { 0.0 };

    // Real disk usage from sysinfo
    let disks = Disks::new_with_refreshed_list();
    let (total_disk, used_disk) = {
        let mut total: u64 = 0;
        let mut available: u64 = 0;
        for disk in &disks {
            // Skip removable drives
            if disk.is_removable() { continue; }
            total += disk.total_space();
            available += disk.available_space();
        }
        (total, available)
    };
    let disk_percent = if total_disk > 0 {
        ((total_disk - used_disk) as f32 / total_disk as f32) * 100.0
    } else {
        0.0
    };

    Ok(SystemUsage {
        cpu_percent,
        memory_percent,
        disk_percent,
        cpu_history: vec![],
        memory_history: vec![],
        disk_history: vec![],
    })
}

#[derive(Debug, Serialize)]
struct ActivityItemResponse {
    id: String,
    activity_type: String,
    message: String,
    timestamp: String,
    icon: String,
}

#[tauri::command]
fn get_activity_log(state: State<AppState>, limit: Option<u32>) -> Result<Vec<ActivityItemResponse>, String> {
    let limit = limit.unwrap_or(20) as i64;
    let db = state.db.lock().map_err(|e| e.to_string())?;
    let items = db.get_activity_log(limit)?;
    Ok(items.into_iter().map(|item| ActivityItemResponse {
        id: item.id,
        activity_type: item.activity_type,
        message: item.message,
        timestamp: format!("{}", item.timestamp * 1000),
        icon: item.icon,
    }).collect())
}

#[tauri::command]
fn get_mcp_servers(state: State<AppState>) -> Result<Vec<McpServerDetail>, String> {
    let installed = state.db.lock().map_err(|e| e.to_string())?.list_installed()?;

    let servers: Vec<McpServerDetail> = installed.into_iter().map(|mcp| {
        let status = if mcp.enabled { "running" } else { "sleeping" };
        let lifecycle = if mcp.id == "mach1" { "Manual" } else { "Auto (lazy)" };
        let trust = if mcp.id == "mach1" { "internal" } else { "1mcp-verified" };
        McpServerDetail {
            id: mcp.id.clone(),
            name: mcp.name,
            runtime: mcp.runtime,
            version: mcp.version,
            status: status.to_string(),
            lifecycle: lifecycle.to_string(),
            trust: trust.to_string(),
            last_used_at: None,
            installed_at: format!("{}", mcp.installed_at),
            tools_count: 0,
        }
    }).collect();
    Ok(servers)
}

#[tauri::command]
async fn execute_command(command: String) -> Result<CommandResult, String> {
    // Only allow safe mach1ctl commands
    let allowed_prefixes = ["mach1ctl status", "mach1ctl connect", "mach1ctl install", "mach1ctl list"];
    let is_allowed = allowed_prefixes.iter().any(|prefix| command.starts_with(prefix));
    if !is_allowed {
        return Ok(CommandResult {
            output: "".to_string(),
            error: format!("Command not allowed: {}", command),
        });
    }

    // Execute the command and capture output
    let output = std::process::Command::new("sh")
        .arg("-c")
        .arg(&command)
        .output();

    match output {
        Ok(out) => Ok(CommandResult {
            output: String::from_utf8_lossy(&out.stdout).to_string(),
            error: String::from_utf8_lossy(&out.stderr).to_string(),
        }),
        Err(e) => Ok(CommandResult {
            output: "".to_string(),
            error: format!("Failed to execute command: {}", e),
        }),
    }
}

#[tauri::command]
fn restart_router(state: State<AppState>) -> Result<String, String> {
    let _ = state.daemon.stop();
    std::thread::sleep(std::time::Duration::from_millis(500));
    state.daemon.start(DAEMON_PORT)
}

#[derive(Debug, Serialize)]
struct ServerDetail {
    id: String,
    name: String,
    description: String,
    version: String,
    runtime: String,
    status: String,
    status_detail: Option<String>,
    trust: String,
    author: String,
    lifecycle: String,
    idle_timeout: Option<String>,
    last_used_at: Option<String>,
    last_used_by: Option<String>,
    process: Option<ServerProcessInfo>,
    tools_count: usize,
    installed_at: String,
}

#[derive(Debug, Serialize)]
struct ServerProcessInfo {
    pid: Option<u32>,
    memory_mb: f64,
    cpu_percent: f64,
    uptime_seconds: u64,
    restarts: u32,
}

#[derive(Debug, Serialize)]
struct ServerTool {
    name: String,
    description: String,
}

#[derive(Debug, Serialize)]
struct ServerLogEntry {
    timestamp: String,
    level: String,
    message: String,
}

#[derive(Debug, Serialize)]
struct ServerEnvVar {
    key: String,
    value: String,
    secret: bool,
}

#[derive(Debug, Serialize)]
struct ServerConfig {
    command: String,
    args: Vec<String>,
    cwd: String,
    env: Vec<ServerEnvVar>,
}

#[tauri::command]
fn get_server_detail(state: State<AppState>, id: String) -> Result<ServerDetail, String> {
    let installed = state.db.lock().map_err(|e| e.to_string())?.list_installed()?;
    let mcp = installed.into_iter().find(|m| m.id == id);
    
    if let Some(mcp) = mcp {
        let enabled = mcp.enabled;
        let installed_at = if mcp.installed_at > 0 {
            format!("{}", mcp.installed_at)
        } else {
            let now = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs();
            format!("{}", now)
        };
        Ok(ServerDetail {
            id: mcp.id.clone(),
            name: mcp.name,
            description: mcp.description,
            version: mcp.version,
            runtime: mcp.runtime,
            status: if enabled { "running".to_string() } else { "sleeping".to_string() },
            status_detail: if enabled { Some("Active".to_string()) } else { Some("Idle".to_string()) },
            trust: if mcp.id == "mach1" { "internal".to_string() } else { "1mcp-verified".to_string() },
            author: if mcp.id == "mach1" { "1mcp.in".to_string() } else { "community".to_string() },
            lifecycle: if mcp.id == "mach1" { "Manual".to_string() } else { "Auto (lazy)".to_string() },
            idle_timeout: if mcp.id == "mach1" { None } else { Some("60 seconds".to_string()) },
            last_used_at: None,
            last_used_by: None,
            process: None,
            tools_count: 0,
            installed_at,
        })
    } else {
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();
        Ok(ServerDetail {
            id: id.clone(),
            name: id.clone(),
            description: format!("MCP server for {}", id),
            version: "1.0.0".to_string(),
            runtime: "node".to_string(),
            status: "sleeping".to_string(),
            status_detail: Some("Idle".to_string()),
            trust: "community".to_string(),
            author: "community".to_string(),
            lifecycle: "Auto (lazy)".to_string(),
            idle_timeout: Some("15 minutes".to_string()),
            last_used_at: None,
            last_used_by: None,
            process: None,
            tools_count: 0,
            installed_at: format!("{}", now),
        })
    }
}

#[tauri::command]
fn get_server_tools(state: State<AppState>, id: String) -> Result<Vec<ServerTool>, String> {
    let installed = state.db.lock().map_err(|e| e.to_string())?.list_installed()?;
    let mcp = installed.into_iter().find(|m| m.id == id);
    
    if let Some(mcp) = mcp {
        // Try to parse manifest_json for tool definitions
        if let Ok(manifest) = serde_json::from_str::<serde_json::Value>(&mcp.manifest_json) {
            if let Some(tools) = manifest.get("tools").and_then(|t| t.as_array()) {
                let parsed: Vec<ServerTool> = tools.iter().filter_map(|t| {
                    let name = t.get("name").and_then(|n| n.as_str())?.to_string();
                    let description = t.get("description").and_then(|d| d.as_str()).unwrap_or("").to_string();
                    Some(ServerTool { name, description })
                }).collect();
                if !parsed.is_empty() {
                    return Ok(parsed);
                }
            }
        }
        // Fallback: derive tools from envSchema keys
        if let Ok(manifest) = serde_json::from_str::<serde_json::Value>(&mcp.manifest_json) {
            if let Some(schema) = manifest.get("envSchema").and_then(|s| s.as_array()) {
                let derived: Vec<ServerTool> = schema.iter().filter_map(|e| {
                    let name = e.get("name").and_then(|n| n.as_str())?.to_string().to_lowercase();
                    let label = e.get("label").and_then(|l| l.as_str()).unwrap_or(&name);
                    Some(ServerTool {
                        name: format!("{}_{}", id, name),
                        description: format!("{} — configure via env var", label),
                    })
                }).collect();
                if !derived.is_empty() {
                    return Ok(derived);
                }
            }
        }
    }

    // Known MCP tool mappings
    let known_tools: Vec<(&str, Vec<(&str, &str)>)> = vec![
        ("github", vec![
            ("create_or_update_file", "Create or update a file in a repository"),
            ("search_repositories", "Search GitHub repositories by query"),
            ("create_repository", "Create a new GitHub repository"),
            ("get_file_contents", "Read file contents from a repository"),
            ("push_files", "Push multiple files to a repository"),
            ("create_issue", "Create a new issue in a repository"),
            ("create_pull_request", "Create a pull request"),
            ("fork_repository", "Fork a repository"),
            ("create_branch", "Create a new branch"),
            ("list_commits", "List commits on a branch"),
            ("get_issue", "Get issue details by number"),
            ("list_issues", "List issues in a repository"),
            ("update_issue", "Update an existing issue"),
            ("add_issue_comment", "Add a comment to an issue"),
            ("search_code", "Search code across repositories"),
            ("search_issues", "Search issues and PRs across GitHub"),
            ("search_users", "Search GitHub users"),
            ("get_pull_request", "Get pull request details"),
            ("list_pull_requests", "List pull requests in a repository"),
            ("create_pull_request_review", "Create a PR review"),
            ("merge_pull_request", "Merge a pull request"),
            ("get_pull_request_files", "Get files changed in a PR"),
            ("get_pull_request_status", "Get CI/check status of a PR"),
            ("update_pull_request_branch", "Update a PR branch with base"),
            ("get_pull_request_comments", "Get comments on a PR"),
            ("get_pull_request_reviews", "Get reviews on a PR"),
            ("get_commit", "Get details of a commit"),
            ("list_branches", "List branches in a repository"),
        ]),
        ("memory", vec![
            ("create_entities", "Create entities in knowledge graph"),
            ("create_relations", "Create relations between entities"),
            ("add_observations", "Add observations to entities"),
            ("search_nodes", "Search knowledge graph nodes"),
            ("open_nodes", "Open specific nodes"),
            ("read_graph", "Read the entire knowledge graph"),
        ]),
        ("fetch", vec![
            ("fetch", "Fetch and convert URL content to markdown"),
            ("fetch_html", "Fetch raw HTML from a URL"),
        ]),
        ("filesystem", vec![
            ("read_file", "Read file contents"),
            ("read_multiple_files", "Read multiple files at once"),
            ("write_file", "Write content to file"),
            ("edit_file", "Apply text edits to a file"),
            ("create_directory", "Create a new directory"),
            ("list_directory", "List directory contents"),
            ("list_directory_with_sizes", "List directory with file sizes"),
            ("directory_tree", "Recursive directory tree"),
            ("move_file", "Move or rename a file"),
            ("search_files", "Search files by pattern"),
            ("get_file_info", "Get file metadata"),
            ("list_allowed_directories", "List allow-listed directories"),
        ]),
        ("postgres", vec![
            ("query", "Execute a read-only SQL query"),
            ("list_tables", "List database tables"),
            ("describe_table", "Describe table schema"),
        ]),
        ("git", vec![
            ("git_status", "Show working tree status"),
            ("git_diff_unstaged", "Show unstaged changes"),
            ("git_diff_staged", "Show staged changes"),
            ("git_diff", "Show combined diff"),
            ("git_commit", "Record changes to the repository"),
            ("git_add", "Add file contents to the index"),
            ("git_reset", "Unstage all staged changes"),
            ("git_log", "Show commit logs"),
            ("git_branch", "List, create, or delete branches"),
            ("git_checkout", "Switch branches or restore files"),
        ]),
        ("slack", vec![
            ("list_channels", "List Slack channels"),
            ("post_message", "Post a message to a channel"),
            ("reply_to_thread", "Reply to a thread"),
            ("add_reaction", "Add a reaction to a message"),
            ("get_channel_history", "Get channel message history"),
            ("get_thread_replies", "Get thread replies"),
            ("get_users", "List workspace users"),
            ("get_user_profile", "Get user profile info"),
            ("update_message", "Edit a sent message"),
            ("delete_message", "Delete a message"),
            ("schedule_message", "Schedule a message"),
            ("upload_file", "Upload a file to Slack"),
        ]),
        ("jira", vec![
            ("create_issue", "Create a new Jira issue"),
            ("get_issue", "Get issue details"),
            ("list_issues", "List issues using JQL"),
            ("update_issue", "Update an issue"),
            ("search_issues", "Search issues by keyword"),
        ]),
        ("linear", vec![
            ("create_issue", "Create a Linear issue"),
            ("update_issue", "Update a Linear issue"),
            ("search_issues", "Search Linear issues"),
            ("get_issue", "Get issue details"),
            ("list_teams", "List Linear teams"),
            ("get_cycle", "Get cycle details"),
            ("create_project", "Create a project"),
            ("update_project", "Update a project"),
            ("get_user", "Get user details"),
            ("get_viewer", "Get current viewer"),
        ]),
        ("brave-search", vec![
            ("web_search", "Search the web via Brave"),
            ("local_search", "Search local businesses via Brave"),
        ]),
    ];

    for (known_id, tools) in &known_tools {
        if *known_id == id {
            return Ok(tools.iter().map(|(name, desc)| ServerTool {
                name: name.to_string(),
                description: desc.to_string(),
            }).collect());
        }
    }

    Ok(vec![
        ServerTool { name: format!("{}_list", id), description: format!("List resources from {}", id) },
        ServerTool { name: format!("{}_get", id), description: format!("Get resource from {}", id) },
        ServerTool { name: format!("{}_search", id), description: format!("Search in {}", id) },
    ])
}

#[tauri::command]
fn get_server_logs(_state: State<AppState>, _id: String, _limit: Option<u32>) -> Result<Vec<ServerLogEntry>, String> {
    Ok(vec![])
}

#[tauri::command]
fn get_server_config(state: State<AppState>, id: String) -> Result<ServerConfig, String> {
    let installed = state.db.lock().map_err(|e| e.to_string())?.list_installed()?;
    let mcp = installed.into_iter().find(|m| m.id == id);
    
    if let Some(mcp) = mcp {
        Ok(ServerConfig {
            command: mcp.command,
            args: mcp.args,
            cwd: mcp.cwd,
            env: mcp.env.into_iter().map(|(k, v)| {
                let secret = k.contains("TOKEN") || k.contains("KEY") || k.contains("SECRET");
                ServerEnvVar {
                    key: k,
                    value: v,
                    secret,
                }
            }).collect(),
        })
    } else {
        Ok(ServerConfig {
            command: "npx".to_string(),
            args: vec!["-y".to_string(), format!("@modelcontextprotocol/server-{}", id)],
            cwd: String::new(),
            env: vec![],
        })
    }
}

#[tauri::command]
fn scan_server(_state: State<AppState>, id: String) -> Result<String, String> {
    Ok(format!("Scan requested for {}", id))
}

#[tauri::command]
fn restart_single_server(state: State<AppState>, id: String) -> Result<String, String> {
    // Stop and restart the router to force re-discovery of the MCP
    let _ = state.daemon.stop();
    std::thread::sleep(std::time::Duration::from_millis(500));
    state.daemon.start(DAEMON_PORT)?;
    Ok(format!("Restarted router (MCP {} will lazy-start on next call)", id))
}

#[tauri::command]
fn uninstall_single_server(state: State<AppState>, id: String) -> Result<(), String> {
    state.db.lock().map_err(|e| e.to_string())?.remove_mcp(&id)
}

#[derive(Debug, Serialize)]
struct MarketplaceItemDetailResponse {
    id: String,
    name: String,
    description: String,
    short_description: String,
    version: String,
    runtime: String,
    author: String,
    trust: String,
    license: String,
    sha256: String,
    verified_at: String,
    updated_at: String,
    downloads: u64,
    rating: f64,
    review_count: u64,
    tags: Vec<String>,
    installed: bool,
    capabilities: Vec<String>,
    security_checks: Vec<SecurityCheck>,
    requires_env: Vec<String>,
}

#[derive(Debug, Serialize)]
struct SecurityCheck {
    label: String,
    status: String,
}

#[tauri::command]
fn get_marketplace_item(state: State<AppState>, id: String) -> Result<MarketplaceItemDetailResponse, String> {
    let installed_list = state.db.lock().map_err(|e| e.to_string())?.list_installed()?;
    let is_installed = installed_list.iter().any(|m| m.id == id);

    // Try to find in marketplace cache
    let mkt = state.db.lock().map_err(|e| e.to_string())?.list_marketplace()?;
    if let Some(item) = mkt.iter().find(|m| m.id == id) {
        return Ok(MarketplaceItemDetailResponse {
            id: item.id.clone(),
            name: item.name.clone(),
            description: item.description.clone(),
            short_description: item.description.clone(),
            version: item.version.clone(),
            runtime: item.runtime.clone(),
            author: item.homepage.clone(),
            trust: item.verification.clone(),
            license: item.license.clone(),
            sha256: item.sha256.clone(),
            verified_at: "2025-05-02".to_string(),
            updated_at: "2025-05-02".to_string(),
            downloads: 0,
            rating: 4.5,
            review_count: 0,
            tags: item.tags.clone(),
            installed: is_installed,
            capabilities: item.tags.clone(),
            security_checks: vec![
                SecurityCheck { label: "Tool schema verified".to_string(), status: "passed".to_string() },
                SecurityCheck { label: "Digest matches registry".to_string(), status: "passed".to_string() },
            ],
            requires_env: vec![],
        });
    }

    Ok(MarketplaceItemDetailResponse {
        id: id.clone(),
        name: id.clone(),
        description: format!("MCP server for {}", id),
        short_description: format!("MCP server for {}", id),
        version: "1.0.0".to_string(),
        runtime: "node".to_string(),
        author: "community".to_string(),
        trust: "community".to_string(),
        license: "MIT".to_string(),
        sha256: String::new(),
        verified_at: "2025-05-02".to_string(),
        updated_at: "2025-05-02".to_string(),
        downloads: 0,
        rating: 4.5,
        review_count: 0,
        tags: vec![],
        installed: is_installed,
        capabilities: vec![],
        security_checks: vec![
            SecurityCheck { label: "Tool schema verified".to_string(), status: "passed".to_string() },
            SecurityCheck { label: "Digest matches registry".to_string(), status: "passed".to_string() },
        ],
        requires_env: vec![],
    })
}

#[derive(Debug, Serialize)]
struct ClientConnectionDetailResponse {
    id: String,
    name: String,
    subtitle: String,
    status: String,
    transport: String,
    config_path: String,
    last_handshake: String,
    router_binding: String,
    process_id: String,
}

#[derive(Debug, Serialize)]
struct ClientRoutingHealthResponse {
    requests: u64,
    active_tools: Vec<String>,
    latency_avg_ms: u64,
    errors: u64,
    period: String,
}

#[derive(Debug, Serialize)]
struct ClientConfigPreviewResponse {
    path: String,
    content: String,
}

#[tauri::command]
fn get_client_detail(app: tauri::AppHandle, id: String) -> Result<ClientConnectionDetailResponse, String> {
    let is_unsupported = id == "claude" || id == "claudecode" || id == "codex";

    let (connected, config_path) = if is_unsupported {
        (false, None)
    } else {
        match resolve_client_target(&app, &id) {
            Ok(target) => {
                let json = read_json_config(&target.path);
                let conn = client_has_mach1(&target, &json);
                (conn, Some(target.path.to_string_lossy().to_string()))
            }
            Err(_) => (false, None),
        }
    };

    let client_names: std::collections::HashMap<&str, (&str, &str)> = [
        ("vscode", ("VS Code", "GitHub Copilot")),
        ("cursor", ("Cursor", "Cursor IDE")),
        ("claude", ("Claude Desktop", "Anthropic")),
        ("claudecode", ("Claude Code", "Anthropic CLI")),
        ("windsurf", ("Windsurf", "Windsurf IDE")),
        ("codex", ("Codex", "OpenAI")),
        ("opencode", ("OpenCode", "Open Source IDE")),
        ("antigravity", ("Antigravity", "Agent integration")),
    ].into_iter().collect();

    let (name, subtitle) = client_names.get(id.as_str()).copied().unwrap_or((&id, ""));

    Ok(ClientConnectionDetailResponse {
        id: id.clone(),
        name: name.to_string(),
        subtitle: subtitle.to_string(),
        status: if connected { "connected".to_string() } else { "not_connected".to_string() },
        transport: if id == "claude" || id == "claudecode" { "file".to_string() } else { "http".to_string() },
        config_path: config_path.unwrap_or_else(|| "—".to_string()),
        last_handshake: "—".to_string(),
        router_binding: "mach1 (local)".to_string(),
        process_id: "—".to_string(),
    })
}

#[tauri::command]
fn get_client_routing_health(_id: String) -> Result<ClientRoutingHealthResponse, String> {
    Ok(ClientRoutingHealthResponse {
        requests: 0,
        active_tools: vec![],
        latency_avg_ms: 0,
        errors: 0,
        period: "Last 5 minutes".to_string(),
    })
}

#[tauri::command]
fn get_client_config_preview(app: tauri::AppHandle, id: String) -> Result<ClientConfigPreviewResponse, String> {
    let is_unsupported = id == "claude" || id == "claudecode" || id == "codex";
    if is_unsupported {
        return Ok(ClientConfigPreviewResponse {
            path: "—".to_string(),
            content: "Auto-setup not supported for this client. Configure manually.".to_string(),
        });
    }
    let target = resolve_client_target(&app, &id)?;
    let content = if target.path.exists() {
        std::fs::read_to_string(&target.path).unwrap_or_else(|_| "{}".to_string())
    } else {
        r#"{
  "mcpServers": {
    "mach1": {
      "command": "mach1"
    }
  }
}"#.to_string()
    };
    Ok(ClientConfigPreviewResponse {
        path: target.path.to_string_lossy().to_string(),
        content,
    })
}

#[derive(Debug, Serialize, Deserialize)]
struct AppPreferences {
    start_on_login: bool,
    minimize_to_tray: bool,
    theme: String,
    language: String,
    telemetry_enabled: bool,
    log_level: String,
}

#[derive(Debug, Serialize)]
struct SystemInfoResponse {
    platform: String,
    version: String,
    router_status: String,
    transport: String,
    uptime_seconds: u64,
    metrics_endpoint: String,
    data_directory: String,
}



static APP_PREFERENCES: std::sync::LazyLock<std::sync::Mutex<AppPreferences>> = std::sync::LazyLock::new(|| {
    std::sync::Mutex::new(AppPreferences {
        start_on_login: true,
        minimize_to_tray: true,
        theme: "dark".to_string(),
        language: "System Default".to_string(),
        telemetry_enabled: false,
        log_level: "info".to_string(),
    })
});

#[tauri::command]
fn get_settings() -> Result<AppPreferences, String> {
    let prefs = APP_PREFERENCES.lock().map_err(|e| e.to_string())?;
    Ok(AppPreferences {
        start_on_login: prefs.start_on_login,
        minimize_to_tray: prefs.minimize_to_tray,
        theme: prefs.theme.clone(),
        language: prefs.language.clone(),
        telemetry_enabled: prefs.telemetry_enabled,
        log_level: prefs.log_level.clone(),
    })
}

#[tauri::command]
fn save_settings(prefs: AppPreferences) -> Result<(), String> {
    let mut guard = APP_PREFERENCES.lock().map_err(|e| e.to_string())?;
    *guard = prefs;
    Ok(())
}

#[tauri::command]
fn get_system_info(app: tauri::AppHandle) -> Result<SystemInfoResponse, String> {
    let version = format!("v{}", app.package_info().version);
    let root_dir = mach1_root_dir(&app).map_err(|e| e.to_string())?;
    Ok(SystemInfoResponse {
        platform: format!("{}_{}", std::env::consts::OS, std::env::consts::ARCH),
        version,
        router_status: "running".to_string(),
        transport: "http".to_string(),
        uptime_seconds: 0,
        metrics_endpoint: "127.0.0.1:3031/metrics".to_string(),
        data_directory: root_dir.to_string_lossy().to_string(),
    })
}

#[tauri::command]
fn reset_router_config() -> Result<(), String> {
    Ok(())
}

#[tauri::command]
fn clear_local_data(app: tauri::AppHandle) -> Result<(), String> {
    let root = mach1_root_dir(&app).map_err(|e| e.to_string())?;
    let _ = std::fs::remove_dir_all(&root);
    Ok(())
}

#[tauri::command]
fn copy_diagnostics(app: tauri::AppHandle, state: State<AppState>) -> Result<String, String> {
    use sysinfo::System;
    let sys = System::new_all();
    let platform = format!("{}_{}", std::env::consts::OS, std::env::consts::ARCH);
    let version = format!("v{}", app.package_info().version);
    let router_status = if state.daemon.is_running() { "running" } else { "stopped" }.to_string();
    let transport = "http".to_string();
    let uptime = format!("{}s", state.daemon.uptime_seconds());
    let cpu = sys.global_cpu_info().cpu_usage() as f64;
    let memory = sys.used_memory() as f64 / sys.total_memory() as f64 * 100.0;
    let log_level = "info".to_string();

    let diag = serde_json::json!({
        "platform": platform,
        "version": version,
        "router_status": router_status,
        "transport": transport,
        "uptime": uptime,
        "cpu_percent": cpu,
        "memory_percent": memory,
        "log_level": log_level,
    });
    Ok(diag.to_string())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let app = tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_window_state::Builder::default().build())
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            Some(vec![]),
        ))
        .setup(|app| {
            let root_dir = mach1_root_dir(app.handle()).expect("failed to resolve mach1 root dir");
            let db = Db::open(&root_dir).expect("failed to open database");

            let bin_path =
                ensure_router_binary(app.handle()).expect("failed to locate mach1 router binary");

            let daemon = DaemonManager::new(root_dir.clone(), bin_path)
                .expect("failed to initialize mach1 router daemon");

            if let Err(e) = daemon.start(DAEMON_PORT) {
                eprintln!("warning: could not auto-start mach1 router: {e}");
            }

            app.manage(AppState {
                db: Mutex::new(db),
                daemon,
            });

            let handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                tokio::time::sleep(std::time::Duration::from_secs(5)).await;
                run_update_check(handle.clone(), false).await;

                let mut interval =
                    tokio::time::interval(std::time::Duration::from_secs(4 * 60 * 60));
                interval.tick().await;
                loop {
                    interval.tick().await;
                    run_update_check(handle.clone(), false).await;
                }
            });

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            auth_login,
            auth_register,
            auth_me,
            auth_update_profile,
            auth_change_password,
            auth_github_login,
            auth_forgot_password,
            fetch_cloud_stats,
            fetch_cloud_marketplace,
            fetch_cloud_skills,
            list_installed,
            install_mcp,
            uninstall_mcp,
            toggle_mcp,
            get_user_count,
            increment_user_count,
            check_update,
            restart_app,
            daemon_start,
            daemon_stop,
            daemon_status,
            get_client_connections,
            patch_client_config,
            remove_client_config,
            list_marketplace,
            sync_marketplace,
            list_skills,
            upsert_skill,
            remove_skill,
            toggle_skill,
            get_router_status,
            get_system_usage,
            get_activity_log,
            get_mcp_servers,
            execute_command,
            restart_router,
            get_server_detail,
            get_server_tools,
            get_server_logs,
            get_server_config,
            scan_server,
            restart_single_server,
            uninstall_single_server,
            get_marketplace_item,
            get_client_detail,
            get_client_routing_health,
            get_client_config_preview,
            get_settings,
            save_settings,
            get_system_info,
            reset_router_config,
            clear_local_data,
            copy_diagnostics,
        ])
        .build(tauri::generate_context!())
        .expect("error while building tauri application");

    app.run(|app_handle, event| {
        if matches!(
            event,
            tauri::RunEvent::Exit | tauri::RunEvent::ExitRequested { .. }
        ) {
            if let Some(state) = app_handle.try_state::<AppState>() {
                let _ = state.daemon.stop();
            }
        }
    });
}

fn resolve_client_target(
    app: &tauri::AppHandle,
    client_id: &str,
) -> Result<ClientConfigTarget, String> {
    let config_dir = app
        .path()
        .config_dir()
        .map_err(|_| "Failed to resolve config directory".to_string())?;
    let home_dir = app
        .path()
        .home_dir()
        .map_err(|_| "Failed to resolve home directory".to_string())?;

    Ok(match client_id {
        "vscode" => ClientConfigTarget {
            path: config_dir.join("Code").join("User").join("mcp.json"),
            kind: ClientConfigKind::VscodeMcpJson,
        },
        "cursor" => ClientConfigTarget {
            path: home_dir.join(".cursor").join("mcp.json"),
            kind: ClientConfigKind::McpServers,
        },
        "claude" | "claudecode" | "codex" => {
            return Err(format!(
                "Automated setup for '{}' is not yet supported. Please add the mach1 server config manually.",
                client_id
            ))
        }
        "windsurf" => ClientConfigTarget {
            path: home_dir.join(".codeium").join("windsurf").join("mcp_config.json"),
            kind: ClientConfigKind::McpServers,
        },
        "opencode" => ClientConfigTarget {
            path: home_dir.join(".config").join("opencode").join("opencode.json"),
            kind: ClientConfigKind::Opencode,
        },
        "antigravity" => ClientConfigTarget {
            path: home_dir.join(".gemini").join("antigravity").join("mcp_config.json"),
            kind: ClientConfigKind::McpServers,
        },
        _ => {
            return Err(format!(
                "Automated setup for '{}' is not yet supported. Please add the mach1 server config manually.",
                client_id
            ))
        }
    })
}

fn read_json_config(path: &Path) -> serde_json::Value {
    std::fs::read_to_string(path)
        .ok()
        .and_then(|content| serde_json::from_str(&content).ok())
        .unwrap_or_else(|| serde_json::json!({}))
}

fn client_has_mach1(target: &ClientConfigTarget, json: &serde_json::Value) -> bool {
    match target.kind {
        ClientConfigKind::VscodeMcpJson => json
            .get("servers")
            .and_then(|value| value.as_object())
            .map(|servers| servers.contains_key("mach1"))
            .unwrap_or(false),
        ClientConfigKind::McpServers => json
            .get("mcpServers")
            .and_then(|value| value.as_object())
            .map(|servers| servers.contains_key("mach1"))
            .unwrap_or(false),
        ClientConfigKind::ClaudeStdio => json
            .get("mcpServers")
            .and_then(|value| value.as_object())
            .map(|servers| servers.contains_key("mach1"))
            .unwrap_or(false),
        ClientConfigKind::Opencode => json
            .get("mcp")
            .and_then(|value| value.as_object())
            .map(|servers| servers.contains_key("mach1"))
            .unwrap_or(false),
        ClientConfigKind::CodexToml => false,
    }
}

fn codex_has_mach1(path: &Path) -> bool {
    std::fs::read_to_string(path)
        .ok()
        .and_then(|content| content.parse::<toml::Value>().ok())
        .and_then(|doc| doc.get("mcp_servers").cloned())
        .and_then(|servers| servers.get("mach1").cloned())
        .is_some()
}

#[tauri::command]
fn get_client_connections(app: tauri::AppHandle) -> Result<Vec<ClientConnectionState>, String> {
    let client_ids = [
        "vscode",
        "cursor",
        "claude",
        "claudecode",
        "codex",
        "windsurf",
        "opencode",
        "antigravity",
    ];

    let states = client_ids
        .into_iter()
        .map(|client_id| match resolve_client_target(&app, client_id) {
            Ok(target) => {
                let connected = if matches!(target.kind, ClientConfigKind::CodexToml) {
                    codex_has_mach1(&target.path)
                } else {
                    let json = read_json_config(&target.path);
                    client_has_mach1(&target, &json)
                };
                ClientConnectionState {
                    id: client_id.to_string(),
                    connected,
                    config_path: Some(target.path.to_string_lossy().to_string()),
                }
            }
            Err(_) => ClientConnectionState {
                id: client_id.to_string(),
                connected: false,
                config_path: None,
            },
        })
        .collect();

    Ok(states)
}

fn persist_mach1_http_token(token: &str) -> Result<(), String> {
    std::env::set_var("MACH1_HTTP_TOKEN", token);

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        const CREATE_NO_WINDOW: u32 = 0x08000000;
        let status = std::process::Command::new("setx")
            .arg("MACH1_HTTP_TOKEN")
            .arg(token)
            .stdout(std::process::Stdio::null())
            .stderr(std::process::Stdio::null())
            .creation_flags(CREATE_NO_WINDOW)
            .status()
            .map_err(|e| format!("failed to persist MACH1_HTTP_TOKEN: {e}"))?;
        if !status.success() {
            return Err("failed to persist MACH1_HTTP_TOKEN for Codex".to_string());
        }
    }

    Ok(())
}

#[tauri::command]
fn patch_client_config(
    app: tauri::AppHandle,
    state: State<AppState>,
    client_id: String,
) -> Result<String, String> {
    use std::fs;

    // Ensure mach1 daemon is running before configuring client
    if !state.daemon.is_running() {
        if let Err(e) = state.daemon.start(DAEMON_PORT) {
            eprintln!("warning: failed to auto-start mach1 daemon: {e}");
        }
    }

    let target = resolve_client_target(&app, &client_id)?;

    if !target.path.exists() {
        if let Some(parent) = target.path.parent() {
            fs::create_dir_all(parent).map_err(|e| e.to_string())?;
        }
        let initial = match target.kind {
            ClientConfigKind::VscodeMcpJson => "{\"servers\": {}}",
            ClientConfigKind::McpServers => "{\"mcpServers\": {}}",
            ClientConfigKind::ClaudeStdio => "{\"mcpServers\": {}}",
            ClientConfigKind::Opencode => {
                "{\"$schema\":\"https://opencode.ai/config.json\",\"mcp\": {}}"
            }
            ClientConfigKind::CodexToml => "",
        };
        if !initial.is_empty() {
            fs::write(&target.path, initial).map_err(|e| e.to_string())?;
        }
    }

    let daemon_url = format!("http://127.0.0.1:{}", DAEMON_PORT);
    let http_token = state.daemon.http_token().to_string();

    // ---- Codex (TOML) ----
    if matches!(target.kind, ClientConfigKind::CodexToml) {
        let mut doc: toml::Value = fs::read_to_string(&target.path)
            .ok()
            .and_then(|c| c.parse().ok())
            .unwrap_or_else(|| {
                let mut table = toml::map::Map::new();
                table.insert(
                    "mcp_servers".to_string(),
                    toml::Value::Table(toml::map::Map::new()),
                );
                toml::Value::Table(table)
            });

        let root = doc
            .as_table_mut()
            .ok_or("Invalid Codex config: root must be a TOML table")?;

        if !root
            .get("mcp_servers")
            .is_some_and(|value| value.is_table())
        {
            root.insert(
                "mcp_servers".to_string(),
                toml::Value::Table(toml::map::Map::new()),
            );
        }

        let servers = root
            .get_mut("mcp_servers")
            .and_then(|value| value.as_table_mut())
            .ok_or("Invalid Codex config: could not create mcp_servers table")?;

        let mut mach1_table = toml::map::Map::new();
        mach1_table.insert(
            "url".to_string(),
            toml::Value::String(format!("{}/mcp", daemon_url)),
        );
        mach1_table.insert(
            "bearer_token_env_var".to_string(),
            toml::Value::String("MACH1_HTTP_TOKEN".to_string()),
        );
        persist_mach1_http_token(&http_token)?;
        servers.insert("mach1".to_string(), toml::Value::Table(mach1_table));

        fs::write(&target.path, doc.to_string()).map_err(|e| e.to_string())?;
        return Ok(target.path.to_string_lossy().to_string());
    }

    // ---- JSON clients ----
    // All clients connect to the single mach1 daemon via HTTP.
    // For clients that support MCP streamable HTTP, we use url+type:http.
    // Claude Desktop only supports stdio, so it gets a spawn fallback.
    let content = fs::read_to_string(&target.path).unwrap_or_else(|_| "{}".to_string());
    let mut json: serde_json::Value =
        serde_json::from_str(&content).unwrap_or(serde_json::json!({}));
    let headers = serde_json::json!({
        "Authorization": format!("Bearer {}", http_token)
    });

    let http_entry = serde_json::json!({
        "type": "http",
        "url": format!("{}/mcp", daemon_url),
        "headers": headers.clone()
    });

    let cursor_entry = serde_json::json!({
        "url": format!("{}/mcp", daemon_url),
        "headers": headers.clone()
    });

    let windsurf_entry = serde_json::json!({
        "serverUrl": format!("{}/mcp", daemon_url),
        "headers": headers.clone()
    });

    let antigravity_entry = serde_json::json!({
        "serverUrl": format!("{}/mcp", daemon_url),
        "headers": headers.clone()
    });

    let opencode_entry = serde_json::json!({
        "type": "remote",
        "url": format!("{}/mcp", daemon_url),
        "headers": headers.clone(),
        "enabled": true
    });

    // Stdio fallback for Claude Desktop (doesn't support HTTP)
    let root_dir = mach1_root_dir(&app)?;
    let db_path = root_dir.join("registry.db");
    let bin_path = ensure_router_binary(&app)?;
    let bin = bin_path.to_string_lossy().to_string();
    let db = db_path.to_string_lossy().to_string();

    let plain_entry = serde_json::json!({
        "command": &bin,
        "args": ["--db", &db]
    });

    match target.kind {
        ClientConfigKind::VscodeMcpJson => {
            if let Some(obj) = json.as_object_mut() {
                if let Some(servers) = obj.get_mut("servers").and_then(|v| v.as_object_mut()) {
                    servers.insert("mach1".to_string(), http_entry);
                } else {
                    let mut servers = serde_json::Map::new();
                    servers.insert("mach1".to_string(), http_entry);
                    obj.insert("servers".to_string(), serde_json::Value::Object(servers));
                }
            }
        }
        ClientConfigKind::McpServers => {
            let entry = match client_id.as_str() {
                "cursor" => cursor_entry,
                "windsurf" => windsurf_entry,
                "antigravity" => antigravity_entry,
                _ => http_entry,
            };
            if let Some(obj) = json.as_object_mut() {
                if let Some(servers) = obj.get_mut("mcpServers").and_then(|v| v.as_object_mut()) {
                    servers.insert("mach1".to_string(), entry.clone());
                } else {
                    let mut servers = serde_json::Map::new();
                    servers.insert("mach1".to_string(), entry);
                    obj.insert("mcpServers".to_string(), serde_json::Value::Object(servers));
                }
            }
        }
        ClientConfigKind::ClaudeStdio => {
            if let Some(obj) = json.as_object_mut() {
                if let Some(servers) = obj.get_mut("mcpServers").and_then(|v| v.as_object_mut()) {
                    servers.insert("mach1".to_string(), plain_entry);
                } else {
                    let mut servers = serde_json::Map::new();
                    servers.insert("mach1".to_string(), plain_entry);
                    obj.insert("mcpServers".to_string(), serde_json::Value::Object(servers));
                }
            }
        }
        ClientConfigKind::Opencode => {
            if let Some(obj) = json.as_object_mut() {
                if let Some(servers) = obj.get_mut("mcp").and_then(|v| v.as_object_mut()) {
                    servers.insert("mach1".to_string(), opencode_entry);
                } else {
                    let mut servers = serde_json::Map::new();
                    servers.insert("mach1".to_string(), opencode_entry);
                    obj.insert("mcp".to_string(), serde_json::Value::Object(servers));
                }
            }
        }
        ClientConfigKind::CodexToml => unreachable!(),
    }

    fs::write(
        &target.path,
        serde_json::to_string_pretty(&json).unwrap_or_default(),
    )
    .map_err(|e| e.to_string())?;

    // Inject 1MCP system directive into client's rule file
    inject_rules(&app, &client_id);

    // For VSCode, also inject instructions reference into user settings.json
    if client_id == "vscode" {
        patch_vscode_settings(&app);
    }

    Ok(target.path.to_string_lossy().to_string())
}

#[tauri::command]
fn remove_client_config(app: tauri::AppHandle, client_id: String) -> Result<bool, String> {
    let target = resolve_client_target(&app, &client_id)?;
    if !target.path.exists() {
        return Ok(false);
    }

    // ---- Codex (TOML) ----
    if matches!(target.kind, ClientConfigKind::CodexToml) {
        let mut doc: toml::Value = std::fs::read_to_string(&target.path)
            .map_err(|e| e.to_string())?
            .parse()
            .map_err(|e: toml::de::Error| e.to_string())?;

        let removed = doc
            .get_mut("mcp_servers")
            .and_then(|v| v.as_table_mut())
            .and_then(|servers| servers.remove("mach1"))
            .is_some();

        if !removed {
            return Ok(false);
        }

        std::fs::write(&target.path, doc.to_string()).map_err(|e| e.to_string())?;
        remove_rules(&app, &client_id);
        if client_id == "vscode" {
            unpatch_vscode_settings(&app);
        }
        return Ok(true);
    }

    // ---- JSON clients ----
    let mut json = read_json_config(&target.path);
    let removed = match target.kind {
        ClientConfigKind::VscodeMcpJson => json
            .get_mut("servers")
            .and_then(|v| v.as_object_mut())
            .and_then(|servers| servers.remove("mach1"))
            .is_some(),
        ClientConfigKind::McpServers => json
            .get_mut("mcpServers")
            .and_then(|v| v.as_object_mut())
            .and_then(|servers| servers.remove("mach1"))
            .is_some(),
        ClientConfigKind::ClaudeStdio => json
            .get_mut("mcpServers")
            .and_then(|v| v.as_object_mut())
            .and_then(|servers| servers.remove("mach1"))
            .is_some(),
        ClientConfigKind::Opencode => json
            .get_mut("mcp")
            .and_then(|v| v.as_object_mut())
            .and_then(|servers| servers.remove("mach1"))
            .is_some(),
        ClientConfigKind::CodexToml => unreachable!(),
    };

    if !removed {
        return Ok(false);
    }

    std::fs::write(
        &target.path,
        serde_json::to_string_pretty(&json).unwrap_or_default(),
    )
    .map_err(|e| e.to_string())?;

    remove_rules(&app, &client_id);
    if client_id == "vscode" {
        unpatch_vscode_settings(&app);
    }

    Ok(true)
}

fn router_binary_name() -> &'static str {
    if cfg!(target_os = "windows") {
        "mach1.exe"
    } else {
        "mach1"
    }
}

fn maybe_copy_file(from: &Path, to: &Path) -> Result<(), String> {
    if let Some(parent) = to.parent() {
        std::fs::create_dir_all(parent).map_err(|e| e.to_string())?;
    }

    // If source and dest are the same file (same inode / already staged), skip.
    if from == to {
        return Ok(());
    }

    // If destination already exists and has the same size, assume it's up-to-date.
    // This avoids os error 32 (file locked) when the source is in use.
    if to.exists() {
        if let (Ok(src_meta), Ok(dst_meta)) = (std::fs::metadata(from), std::fs::metadata(to)) {
            if src_meta.len() == dst_meta.len() {
                return Ok(());
            }
        }
        // Try to remove the old file first. If it's locked (in use), the rename trick
        // below will still work because rename is atomic on the same filesystem.
        let _ = std::fs::remove_file(to);
    }

    // Write to a temp file first, then atomically rename. This avoids os error 32
    // when the destination might be locked by another process, and avoids partial
    // writes if we crash mid-copy.
    let tmp_path = to.with_extension("tmp");
    std::fs::copy(from, &tmp_path)
        .map_err(|e| format!("copy {} -> {}: {e}", from.display(), tmp_path.display()))?;
    std::fs::rename(&tmp_path, to).map_err(|e| {
        let _ = std::fs::remove_file(&tmp_path);
        format!("rename {} -> {}: {e}", tmp_path.display(), to.display())
    })?;
    Ok(())
}

fn ensure_router_binary(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let staged_dir = mach1_root_dir(app)?.join("bin");
    let staged_path = staged_dir.join(router_binary_name());

    // Get our own exe path so we never accidentally stage the Tauri GUI binary
    // as the router. On Windows the installed layout is:
    //   appdir/mach1.exe           ← Tauri GUI (MUST NOT be used as MCP server)
    //   appdir/resources/mach1.exe ← Go router binary (the one we want)
    let own_exe = std::env::current_exe().ok();

    let mut candidates = Vec::new();

    // 1. Check resources subdirectory FIRST — this is where the actual Go router
    //    binary lives in production installs. resource_dir/ itself contains the
    //    Tauri app binary on some platforms, so we only check the resources/ subfolder.
    if let Ok(resource_dir) = app.path().resource_dir() {
        candidates.push(resource_dir.join("resources").join(router_binary_name()));
    }
    if let Some(exe_path) = &own_exe {
        if let Some(parent) = exe_path.parent() {
            candidates.push(parent.join("resources").join(router_binary_name()));
        }
    }

    // 2. Dev fallback — relative to CWD (for cargo tauri dev)
    if let Ok(current_dir) = std::env::current_dir() {
        candidates.push(current_dir.join("../../bin").join(router_binary_name()));
        candidates.push(current_dir.join("../../../bin").join(router_binary_name()));
    }

    // 3. Resource dir root (lower priority — might be the Tauri app itself)
    if let Ok(resource_dir) = app.path().resource_dir() {
        candidates.push(resource_dir.join(router_binary_name()));
    }

    for candidate in candidates.iter() {
        if candidate == &staged_path {
            continue;
        }
        if !candidate.exists() {
            continue;
        }
        // Never stage our own exe as the router — it would open a GUI window
        // when an MCP client tries to spawn the router over stdio.
        if let Some(own) = &own_exe {
            if candidate == own {
                continue;
            }
        }
        maybe_copy_file(candidate, &staged_path)?;
        return Ok(staged_path);
    }

    if staged_path.exists() {
        return Ok(staged_path);
    }

    Err("mach1 binary was not bundled with the app. Reinstall 1mcp.in or configure the client manually.".to_string())
}

fn find_cli_binary(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let cli_name = if cfg!(target_os = "windows") { "mach1ctl.exe" } else { "mach1ctl" };

    let mut candidates = Vec::new();

    // Check resources subdirectory (where all Go binaries are bundled)
    if let Ok(resource_dir) = app.path().resource_dir() {
        candidates.push(resource_dir.join("resources").join(cli_name));
    }
    if let Some(exe_path) = std::env::current_exe().ok() {
        if let Some(parent) = exe_path.parent() {
            candidates.push(parent.join("resources").join(cli_name));
        }
    }

    // Windows: check beside the binary itself (NSIS installs resources at exe level)
    if cfg!(target_os = "windows") {
        if let Some(exe_path) = std::env::current_exe().ok() {
            if let Some(parent) = exe_path.parent() {
                candidates.push(parent.join(cli_name));
            }
        }
    }

    // Dev fallback — relative to CWD
    if let Ok(current_dir) = std::env::current_dir() {
        candidates.push(current_dir.join("../../bin").join(cli_name));
        candidates.push(current_dir.join("../../../bin").join(cli_name));
    }

    for candidate in &candidates {
        if candidate.exists() {
            return Ok(candidate.clone());
        }
    }

    Err("mach1ctl binary not found".to_string())
}

fn inject_rules(app: &tauri::AppHandle, client_id: &str) {
    let cli_path = match find_cli_binary(app) {
        Ok(p) => p,
        Err(e) => {
            eprintln!("inject_rules: {e}");
            return;
        }
    };

    let output = std::process::Command::new(&cli_path)
        .arg("inject-rules")
        .arg(client_id)
        .output();

    match output {
        Ok(out) => {
            if !out.status.success() {
                let stderr = String::from_utf8_lossy(&out.stderr);
                let stdout = String::from_utf8_lossy(&out.stdout);
                eprintln!("inject_rules failed (exit={:?}): {stdout}{stderr}", out.status.code());
            }
        }
        Err(e) => {
            eprintln!("inject_rules: cannot spawn {cli_path:?}: {e}");
        }
    }
}

fn remove_rules(app: &tauri::AppHandle, client_id: &str) {
    let cli_path = match find_cli_binary(app) {
        Ok(p) => p,
        Err(e) => {
            eprintln!("remove_rules: {e}");
            return;
        }
    };

    let output = std::process::Command::new(&cli_path)
        .arg("remove-rules")
        .arg(client_id)
        .output();

    match output {
        Ok(out) => {
            if !out.status.success() {
                let stderr = String::from_utf8_lossy(&out.stderr);
                let stdout = String::from_utf8_lossy(&out.stdout);
                eprintln!("remove_rules failed (exit={:?}): {stdout}{stderr}", out.status.code());
            }
        }
        Err(e) => {
            eprintln!("remove_rules: cannot spawn {cli_path:?}: {e}");
        }
    }
}

fn vscode_settings_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let config_dir = app
        .path()
        .config_dir()
        .map_err(|_| "Failed to resolve config directory".to_string())?;
    Ok(config_dir.join("Code").join("User").join("settings.json"))
}

fn vscode_instructions_path(app: &tauri::AppHandle) -> PathBuf {
    let home = app.path().home_dir().unwrap_or_else(|_| PathBuf::from("."));
    home.join(".copilot").join("instructions").join("copilot-instructions.md")
}

fn patch_vscode_settings(app: &tauri::AppHandle) {
    let settings_path = match vscode_settings_path(app) {
        Ok(p) => p,
        Err(e) => {
            eprintln!("patch_vscode_settings: {e}");
            return;
        }
    };

    let ins_path = vscode_instructions_path(app);
    let ins_path_str = ins_path.to_string_lossy().to_string();

    let mut settings: serde_json::Value = std::fs::read_to_string(&settings_path)
        .ok()
        .and_then(|c| serde_json::from_str(&c).ok())
        .unwrap_or(serde_json::json!({}));

    let instruction_entry = serde_json::json!({"file": ins_path_str});

    if let Some(obj) = settings.as_object_mut() {
        obj.insert(
            "github.copilot.chat.codeGeneration.instructions".to_string(),
            serde_json::json!([instruction_entry]),
        );
    }

    if let Some(parent) = settings_path.parent() {
        if let Err(e) = std::fs::create_dir_all(parent) {
            eprintln!("patch_vscode_settings: create dir: {e}");
            return;
        }
    }

    if let Err(e) = std::fs::write(&settings_path, serde_json::to_string_pretty(&settings).unwrap_or_default()) {
        eprintln!("patch_vscode_settings: write: {e}");
    }
}

fn unpatch_vscode_settings(app: &tauri::AppHandle) {
    let settings_path = match vscode_settings_path(app) {
        Ok(p) => p,
        Err(e) => {
            eprintln!("unpatch_vscode_settings: {e}");
            return;
        }
    };

    let mut settings: serde_json::Value = std::fs::read_to_string(&settings_path)
        .ok()
        .and_then(|c| serde_json::from_str(&c).ok())
        .unwrap_or(serde_json::json!({}));

    if let Some(obj) = settings.as_object_mut() {
        obj.remove("github.copilot.chat.codeGeneration.instructions");
    }

    if let Err(e) = std::fs::write(&settings_path, serde_json::to_string_pretty(&settings).unwrap_or_default()) {
        eprintln!("unpatch_vscode_settings: write: {e}");
    }
}
