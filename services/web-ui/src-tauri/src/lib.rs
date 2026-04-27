mod db;

use db::{Db, InstalledMcp, MarketplaceItem};
use reqwest::Response;
use serde::{Deserialize, Serialize};
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

// ──────────────────────────────────────────────
// DB commands
// ──────────────────────────────────────────────

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

// ──────────────────────────────────────────────
// Updater — called from frontend "check for updates"
// ──────────────────────────────────────────────

#[tauri::command]
async fn check_update(app: tauri::AppHandle) -> Result<(), String> {
    spawn_update_check(app);
    Ok(())
}

fn spawn_update_check(app: tauri::AppHandle) {
    tauri::async_runtime::spawn(async move {
        run_update_check(app).await;
    });
}

async fn run_update_check(app: tauri::AppHandle) {
    use tauri_plugin_updater::UpdaterExt;

    let updater = match app.updater() {
        Ok(u) => u,
        Err(e) => {
            eprintln!("updater init failed: {e}");
            return;
        }
    };

    match updater.check().await {
        Ok(Some(update)) => {
            let version = update.version.clone();

            // OS notification — update available
            let _ = app
                .notification()
                .builder()
                .title("1mcp.in Update Available")
                .body(format!("Version {} is downloading in the background…", version))
                .show();

            // Tell the UI a download is starting
            let _ = app.emit("update-downloading", &version);

            // Download and stage the update
            match update.download_and_install(|_chunk, _total| {}, || {}).await {
                Ok(_) => {
                    // OS notification — ready to restart
                    let _ = app
                        .notification()
                        .builder()
                        .title("1mcp.in Ready to Restart")
                        .body(format!("v{} is installed. Click Restart in the app to apply.", version))
                        .show();

                    // Tell the UI to show the "Restart" banner
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

// Called by the frontend "Restart" button
#[tauri::command]
fn restart_app(app: tauri::AppHandle) {
    app.restart();
}

// ──────────────────────────────────────────────
// Marketplace commands
// ──────────────────────────────────────────────

#[tauri::command]
fn list_marketplace(state: State<AppState>) -> Result<Vec<MarketplaceItem>, String> {
    state.db.lock().map_err(|e| e.to_string())?.list_marketplace()
}

/// Sync a batch of marketplace items received from the cloud API into local SQLite.
#[tauri::command]
fn sync_marketplace(state: State<AppState>, items: Vec<MarketplaceItem>) -> Result<(), String> {
    state.db.lock().map_err(|e| e.to_string())?.sync_marketplace(&items)
}

// ──────────────────────────────────────────────
// App bootstrap
// ──────────────────────────────────────────────

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

            // Check for updates ~5 s after startup, then every 4 hours
            let handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                tokio::time::sleep(std::time::Duration::from_secs(5)).await;
                run_update_check(handle.clone()).await;

                let mut interval = tokio::time::interval(std::time::Duration::from_secs(4 * 60 * 60));
                interval.tick().await; // consume the first immediate tick
                loop {
                    interval.tick().await;
                    run_update_check(handle.clone()).await;
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
    use tauri::Manager;

    let config_dir = app.path().config_dir().map_err(|_| "Failed to resolve config directory".to_string())?;
    let home_dir = app.path().home_dir().map_err(|_| "Failed to resolve home directory".to_string())?;
    
    // Resolve the correct config file for each client
    // See https://modelcontextprotocol.io/quickstart/user for official paths
    let (path, key_name) = match client_id.as_str() {
        "vscode" => {
            // VS Code uses %APPDATA%/Code/User/settings.json (mcp.servers) or .vscode/mcp.json
            // For global setup, we write to the user-level mcp settings
            (config_dir.join("Code").join("User").join("settings.json"), "mcp.servers")
        },
        "cursor" => {
            // Cursor: ~/.cursor/mcp.json
            (home_dir.join(".cursor").join("mcp.json"), "mcpServers")
        },
        "claude" => {
            // Claude Desktop: %APPDATA%/Claude/claude_desktop_config.json
            (config_dir.join("Claude").join("claude_desktop_config.json"), "mcpServers")
        },
        "claudecode" => {
            // Claude Code: ~/.claude.json
            (home_dir.join(".claude.json"), "mcpServers")
        },
        "windsurf" | "codex" => {
            // Windsurf/Codex: ~/.codeium/windsurf/mcp_config.json or similar
            // Codex: ~/.codex/mcp.json
            let p = if client_id == "codex" {
                home_dir.join(".codex").join("mcp.json")
            } else {
                home_dir.join(".codeium").join("windsurf").join("mcp_config.json")
            };
            (p, "mcpServers")
        },
        _ => return Err(format!("Automated setup for '{}' is not yet supported. Please add the 1mcp server config manually.", client_id)),
    };

    if !path.exists() {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).unwrap_or_default();
        }
        if key_name == "mcp.servers" {
            fs::write(&path, "{}").unwrap_or_default();
        } else {
            fs::write(&path, "{\"mcpServers\": {}}").unwrap_or_default();
        }
    }

    let content = fs::read_to_string(&path).unwrap_or_else(|_| "{}".to_string());
    let mut json: serde_json::Value = serde_json::from_str(&content).unwrap_or(serde_json::json!({}));

    // Resolve centralmcpd binary path
    let data_dir = app.path().app_data_dir().map_err(|_| "Failed to resolve app data dir".to_string())?;
    let db_path = data_dir.join("1mcp.db");

    let current_dir = std::env::current_dir().unwrap_or_default();
    let bin_path = current_dir.join("../../bin/centralmcpd.exe");
    let bin_str = if bin_path.exists() {
        bin_path.to_string_lossy().to_string()
    } else {
        "centralmcpd".to_string()
    };

    let mcp_entry = serde_json::json!({
        "command": bin_str,
        "args": ["--db", db_path.to_string_lossy().to_string()]
    });

    if key_name == "mcp.servers" {
        // VS Code settings.json uses "mcp.servers" as a top-level key
        // Format: { "mcp.servers": { "1mcp": { "command": ..., "args": [...] } } }
        if let Some(obj) = json.as_object_mut() {
            if let Some(servers) = obj.get_mut("mcp.servers").and_then(|s| s.as_object_mut()) {
                servers.insert("1mcp".to_string(), mcp_entry);
            } else {
                let mut servers = serde_json::Map::new();
                servers.insert("1mcp".to_string(), mcp_entry);
                obj.insert("mcp.servers".to_string(), serde_json::Value::Object(servers));
            }
        }
    } else {
        // mcpServers format for Claude, Cursor, etc.
        if let Some(obj) = json.as_object_mut() {
            if let Some(servers) = obj.get_mut("mcpServers").and_then(|s| s.as_object_mut()) {
                servers.insert("1mcp".to_string(), mcp_entry);
            } else {
                let mut servers = serde_json::Map::new();
                servers.insert("1mcp".to_string(), mcp_entry);
                obj.insert("mcpServers".to_string(), serde_json::Value::Object(servers));
            }
        }
    }

    fs::write(&path, serde_json::to_string_pretty(&json).unwrap_or_default()).map_err(|e| e.to_string())?;

    Ok(path.to_string_lossy().to_string())
}
