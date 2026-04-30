use rusqlite::{params, Connection};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::path::PathBuf;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct InstalledMcp {
    pub id: String,
    pub name: String,
    pub version: String,
    pub runtime: String,
    pub enabled: bool,
    pub command: String,
    #[serde(default)]
    pub args: Vec<String>,
    #[serde(default)]
    pub env: HashMap<String, String>,
    #[serde(default)]
    pub cwd: String,
    pub description: String,
    #[serde(default)]
    pub manifest_json: String,
    #[serde(default)]
    pub installed_at: i64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct MarketplaceItem {
    pub id: String,
    pub name: String,
    pub description: String,
    pub version: String,
    pub runtime: String,
    #[serde(default)]
    pub transport: String,
    pub tags: Vec<String>,
    pub homepage: String,
    pub license: String,
    #[serde(default)]
    pub verification: String,
    #[serde(default)]
    pub sha256: String,
    #[serde(default)]
    pub signature: String,
    #[serde(default)]
    pub entrypoint: MarketplaceEntrypoint,
}

#[derive(Debug, Serialize, Deserialize, Clone, Default)]
pub struct MarketplaceEntrypoint {
    #[serde(default)]
    pub command: String,
    #[serde(default)]
    pub args: Vec<String>,
    #[serde(default)]
    pub cwd: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ActivityItem {
    pub id: String,
    pub activity_type: String,
    pub message: String,
    pub timestamp: i64,
    pub icon: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Skill {
    pub id: String,
    pub name: String,
    pub description: String,
    pub icon: String,
    #[serde(default)]
    pub mcp_ids: Vec<String>,
    #[serde(default)]
    pub client_ids: Vec<String>,
    pub installed: bool,
    pub enabled: bool,
    #[serde(default)]
    pub created_at: i64,
}

pub struct Db {
    conn: Connection,
}

impl Db {
    pub fn open(root_dir: &PathBuf) -> Result<Self, String> {
        std::fs::create_dir_all(root_dir).map_err(|e| e.to_string())?;
        let path = root_dir.join("registry.db");
        let conn = Connection::open(&path).map_err(|e| e.to_string())?;

        // Enable WAL mode for concurrent readers (mach1 accesses this same DB).
        conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;")
            .map_err(|e| e.to_string())?;

        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS installed_mcps (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                version TEXT NOT NULL,
                runtime TEXT NOT NULL,
                enabled INTEGER NOT NULL DEFAULT 1,
                command TEXT NOT NULL,
                description TEXT NOT NULL DEFAULT '',
                args_json TEXT NOT NULL DEFAULT '[]',
                env_json TEXT NOT NULL DEFAULT '{}',
                cwd TEXT NOT NULL DEFAULT '',
                manifest_json TEXT NOT NULL DEFAULT '{}',
                installed_at INTEGER NOT NULL DEFAULT 0
            );
            CREATE TABLE IF NOT EXISTS marketplace_items (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                description TEXT NOT NULL DEFAULT '',
                version TEXT NOT NULL DEFAULT '0.0.1',
                runtime TEXT NOT NULL DEFAULT 'node',
                transport TEXT NOT NULL DEFAULT 'stdio',
                tags TEXT NOT NULL DEFAULT '[]',
                homepage TEXT NOT NULL DEFAULT '',
                license TEXT NOT NULL DEFAULT 'MIT',
                verification TEXT NOT NULL DEFAULT 'community',
                sha256 TEXT NOT NULL DEFAULT '',
                signature TEXT NOT NULL DEFAULT '',
                entrypoint_command TEXT NOT NULL DEFAULT '',
                entrypoint_args TEXT NOT NULL DEFAULT '[]',
                synced_at DATETIME NOT NULL DEFAULT (datetime('now'))
            );
            CREATE TABLE IF NOT EXISTS user_counter (
                id INTEGER PRIMARY KEY CHECK (id = 1),
                base_count INTEGER NOT NULL DEFAULT 1000,
                real_count INTEGER NOT NULL DEFAULT 0
            );
            CREATE TABLE IF NOT EXISTS skills (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                description TEXT NOT NULL DEFAULT '',
                icon TEXT NOT NULL DEFAULT '⚡',
                mcp_ids TEXT NOT NULL DEFAULT '[]',
                client_ids TEXT NOT NULL DEFAULT '[]',
                installed INTEGER NOT NULL DEFAULT 0,
                enabled INTEGER NOT NULL DEFAULT 1,
                created_at INTEGER NOT NULL DEFAULT 0
            );
            INSERT OR IGNORE INTO user_counter (id, base_count, real_count) VALUES (1, 1000, 0);",
        )
        .map_err(|e| e.to_string())?;

        // Migrate legacy schema: add columns that may be missing from older DBs.
        let migrations = [
            "ALTER TABLE installed_mcps ADD COLUMN description TEXT NOT NULL DEFAULT ''",
            "ALTER TABLE installed_mcps ADD COLUMN args_json TEXT NOT NULL DEFAULT '[]'",
            "ALTER TABLE installed_mcps ADD COLUMN env_json TEXT NOT NULL DEFAULT '{}'",
            "ALTER TABLE installed_mcps ADD COLUMN cwd TEXT NOT NULL DEFAULT ''",
            "ALTER TABLE installed_mcps ADD COLUMN manifest_json TEXT NOT NULL DEFAULT '{}'",
            "ALTER TABLE installed_mcps ADD COLUMN installed_at INTEGER NOT NULL DEFAULT 0",
            "ALTER TABLE marketplace_items ADD COLUMN transport TEXT NOT NULL DEFAULT 'stdio'",
            "ALTER TABLE marketplace_items ADD COLUMN verification TEXT NOT NULL DEFAULT 'community'",
            "ALTER TABLE marketplace_items ADD COLUMN sha256 TEXT NOT NULL DEFAULT ''",
            "ALTER TABLE marketplace_items ADD COLUMN signature TEXT NOT NULL DEFAULT ''",
            "ALTER TABLE marketplace_items ADD COLUMN entrypoint_command TEXT NOT NULL DEFAULT ''",
            "ALTER TABLE marketplace_items ADD COLUMN entrypoint_args TEXT NOT NULL DEFAULT '[]'",
        ];
        for sql in migrations {
            let _ = conn.execute(sql, []);
        }

        // Create skills table if migrating from older DBs.
        let _ = conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS skills (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                description TEXT NOT NULL DEFAULT '',
                icon TEXT NOT NULL DEFAULT '⚡',
                mcp_ids TEXT NOT NULL DEFAULT '[]',
                client_ids TEXT NOT NULL DEFAULT '[]',
                installed INTEGER NOT NULL DEFAULT 0,
                enabled INTEGER NOT NULL DEFAULT 1,
                created_at INTEGER NOT NULL DEFAULT 0
            );",
        );
        // Migrate skills table: add installed column if missing.
        let _ = conn.execute(
            "ALTER TABLE skills ADD COLUMN installed INTEGER NOT NULL DEFAULT 0",
            [],
        );

        // Activity log table
        let _ = conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS activity_log (
                id TEXT PRIMARY KEY,
                activity_type TEXT NOT NULL DEFAULT '',
                message TEXT NOT NULL DEFAULT '',
                timestamp INTEGER NOT NULL DEFAULT 0,
                icon TEXT NOT NULL DEFAULT ''
            );",
        );

        // If legacy rows exist without manifest_json, backfill with a minimal manifest.
        let _ = conn.execute(
            "UPDATE installed_mcps
             SET manifest_json = json_object(
                 'id', id,
                 'name', name,
                 'version', version,
                 'description', description,
                 'runtime', runtime,
                 'entrypoint', json_object('command', command, 'args', json(args_json), 'cwd', cwd),
                 'envSchema', '[]'
             )
             WHERE manifest_json = '{}' OR manifest_json IS NULL",
            [],
        );

        // Seed built-in MCPs if table is empty (first run).
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i64;
        let count: i64 = conn
            .query_row("SELECT COUNT(*) FROM installed_mcps", [], |r| r.get(0))
            .unwrap_or(0);
        if count == 0 {
            let defaults = [
                ("mach1", "Mach1 Router", "1.0.0", "binary", true, "mach1", "Semantic router for 1mcp.in. Auto-activates the required MCPs using prompt-aware matching.", "[]", "{}", "", r#"{"id":"mach1","name":"Mach1 Router","version":"1.0.0","runtime":"binary","description":"Semantic router for 1mcp.in","entrypoint":{"command":"mach1","args":[],"cwd":""},"envSchema":[]}"#),
                ("github", "GitHub", "0.6.2", "node", false, "npx", "Search code, read issues/PRs, and create issues on GitHub via the GitHub API.", r#"["-y","@modelcontextprotocol/server-github"]"#, "{}", "", r#"{"id":"github","name":"GitHub","version":"0.6.2","runtime":"node","description":"Search code, read issues/PRs, and create issues on GitHub via the GitHub API.","entrypoint":{"command":"npx","args":["-y","@modelcontextprotocol/server-github"],"cwd":""},"envSchema":[{"name":"GITHUB_PERSONAL_ACCESS_TOKEN","label":"GitHub Personal Access Token","secret":true,"required":true}]}"#),
                ("memory", "Knowledge Graph Memory", "0.6.0", "node", false, "npx", "Persistent knowledge graph the agent can query and update across sessions.", r#"["-y","@modelcontextprotocol/server-memory"]"#, "{}", "", r#"{"id":"memory","name":"Knowledge Graph Memory","version":"0.6.0","runtime":"node","description":"Persistent knowledge graph the agent can query and update across sessions.","entrypoint":{"command":"npx","args":["-y","@modelcontextprotocol/server-memory"],"cwd":""},"envSchema":[]}"#),
            ];
            for (id, name, version, runtime, enabled, cmd, desc, args, env, cwd, manifest) in
                defaults
            {
                let _ = conn.execute(
                    "INSERT OR IGNORE INTO installed_mcps (id, name, version, runtime, enabled, command, description, args_json, env_json, cwd, manifest_json, installed_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12)",
                    rusqlite::params![id, name, version, runtime, enabled as i32, cmd, desc, args, env, cwd, manifest, now],
                );
            }
        }

        // Seed built-in skills if table is empty.
        let skill_count: i64 = conn
            .query_row("SELECT COUNT(*) FROM skills", [], |r| r.get(0))
            .unwrap_or(0);
        if skill_count == 0 {
            let seed_skills = [
                (
                    "frontend-dev",
                    "Frontend Developer",
                    "GitHub, filesystem, and memory for frontend workflows",
                    "🎨",
                    r#"["github","filesystem","memory"]"#,
                    "[]",
                ),
                (
                    "backend-dev",
                    "Backend Developer",
                    "GitHub, Postgres, and fetch for backend and API work",
                    "⚙️",
                    r#"["github","postgres","fetch"]"#,
                    "[]",
                ),
                (
                    "devops",
                    "DevOps",
                    "GitHub, fetch, and memory for infrastructure and deployment",
                    "🛠️",
                    r#"["github","fetch","memory"]"#,
                    "[]",
                ),
                (
                    "writer",
                    "Writer",
                    "Fetch, memory, and filesystem for research and content creation",
                    "✍️",
                    r#"["fetch","memory","filesystem"]"#,
                    "[]",
                ),
                (
                    "full-stack",
                    "Full Stack",
                    "GitHub, filesystem, memory, and fetch for end-to-end development",
                    "🚀",
                    r#"["github","filesystem","memory","fetch"]"#,
                    "[]",
                ),
            ];
            for (id, name, desc, icon, mcp_ids, client_ids) in seed_skills {
                let _ = conn.execute(
                    "INSERT OR IGNORE INTO skills (id, name, description, icon, mcp_ids, client_ids, installed, enabled, created_at) VALUES (?1,?2,?3,?4,?5,?6,0,1,?7)",
                    rusqlite::params![id, name, desc, icon, mcp_ids, client_ids, now],
                );
            }
        }

        Ok(Self { conn })
    }

    pub fn list_installed(&self) -> Result<Vec<InstalledMcp>, String> {
        let mut stmt = self.conn.prepare(
            "SELECT id, name, version, runtime, enabled, command, description, args_json, env_json, cwd, manifest_json, installed_at FROM installed_mcps"
        ).map_err(|e| e.to_string())?;

        let rows = stmt
            .query_map([], |row| {
                let args_json: String = row.get(7)?;
                let args: Vec<String> = serde_json::from_str(&args_json).unwrap_or_default();
                let env_json: String = row.get(8)?;
                let env: HashMap<String, String> =
                    serde_json::from_str(&env_json).unwrap_or_default();
                let installed_at: i64 = row.get(11)?;
                Ok(InstalledMcp {
                    id: row.get(0)?,
                    name: row.get(1)?,
                    version: row.get(2)?,
                    runtime: row.get(3)?,
                    enabled: row.get::<_, i32>(4)? != 0,
                    command: row.get(5)?,
                    args,
                    env,
                    cwd: row.get(9)?,
                    description: row.get(6)?,
                    manifest_json: row.get(10)?,
                    installed_at,
                })
            })
            .map_err(|e| e.to_string())?;

        rows.collect::<Result<Vec<_>, _>>()
            .map_err(|e| e.to_string())
    }

    pub fn upsert_mcp(&self, mcp: &InstalledMcp) -> Result<(), String> {
        let args_json = serde_json::to_string(&mcp.args).unwrap_or_else(|_| "[]".to_string());
        let env_json = serde_json::to_string(&mcp.env).unwrap_or_else(|_| "{}".to_string());
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i64;

        // If manifest_json is empty, build a minimal manifest.
        let manifest = if mcp.manifest_json.is_empty() || mcp.manifest_json == "{}" {
            serde_json::json!({
                "id": mcp.id,
                "name": mcp.name,
                "version": mcp.version,
                "description": mcp.description,
                "runtime": mcp.runtime,
                "entrypoint": {
                    "command": mcp.command,
                    "args": mcp.args,
                    "cwd": mcp.cwd,
                },
                "envSchema": [],
            })
            .to_string()
        } else {
            mcp.manifest_json.clone()
        };

        self.conn.execute(
            "INSERT INTO installed_mcps (id, name, version, runtime, enabled, command, description, args_json, env_json, cwd, manifest_json, installed_at)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)
             ON CONFLICT(id) DO UPDATE SET
                name=excluded.name, version=excluded.version, runtime=excluded.runtime,
                enabled=excluded.enabled, command=excluded.command, description=excluded.description,
                args_json=excluded.args_json, env_json=excluded.env_json, cwd=excluded.cwd,
                manifest_json=excluded.manifest_json",
            params![
                mcp.id, mcp.name, mcp.version, mcp.runtime, mcp.enabled as i32,
                mcp.command, mcp.description, args_json, env_json, mcp.cwd, manifest,
                if mcp.installed_at > 0 { mcp.installed_at } else { now }
            ],
        ).map_err(|e| e.to_string())?;
        Ok(())
    }

    pub fn remove_mcp(&self, id: &str) -> Result<(), String> {
        self.conn
            .execute("DELETE FROM installed_mcps WHERE id = ?1", params![id])
            .map_err(|e| e.to_string())?;
        Ok(())
    }

    pub fn toggle_mcp(&self, id: &str) -> Result<bool, String> {
        self.conn.execute(
            "UPDATE installed_mcps SET enabled = CASE WHEN enabled = 1 THEN 0 ELSE 1 END WHERE id = ?1",
            params![id],
        ).map_err(|e| e.to_string())?;
        let enabled: bool = self
            .conn
            .query_row(
                "SELECT enabled FROM installed_mcps WHERE id = ?1",
                params![id],
                |r| r.get::<_, i32>(0).map(|v| v != 0),
            )
            .map_err(|e| e.to_string())?;
        Ok(enabled)
    }

    /// Upsert marketplace items received from the cloud API into local cache.
    pub fn sync_marketplace(&self, items: &[MarketplaceItem]) -> Result<(), String> {
        for item in items {
            let tags_json = serde_json::to_string(&item.tags).unwrap_or_else(|_| "[]".to_string());
            let entrypoint_args =
                serde_json::to_string(&item.entrypoint.args).unwrap_or_else(|_| "[]".to_string());
            self.conn.execute(
                "INSERT INTO marketplace_items (id, name, description, version, runtime, transport, tags, homepage, license, verification, sha256, signature, entrypoint_command, entrypoint_args, synced_at)
                 VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, datetime('now'))
                 ON CONFLICT(id) DO UPDATE SET
                    name=excluded.name, description=excluded.description,
                    version=excluded.version, runtime=excluded.runtime, transport=excluded.transport,
                    tags=excluded.tags, homepage=excluded.homepage,
                    license=excluded.license, verification=excluded.verification,
                    sha256=excluded.sha256, signature=excluded.signature,
                    entrypoint_command=excluded.entrypoint_command,
                    entrypoint_args=excluded.entrypoint_args, synced_at=datetime('now')",
                params![item.id, item.name, item.description, item.version, item.runtime, item.transport, tags_json, item.homepage, item.license, item.verification, item.sha256, item.signature, item.entrypoint.command, entrypoint_args],
            ).map_err(|e| e.to_string())?;
        }
        Ok(())
    }

    /// List all cached marketplace items.
    pub fn list_marketplace(&self) -> Result<Vec<MarketplaceItem>, String> {
        let mut stmt = self.conn.prepare(
            "SELECT id, name, description, version, runtime, transport, tags, homepage, license, verification, sha256, signature, entrypoint_command, entrypoint_args FROM marketplace_items ORDER BY name"
        ).map_err(|e| e.to_string())?;

        let rows = stmt
            .query_map([], |row| {
                let tags_json: String = row.get(6)?;
                let tags: Vec<String> = serde_json::from_str(&tags_json).unwrap_or_default();
                let entrypoint_args_json: String = row.get(13)?;
                let entrypoint_args: Vec<String> =
                    serde_json::from_str(&entrypoint_args_json).unwrap_or_default();
                Ok(MarketplaceItem {
                    id: row.get(0)?,
                    name: row.get(1)?,
                    description: row.get(2)?,
                    version: row.get(3)?,
                    runtime: row.get(4)?,
                    transport: row.get(5)?,
                    tags,
                    homepage: row.get(7)?,
                    license: row.get(8)?,
                    verification: row.get(9)?,
                    sha256: row.get(10)?,
                    signature: row.get(11)?,
                    entrypoint: MarketplaceEntrypoint {
                        command: row.get(12)?,
                        args: entrypoint_args,
                        cwd: String::new(),
                    },
                })
            })
            .map_err(|e| e.to_string())?;

        rows.collect::<Result<Vec<_>, _>>()
            .map_err(|e| e.to_string())
    }

    pub fn get_user_count(&self) -> Result<u64, String> {
        let (base, real): (u64, u64) = self
            .conn
            .query_row(
                "SELECT base_count, real_count FROM user_counter WHERE id = 1",
                [],
                |r| Ok((r.get(0)?, r.get(1)?)),
            )
            .map_err(|e| e.to_string())?;
        Ok(base + real)
    }

    pub fn increment_user_count(&self) -> Result<u64, String> {
        self.conn
            .execute(
                "UPDATE user_counter SET real_count = real_count + 1 WHERE id = 1",
                [],
            )
            .map_err(|e| e.to_string())?;
        self.get_user_count()
    }

    // ── Skills CRUD ──

    pub fn list_skills(&self) -> Result<Vec<Skill>, String> {
        let mut stmt = self.conn.prepare(
            "SELECT id, name, description, icon, mcp_ids, client_ids, installed, enabled, created_at FROM skills ORDER BY name"
        ).map_err(|e| e.to_string())?;

        let rows = stmt
            .query_map([], |row| {
                let mcp_json: String = row.get(4)?;
                let mcp_ids: Vec<String> = serde_json::from_str(&mcp_json).unwrap_or_default();
                let client_json: String = row.get(5)?;
                let client_ids: Vec<String> =
                    serde_json::from_str(&client_json).unwrap_or_default();
                Ok(Skill {
                    id: row.get(0)?,
                    name: row.get(1)?,
                    description: row.get(2)?,
                    icon: row.get(3)?,
                    mcp_ids,
                    client_ids,
                    installed: row.get::<_, i32>(6)? != 0,
                    enabled: row.get::<_, i32>(7)? != 0,
                    created_at: row.get(8)?,
                })
            })
            .map_err(|e| e.to_string())?;

        rows.collect::<Result<Vec<_>, _>>()
            .map_err(|e| e.to_string())
    }

    pub fn upsert_skill(&self, skill: &Skill) -> Result<(), String> {
        let mcp_json = serde_json::to_string(&skill.mcp_ids).unwrap_or_else(|_| "[]".to_string());
        let client_json =
            serde_json::to_string(&skill.client_ids).unwrap_or_else(|_| "[]".to_string());
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i64;

        self.conn.execute(
            "INSERT INTO skills (id, name, description, icon, mcp_ids, client_ids, installed, enabled, created_at)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)
             ON CONFLICT(id) DO UPDATE SET
                name=excluded.name, description=excluded.description,
                icon=excluded.icon, mcp_ids=excluded.mcp_ids,
                client_ids=excluded.client_ids, installed=excluded.installed,
                enabled=excluded.enabled",
            params![
                skill.id, skill.name, skill.description, skill.icon,
                mcp_json, client_json, skill.installed as i32, skill.enabled as i32,
                if skill.created_at > 0 { skill.created_at } else { now }
            ],
        ).map_err(|e| e.to_string())?;
        Ok(())
    }

    pub fn remove_skill(&self, id: &str) -> Result<(), String> {
        self.conn
            .execute("DELETE FROM skills WHERE id = ?1", params![id])
            .map_err(|e| e.to_string())?;
        Ok(())
    }

    pub fn toggle_skill(&self, id: &str) -> Result<bool, String> {
        self.conn
            .execute(
                "UPDATE skills SET enabled = CASE WHEN enabled = 1 THEN 0 ELSE 1 END WHERE id = ?1",
                params![id],
            )
            .map_err(|e| e.to_string())?;
        let enabled: bool = self
            .conn
            .query_row(
                "SELECT enabled FROM skills WHERE id = ?1",
                params![id],
                |r| r.get::<_, i32>(0).map(|v| v != 0),
            )
            .map_err(|e| e.to_string())?;
        Ok(enabled)
    }

    // ── Activity Log ──

    pub fn add_activity(&self, item: &ActivityItem) -> Result<(), String> {
        self.conn.execute(
            "INSERT INTO activity_log (id, activity_type, message, timestamp, icon) VALUES (?1, ?2, ?3, ?4, ?5)",
            params![item.id, item.activity_type, item.message, item.timestamp, item.icon],
        ).map_err(|e| e.to_string())?;
        Ok(())
    }

    pub fn get_activity_log(&self, limit: i64) -> Result<Vec<ActivityItem>, String> {
        let mut stmt = self.conn.prepare(
            "SELECT id, activity_type, message, timestamp, icon FROM activity_log ORDER BY timestamp DESC LIMIT ?1"
        ).map_err(|e| e.to_string())?;

        let rows = stmt
            .query_map([limit], |row| {
                Ok(ActivityItem {
                    id: row.get(0)?,
                    activity_type: row.get(1)?,
                    message: row.get(2)?,
                    timestamp: row.get(3)?,
                    icon: row.get(4)?,
                })
            })
            .map_err(|e| e.to_string())?;

        rows.collect::<Result<Vec<_>, _>>()
            .map_err(|e| e.to_string())
    }
}
