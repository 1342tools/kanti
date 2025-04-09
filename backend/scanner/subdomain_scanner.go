package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors" // Ensure errors package is imported
	"fmt"
	"io"
	"io/ioutil" // Added for TempFile
	"log"
	"net"               // Added for IP parsing
	"os"                // Import os package for file operations
	"rewrite-go/config" // Import the config package
	"rewrite-go/database"
	"rewrite-go/models"
	"strconv" // Add strconv import
	"strings"
	"sync"
	"time"

	"github.com/projectdiscovery/subfinder/v2/pkg/runner"
	"gopkg.in/yaml.v3" // Import yaml package
	"gorm.io/gorm"
	"gorm.io/gorm/clause" // Import the clause package

	httpxrunner "github.com/projectdiscovery/httpx/runner"
)

// --- Scanner Functions ---

// Helper function to safely extract integer options from a map
func getIntOption(options map[string]interface{}, key string, defaultValue int) int {
	if val, ok := options[key]; ok {
		// Try converting from float64 (common for JSON numbers) or int
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string: // Handle string representation if needed
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return defaultValue
}

// Helper function to safely extract boolean options from a map
func getBoolOption(options map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := options[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// Helper function to parse generic tool options into a map[string]interface{}
// This is a basic example; more robust parsing might be needed for complex options.
func parseToolOptions(options []string) map[string]interface{} {
	parsed := make(map[string]interface{})
	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		// Use a more specific key extraction, removing leading dashes carefully
		key := strings.TrimLeft(parts[0], "-")
		key = strings.TrimSpace(key) // Trim spaces after removing dashes

		if len(parts) == 2 {
			valueStr := strings.TrimSpace(parts[1])
			// Attempt to parse as int, float, bool, otherwise keep as string
			if i, err := strconv.Atoi(valueStr); err == nil {
				parsed[key] = i
			} else if f, err := strconv.ParseFloat(valueStr, 64); err == nil {
				parsed[key] = f
			} else if b, err := strconv.ParseBool(valueStr); err == nil {
				parsed[key] = b
			} else {
				// Store the raw string value, removing potential quotes
				parsed[key] = strings.Trim(valueStr, "\"'")
			}
		} else if key != "" {
			// Handle boolean flags (presence implies true)
			parsed[key] = true
		}
	}
	return parsed
}

// runSubfinder executes subfinder for the given domain using provided configuration.
// Renamed config parameter to toolOptions to avoid collision with imported config package.
func runSubfinder(ctx context.Context, domain string, toolOptions map[string]interface{}) (map[string]struct{}, error) {
	// Extract specific options with defaults using the new parameter name
	threads := getIntOption(toolOptions, "threads", 10)
	timeout := getIntOption(toolOptions, "timeout", 30)
	// Match the key used in parseToolOptions (which removes dashes)
	maxEnumTime := getIntOption(toolOptions, "maxEnumerationTime", 5) // Assuming key is maxEnumerationTime after parsing

	// --- Load API Keys from Config and Prepare Provider Config File ---
	providerConfigMap := make(map[string][]string)
	providerConfigFile := "" // Path to the temporary config file

	// Define keys to check based on frontend settings and subfinder provider names
	apiKeysToCheck := map[string]string{
		"shodan":         "SHODAN_API_KEY",
		"censys":         "CENSYS_API_ID", // Needs ID and Secret
		"binaryedge":     "BINARYEDGE_API_KEY",
		"virustotal":     "VIRUSTOTAL_API_KEY",
		"securitytrails": "SECURITYTRAILS_API_KEY",
		"chaos":          "CHAOS_API_KEY",
		"github":         "GITHUB_TOKEN",
		"passivetotal":   "PASSIVETOTAL_USERNAME", // Needs Username and Key
		"zoomeye":        "ZOOMEYE_API_KEY",
		"fofa":           "FOFA_EMAIL", // Needs Email and Key
		"hunter":         "HUNTER_API_KEY",
		"quake":          "QUAKE_API_KEY",
		"netlas":         "NETLAS_API_KEY",
		"intelx":         "INTELX_API_KEY", // Needs Key (Host optional, defaults usually ok)
		"leakix":         "LEAKIX_API_KEY",
		// Add others like anubis, bevigil, criminalip, fullhunt, publicwww, shodan-idb if needed
	}

	log.Println("Loading API keys for Subfinder sources...")
	for source, configKey := range apiKeysToCheck {
		// Use the imported 'config' package
		apiKey := config.Get(configKey) // Primary key/ID/Username/Email
		if apiKey != "" {
			// Handle multi-key providers
			if source == "censys" {
				apiSecret := config.Get("CENSYS_API_SECRET")
				if apiSecret != "" {
					providerConfigMap[source] = []string{apiKey, apiSecret} // ID, Secret
					log.Printf("  - Loaded Censys API ID and Secret")
				} else {
					log.Printf("  - Warning: Censys API ID found but Secret is missing.")
				}
			} else if source == "passivetotal" {
				apiKeyVal := config.Get("PASSIVETOTAL_API_KEY")
				if apiKeyVal != "" {
					providerConfigMap[source] = []string{apiKey, apiKeyVal} // Username, Key
					log.Printf("  - Loaded PassiveTotal Username and Key")
				} else {
					log.Printf("  - Warning: PassiveTotal Username found but Key is missing.")
				}
			} else if source == "fofa" {
				apiKeyVal := config.Get("FOFA_API_KEY")
				if apiKeyVal != "" {
					providerConfigMap[source] = []string{apiKey, apiKeyVal} // Email, Key
					log.Printf("  - Loaded Fofa Email and Key")
				} else {
					log.Printf("  - Warning: Fofa Email found but Key is missing.")
				}
			} else if source == "intelx" {
				// IntelX host is optional, defaults usually work. Key is required.
				providerConfigMap[source] = []string{apiKey} // Just the key
				log.Printf("  - Loaded IntelX API Key")
			} else {
				// Single key providers
				providerConfigMap[source] = []string{apiKey}
				log.Printf("  - Loaded %s API Key/Token", strings.Title(source))
			}
		} else {
			// Log if a key is expected but not found (optional)
			// log.Printf("  - %s API Key not found in config.", strings.Title(source))
		}
	}

	// Create temporary YAML file if any keys were loaded
	if len(providerConfigMap) > 0 {
		yamlData, err := yaml.Marshal(providerConfigMap)
		if err != nil {
			log.Printf("Warning: Failed to marshal provider config to YAML: %v. Proceeding without API keys.", err)
		} else {
			tmpFile, err := os.CreateTemp("", "subfinder-provider-*.yaml")
			if err != nil {
				log.Printf("Warning: Failed to create temporary provider config file: %v. Proceeding without API keys.", err)
			} else {
				providerConfigFile = tmpFile.Name()
				log.Printf("Writing Subfinder provider config to temporary file: %s", providerConfigFile)
				if _, err := tmpFile.Write(yamlData); err != nil {
					log.Printf("Warning: Failed to write to temporary provider config file %s: %v. Proceeding without API keys.", providerConfigFile, err)
					providerConfigFile = "" // Reset path if write failed
				}
				if err := tmpFile.Close(); err != nil {
					log.Printf("Warning: Failed to close temporary provider config file %s: %v.", providerConfigFile, err)
				}
				// Ensure the temporary file is removed after the function returns
				defer func() {
					if providerConfigFile != "" {
						log.Printf("Removing temporary Subfinder provider config file: %s", providerConfigFile)
						os.Remove(providerConfigFile)
					}
				}()
			}
		}
	}
	// --- End API Key Loading and File Creation ---

	log.Printf("Configuring Subfinder: Threads=%d, Timeout=%ds, MaxEnumTime=%dm", threads, timeout, maxEnumTime)
	subfinderOpts := &runner.Options{
		Threads:            threads,
		Timeout:            timeout,
		MaxEnumerationTime: maxEnumTime,
		Silent:             true,               // Keep silent to avoid cluttering logs
		ProviderConfig:     providerConfigFile, // Pass the *path* to the config file
	}

	subfinderRunner, err := runner.NewRunner(subfinderOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create subfinder runner: %w", err)
	}

	output := &bytes.Buffer{} // Discard output, we use the map
	sourceMap, err := subfinderRunner.EnumerateSingleDomainWithCtx(ctx, domain, []io.Writer{output})
	if err != nil {
		// Don't treat context deadline exceeded as fatal, just return what was found
		uniqueSubdomains := make(map[string]struct{}) // Initialize map even on error
		if sourceMap != nil {                         // Check if sourceMap is not nil before iterating
			for subdomain := range sourceMap {
				uniqueSubdomains[subdomain] = struct{}{}
			}
		}
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Subfinder timed out for domain %s, returning partial results (%d found)", domain, len(uniqueSubdomains))
			return uniqueSubdomains, nil // Return potentially partial results
		}
		return uniqueSubdomains, fmt.Errorf("failed to enumerate domain %s: %w", domain, err) // Return found results along with error
	}

	// Extract unique subdomains from the sourceMap
	uniqueSubdomains := make(map[string]struct{})
	for subdomain := range sourceMap {
		uniqueSubdomains[subdomain] = struct{}{}
	}

	return uniqueSubdomains, nil
}

// verifyActiveSubdomains uses httpx library to check which subdomains are responding.
func verifyActiveSubdomains(ctx context.Context, subdomains map[string]struct{}) (map[string]struct{}, error) {
	activeSubdomains := make(map[string]struct{})
	if len(subdomains) == 0 {
		return activeSubdomains, nil
	}

	log.Printf("Verifying %d potential subdomains using httpx...", len(subdomains))

	// --- Create Temporary Input File for httpx ---
	tmpFile, err := ioutil.TempFile("", "httpx-input-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary input file for httpx: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the file afterwards

	hostsList := make([]string, 0, len(subdomains)) // Keep a list for logging
	for host := range subdomains {
		if _, err := tmpFile.WriteString(host + "\n"); err != nil {
			tmpFile.Close() // Close before returning error
			return nil, fmt.Errorf("failed to write to temporary httpx input file: %w", err)
		}
		hostsList = append(hostsList, host)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temporary httpx input file: %w", err)
	}
	// --- End Temp File Creation ---

	// Configure httpx options
	// We want basic probing, silent operation, and capture results via callback
	options := httpxrunner.Options{
		Methods:         "GET",          // Use GET for basic check
		InputFile:       tmpFile.Name(), // Use the temporary file path
		Threads:         50,             // Increase threads for faster checking
		Timeout:         10,             // Timeout in seconds (int)
		Retries:         1,              // Number of retries
		NoColor:         true,
		Silent:          true,  // Keep httpx quiet
		ExtractTitle:    false, // Don't need title
		StatusCode:      true,  // Get status code
		ContentLength:   false, // Don't need content length
		FollowRedirects: true,  // Follow redirects to catch more live hosts
		RandomAgent:     true,
		// Define the callback to process results
		OnResult: func(result httpxrunner.Result) {
			// Check if the probe was successful (no error and maybe filter by status code if needed)
			// For now, any successful probe (non-error) marks it as active.
			// You could add checks like result.StatusCode < 400 if needed.
			if result.Err == nil && result.StatusCode > 0 { // Check for error and valid status code
				// Use a mutex if running httpx concurrently within this function,
				// but httpx runner handles internal concurrency.
				// We just need to safely add to our result map.
				// Since OnResult might be called concurrently, protect the map write.
				// (Although, with a single runner instance, maybe not strictly needed? Better safe)
				// Let's assume httpx calls this sequentially or handles safety. If issues arise, add mutex here.
				activeSubdomains[result.Input] = struct{}{} // Use result.Input (original hostname)
				// log.Printf("httpx verified active: %s (Status: %d)", result.Input, result.StatusCode) // Optional detailed logging
			} else if result.Err != nil {
				// log.Printf("httpx error for %s: %v", result.Input, result.Err) // Optional error logging
			} else {
				// log.Printf("httpx inactive: %s (Status: %d)", result.Input, result.StatusCode) // Optional inactive logging
			}
		},
	}

	// Create and run httpx runner
	runner, err := httpxrunner.New(&options)
	if err != nil {
		return nil, fmt.Errorf("failed to create httpx runner: %w", err)
	}
	defer runner.Close()

	// Run the enumeration
	// RunEnumeration doesn't take context or return an error directly based on compiler feedback
	runner.RunEnumeration()
	// Error handling happens within the OnResult callback or via panics/logs from the runner itself.

	log.Printf("httpx verification complete. Found %d active subdomains.", len(activeSubdomains))
	return activeSubdomains, nil // Assume success unless OnResult logged errors or runner panicked
}

// updateScanStatus updates the status and potentially summary/completion time of a scan.
func updateScanStatus(db *gorm.DB, scanID uint, status string, errMsg ...string) {
	updateData := map[string]interface{}{"status": status}
	message := ""
	if len(errMsg) > 0 && errMsg[0] != "" {
		message = errMsg[0]
		// Use ResultsSummary to store error/completion messages
		updateData["results_summary"] = message
	}

	now := time.Now()
	if status == "running" {
		// Only update StartedAt if it's not already set (or handle re-runs if needed)
		// For simplicity, we'll just set it here. GORM might handle default values too.
		updateData["started_at"] = now
	} else if status == "completed" || status == "failed" {
		updateData["completed_at"] = &now // CompletedAt is a pointer (*time.Time)
	}

	// Perform the update
	if err := db.Model(&models.Scan{}).Where("id = ?", scanID).Updates(updateData).Error; err != nil {
		log.Printf("Error updating scan %d status to %s (message: %s): %v", scanID, status, message, err)
	} else {
		log.Printf("Updated scan %d status to %s", scanID, status)
	}
}

// saveSubdomains saves the found subdomains to the database and returns a map of hostname -> ID for saved/existing ones.
func saveSubdomains(db *gorm.DB, rootDomainID uint, scanID uint, subdomains map[string]struct{}) (map[string]uint, error) {
	savedSubdomainIDs := make(map[string]uint) // Map to return
	if len(subdomains) == 0 {
		log.Printf("No active subdomains to save for scan %d.", scanID)
		return savedSubdomainIDs, nil
	}

	var modelsToCreate []models.Subdomain
	for sub := range subdomains {
		// --- IP Address Filtering ---
		// Check if the 'sub' string is a valid IP address. If so, skip it.
		if net.ParseIP(sub) != nil {
			log.Printf("Skipping potential IP address found during verification: %s", sub)
			continue // Don't save IP addresses as subdomains
		}
		// --- End IP Filtering ---

		// Correct field name is Hostname, ScanID is a pointer
		modelsToCreate = append(modelsToCreate, models.Subdomain{
			Hostname:     sub,
			RootDomainID: rootDomainID,
			ScanID:       &scanID,    // Pass address of scanID
			DiscoveredAt: time.Now(), // Set discovery time
			IsActive:     true,       // Assume active initially, maybe verify later?
		})
	}

	// Use GORM's batch insert with conflict handling (ignore duplicates based on domain and root_domain_id)
	// Note: This requires a unique constraint on (domain, root_domain_id) in your DB schema.
	// If the constraint doesn't exist, duplicates might be inserted or errors might occur depending on the DB.
	// Adjust the conflict handling as needed for your specific database and schema.
	// For PostgreSQL: Clauses(clause.OnConflict{DoNothing: true})
	// For SQLite/MySQL: Clauses(clause.Insert{Modifier: "IGNORE"}) - Check GORM docs for specifics
	// Use GORM's batch insert with conflict handling (ignore duplicates based on hostname and root_domain_id)
	// This requires a unique constraint on (hostname, root_domain_id) in the DB schema.
	log.Printf("Attempting to save %d discovered subdomains for scan %d (duplicates will be ignored)...", len(modelsToCreate), scanID)
	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hostname"}, {Name: "root_domain_id"}}, // Specify conflict columns
		DoNothing: true,                                                          // Ignore duplicates
	}).Create(&modelsToCreate)
	if result.Error != nil {
		return savedSubdomainIDs, fmt.Errorf("failed to save subdomains: %w", result.Error)
	}

	log.Printf("Attempted to save/update %d subdomains for scan %d (%d actually created/updated).", len(modelsToCreate), scanID, result.RowsAffected)

	// After attempting to create, fetch the IDs for all intended subdomains (both new and existing)
	// This ensures we have the correct IDs for linking screenshots later.
	hostnamesToQuery := make([]string, 0, len(modelsToCreate))
	for _, subModel := range modelsToCreate {
		hostnamesToQuery = append(hostnamesToQuery, subModel.Hostname)
	}

	if len(hostnamesToQuery) > 0 {
		var fetchedSubdomains []models.Subdomain
		// Fetch subdomains matching the hostnames and root domain ID
		fetchResult := db.Where("root_domain_id = ? AND hostname IN ?", rootDomainID, hostnamesToQuery).Find(&fetchedSubdomains)
		if fetchResult.Error != nil {
			log.Printf("Warning: Failed to fetch IDs after saving subdomains for scan %d: %v", scanID, fetchResult.Error)
			// Return the error, as we need these IDs for potential screenshots
			return savedSubdomainIDs, fmt.Errorf("failed to fetch subdomain IDs after save: %w", fetchResult.Error)
		}
		for _, sub := range fetchedSubdomains {
			savedSubdomainIDs[sub.Hostname] = sub.ID
		}
		log.Printf("Fetched %d subdomain IDs for potential screenshot linking (Scan ID: %d).", len(savedSubdomainIDs), scanID)
	}

	return savedSubdomainIDs, nil
}

