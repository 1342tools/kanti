package models

import "time"

// Organization represents an organization entity.
type Organization struct {
	ID               uint         `json:"id"`
	Name             string       `json:"name"`
	Notes            string       `json:"notes,omitempty"`           // Optional notes
	BugBountyLink    string       `json:"bug_bounty_link,omitempty"` // Optional link
	CreatedAt        time.Time    `json:"created_at"`
	RootDomains      []RootDomain `json:"root_domains,omitempty" gorm:"foreignKey:OrganizationID"` // Relationship
	TotalRootDomains int64        `json:"total_root_domains" gorm:"-"`                             // Calculated field
	TotalSubdomains  int64        `json:"total_subdomains" gorm:"-"`                               // Calculated field
	TotalEndpoints   int64        `json:"total_endpoints" gorm:"-"`                                // Calculated field
}

// RootDomain represents a root domain associated with an organization.
type RootDomain struct {
	ID              uint          `json:"id"`
	OrganizationID  uint          `json:"organization_id"` // Foreign Key
	Domain          string        `json:"domain"`
	CreatedAt       time.Time     `json:"created_at"`
	LastScannedAt   *time.Time    `json:"last_scanned_at,omitempty"` // Nullable DateTime
	Organization    *Organization `json:"organization,omitempty"`    // Relationship
	Subdomains      []Subdomain   `json:"subdomains,omitempty"`      // Relationship
	Scans           []Scan        `json:"scans,omitempty"`           // Relationship
	TotalSubdomains int64         `json:"total_subdomains" gorm:"-"` // Calculated field
	TotalEndpoints  int64         `json:"total_endpoints" gorm:"-"`  // Calculated field
}

// Subdomain represents a subdomain discovered under a root domain.
type Subdomain struct {
	ID           uint         `json:"id"`
	RootDomainID uint         `json:"root_domain_id" gorm:"uniqueIndex:idx_hostname_rootdomain"` // Foreign Key + Unique Index
	Hostname     string       `json:"hostname" gorm:"uniqueIndex:idx_hostname_rootdomain"`       // Unique Index
	IPAddress    string       `json:"ip_address,omitempty"`
	IsActive     bool         `json:"is_active"`
	DiscoveredAt time.Time    `json:"discovered_at"`
	RootDomain   *RootDomain  `json:"root_domain,omitempty"`                                           // Relationship
	ScanID       *uint        `json:"scan_id,omitempty"`                                               // Nullable Foreign Key
	Scan         *Scan        `json:"scan,omitempty"`                                                  // Relationship
	Endpoints    []Endpoint   `json:"endpoints,omitempty"`                                             // Relationship
	Technologies []Technology `json:"technologies,omitempty" gorm:"many2many:subdomain_technologies;"` // Many-to-Many relationship
}

// Endpoint represents a specific path/method discovered on a subdomain.
type Endpoint struct {
	ID               uint              `json:"id"`
	SubdomainID      uint              `json:"subdomain_id"` // Foreign Key
	Path             string            `json:"path"`
	Method           string            `json:"method"`
	StatusCode       int               `json:"status_code,omitempty"`
	ContentType      string            `json:"content_type,omitempty"`
	DiscoveredAt     time.Time         `json:"discovered_at"`
	ScanID           *uint             `json:"scan_id,omitempty"`                                              // Nullable Foreign Key
	Scan             *Scan             `json:"scan,omitempty"`                                                 // Relationship
	Subdomain        *Subdomain        `json:"subdomain,omitempty"`                                            // Relationship
	Parameters       []Parameter       `json:"parameters,omitempty"`                                           // Relationship
	Technologies     []Technology      `json:"technologies,omitempty" gorm:"many2many:endpoint_technologies;"` // Many-to-Many relationship
	RequestResponses []RequestResponse `json:"request_responses,omitempty"`                                    // Relationship
}

// Parameter represents a parameter associated with an endpoint.
type Parameter struct {
	ID           uint      `json:"id"`
	EndpointID   uint      `json:"endpoint_id"` // Foreign Key
	Name         string    `json:"name"`
	ParamType    string    `json:"param_type"` // 'query', 'body', 'cookie', 'header'
	DiscoveredAt time.Time `json:"discovered_at"`
	Endpoint     *Endpoint `json:"endpoint,omitempty"` // Relationship
}

// Technology represents a web technology identified.
type Technology struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category,omitempty"`
	// Relationships Subdomains and Endpoints are Many-to-Many, handled via join tables
}

// SubdomainTechnology represents the join table between Subdomains and Technologies.
type SubdomainTechnology struct {
	SubdomainID  uint      `json:"subdomain_id"`         // Foreign Key & Primary Key
	TechnologyID uint      `json:"technology_id"`        // Foreign Key & Primary Key
	Confidence   *float64  `json:"confidence,omitempty"` // Nullable Float
	DetectedAt   time.Time `json:"detected_at"`
}

// EndpointTechnology represents the join table between Endpoints and Technologies.
type EndpointTechnology struct {
	EndpointID   uint      `json:"endpoint_id"`          // Foreign Key & Primary Key
	TechnologyID uint      `json:"technology_id"`        // Foreign Key & Primary Key
	Confidence   *float64  `json:"confidence,omitempty"` // Nullable Float
	DetectedAt   time.Time `json:"detected_at"`
}

