use rusqlite::{Connection, params};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct InstalledMcp {
    pub id: String,
    pub name: String,
    pub version: String,
    pub runtime: String,
    pub enabled: bool,
    pub command: String,
    pub description: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct MarketplaceItem {
    pub id: String,
    pub name: String,
    pub description: String,
    pub version: String,
    pub runtime: String,
    pub tags: Vec<String>,
    pub homepage: String,
    pub license: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct UserStats {
    pub total_users: u64,
    pub mcps_installed: u64,
    pub active_connections: u64,
}

pub struct Db {
    conn: Connection,
}

impl Db {
    pub fn open(data_dir: &PathBuf) -> Result<Self, String> {
        std::fs::create_dir_all(data_dir).map_err(|e| e.to_string())?;
        let path = data_dir.join("1mcp.db");
        let conn = Connection::open(&path).map_err(|e| e.to_string())?;

        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS installed_mcps (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                version TEXT NOT NULL,
                runtime TEXT NOT NULL,
                enabled INTEGER NOT NULL DEFAULT 1,
                command TEXT NOT NULL,
                description TEXT NOT NULL DEFAULT ''
            );
            CREATE TABLE IF NOT EXISTS tokens (
                mcp_id TEXT NOT NULL,
                key TEXT NOT NULL,
                value TEXT NOT NULL,
                PRIMARY KEY (mcp_id, key),
                FOREIGN KEY (mcp_id) REFERENCES installed_mcps(id) ON DELETE CASCADE
            );
            CREATE TABLE IF NOT EXISTS marketplace_items (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                description TEXT NOT NULL DEFAULT '',
                version TEXT NOT NULL DEFAULT '0.0.1',
                runtime TEXT NOT NULL DEFAULT 'node',
                tags TEXT NOT NULL DEFAULT '[]',
                homepage TEXT NOT NULL DEFAULT '',
                license TEXT NOT NULL DEFAULT 'MIT',
                synced_at DATETIME NOT NULL DEFAULT (datetime('now'))
            );
            CREATE TABLE IF NOT EXISTS user_counter (
                id INTEGER PRIMARY KEY CHECK (id = 1),
                base_count INTEGER NOT NULL DEFAULT 1000,
                real_count INTEGER NOT NULL DEFAULT 0
            );
            INSERT OR IGNORE INTO user_counter (id, base_count, real_count) VALUES (1, 1000, 0);"
        ).map_err(|e| e.to_string())?;

        Ok(Self { conn })
    }

    pub fn list_installed(&self) -> Result<Vec<InstalledMcp>, String> {
        let mut stmt = self.conn.prepare(
            "SELECT id, name, version, runtime, enabled, command, description FROM installed_mcps"
        ).map_err(|e| e.to_string())?;

        let rows = stmt.query_map([], |row| {
            Ok(InstalledMcp {
                id: row.get(0)?,
                name: row.get(1)?,
                version: row.get(2)?,
                runtime: row.get(3)?,
                enabled: row.get::<_, i32>(4)? != 0,
                command: row.get(5)?,
                description: row.get(6)?,
            })
        }).map_err(|e| e.to_string())?;

        rows.collect::<Result<Vec<_>, _>>().map_err(|e| e.to_string())
    }

    pub fn upsert_mcp(&self, mcp: &InstalledMcp) -> Result<(), String> {
        self.conn.execute(
            "INSERT INTO installed_mcps (id, name, version, runtime, enabled, command, description)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)
             ON CONFLICT(id) DO UPDATE SET
                name=excluded.name, version=excluded.version, runtime=excluded.runtime,
                enabled=excluded.enabled, command=excluded.command, description=excluded.description",
            params![mcp.id, mcp.name, mcp.version, mcp.runtime, mcp.enabled as i32, mcp.command, mcp.description],
        ).map_err(|e| e.to_string())?;
        Ok(())
    }

    pub fn remove_mcp(&self, id: &str) -> Result<(), String> {
        self.conn.execute("DELETE FROM installed_mcps WHERE id = ?1", params![id])
            .map_err(|e| e.to_string())?;
        Ok(())
    }

    pub fn toggle_mcp(&self, id: &str) -> Result<bool, String> {
        self.conn.execute(
            "UPDATE installed_mcps SET enabled = CASE WHEN enabled = 1 THEN 0 ELSE 1 END WHERE id = ?1",
            params![id],
        ).map_err(|e| e.to_string())?;
        let enabled: bool = self.conn.query_row(
            "SELECT enabled FROM installed_mcps WHERE id = ?1", params![id],
            |r| r.get::<_, i32>(0).map(|v| v != 0)
        ).map_err(|e| e.to_string())?;
        Ok(enabled)
    }

    pub fn set_token(&self, mcp_id: &str, key: &str, value: &str) -> Result<(), String> {
        self.conn.execute(
            "INSERT INTO tokens (mcp_id, key, value) VALUES (?1, ?2, ?3)
             ON CONFLICT(mcp_id, key) DO UPDATE SET value=excluded.value",
            params![mcp_id, key, value],
        ).map_err(|e| e.to_string())?;
        Ok(())
    }

    /// Upsert marketplace items received from the cloud API into local cache.
    pub fn sync_marketplace(&self, items: &[MarketplaceItem]) -> Result<(), String> {
        for item in items {
            let tags_json = serde_json::to_string(&item.tags).unwrap_or_else(|_| "[]".to_string());
            self.conn.execute(
                "INSERT INTO marketplace_items (id, name, description, version, runtime, tags, homepage, license, synced_at)
                 VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, datetime('now'))
                 ON CONFLICT(id) DO UPDATE SET
                    name=excluded.name, description=excluded.description,
                    version=excluded.version, runtime=excluded.runtime,
                    tags=excluded.tags, homepage=excluded.homepage,
                    license=excluded.license, synced_at=datetime('now')",
                params![item.id, item.name, item.description, item.version, item.runtime, tags_json, item.homepage, item.license],
            ).map_err(|e| e.to_string())?;
        }
        Ok(())
    }

    /// List all cached marketplace items.
    pub fn list_marketplace(&self) -> Result<Vec<MarketplaceItem>, String> {
        let mut stmt = self.conn.prepare(
            "SELECT id, name, description, version, runtime, tags, homepage, license FROM marketplace_items ORDER BY name"
        ).map_err(|e| e.to_string())?;

        let rows = stmt.query_map([], |row| {
            let tags_json: String = row.get(5)?;
            let tags: Vec<String> = serde_json::from_str(&tags_json).unwrap_or_default();
            Ok(MarketplaceItem {
                id: row.get(0)?,
                name: row.get(1)?,
                description: row.get(2)?,
                version: row.get(3)?,
                runtime: row.get(4)?,
                tags,
                homepage: row.get(6)?,
                license: row.get(7)?,
            })
        }).map_err(|e| e.to_string())?;

        rows.collect::<Result<Vec<_>, _>>().map_err(|e| e.to_string())
    }

    pub fn get_user_count(&self) -> Result<u64, String> {
        let (base, real): (u64, u64) = self.conn.query_row(
            "SELECT base_count, real_count FROM user_counter WHERE id = 1",
            [],
            |r| Ok((r.get(0)?, r.get(1)?))
        ).map_err(|e| e.to_string())?;
        Ok(base + real)
    }

    pub fn increment_user_count(&self) -> Result<u64, String> {
        self.conn.execute(
            "UPDATE user_counter SET real_count = real_count + 1 WHERE id = 1",
            [],
        ).map_err(|e| e.to_string())?;
        self.get_user_count()
    }
}
