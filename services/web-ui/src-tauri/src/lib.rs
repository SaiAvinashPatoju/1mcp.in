mod db;

use db::{Db, InstalledMcp, MarketplaceItem};
use reqwest::Response;
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::Duration;
use tauri::{Emitter, Manager, State};
use tauri_plugin_notification::NotificationExt;

struct AppState {
    db: Mutex<Db>,
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

#[derive(Copy, Clone)]
enum ClientConfigKind {
    VscodeMcpJson,
    McpServers,
    Opencode,
}

struct ClientConfigTarget {
    path: PathBuf,
    kind: ClientConfigKind,
}

fn cloud_api_url() -> &'static str {
    option_env!("ONEMCP_API_URL").unwrap_or(if cfg!(debug_assertions) {
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
async fn auth_register(name: String, email: String, password: String) -> Result<AuthResponse, String> {
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
fn set_token(state: State<AppState>, mcp_id: String, key: String, value: String) -> Result<(), String> {
    state.db.lock().map_err(|e| e.to_string())?.set_token(&mcp_id, &key, &value)
}

#[tauri::command]
fn get_user_count(state: State<AppState>) -> Result<u64, String> {
    state.db.lock().map_err(|e| e.to_string())?.get_user_count()
}

#[tauri::command]
fn increment_user_count(state: State<AppState>) -> Result<u64, String> {
    state.db.lock().map_err(|e| e.to_string())?.increment_user_count()
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

            if !interactive && read_staged_update_version(&app).as_deref() == Some(version.as_str()) {
                let _ = app.emit("update-none", ());
                return;
            }

            let _ = app
                .notification()
                .builder()
                .title("1mcp.in Update Available")
                .body(format!("Version {} is downloading in the background…", version))
                .show();

            let _ = app.emit("update-downloading", &version);

            match update.download_and_install(|_chunk, _total| {}, || {}).await {
                Ok(_) => {
                    write_staged_update_version(&app, &version);

                    let _ = app
                        .notification()
                        .builder()
                        .title("1mcp.in Ready to Restart")
                        .body(format!("v{} is installed. Click Restart in the app to apply.", version))
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
fn list_marketplace(state: State<AppState>) -> Result<Vec<MarketplaceItem>, String> {
    state.db.lock().map_err(|e| e.to_string())?.list_marketplace()
}

#[tauri::command]
fn sync_marketplace(state: State<AppState>, items: Vec<MarketplaceItem>) -> Result<(), String> {
    state.db.lock().map_err(|e| e.to_string())?.sync_marketplace(&items)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            Some(vec![]),
        ))
        .setup(|app| {
            let data_dir = app
                .path()
                .app_data_dir()
                .expect("failed to resolve app data dir");
            let db = Db::open(&data_dir).expect("failed to open database");
            app.manage(AppState { db: Mutex::new(db) });

            let handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                tokio::time::sleep(std::time::Duration::from_secs(5)).await;
                run_update_check(handle.clone(), false).await;

                let mut interval = tokio::time::interval(std::time::Duration::from_secs(4 * 60 * 60));
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
            fetch_cloud_stats,
            fetch_cloud_marketplace,
            list_installed,
            install_mcp,
            uninstall_mcp,
            toggle_mcp,
            set_token,
            get_user_count,
            increment_user_count,
            check_update,
            restart_app,
            patch_client_config,
            list_marketplace,
            sync_marketplace,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

#[tauri::command]
fn patch_client_config(app: tauri::AppHandle, client_id: String) -> Result<String, String> {
    use std::fs;

    let config_dir = app
        .path()
        .config_dir()
        .map_err(|_| "Failed to resolve config directory".to_string())?;
    let home_dir = app
        .path()
        .home_dir()
        .map_err(|_| "Failed to resolve home directory".to_string())?;

    let target = match client_id.as_str() {
        "vscode" => ClientConfigTarget {
            path: config_dir.join("Code").join("User").join("mcp.json"),
            kind: ClientConfigKind::VscodeMcpJson,
        },
        "cursor" => ClientConfigTarget {
            path: home_dir.join(".cursor").join("mcp.json"),
            kind: ClientConfigKind::McpServers,
        },
        "claude" => ClientConfigTarget {
            path: config_dir.join("Claude").join("claude_desktop_config.json"),
            kind: ClientConfigKind::McpServers,
        },
        "claudecode" => ClientConfigTarget {
            path: home_dir.join(".claude.json"),
            kind: ClientConfigKind::McpServers,
        },
        "windsurf" => ClientConfigTarget {
            path: home_dir.join(".codeium").join("windsurf").join("mcp_config.json"),
            kind: ClientConfigKind::McpServers,
        },
        "codex" => ClientConfigTarget {
            path: home_dir.join(".codex").join("mcp.json"),
            kind: ClientConfigKind::McpServers,
        },
        "opencode" => ClientConfigTarget {
            path: home_dir.join(".config").join("opencode").join("opencode.json"),
            kind: ClientConfigKind::Opencode,
        },
        _ => {
            return Err(format!(
                "Automated setup for '{}' is not yet supported. Please add the 1mcp server config manually.",
                client_id
            ))
        }
    };

    if !target.path.exists() {
        if let Some(parent) = target.path.parent() {
            fs::create_dir_all(parent).unwrap_or_default();
        }
        let initial = match target.kind {
            ClientConfigKind::VscodeMcpJson => "{\"servers\": {}}",
            ClientConfigKind::McpServers => "{\"mcpServers\": {}}",
            ClientConfigKind::Opencode => "{\"$schema\":\"https://opencode.ai/config.json\",\"mcp\": {}}",
        };
        fs::write(&target.path, initial).unwrap_or_default();
    }

    let content = fs::read_to_string(&target.path).unwrap_or_else(|_| "{}".to_string());
    let mut json: serde_json::Value = serde_json::from_str(&content).unwrap_or(serde_json::json!({}));

    let data_dir = app
        .path()
        .app_data_dir()
        .map_err(|_| "Failed to resolve app data dir".to_string())?;
    let db_path = data_dir.join("1mcp.db");
    let bin_path = ensure_router_binary(&app)?;

    let mcp_entry = serde_json::json!({
        "command": bin_path.to_string_lossy().to_string(),
        "args": ["--db", db_path.to_string_lossy().to_string()]
    });

    let opencode_entry = serde_json::json!({
        "type": "local",
        "command": [
            bin_path.to_string_lossy().to_string(),
            "--db",
            db_path.to_string_lossy().to_string()
        ],
        "enabled": true
    });

    match target.kind {
        ClientConfigKind::VscodeMcpJson => {
            if let Some(obj) = json.as_object_mut() {
                if let Some(servers) = obj.get_mut("servers").and_then(|value| value.as_object_mut()) {
                    servers.insert("1mcp".to_string(), mcp_entry);
                } else {
                    let mut servers = serde_json::Map::new();
                    servers.insert("1mcp".to_string(), mcp_entry);
                    obj.insert("servers".to_string(), serde_json::Value::Object(servers));
                }
            }
        }
        ClientConfigKind::McpServers => {
            if let Some(obj) = json.as_object_mut() {
                if let Some(servers) = obj.get_mut("mcpServers").and_then(|value| value.as_object_mut()) {
                    servers.insert("1mcp".to_string(), mcp_entry);
                } else {
                    let mut servers = serde_json::Map::new();
                    servers.insert("1mcp".to_string(), mcp_entry);
                    obj.insert("mcpServers".to_string(), serde_json::Value::Object(servers));
                }
            }
        }
        ClientConfigKind::Opencode => {
            if let Some(obj) = json.as_object_mut() {
                if let Some(servers) = obj.get_mut("mcp").and_then(|value| value.as_object_mut()) {
                    servers.insert("1mcp".to_string(), opencode_entry);
                } else {
                    let mut servers = serde_json::Map::new();
                    servers.insert("1mcp".to_string(), opencode_entry);
                    obj.insert("mcp".to_string(), serde_json::Value::Object(servers));
                }
            }
        }
    }

    fs::write(
        &target.path,
        serde_json::to_string_pretty(&json).unwrap_or_default(),
    )
    .map_err(|e| e.to_string())?;

    Ok(target.path.to_string_lossy().to_string())
}

fn router_binary_name() -> &'static str {
    if cfg!(target_os = "windows") {
        "centralmcpd.exe"
    } else {
        "centralmcpd"
    }
}

fn maybe_copy_file(from: &Path, to: &Path) -> Result<(), String> {
    if let Some(parent) = to.parent() {
        std::fs::create_dir_all(parent).map_err(|e| e.to_string())?;
    }
    std::fs::copy(from, to)
        .map_err(|e| format!("copy {} -> {}: {e}", from.display(), to.display()))?;
    Ok(())
}

fn ensure_router_binary(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let staged_dir = app
        .path()
        .app_data_dir()
        .map_err(|_| "Failed to resolve app data dir".to_string())?
        .join("bin");
    let staged_path = staged_dir.join(router_binary_name());

    let mut candidates = Vec::new();

    // 1. Check the staged directory first (most reliable for already-extracted binaries)
    candidates.push(staged_path.clone());

    // 2. Check resource_dir (bundled with app)
    if let Ok(resource_dir) = app.path().resource_dir() {
        candidates.push(resource_dir.join(router_binary_name()));
        // Also check for resources subdirectory
        candidates.push(resource_dir.join("resources").join(router_binary_name()));
    }

    // 3. Check alongside the executable
    if let Ok(exe_path) = std::env::current_exe() {
        if let Some(parent) = exe_path.parent() {
            candidates.push(parent.join(router_binary_name()));
            // Also check in resources subfolder next to exe
            candidates.push(parent.join("resources").join(router_binary_name()));
        }
    }

    // 4. Check relative to current directory (dev fallback)
    if let Ok(current_dir) = std::env::current_dir() {
        candidates.push(current_dir.join("../../bin").join(router_binary_name()));
    }

    for candidate in candidates.iter().filter(|c| c != &&staged_path) {
        if candidate.exists() {
            maybe_copy_file(candidate, &staged_path)?;
            return Ok(staged_path);
        }
    }

    if staged_path.exists() {
        return Ok(staged_path);
    }

    Err("centralmcpd binary was not bundled with the app. Reinstall 1mcp.in or configure the client manually.".to_string())
}
