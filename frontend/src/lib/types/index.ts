// Add Organization type
export interface Organization {
	id: number;
	name: string;
	notes?: string | null; // Optional notes
	bug_bounty_link?: string | null; // Optional link
	created_at: string;
	total_root_domains: number; // Added count
	total_subdomains: number; // Added count
	total_endpoints: number; // Added count
	root_domains?: RootDomain[]; // Optional: Include if backend preloads them
}

// Modify RootDomain type (Consider renaming or aliasing if it represents the same concept as Domain)
export interface RootDomain {
	id: number;
	organization_id: number; // Add organization ID
	domain: string;
	created_at: string;
	last_scanned_at: string | null;
	total_subdomains: number; // Added count
	total_endpoints: number; // Added count
}

// Define Domain type (similar to RootDomain for now)
export interface Domain {
	id: number;
	organization_id: number;
	name: string; // Using 'name' as used in the page component
	created_at: string;
	last_scanned_at: string | null;
	total_subdomains: number;
	total_endpoints: number;
}

export interface Subdomain {
    id: number;
    root_domain_id: number;
    hostname: string;
    ip_address: string | null;
    is_active: boolean;
    discovered_at: string;
    technologies?: Technology[];
    latest_screenshot_path?: string | null; // Add screenshot path
}

export interface Endpoint {
    id: number;
    subdomain_id: number;
    path: string;
    method: string;
    status_code: number | null;
    content_type: string | null;
    discovered_at: string;
    parameters?: Parameter[];
    technologies?: Technology[];
    latest_screenshot_path?: string | null; // Add screenshot path
}

export interface Parameter {
    id: number;
    endpoint_id: number;
    name: string;
    param_type: string;
    discovered_at: string;
}

export interface Technology {
    id: number;
    name: string;
    category: string | null;
    confidence?: number;
}

export interface RequestResponse {
    id: number;
    endpoint_id: number;
    request_headers: string;
    request_body: string;
    response_headers: string;
    response_body: string;
    captured_at: string;
}

export interface Scan {
    id: number;
    root_domain_id: number;
    scan_type: string;
    started_at: string;
    completed_at: string | null;
    status: 'running' | 'completed' | 'failed';
    results_summary: string | null; // This is likely JSON string, might need parsing
    discovered_subdomains?: DiscoveredSubdomain[]; // Optional for older scans?
    discovered_endpoints?: DiscoveredEndpoint[]; // Optional for older scans?
}

// Type for subdomains returned within a Scan result
export interface DiscoveredSubdomain {
    id: number;
    hostname: string;
    ip_address: string | null;
    is_active: boolean;
    discovered_at: string | null;
    scan_id: number;
}

// Type for endpoints returned within a Scan result
export interface DiscoveredEndpoint {
    id: number;
    subdomain_id: number;
    path: string;
    method: string;
    status_code: number | null;
    content_type: string | null;
    discovered_at: string | null;
    scan_id: number;
    subdomain_hostname: string | null; // Hostname for constructing full URL
}

// --- Scan Template Types ---

export interface ScanToolConfig {
	enabled: boolean;
	options?: string[]; // Optional array of string options
}

export interface ScanSectionConfig {
	enabled: boolean;
	tools: {
		[toolName: string]: ScanToolConfig; // Dictionary of tool configurations
	};
}

export interface ScanTemplate {
	id: number;
	name: string;
	description?: string | null;
	subdomain_scan_config: ScanSectionConfig;
	url_scan_config: ScanSectionConfig;
	parameter_scan_config: ScanSectionConfig;
	tech_detect_enabled: boolean;
	screenshot_enabled: boolean; // Add the new field for screenshots
	created_at?: string; // Optional on create/update
	updated_at?: string; // Optional on create/update
}

// --- End Scan Template Types ---

export interface ApiError {
    detail: string;
}

export interface GraphNode {
    id: string;
    name: string;
    type: 'domain' | 'subdomain' | 'endpoint' | 'technology' | 'parameter';
    radius?: number;
    level: number;
}

export interface GraphLink {
    source: string;
    target: string;
    value: number;
}

export interface GraphData {
    nodes: GraphNode[];
    links: GraphLink[];
    hierarchyLevels: number;
}

export type Theme = 'light' | 'dark';
