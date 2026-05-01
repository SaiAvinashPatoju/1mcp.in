export type Runtime = 'node' | 'python' | 'go' | 'binary';
export type PatProvider = 'github' | 'gitlab' | 'linear' | 'custom';
export type VerificationStatus = 'anthropic-official' | '1mcp.in-verified' | 'community' | 'pending' | 'verified' | 'unverified';

export interface User {
	id: string;
	name: string;
	email: string;
}

export interface MarketplaceEntrypoint {
	command: string;
	args?: string[];
	cwd?: string;
}

export interface InstalledMcp {
	id: string;
	name: string;
	version: string;
	runtime: Runtime;
	enabled: boolean;
	command: string;
	args?: string[];
	env?: Record<string, string>;
	cwd?: string;
	description: string;
	manifest_json?: string;
	installed_at?: number;
	patProvider?: PatProvider;
}

export interface MarketplaceMcp {
	id: string;
	name: string;
	shortDescription: string;
	version: string;
	runtime: Runtime;
	transport?: string;
	author: string;
	tags: string[];
	rating: number;
	reviewCount: number;
	downloads: number;
	verificationStatus: VerificationStatus;
	sha256?: string;
	signature?: string;
	publishedAt: string;
	installed: boolean;
	entrypoint?: MarketplaceEntrypoint;
	patProvider?: PatProvider;
}

export interface Skill {
	id: string;
	name: string;
	description: string;
	icon: string;
	mcp_ids: string[];
	client_ids: string[];
	installed: boolean;
	enabled: boolean;
	created_at: number;
}

export interface ClientApp {
	id: string;
	name: string;
	icon: string;
	description: string;
	connected: boolean;
	connectCommand: string;
	status_detail?: string;
	transport?: string;
	config_path?: string;
	last_seen?: string;
	routing_status?: 'active' | 'idle' | 'inactive';
	routing_detail?: string;
}

export interface ClientConnectionDetail {
	id: string;
	name: string;
	subtitle: string;
	status: 'connected' | 'not_connected' | 'disconnected' | 'connected_idle';
	transport: string;
	config_path: string;
	last_handshake: string;
	router_binding: string;
	process_id: string;
}

export interface ClientRoutingHealth {
	requests: number;
	active_tools: string[];
	latency_avg_ms: number;
	errors: number;
	period: string;
}

export interface ClientConfigPreview {
	path: string;
	content: string;
}

export interface RouterStatus {
	status: string;
	version: string;
	transport: string;
	uptime_seconds: number;
	port: number;
	metrics_endpoint: string;
}

export interface SystemUsage {
	cpu_percent: number;
	memory_percent: number;
	disk_percent: number;
	cpu_history: number[];
	memory_history: number[];
	disk_history: number[];
}

export interface ActivityItem {
	id: string;
	type: 'router_started' | 'client_connected' | 'mcp_started' | 'mcp_stopped' | 'user_registered' | 'command_executed' | 'error';
	message: string;
	timestamp: string;
	icon: string;
}

export interface McpServerDetail {
	id: string;
	name: string;
	description: string;
	runtime: Runtime;
	version: string;
	status: 'running' | 'sleeping' | 'error';
	status_detail?: string;
	lifecycle: string;
	trust: string;
	author: string;
	idle_timeout?: string;
	last_used_at: string | null;
	last_used_by?: string;
	process?: ServerProcessInfo;
	tools_count: number;
	installed_at: string;
}

export interface CommandResult {
	output: string;
	error: string;
}

export interface MarketplaceItemDetail {
	id: string;
	name: string;
	description: string;
	shortDescription: string;
	version: string;
	runtime: Runtime;
	author: string;
	trust: string;
	license: string;
	sha256: string;
	verified_at: string;
	updated_at: string;
	downloads: number;
	rating: number;
	reviewCount: number;
	tags: string[];
	installed: boolean;
	capabilities: string[];
	security_checks: { label: string; status: 'passed' | 'warning' | 'failed' }[];
	requires_env: string[];
}

export interface ServerProcessInfo {
	pid?: number;
	memory_mb: number;
	cpu_percent: number;
	uptime_seconds: number;
	restarts: number;
}

export interface ServerTool {
	name: string;
	description: string;
	input_schema?: Record<string, unknown>;
}

export interface ServerLogEntry {
	timestamp: string;
	level: 'info' | 'warn' | 'error' | 'debug';
	message: string;
}

export interface ServerEnvironment {
	key: string;
	value: string;
	secret: boolean;
}

export interface ServerConfig {
	command: string;
	args: string[];
	cwd: string;
	env: ServerEnvironment[];
}

export interface ServerDetail {
	id: string;
	name: string;
	description: string;
	version: string;
	runtime: Runtime;
	status: 'running' | 'sleeping' | 'error';
	status_detail?: string;
	trust: string;
	author: string;
	lifecycle: string;
	idle_timeout?: string;
	last_used_at: string | null;
	last_used_by?: string;
	process?: ServerProcessInfo;
	tools_count: number;
	installed_at: string;
}

export interface AppPreferences {
	start_on_login: boolean;
	minimize_to_tray: boolean;
	theme: 'dark' | 'light' | 'system';
	language: string;
	telemetry_enabled: boolean;
	log_level: 'debug' | 'info' | 'warn' | 'error';
}

export interface SystemInfo {
	platform: string;
	version: string;
	router_status: 'running' | 'stopped' | 'error';
	transport: string;
	uptime_seconds: number;
	metrics_endpoint: string;
	data_directory: string;
}

export interface DiagnosticsData {
	platform: string;
	version: string;
	router_status: string;
	transport: string;
	uptime: string;
	cpu_percent: number;
	memory_percent: number;
	log_level: string;
	installed_mcps: number;
	connected_clients: number;
}
