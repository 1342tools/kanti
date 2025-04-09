package scanner

import (
	"context" // Ensure context is imported
	"fmt"
	"log"
	"net/url"
	"rewrite-go/database"
	"rewrite-go/models"

	// "strconv" // Removed
	// "strings" // Removed unused import
	"sync"
	"time"

	"github.com/projectdiscovery/katana/pkg/engine/standard"
	"github.com/projectdiscovery/katana/pkg/output"
	"github.com/projectdiscovery/katana/pkg/types"

	// wappalyzer "github.com/projectdiscovery/wappalyzergo" // No longer needed here
	"github.com/weppos/publicsuffix-go/publicsuffix"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- URL Scan Specific Structs and Functions ---

// urlScanResult holds processed data from a Katana result.
type urlScanResult struct {
	Hostname string // Store the actual hostname found
	Endpoint models.Endpoint
	Params   []models.Parameter
	FullURL  string // Store the original full URL for screenshotting
}

// processKatanaOutput is the callback function for Katana results.
// It parses the URL, extracts relevant information, and sends it to a channel for processing.
// It should NOT modify existingSubdomains map.
func processKatanaOutput(result output.Result, rootDomain string, rootDomainID uint, scanID uint, resultsChan chan<- urlScanResult, existingSubdomains *sync.Map) { // existingSubdomains map is read-only here now
	// Basic filtering
	if result.Request == nil || result.Response == nil || result.Response.StatusCode < 200 || result.Response.StatusCode >= 400 {
		return
	}

	parsedURL, err := url.Parse(result.Request.URL)
	if err != nil {
		log.Printf("Error parsing URL %s: %v", result.Request.URL, err)
		return
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		log.Printf("Could not extract hostname from URL: %s", result.Request.URL)
		return
	}

	// Check if the hostname belongs to the target root domain using publicsuffix
	parsedHostDomain, err := publicsuffix.Parse(hostname)
	if err != nil {
		// log.Printf("Could not parse hostname %s for root domain check: %v", hostname, err)
		return // Skip if we can't parse
	}
	// Handle cases like "domain.co.uk" where SLD is "domain"
	hostRootDomain := parsedHostDomain.SLD + "." + parsedHostDomain.TLD
	if parsedHostDomain.SLD == "" { // Handle cases like "com.au" if parsed directly
		hostRootDomain = hostname
	}

	if hostRootDomain != rootDomain {
		// log.Printf("Skipping URL %s: Host %s (root: %s) does not belong to target root domain %s", result.Request.URL, hostname, hostRootDomain, rootDomain)
		return // Skip URLs not belonging to the target root domain
	}

	// Don't modify existingSubdomains here. Let saveURLScanResults handle it.

	res := urlScanResult{
		Hostname: hostname,           // Pass the actual hostname
		FullURL:  result.Request.URL, // Store the original URL
		Endpoint: models.Endpoint{
			// SubdomainID will be filled later by saveURLScanResults
			Path:         parsedURL.Path,
			Method:       result.Request.Method,
			StatusCode:   result.Response.StatusCode,
			ContentType:  result.Response.Headers["Content-Type"],
			DiscoveredAt: time.Now(),
			ScanID:       &scanID,
		},
	}

	// Extract Parameters
	queryParams := parsedURL.Query()
	for name, values := range queryParams {
		// Store only the first value for simplicity, or handle multiple values if needed
		if len(values) > 0 {
			res.Params = append(res.Params, models.Parameter{
				Name:      name,
				ParamType: "query", // Katana primarily finds query params
				// EndpointID will be set after Endpoint creation
				DiscoveredAt: time.Now(),
			})
		}
	}
	// TODO: Potentially parse body for parameters if needed and available in result

	resultsChan <- res
}

// saveURLScanResults processes results from the channel and saves them to the DB.
// Added screenshotEnabled bool parameter.
func saveURLScanResults(db *gorm.DB, rootDomain string, rootDomainID uint, scanID uint, resultsChan <-chan urlScanResult, wg *sync.WaitGroup, existingSubdomains *sync.Map, screenshotEnabled bool) {
	defer wg.Done()
	var newSubdomainsToCreate []models.Subdomain
	var endpointsToCreate []models.Endpoint                  // Holds endpoints collected during the run
	var endpointOriginalURLs = make(map[int]string)          // Map index in endpointsToCreate to its original URL
	var endpointParamsMap = make(map[int][]models.Parameter) // Map index in endpointsToCreate to its params
	var endpointHostnameMap = make(map[int]string)           // Map index in endpointsToCreate to its hostname

	subdomainMap := make(map[string]uint) // Map hostname to known Subdomain ID (from DB or newly created)
	var screenshotWG sync.WaitGroup       // WaitGroup for screenshot goroutines

	// Load existing subdomain IDs from DB into both maps
	var existingDBSubdomains []models.Subdomain
	db.Where("root_domain_id = ?", rootDomainID).Find(&existingDBSubdomains)
	for _, sub := range existingDBSubdomains {
		subdomainMap[sub.Hostname] = sub.ID
		existingSubdomains.Store(sub.Hostname, sub.ID) // Store actual uint ID
	}

	endpointIndex := 0 // Counter for maps keyed by index

	// --- Collect results from channel ---
	for res := range resultsChan {
		currentHostname := res.Hostname

		// Use LoadOrStore to atomically check/create placeholder uint(0)
		// This map tracks subdomains seen *during this URL scan* or loaded from DB.
		idVal, loaded := existingSubdomains.LoadOrStore(currentHostname, uint(0))

		// Check the type returned by LoadOrStore - crucial step!
		_, idIsUint := idVal.(uint) // We don't need the ID here, just the type check
		if !idIsUint {
			log.Printf("CRITICAL: Non-uint value found in existingSubdomains map after LoadOrStore for key '%s': %T. Skipping result.", currentHostname, idVal)
			continue // Skip this result as the map state is corrupted
		}

		// If LoadOrStore stored uint(0), it means it was new *to this map*.
		// Add it to the creation list if it's not the root domain.
		if !loaded && currentHostname != rootDomain {
			// Check again to prevent duplicates in the slice if processed concurrently
			isAlreadyInCreateList := false
			for _, existingSub := range newSubdomainsToCreate {
				if existingSub.Hostname == currentHostname {
					isAlreadyInCreateList = true
					break
				}
			}
			if !isAlreadyInCreateList {
				newSubdomainsToCreate = append(newSubdomainsToCreate, models.Subdomain{
					Hostname: currentHostname, RootDomainID: rootDomainID, ScanID: &scanID, DiscoveredAt: time.Now(), IsActive: true,
				})
			}
		}

		// Store endpoint, params, hostname, and original URL together for later processing
		// SubdomainID is not set here yet.
		endpointsToCreate = append(endpointsToCreate, res.Endpoint)
		endpointParamsMap[endpointIndex] = res.Params
		endpointHostnameMap[endpointIndex] = currentHostname // Store hostname for this endpoint index
		endpointOriginalURLs[endpointIndex] = res.FullURL    // Store original URL
		endpointIndex++
	}
	// --- End collecting results ---

	// --- Batch Create New Subdomains ---
	if len(newSubdomainsToCreate) > 0 {
		log.Printf("URL Scan: Saving %d new subdomains for scan %d...", len(newSubdomainsToCreate), scanID)
		result := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "hostname"}, {Name: "root_domain_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"scan_id", "discovered_at", "is_active"}),
		}).Create(&newSubdomainsToCreate) // Create the list

		if result.Error != nil {
			log.Printf("Error saving new subdomains from URL scan %d: %v", scanID, result.Error)
		} else {
			log.Printf("URL Scan: Saved %d new subdomains for scan %d.", result.RowsAffected, scanID)
			// Update the maps with actual IDs for just created ones
			for _, sub := range newSubdomainsToCreate { // Iterate over the created slice
				if sub.ID != 0 {
					subdomainMap[sub.Hostname] = sub.ID
					existingSubdomains.Store(sub.Hostname, sub.ID) // Update sync.Map too
				}
			}
			// Optional: Re-fetch IDs for any potentially missed during conflict resolution
			var createdSubs []models.Subdomain
			hostnames := make([]string, 0, len(newSubdomainsToCreate))
			for _, s := range newSubdomainsToCreate {
				hostnames = append(hostnames, s.Hostname)
			}
			if len(hostnames) > 0 {
				db.Where("root_domain_id = ? AND hostname IN ?", rootDomainID, hostnames).Find(&createdSubs)
				for _, sub := range createdSubs {
					// Ensure map has the latest ID
					if _, ok := subdomainMap[sub.Hostname]; !ok || subdomainMap[sub.Hostname] == 0 {
						subdomainMap[sub.Hostname] = sub.ID
						existingSubdomains.Store(sub.Hostname, sub.ID)
					}
				}
			}
		}
		// --- Force Refresh Subdomain Map After Creation/Updates ---
		log.Printf("URL Scan: Refreshing subdomain map for root domain ID %d after potential creations...", rootDomainID)
		subdomainMap = make(map[string]uint) // Clear the map
		var allCurrentSubdomains []models.Subdomain
		db.Where("root_domain_id = ?", rootDomainID).Find(&allCurrentSubdomains)
		for _, sub := range allCurrentSubdomains {
			subdomainMap[sub.Hostname] = sub.ID
			existingSubdomains.Store(sub.Hostname, sub.ID) // Ensure sync.Map is also up-to-date
		}
		log.Printf("URL Scan: Refreshed subdomain map, found %d subdomains.", len(allCurrentSubdomains))
		// --- End Refresh ---

	}
	// --- End Batch Create New Subdomains ---

	// --- Prepare Final Endpoint List for Batch Create ---
	var finalEndpointsToCreate []models.Endpoint
	var finalEndpointParamsMap = make(map[int][]models.Parameter) // Map final index to original params
	var finalEndpointURLsMap = make(map[int]string)               // Map final index to original URL
	finalEndpointIndex := 0                                       // Index for the final lists

	// Note: The root domain check previously here is now implicitly handled
	// by the full refresh of subdomainMap above.

	// Iterate through originally collected endpoints and resolve SubdomainID
	for i, ep := range endpointsToCreate { // Iterate with original index 'i' from collection phase
		hostname := endpointHostnameMap[i] // Get the associated hostname using the original index

		// Find the correct SubdomainID from the refreshed map
		resolvedSubID, found := subdomainMap[hostname]

		// Fallback check to sync.Map (less likely needed now, but safe)
		if !found || resolvedSubID == 0 {
			idVal, loaded := existingSubdomains.Load(hostname)
			if loaded {
				id, isUint := idVal.(uint)
				if isUint && id != 0 {
					resolvedSubID = id
					found = true
					log.Printf("Debug: Resolved SubdomainID %d for host %s from sync.Map fallback.", resolvedSubID, hostname)
				}
			}
		}

		if !found || resolvedSubID == 0 {
			// This should be much rarer now after the map refresh
			log.Printf("Warning: Could not resolve SubdomainID for endpoint %s on host %s (original index %d) even after map refresh. Skipping.", ep.Path, hostname, i)
			continue // Skip endpoint if we can't link it
		}

		ep.SubdomainID = resolvedSubID // Set the resolved ID

		// Clean up path if it contains the full URL
		parsedFinalURL, err := url.Parse(ep.Path)
		if err == nil && parsedFinalURL.IsAbs() {
			ep.Path = parsedFinalURL.Path
			if ep.Path == "" {
				ep.Path = "/"
			}
		}

		finalEndpointsToCreate = append(finalEndpointsToCreate, ep)
		finalEndpointParamsMap[finalEndpointIndex] = endpointParamsMap[i]  // Use the new index for params map
		finalEndpointURLsMap[finalEndpointIndex] = endpointOriginalURLs[i] // Use the new index for URL map
		finalEndpointIndex++
	}
	// --- End Preparing Final Endpoint List ---

	// --- Process Endpoints Individually ---
	log.Printf("URL Scan: Processing %d potential endpoints for scan %d...", len(finalEndpointsToCreate), scanID)
	savedEndpointCount := 0
	for i, ep := range finalEndpointsToCreate { // Use final index 'i'
		originalURL := finalEndpointURLsMap[i] // Get the original URL for screenshotting

		// Assign fields that should always be updated if found, or set if created
		updateAttrs := models.Endpoint{
			StatusCode:   ep.StatusCode,
			ContentType:  ep.ContentType,
			DiscoveredAt: ep.DiscoveredAt, // Update discovery time
			ScanID:       ep.ScanID,       // Update last scan ID
		}

		// Find based on unique key, create with all fields if not found, update specific fields if found
		// The 'ep' variable will be populated with the found or created record, including its ID.
		result := db.Where(models.Endpoint{
			SubdomainID: ep.SubdomainID,
			Path:        ep.Path,
			Method:      ep.Method,
		}).Assign(updateAttrs).FirstOrCreate(&ep)

		if result.Error != nil {
			log.Printf("Error saving/finding endpoint %s %s for subdomain %d: %v", ep.Method, ep.Path, ep.SubdomainID, result.Error)
			continue // Skip parameters and screenshots if endpoint failed
		}

		// Check if a record was actually affected (created or updated)
		if result.RowsAffected > 0 {
			savedEndpointCount++
		}

		// Ensure we have an ID before processing parameters or screenshots
		if ep.ID == 0 {
			log.Printf("Warning: Endpoint %s %s for subdomain %d did not get an ID after FirstOrCreate. Skipping parameter associations and screenshots.", ep.Method, ep.Path, ep.SubdomainID)
			continue
		}

		// --- Take Screenshot (if enabled and eligible) ---
		if screenshotEnabled && ShouldScreenshot(originalURL) {
			screenshotWG.Add(1)
			go func(targetURL string, currentEndpointID uint) {
				defer screenshotWG.Done()
				screenshotCtx := context.Background()
				// Pass nil for subdomainID, pass endpointID
				err := TakeScreenshot(screenshotCtx, targetURL, scanID, nil, &currentEndpointID)
				if err != nil {
					log.Printf("Screenshot attempt finished for %s (Endpoint ID: %d, Scan ID: %d) - see previous logs for details.", targetURL, currentEndpointID, scanID)
				}
			}(originalURL, ep.ID) // Pass the original URL and the confirmed endpoint ID
			time.Sleep(1 * time.Second) // Rate limit screenshots to 1 per second
		}
		// --- End Screenshot ---

		// --- Save Parameters (Associated with the current endpoint ID) ---
		if params, ok := finalEndpointParamsMap[i]; ok && len(params) > 0 { // Use final index 'i'
			for _, param := range params { // Process each parameter individually for simplicity
				param.EndpointID = ep.ID // Set the correct EndpointID

				paramUpdateAttrs := models.Parameter{
					DiscoveredAt: param.DiscoveredAt, // Update discovery time
					// Add other fields to update if needed
				}

				paramResult := db.Where(models.Parameter{
					EndpointID: param.EndpointID,
					Name:       param.Name,
					ParamType:  param.ParamType,
				}).Assign(paramUpdateAttrs).FirstOrCreate(&param) // param gets populated with ID

				if paramResult.Error != nil {
					log.Printf("Error saving/finding parameter '%s' (%s) for endpoint ID %d: %v", param.Name, param.ParamType, ep.ID, paramResult.Error)
					// Continue processing other parameters even if one fails
				}
			}
		}
	}
	log.Printf("URL Scan: Finished processing endpoints for scan %d. Saved/Updated %d endpoints.", scanID, savedEndpointCount)
	// --- End Process Endpoints Individually ---

	log.Printf("URL Scan: Waiting for screenshot tasks to complete for scan %d...", scanID)
	screenshotWG.Wait() // Wait for all screenshot goroutines to finish
	log.Printf("URL Scan: Screenshot tasks finished for scan %d.", scanID)

} // <<< Correct closing brace for saveURLScanResults