// RequestResponse stores captured HTTP request/response pairs for an endpoint.
type RequestResponse struct {
	ID              uint      `json:"id"`
	EndpointID      uint      `json:"endpoint_id"`                // Foreign Key
	RequestHeaders  string    `json:"request_headers,omitempty"`  // Text -> string
	RequestBody     string    `json:"request_body,omitempty"`     // Text -> string
	ResponseHeaders string    `json:"response_headers,omitempty"` // Text -> string
	ResponseBody    string    `json:"response_body,omitempty"`    // Text -> string
	CapturedAt      time.Time `json:"captured_at"`
	Endpoint        *Endpoint `json:"endpoint,omitempty"` // Relationship
}

// Scan represents a scan task performed on a root domain or subdomain.
type Scan struct {
	ID                   uint          `json:"id"`
	RootDomainID         uint          `json:"root_domain_id"`         // Foreign Key (always set, even for subdomain scans)
	SubdomainID          *uint         `json:"subdomain_id,omitempty"` // Nullable Foreign Key for subdomain-specific scans
	ScanType             string        `json:"scan_type"`
	StartedAt            time.Time     `json:"started_at"`
	CompletedAt          *time.Time    `json:"completed_at,omitempty"` // Nullable DateTime
	Status               string        `json:"status,omitempty"`
	ResultsSummary       string        `json:"results_summary,omitempty"`       // Text -> string
	RootDomain           *RootDomain   `json:"root_domain,omitempty"`           // Relationship
	Subdomain            *Subdomain    `json:"subdomain,omitempty"`             // Relationship (for subdomain scans)
	DiscoveredSubdomains []Subdomain   `json:"discovered_subdomains,omitempty"` // Relationship (relevant for root domain scans)
	DiscoveredEndpoints  []Endpoint    `json:"discovered_endpoints,omitempty"`  // Relationship
	ScanTemplateID       *uint         `json:"scan_template_id,omitempty"`      // Nullable Foreign Key
	ScanTemplate         *ScanTemplate `json:"scan_template,omitempty"`         // Relationship
}

// ScanTemplate defines the configuration for a scan.
type ScanTemplate struct {
	ID                  uint       `json:"id"`
	Name                string     `json:"name"`
	Description         string     `json:"description,omitempty"`           // Text -> string
	SubdomainScanConfig string     `json:"subdomain_scan_config,omitempty"` // Text (JSON string) -> string
	URLScanConfig       string     `json:"url_scan_config,omitempty"`       // Text (JSON string) -> string
	ParameterScanConfig string     `json:"parameter_scan_config,omitempty"` // Text (JSON string) -> string
	TechDetectEnabled   bool       `json:"tech_detect_enabled"`
	ScreenshotEnabled   bool       `json:"screenshot_enabled"` // New field for enabling screenshots
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty"` // Nullable DateTime (onupdate)
	Scans               []Scan     `json:"scans,omitempty"`      // Relationship
}

// Screenshot stores information about captured screenshots.
type Screenshot struct {
	ID          uint       `json:"id"`
	SubdomainID *uint      `json:"subdomain_id,omitempty"` // Optional Foreign Key to Subdomain
	EndpointID  *uint      `json:"endpoint_id,omitempty"`  // Optional Foreign Key to Endpoint
	URL         string     `json:"url"`                    // The URL that was screenshotted
	FilePath    string     `json:"file_path"`              // Path to the saved screenshot image file
	ScanID      uint       `json:"scan_id"`                // Foreign Key to Scan
	CapturedAt  time.Time  `json:"captured_at"`
	Subdomain   *Subdomain `json:"subdomain,omitempty"` // Relationship
	Endpoint    *Endpoint  `json:"endpoint,omitempty"`  // Relationship
	Scan        *Scan      `json:"scan,omitempty"`      // Relationship
}

// --- Request/Response Structs for Handlers ---
// (Moved from handlers package to avoid circular dependencies and redeclarations)

// ScanStartRequest represents the request body for starting any scan.
type ScanStartRequest struct {
	RootDomainID   uint  `json:"root_domain_id" binding:"required"`
	SubdomainID    *uint `json:"subdomain_id"`     // Optional: ID of the specific subdomain to scan
	ScanTemplateID *uint `json:"scan_template_id"` // Optional: ID of the template to use
}

// ScanConfig holds parsed configuration from a ScanTemplate.
type ScanConfig struct {
	SubdomainScanConfig map[string]interface{} `json:"subdomain_scan_config"`
	URLScanConfig       map[string]interface{} `json:"url_scan_config"`
	ParameterScanConfig map[string]interface{} `json:"parameter_scan_config"`
	TechDetectEnabled   bool                   `json:"tech_detect_enabled"`
	ScreenshotEnabled   bool                   `json:"screenshot_enabled"` // Added based on template model
}

// --- Shared Scanner Configuration Structs ---
// These structs define the expected JSON structure within ScanTemplate config fields.

// ScanToolConfig represents tool-specific configuration within a scan section.
type ScanToolConfig struct {
	Enabled bool     `json:"enabled"`           // Is this specific tool enabled?
	Options []string `json:"options,omitempty"` // Tool-specific command-line style options (e.g., "--threads=10")
}

// ScanSectionConfig represents configuration for a scan section (e.g., subdomains, urls).
// This structure is expected to be marshalled into the JSON strings stored in ScanTemplate.
type ScanSectionConfig struct {
	Enabled bool                      `json:"enabled"`         // Is this entire section (e.g., Subdomain Scan) enabled?
	Tools   map[string]ScanToolConfig `json:"tools,omitempty"` // Map of tool names (e.g., "subfinder", "katana") to their configs
}

// Note: The original SubdomainScannerConfig and URLScannerConfig structs are removed
// as their structure did not match the parsing logic in the scanner.
// The ScanTemplate fields (SubdomainScanConfig, URLScanConfig, etc.) will store
// JSON strings marshalled from ScanSectionConfig instances.
