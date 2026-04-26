mod db;

use db::{Db, InstalledMcp};
use std::sync::Mutex;
use tauri::State;
use tauri_plugin_notification::NotificationExt;

struct AppState {
    db: Mutex<Db>,
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

            // OS notification
            let _ = app
                .notification()
                .builder()
                .title("Mach1 Update Available")
                .body(format!("Version {} is downloading…", version))
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
                        .title("Mach1 Ready to Restart")
                        .body(format!("Version {} downloaded. Click Restart to apply.", version))
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

            // Check for updates ~5 s after startup (non-blocking)
            let handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                tokio::time::sleep(std::time::Duration::from_secs(5)).await;
                run_update_check(handle).await;
            });

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            list_installed,
            install_mcp,
            uninstall_mcp,
            toggle_mcp,
            set_token,
            get_user_count,
            increment_user_count,
            check_update,
            restart_app,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
