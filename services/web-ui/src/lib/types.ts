export type Runtime = 'node' | 'python' | 'go' | 'binary';
export type PatProvider = 'github' | 'gitlab' | 'linear' | 'custom';
export type VerificationStatus = 'anthropic-official' | '1mcp.in-verified' | 'community' | 'pending' | 'verified' | 'unverified';

export interface User {
	id: string;
	name: string;
	email: string;
}

export interface InstalledMcp {
	id: string;
	name: string;
	version: string;
	runtime: Runtime;
	enabled: boolean;
	command: string;
	description: string;
	patProvider?: PatProvider;
}

export interface MarketplaceMcp {
	id: string;
	name: string;
	shortDescription: string;
	version: string;
	runtime: Runtime;
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
	patProvider?: PatProvider;
}

export interface ClientApp {
	id: string;
	name: string;
	icon: string;
	description: string;
	connected: boolean;
	connectCommand: string;
}
