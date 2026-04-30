use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::sync::Mutex;
use std::time::{Instant, SystemTime, UNIX_EPOCH};

pub struct DaemonManager {
    inner: Mutex<Option<std::process::Child>>,
    root_dir: PathBuf,
    bin_path: PathBuf,
    port: std::sync::atomic::AtomicU16,
    http_token: String,
    start_time: Mutex<Option<Instant>>,
    start_timestamp: Mutex<Option<u64>>,
}

impl DaemonManager {
    pub fn new(root_dir: PathBuf, bin_path: PathBuf) -> Result<Self, String> {
        let http_token = load_or_create_http_token(&root_dir)?;
        Ok(Self {
            inner: Mutex::new(None),
            root_dir,
            bin_path,
            port: std::sync::atomic::AtomicU16::new(3200),
            http_token,
            start_time: Mutex::new(None),
            start_timestamp: Mutex::new(None),
        })
    }

    pub fn start(&self, port: u16) -> Result<String, String> {
        let mut guard = self.inner.lock().map_err(|e| e.to_string())?;

        if let Some(ref mut child) = *guard {
            match child.try_wait() {
                Ok(None) => return Ok(format!("mach1 router is already running on port {}", port)),
                Ok(Some(_)) => { /* process exited, restart */ }
                Err(_) => { /* process inaccessible, restart */ }
            }
        }

        let db_path = self.root_dir.join("registry.db");

        let mut cmd = Command::new(&self.bin_path);
        cmd.arg("--db")
            .arg(&db_path)
            .arg("--transport")
            .arg("http")
            .arg("--listen")
            .arg(format!("127.0.0.1:{}", port))
            .arg("--http-token")
            .arg(&self.http_token)
            .arg("--log")
            .arg("info");

        // On Windows, suppress the console window that would otherwise appear.
        #[cfg(target_os = "windows")]
        {
            use std::os::windows::process::CommandExt;
            const CREATE_NO_WINDOW: u32 = 0x08000000;
            cmd.creation_flags(CREATE_NO_WINDOW);
        }

        cmd.stdout(Stdio::null()).stderr(Stdio::null());

        let child = cmd
            .spawn()
            .map_err(|e| format!("Failed to start mach1 router: {e}"))?;

        self.port.store(port, std::sync::atomic::Ordering::SeqCst);
        *guard = Some(child);
        *self.start_time.lock().map_err(|e| e.to_string())? = Some(Instant::now());
        *self.start_timestamp.lock().map_err(|e| e.to_string())? = Some(
            SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs(),
        );

        Ok(format!("mach1 router started on 127.0.0.1:{}", port))
    }

    pub fn stop(&self) -> Result<String, String> {
        let mut guard = self.inner.lock().map_err(|e| e.to_string())?;

        if let Some(mut child) = guard.take() {
            let _ = child.kill();
            let _ = child.wait();
            Ok("mach1 router stopped".to_string())
        } else {
            Ok("mach1 router is not running".to_string())
        }
    }

    pub fn status(&self) -> String {
        if self.is_running() {
            format!(
                "running on 127.0.0.1:{}",
                self.port.load(std::sync::atomic::Ordering::SeqCst)
            )
        } else {
            "stopped".to_string()
        }
    }

    pub fn base_url(&self) -> String {
        format!(
            "http://127.0.0.1:{}",
            self.port.load(std::sync::atomic::Ordering::SeqCst)
        )
    }

    pub fn http_token(&self) -> &str {
        &self.http_token
    }

    pub fn uptime_seconds(&self) -> u64 {
        self.start_time
            .lock()
            .ok()
            .and_then(|t| t.map(|s| s.elapsed().as_secs()))
            .unwrap_or(0)
    }

    pub fn start_timestamp(&self) -> Option<u64> {
        self.start_timestamp.lock().ok().and_then(|t| *t)
    }

    pub fn is_running(&self) -> bool {
        let url = format!("{}/health", self.base_url());
        reqwest::blocking::Client::new()
            .get(&url)
            .timeout(std::time::Duration::from_millis(500))
            .send()
            .map(|r| r.status().is_success())
            .unwrap_or(false)
    }
}

fn load_or_create_http_token(root_dir: &PathBuf) -> Result<String, String> {
    use rand::rngs::OsRng;
    use rand::RngCore;

    std::fs::create_dir_all(root_dir).map_err(|e| e.to_string())?;
    let path = root_dir.join("http-token");
    if path.exists() {
        let token = std::fs::read_to_string(&path)
            .map_err(|e| format!("failed to read mach1 HTTP token: {e}"))?
            .trim()
            .to_string();
        if token.len() >= 32 {
            return Ok(token);
        }
    }

    let mut bytes = [0u8; 32];
    OsRng.fill_bytes(&mut bytes);
    let token = bytes
        .iter()
        .map(|byte| format!("{:02x}", byte))
        .collect::<String>();
    std::fs::write(&path, &token).map_err(|e| format!("failed to write mach1 HTTP token: {e}"))?;
    Ok(token)
}

impl Drop for DaemonManager {
    fn drop(&mut self) {
        if let Ok(mut guard) = self.inner.lock() {
            if let Some(mut child) = guard.take() {
                let _ = child.kill();
                let _ = child.wait();
            }
        }
    }
}