// ExecuteURLScan performs URL crawling starting from a list of seed URLs, using provided configuration.
// Added scanTemplate parameter.
func ExecuteURLScan(seedURLs []string, rootDomain string, rootDomainID uint, scanID uint, existingSubdomains *sync.Map, scanTemplate *models.ScanTemplate, config map[string]interface{}, outputFile string) error {
	log.Printf("Starting URL scan for scan %d with %d seed URLs...", scanID, len(seedURLs))
	if outputFile != "" {
		log.Printf("URL scan %d will output results to: %s", scanID, outputFile)
	}
	if scanTemplate == nil {
		return fmt.Errorf("internal error: ExecuteURLScan called with nil scanTemplate for Scan ID: %d", scanID)
	}
	if len(seedURLs) == 0 {
		log.Printf("No seed URLs provided for URL scan %d. Skipping.", scanID)
		return nil
	}

	db := database.GetDB()
	resultsChan := make(chan urlScanResult, 100) // Buffered channel
	var saveWg sync.WaitGroup

	// Start a goroutine to save results from the channel
	saveWg.Add(1)
	// Pass rootDomain string and screenshotEnabled flag to saveURLScanResults
	go saveURLScanResults(db, rootDomain, rootDomainID, scanID, resultsChan, &saveWg, existingSubdomains, scanTemplate.ScreenshotEnabled)

	// Extract Katana options from the config map using helpers
	maxDepth := getIntOption(config, "maxDepth", 3)
	concurrency := getIntOption(config, "concurrency", 10)
	parallelism := getIntOption(config, "parallelism", 10)
	rateLimit := getIntOption(config, "rateLimit", 150)
	timeout := getIntOption(config, "timeout", 10)
	// TODO: Add other Katana options if needed (e.g., strategy, fieldScope)

	log.Printf("Configuring Katana: Depth=%d, Concurrency=%d, Parallelism=%d, RateLimit=%d, Timeout=%ds",
		maxDepth, concurrency, parallelism, rateLimit, timeout)

	// Base Katana options
	options := &types.Options{
		MaxDepth:     maxDepth,
		FieldScope:   "rdn",           // Keep scope as root domain name (or make configurable via map?)
		BodyReadSize: 1 * 1024 * 1024, // Keep body read size limit (or make configurable?)
		Timeout:      timeout,
		Concurrency:  concurrency,
		Parallelism:  parallelism,
		RateLimit:    rateLimit,
		Strategy:     "depth-first", // Keep strategy (or make configurable?)
		Silent:       true,          // Keep silent
		NoScope:      false,         // Keep scope enforced
		OutputFile:   outputFile,    // Set the output file path
		OnResult: func(result output.Result) { // Callback for each found URL
			// Technology detection removed from here
			// log.Printf("sumshi") // Removed debug log
			// Send to processing channel (without fingerprints)
			processKatanaOutput(result, rootDomain, rootDomainID, scanID, resultsChan, existingSubdomains)
		},
	}

	crawlerOptions, err := types.NewCrawlerOptions(options)
	if err != nil {
		close(resultsChan) // Close channel before returning error
		saveWg.Wait()      // Wait for saver to finish
		return fmt.Errorf("failed to create crawler options: %w", err)
	}
	defer crawlerOptions.Close()

	crawler, err := standard.New(crawlerOptions)
	if err != nil {
		close(resultsChan)
		saveWg.Wait()
		return fmt.Errorf("failed to create standard crawler: %w", err)
	}
	defer crawler.Close()

	// Crawl each seed URL provided
	var crawlErr error
	for _, seed := range seedURLs {
		err = crawler.Crawl(seed) // Use Crawl method per seed URL
		if err != nil {
			log.Printf("Could not crawl seed %s for scan %d: %v", seed, scanID, err)
			// Collect errors? For now, just log and continue with other seeds.
			crawlErr = err // Store last error?
		}
	}
	if crawlErr != nil {
		log.Printf("URL scan %d finished with errors during crawling.", scanID)
	}

	// Close the results channel and wait for the saver goroutine to finish
	close(resultsChan)
	saveWg.Wait()

	log.Printf("URL scan %d finished.", scanID)
	return nil // Return nil even if crawler had errors, as some results might have been saved
}