// ExecuteSubdomainScan performs subdomain enumeration or targets a specific subdomain based on scanType.
func ExecuteSubdomainScan(targetHost string, scanType string, rootDomainID uint, scanID uint, scanTemplate *models.ScanTemplate) {
	db := database.GetDB()
	if scanTemplate == nil {
		log.Printf("Error: ExecuteSubdomainScan called with nil scanTemplate for Scan ID: %d", scanID)
		updateScanStatus(db, scanID, "failed", "Internal error: Scan template missing")
		return
	}

	// --- Parse Scan Template Configuration (using shared models) ---
	var subdomainSection models.ScanSectionConfig // Use shared model
	var urlSection models.ScanSectionConfig       // Use shared model
	// Parameter section parsing would go here if needed

	// Default values (will be used if section is disabled or parsing fails)
	subfinderEnabled := true                                                                          // Assume enabled by default for root_domain scans
	subfinderOptions := map[string]interface{}{"threads": 10, "timeout": 30, "maxEnumerationTime": 5} // Default options

	urlScanEnabled := true
	// Default options for Katana (assuming it's the primary URL tool)
	katanaOptions := map[string]interface{}{"maxDepth": 3, "concurrency": 10, "parallelism": 10, "rateLimit": 150, "timeout": 10}
	katanaOutputFile := "" // Initialize output file path

	// Parse Subdomain Config only if it's a root domain scan
	if scanType == "root_domain" {
		if scanTemplate.SubdomainScanConfig != "" {
			err := json.Unmarshal([]byte(scanTemplate.SubdomainScanConfig), &subdomainSection) // Unmarshal into models.ScanSectionConfig
			if err != nil {
				log.Printf("Warning: Failed to parse SubdomainScanConfig JSON for template %d: %v. Using defaults.", scanTemplate.ID, err)
			} else {
				if !subdomainSection.Enabled {
					subfinderEnabled = false
					log.Printf("Subdomain discovery disabled by template %d.", scanTemplate.ID)
				} else {
					if toolCfg, ok := subdomainSection.Tools["subfinder"]; ok {
						subfinderEnabled = toolCfg.Enabled
						if subfinderEnabled {
							subfinderOptions = parseToolOptions(toolCfg.Options)
							// Ensure defaults are present if not specified in options
							if _, ok := subfinderOptions["threads"]; !ok {
								subfinderOptions["threads"] = 10
							}
							if _, ok := subfinderOptions["timeout"]; !ok {
								subfinderOptions["timeout"] = 30
							}
							if _, ok := subfinderOptions["maxEnumerationTime"]; !ok {
								subfinderOptions["maxEnumerationTime"] = 5
							}
						}
					} else {
						subfinderEnabled = false // Tool not defined in config
					}
				}
			}
		} else {
			log.Printf("Scan template %d has no SubdomainScanConfig. Using defaults (Subfinder enabled for root domain scan).", scanTemplate.ID)
		}
	} else {
		// If it's a subdomain scan, disable discovery tools regardless of template
		subfinderEnabled = false
		log.Printf("Subdomain discovery skipped for specific subdomain scan (Scan ID: %d, Target: %s)", scanID, targetHost)
	}

	// Parse URL Config (applies to both scan types)
	if scanTemplate.URLScanConfig != "" {
		err := json.Unmarshal([]byte(scanTemplate.URLScanConfig), &urlSection) // Unmarshal into models.ScanSectionConfig
		if err != nil {
			log.Printf("Warning: Failed to parse URLScanConfig JSON for template %d: %v. Using defaults.", scanTemplate.ID, err)
			// Keep default value (urlScanEnabled=true) if parsing fails
		} else {
			urlScanEnabled = urlSection.Enabled // Check if the whole section is enabled
			if urlScanEnabled {
				// Assuming 'katana' is the primary tool for URL scanning based on the key used in db.go
				if toolCfg, ok := urlSection.Tools["katana"]; ok && toolCfg.Enabled {
					katanaOptions = parseToolOptions(toolCfg.Options) // Parse general options first

					// Check specifically for the outputFile option to enable file output
					for _, opt := range toolCfg.Options {
						if strings.HasPrefix(opt, "outputFile") { // Check if option exists (e.g., "outputFile=true", "outputFile")
							katanaOutputFile = fmt.Sprintf("/tmp/scan_%d_katana_results.txt", scanID)
							log.Printf("Katana output file enabled by template, will write to: %s", katanaOutputFile)
							break // Found the option, no need to check further
						}
					}

					// Ensure defaults for other options are present if not specified
					if _, ok := katanaOptions["maxDepth"]; !ok {
						katanaOptions["maxDepth"] = 3
					}
					if _, ok := katanaOptions["concurrency"]; !ok {
						katanaOptions["concurrency"] = 10
					}
					if _, ok := katanaOptions["parallelism"]; !ok {
						katanaOptions["parallelism"] = 10
					}
					if _, ok := katanaOptions["rateLimit"]; !ok {
						katanaOptions["rateLimit"] = 150
					}
					if _, ok := katanaOptions["timeout"]; !ok {
						katanaOptions["timeout"] = 10
					}
				} else {
					urlScanEnabled = false // Disable URL scan if section enabled but katana tool is not defined or disabled
					log.Printf("URL scanning disabled for template %d (Katana tool not enabled).", scanTemplate.ID)
				}
			} else {
				log.Printf("URL scanning disabled by template %d.", scanTemplate.ID)
			}
		}
	} else {
		log.Printf("Scan template %d has no URLScanConfig. Using defaults.", scanTemplate.ID)
	}

	// Parse Parameter Config (Example structure - adapt if needed)
	// var parameterSection ScanSectionConfig
	// parameterScanEnabled := true // Default
	// arjunOptions := map[string]interface{}{} // Default options for arjun
	// if scanTemplate.ParameterScanConfig != "" { ... parse ... }

	updateScanStatus(db, scanID, "running")
	log.Printf("Starting scan for %s (Type: %s, Scan ID: %d, Template: %s)", targetHost, scanType, scanID, scanTemplate.Name)

	// --- Screenshot Existing Assets (if enabled) ---
	// This part screenshots assets *before* discovery/targeting the specific subdomain.
	// Keep this logic as is, it screenshots based on rootDomainID.
	var initialScreenshotWG sync.WaitGroup
	if scanTemplate.ScreenshotEnabled {
		log.Printf("Screenshotting enabled: Fetching existing assets for scan %d...", scanID)

		// Fetch existing subdomains
		var existingSubdomainsDB []models.Subdomain
		if err := db.Where("root_domain_id = ?", rootDomainID).Find(&existingSubdomainsDB).Error; err != nil {
			log.Printf("Error fetching existing subdomains for screenshotting (Scan ID: %d): %v", scanID, err)
			// Optionally add to scanErrors? For now, just log.
		} else {
			log.Printf("Found %d existing subdomains to potentially screenshot.", len(existingSubdomainsDB))
			for _, sub := range existingSubdomainsDB {
				// Need a loop variable copy for the goroutine
				currentSub := sub
				urlsToTry := []string{
					fmt.Sprintf("http://%s", currentSub.Hostname),
					fmt.Sprintf("https://%s", currentSub.Hostname),
				}
				for _, urlStr := range urlsToTry {
					if ShouldScreenshot(urlStr) {
						initialScreenshotWG.Add(1)
						go func(targetURL string, subID uint) {
							defer initialScreenshotWG.Done()
							screenshotCtx := context.Background()
							err := TakeScreenshot(screenshotCtx, targetURL, scanID, &subID, nil)
							if err != nil {
								log.Printf("Initial screenshot attempt finished for %s (Subdomain ID: %d, Scan ID: %d) - see previous logs for details.", targetURL, subID, scanID)
							}
						}(urlStr, currentSub.ID)
					}
				}
			}
		}

		// Fetch existing endpoints (and their subdomains for URL construction)
		var existingEndpointsDB []models.Endpoint
		// Get Subdomain IDs first
		subdomainIDs := make([]uint, len(existingSubdomainsDB))
		for i, sub := range existingSubdomainsDB {
			subdomainIDs[i] = sub.ID
		}

		if len(subdomainIDs) > 0 {
			if err := db.Preload("Subdomain").Where("subdomain_id IN ?", subdomainIDs).Find(&existingEndpointsDB).Error; err != nil {
				log.Printf("Error fetching existing endpoints for screenshotting (Scan ID: %d): %v", scanID, err)
			} else {
				log.Printf("Found %d existing endpoints to potentially screenshot.", len(existingEndpointsDB))
				for _, ep := range existingEndpointsDB {
					// Need loop variable copy
					currentEp := ep
					if currentEp.Subdomain.Hostname == "" || currentEp.Path == "" {
						continue // Skip if essential info is missing
					}
					// Construct URL (try https first, then http?) - Let's try both like subdomains
					path := currentEp.Path
					if !strings.HasPrefix(path, "/") {
						path = "/" + path
					}
					urlsToTry := []string{
						fmt.Sprintf("http://%s%s", currentEp.Subdomain.Hostname, path),
						fmt.Sprintf("https://%s%s", currentEp.Subdomain.Hostname, path),
					}
					for _, urlStr := range urlsToTry {
						if ShouldScreenshot(urlStr) {
							initialScreenshotWG.Add(1)
							go func(targetURL string, endpointID uint) {
								defer initialScreenshotWG.Done()
								screenshotCtx := context.Background()
								err := TakeScreenshot(screenshotCtx, targetURL, scanID, nil, &endpointID)
								if err != nil {
									log.Printf("Initial screenshot attempt finished for %s (Endpoint ID: %d, Scan ID: %d) - see previous logs for details.", targetURL, endpointID, scanID)
								}
							}(urlStr, currentEp.ID)
						}
					}
				}
			}
		}
		// Wait for initial screenshots before proceeding with discovery phases?
		// This ensures existing assets are attempted even if discovery is off.
		log.Printf("Waiting for initial screenshot tasks to complete for scan %d...", scanID)
		initialScreenshotWG.Wait()
		log.Printf("Initial screenshot tasks finished for scan %d.", scanID)
	}
	// --- End Screenshot Existing Assets ---

	// Context with timeout for the entire subdomain scan phase (consider making this configurable too?)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute) // Increased default timeout slightly
	defer cancel()

	allSubdomains := make(map[string]struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex // Mutex to protect access to shared resources (scanErrors, maps)
	var scanErrors []string
	activeSubdomains := make(map[string]struct{}) // Map of active subdomains found/targeted
	savedSubdomainMap := make(map[string]uint)    // Map of hostname -> saved ID

	if scanType == "root_domain" {
		// --- Root Domain Scan: Discover and Verify ---
		// Use the 'allSubdomains' map declared earlier (line 633)
		// allSubdomains := make(map[string]struct{}) // REMOVE THIS REDECLARATION

		// Run Subfinder (if enabled in parsed config)
		if subfinderEnabled {
			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Printf("Running subfinder for %s...", targetHost)
				subfinderTimeout := time.Duration(getIntOption(subfinderOptions, "maxEnumerationTime", 5)+1) * time.Minute
				subfinderCtx, subfinderCancel := context.WithTimeout(ctx, subfinderTimeout)
				defer subfinderCancel()
				subs, err := runSubfinder(subfinderCtx, targetHost, subfinderOptions)
				mu.Lock()
				if err != nil {
					log.Printf("Subfinder error for %s: %v", targetHost, err)
					scanErrors = append(scanErrors, fmt.Sprintf("Subfinder: %v", err))
				} else if subs != nil {
					log.Printf("Subfinder found %d results for %s.", len(subs), targetHost)
					for sub := range subs {
						allSubdomains[sub] = struct{}{}
					}
				}
				mu.Unlock()
			}()
		} else {
			log.Printf("Subfinder skipped for scan %d (disabled in template or not root_domain scan).", scanID)
		}

		wg.Wait() // Wait for discovery phase

		// Ensure the root domain itself is included
		mu.Lock()
		if _, exists := allSubdomains[targetHost]; !exists {
			log.Printf("Explicitly adding root domain '%s' to potential list for scan %d", targetHost, scanID)
			allSubdomains[targetHost] = struct{}{}
		}
		mu.Unlock()

		log.Printf("Found %d unique potential subdomains in total for %s (Scan ID: %d). Verifying active hosts...", len(allSubdomains), targetHost, scanID)

		// Verify Active Subdomains using httpx
		verifiedSubs, verifyErr := verifyActiveSubdomains(ctx, allSubdomains)
		if verifyErr != nil {
			log.Printf("Error verifying active subdomains for scan %d: %v", scanID, verifyErr)
			mu.Lock()
			scanErrors = append(scanErrors, fmt.Sprintf("Subdomain verification: %v", verifyErr))
			mu.Unlock()
		}
		activeSubdomains = verifiedSubs // Assign verified results

		// Ensure the root domain itself is considered "active" if it was in the original list
		mu.Lock()
		if _, existsInOriginal := allSubdomains[targetHost]; existsInOriginal {
			if _, existsInActive := activeSubdomains[targetHost]; !existsInActive {
				log.Printf("Explicitly re-adding root domain '%s' to active list for saving (Scan ID: %d)", targetHost, scanID)
				activeSubdomains[targetHost] = struct{}{}
			}
		}
		mu.Unlock()

	} else if scanType == "subdomain" {
		// --- Specific Subdomain Scan: Target is the only active one ---
		log.Printf("Targeting specific subdomain: %s (Scan ID: %d)", targetHost, scanID)
		activeSubdomains[targetHost] = struct{}{} // Only target the input host
	} else {
		// Should not happen if called correctly from handler
		log.Printf("Error: Unknown scanType '%s' for scan ID %d", scanType, scanID)
		updateScanStatus(db, scanID, "failed", fmt.Sprintf("Internal error: Unknown scanType '%s'", scanType))
		return
	}

	// --- Save Active/Targeted Subdomains ---
	if len(activeSubdomains) > 0 {
		log.Printf("Saving %d active/targeted subdomains for %s (Scan ID: %d)", len(activeSubdomains), targetHost, scanID)
		var saveErr error
		savedSubdomainMap, saveErr = saveSubdomains(db, rootDomainID, scanID, activeSubdomains) // Use activeSubdomains map
		if saveErr != nil {
			log.Printf("Error saving active subdomains or fetching their IDs for scan %d: %v", scanID, saveErr)
			mu.Lock()
			scanErrors = append(scanErrors, fmt.Sprintf("Subdomain Save/ID Fetch: %v", saveErr))
			mu.Unlock()
		}
	} else {
		log.Printf("No active/targeted subdomains to save for scan %d.", scanID)
	}

	// --- Take Screenshots (if enabled and subdomains were saved/fetched) ---
	if scanTemplate.ScreenshotEnabled && len(savedSubdomainMap) > 0 {
		log.Printf("Screenshotting enabled for scan %d. Starting screenshot process for %d saved/fetched subdomains.", scanID, len(savedSubdomainMap))
		var screenshotWG sync.WaitGroup

		for hostname, subID := range savedSubdomainMap { // Iterate over the map of saved hostnames and their IDs
			urlsToTry := []string{
				fmt.Sprintf("http://%s", hostname), // Use hostname from the map key
				fmt.Sprintf("https://%s", hostname),
			}

			for _, urlStr := range urlsToTry {
				if ShouldScreenshot(urlStr) {
					screenshotWG.Add(1)
					go func(targetURL string, currentSubID uint) {
						defer screenshotWG.Done()
						// semaphore <- struct{}{} // Acquire semaphore slot
						// defer func() { <-semaphore }() // Release semaphore slot

						// Use a separate context for each screenshot task? Or reuse the main scan context?
						// Reusing main context might cause issues if it times out early.
						// Create a new background context for robustness.
						screenshotCtx := context.Background()                                       // Use background context for independence
						err := TakeScreenshot(screenshotCtx, targetURL, scanID, &currentSubID, nil) // Pass subdomain ID
						if err != nil {
							// TakeScreenshot already logs errors, no need to log again unless adding context
							log.Printf("Screenshot attempt finished for %s (Subdomain ID: %d, Scan ID: %d) - see previous logs for details.", targetURL, currentSubID, scanID)
							// Optionally add screenshot errors to scanErrors?
							// mu.Lock()
							// scanErrors = append(scanErrors, fmt.Sprintf("Screenshot %s: %v", targetURL, err))
							// mu.Unlock()
						}
					}(urlStr, subID)
				}
			}
		}
		log.Printf("Waiting for screenshot tasks to complete for scan %d...", scanID)
		screenshotWG.Wait()
		log.Printf("Screenshot tasks finished for scan %d.", scanID)
	} else if scanTemplate.ScreenshotEnabled {
		log.Printf("Screenshotting enabled for scan %d, but no active subdomains were successfully saved with IDs.", scanID)
	} else {
		log.Printf("Screenshotting disabled for scan %d.", scanID)
	}
	// --- End Screenshotting ---

	// Update final status
	finalStatus := "completed" // Assume success initially
	errMsg := ""
	if len(scanErrors) > 0 {
		finalStatus = "failed" // Mark as failed if any step had errors
		errMsg = strings.Join(scanErrors, "; ")
		log.Printf("Subdomain scan %d finished with errors: %s", scanID, errMsg)
	} else {
		log.Printf("Subdomain scan %d completed successfully.", scanID)
	}

	// --- Prepare for and Execute URL Scan (if enabled) ---
	if urlScanEnabled {
		// Prepare the map of existing/target subdomains for URL scanner
		urlScanSubdomainMap := &sync.Map{}
		for host, id := range savedSubdomainMap {
			urlScanSubdomainMap.Store(host, id) // Use the IDs we got after saving
		}

		// Prepare seed URLs based on scan type
		var seedURLs []string
		if scanType == "root_domain" {
			// Seed with the root domain and all active/saved subdomains
			seedURLs = append(seedURLs, fmt.Sprintf("http://%s", targetHost))
			seedURLs = append(seedURLs, fmt.Sprintf("https://%s", targetHost))
			for host := range activeSubdomains {
				if host != targetHost { // Avoid adding root domain again
					seedURLs = append(seedURLs, fmt.Sprintf("http://%s", host))
					seedURLs = append(seedURLs, fmt.Sprintf("https://%s", host))
				}
			}
		} else { // scanType == "subdomain"
			// Seed only with the target subdomain
			seedURLs = append(seedURLs, fmt.Sprintf("http://%s", targetHost))
			seedURLs = append(seedURLs, fmt.Sprintf("https://%s", targetHost))
		}

		log.Printf("Starting URL scan phase for scan %d with %d seeds.", scanID, len(seedURLs))
		// Pass the correct targetHost (which is the root domain name for context)
		urlScanErr := ExecuteURLScan(seedURLs, targetHost, rootDomainID, scanID, urlScanSubdomainMap, scanTemplate, katanaOptions, katanaOutputFile)
		if urlScanErr != nil {
			log.Printf("URL scan phase for scan %d finished with error: %v", scanID, urlScanErr)
			mu.Lock()
			scanErrors = append(scanErrors, fmt.Sprintf("URL Scan: %v", urlScanErr))
			mu.Unlock()
		} else {
			log.Printf("URL scan phase for scan %d finished.", scanID)
		}
	} else {
		log.Printf("URL Scan skipped for scan %d (disabled in template).", scanID)
	}

	// --- Execute Technology Detection (if enabled) ---
	if scanTemplate.TechDetectEnabled {
		log.Printf("Technology detection enabled for scan %d. Gathering target URLs...", scanID)

		// --- Gather Target URLs ---
		var urlsToScanSet map[string]struct{} // Use a set to avoid duplicates

		if scanType == "root_domain" {
			// Fetch all subdomains and endpoints for the root domain ID from the DB
			// (This logic remains the same as before for root domain scans)
			var allDbSubdomains []models.Subdomain
			if err := db.Where("root_domain_id = ?", rootDomainID).Find(&allDbSubdomains).Error; err != nil {
				log.Printf("Error fetching subdomains for tech scan (Scan ID: %d): %v", scanID, err)
				mu.Lock()
				scanErrors = append(scanErrors, fmt.Sprintf("Tech Detect Target Fetch (Subdomains): %v", err))
				mu.Unlock()
			}
			var allDbEndpoints []models.Endpoint
			subdomainIDs := make([]uint, len(allDbSubdomains))
			for i, sub := range allDbSubdomains {
				subdomainIDs[i] = sub.ID
			}
			if len(subdomainIDs) > 0 {
				if err := db.Preload("Subdomain").Where("subdomain_id IN ?", subdomainIDs).Find(&allDbEndpoints).Error; err != nil {
					log.Printf("Error fetching endpoints for tech scan (Scan ID: %d): %v", scanID, err)
					mu.Lock()
					scanErrors = append(scanErrors, fmt.Sprintf("Tech Detect Target Fetch (Endpoints): %v", err))
					mu.Unlock()
				}
			} else {
				log.Printf("No subdomains found for RootDomainID %d, skipping endpoint fetch for tech scan.", rootDomainID)
			}

			urlsToScanSet = make(map[string]struct{})
			for _, sub := range allDbSubdomains {
				urlsToScanSet["http://"+sub.Hostname] = struct{}{}
				urlsToScanSet["https://"+sub.Hostname] = struct{}{}
			}
			for _, ep := range allDbEndpoints {
				if ep.Subdomain.Hostname != "" && ep.Path != "" {
					path := ep.Path
					if !strings.HasPrefix(path, "/") {
						path = "/" + path
					}
					urlsToScanSet["http://"+ep.Subdomain.Hostname+path] = struct{}{}
					urlsToScanSet["https://"+ep.Subdomain.Hostname+path] = struct{}{}
				}
			}
		} else { // scanType == "subdomain"
			// Only target the specific subdomain and its discovered endpoints
			urlsToScanSet = make(map[string]struct{})
			urlsToScanSet["http://"+targetHost] = struct{}{}
			urlsToScanSet["https://"+targetHost] = struct{}{}

			// Fetch endpoints ONLY for the target subdomain ID
			targetSubdomainID, ok := savedSubdomainMap[targetHost]
			if !ok {
				log.Printf("Warning: Could not find saved ID for target subdomain %s for tech scan (Scan ID: %d). Fetching endpoints might fail.", targetHost, scanID)
				// Attempt to fetch ID again? Or skip endpoint tech scan? Let's try fetching.
				var subModel models.Subdomain
				if res := db.Where("hostname = ? AND root_domain_id = ?", targetHost, rootDomainID).First(&subModel); res.Error == nil {
					targetSubdomainID = subModel.ID
					ok = true
				} else {
					log.Printf("Error re-fetching ID for target subdomain %s: %v", targetHost, res.Error)
				}
			}

			if ok {
				var targetEndpoints []models.Endpoint
				if err := db.Where("subdomain_id = ?", targetSubdomainID).Find(&targetEndpoints).Error; err != nil {
					log.Printf("Error fetching endpoints for specific subdomain tech scan (Subdomain ID: %d, Scan ID: %d): %v", targetSubdomainID, scanID, err)
					mu.Lock()
					scanErrors = append(scanErrors, fmt.Sprintf("Tech Detect Target Fetch (Endpoints for %s): %v", targetHost, err))
					mu.Unlock()
				} else {
					for _, ep := range targetEndpoints {
						if ep.Path != "" {
							path := ep.Path
							if !strings.HasPrefix(path, "/") {
								path = "/" + path
							}
							urlsToScanSet["http://"+targetHost+path] = struct{}{}
							urlsToScanSet["https://"+targetHost+path] = struct{}{}
						}
					}
				}
			}
		}
		// --- End Target URL Gathering ---

		// Convert set to slice
		finalUrlsToScan := make([]string, 0, len(urlsToScanSet))
		for urlStr := range urlsToScanSet {
			finalUrlsToScan = append(finalUrlsToScan, urlStr)
		}

		if len(finalUrlsToScan) == 0 {
			log.Printf("No target URLs gathered for technology detection (Scan ID: %d). Skipping phase.", scanID)
		} else {
			log.Printf("Starting technology detection phase for scan %d on %d unique URLs.", scanID, len(finalUrlsToScan))
			techScanErr := ExecuteTechScan(finalUrlsToScan, scanID, rootDomainID) // Pass rootDomainID for context
			if techScanErr != nil {
				log.Printf("Technology detection phase for scan %d finished with error: %v", scanID, techScanErr)
				mu.Lock()
				scanErrors = append(scanErrors, fmt.Sprintf("Tech Detect: %v", techScanErr))
				mu.Unlock()
			} else {
				log.Printf("Technology detection phase for scan %d finished.", scanID)
			}
		}
	} else {
		log.Printf("Technology detection skipped for scan %d (disabled in template).", scanID)
	}

	// --- Update Final Status ---
	finalStatus = "completed" // Use '=' as it's already declared
	errMsg = ""               // Use '=' as it's already declared
	mu.Lock()                 // Lock before checking scanErrors
	if len(scanErrors) > 0 {
		finalStatus = "failed"
		errMsg = strings.Join(scanErrors, "; ")
		log.Printf("Scan %d finished with errors: %s", scanID, errMsg)
	} else {
		errMsg = "Scan completed successfully" // Set success message only if no errors
		log.Printf("Scan %d completed successfully.", scanID)
	}
	mu.Unlock() // Unlock after checking scanErrors

	updateScanStatus(db, scanID, finalStatus, errMsg)
}
