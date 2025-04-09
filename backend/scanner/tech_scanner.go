package scanner

import (
	"context"
	"errors" // Ensure errors is imported
	"fmt"
	"io" // Re-add io for sequential processing
	"log"
	"math/rand"
	"net/http"
	"net/url" // Added for URL parsing
	"rewrite-go/database"
	"rewrite-go/models"
	"strings"
	"time"

	wappalyzergo "github.com/projectdiscovery/wappalyzergo" // Revert alias
	"gorm.io/gorm"
)

const techDetectTimeout = 30 // Timeout in seconds for fetching a single URL

// ExecuteTechScan performs technology detection on a list of URLs sequentially.
func ExecuteTechScan(urls []string, scanID uint, rootDomainID uint) error {
	db := database.GetDB()
	if len(urls) == 0 {
		log.Printf("No URLs provided for technology detection (Scan ID: %d). Skipping.", scanID)
		return nil
	}
	log.Printf("Starting technology detection for %d URLs (Scan ID: %d)", len(urls), scanID)

	wappalyzerClient, err := wappalyzergo.New()
	if err != nil {
		log.Printf("Error creating Wappalyzer client for scan %d: %v", scanID, err)
		return fmt.Errorf("failed to create wappalyzer client: %w", err)
	}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Define a list of common user agents
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/109.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/109.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/109.0",
	}

	// --- Sequential Processing ---
	// Store results keyed by the original URL processed
	allResultsByURL := make(map[string]map[string]struct{})
	var scanErrors []error

	httpClient := &http.Client{
		Timeout: time.Duration(techDetectTimeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	log.Printf("Processing %d URLs sequentially for technology detection (Scan ID: %d)...", len(urls), scanID)

	for _, urlStr := range urls {
		var detectedTechs map[string]struct{}
		var fetchErr error

		// Process the single provided URL
		req, err := http.NewRequestWithContext(context.Background(), "GET", urlStr, nil)
		if err != nil {
			fetchErr = fmt.Errorf("failed to create request for %s: %w", urlStr, err)
			log.Printf("Error processing URL %s (Scan ID: %d): %v", urlStr, scanID, fetchErr)
			scanErrors = append(scanErrors, fmt.Errorf("url %s: %w", urlStr, fetchErr))
			continue // Move to next URL
		}
		// Select a random user agent
		randomUserAgent := userAgents[rand.Intn(len(userAgents))]
		req.Header.Set("User-Agent", randomUserAgent)
		// log.Printf("Using User-Agent: %s for URL: %s", randomUserAgent, urlStr) // Optional: Log the user agent being used

		resp, err := httpClient.Do(req)
		if err != nil {
			fetchErr = fmt.Errorf("failed to fetch %s: %w", urlStr, err)
			log.Printf("Error processing URL %s (Scan ID: %d): %v", urlStr, scanID, fetchErr)
			scanErrors = append(scanErrors, fmt.Errorf("url %s: %w", urlStr, fetchErr))
			continue // Move to next URL
		}

		// Read body
		limitedReader := &io.LimitedReader{R: resp.Body, N: 1 * 1024 * 1024} // Limit read size
		data, err := io.ReadAll(limitedReader)
		resp.Body.Close() // Close body immediately
		if err != nil && err != io.EOF {
			fetchErr = fmt.Errorf("failed to read body for %s: %w", urlStr, err)
			log.Printf("Error processing URL %s (Scan ID: %d): %v", urlStr, scanID, fetchErr)
			scanErrors = append(scanErrors, fmt.Errorf("url %s: %w", urlStr, fetchErr))
			continue // Move to next URL
		}

		// Run Wappalyzer fingerprinting
		fingerprints := wappalyzerClient.Fingerprint(resp.Header, data)

		if len(fingerprints) > 0 {
			detectedTechs = fingerprints
			log.Printf("Detected %d technologies on %s (Scan ID: %d)", len(detectedTechs), urlStr, scanID)
			allResultsByURL[urlStr] = detectedTechs // Store results keyed by URL
		} else {
			// Log that no techs were detected, but don't treat as a fatal error for the scan job
			log.Printf("Info: No technologies detected on %s (Scan ID: %d, Status: %d)", urlStr, scanID, resp.StatusCode)
		}
	} // end loop (urlStr)

	// --- Save Results ---
	saveErr := saveTechnologies(db, allResultsByURL, scanID, rootDomainID) // Pass the URL-keyed map
	if saveErr != nil {
		// Append save error to any scan errors encountered
		scanErrors = append(scanErrors, fmt.Errorf("failed to save technologies: %w", saveErr))
	}

	// --- Final Error Handling ---
	if len(scanErrors) > 0 {
		log.Printf("Technology detection for scan %d finished with %d errors.", scanID, len(scanErrors))
		// Combine errors? For now, return the first one.
		// Consider using multierr package if more granular error reporting is needed.
		return fmt.Errorf("technology detection encountered errors: %w", scanErrors[0])
	}

	log.Printf("Technology detection for scan %d completed successfully.", scanID)
	return nil
}

// saveTechnologies saves the detected technologies using join table entries.
// It now accepts results keyed by URL and extracts the hostname for linking.
func saveTechnologies(db *gorm.DB, resultsByURL map[string]map[string]struct{}, scanID uint, rootDomainID uint) error {
	if len(resultsByURL) == 0 {
		log.Printf("No technologies found to save for scan %d.", scanID)
		return nil
	}

	// Use transaction for atomicity
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // Re-panic after rollback
		} else if tx.Error != nil {
			tx.Rollback() // Rollback if any GORM error occurred
		}
	}()

	// --- Pre-fetch necessary data ---
	// Fetch Root Domain info
	var rootDomain models.RootDomain
	if err := tx.First(&rootDomain, rootDomainID).Error; err != nil {
		return fmt.Errorf("failed to fetch root domain %d: %w", rootDomainID, err)
	}

	// Fetch existing subdomains for the root domain to get their IDs
	var existingSubdomains []models.Subdomain
	if err := tx.Where("root_domain_id = ?", rootDomainID).Find(&existingSubdomains).Error; err != nil {
		log.Printf("Warning: Error fetching existing subdomains for root domain %d: %v", rootDomainID, err)
		// Continue, but might miss linking some technologies
	}
	subdomainIDMap := make(map[string]uint)
	for _, sub := range existingSubdomains {
		subdomainIDMap[sub.Hostname] = sub.ID
	}

	// Ensure a Subdomain entry exists for the root domain itself for linking
	var rootSubdomain models.Subdomain
	err := tx.Where("root_domain_id = ? AND hostname = ?", rootDomainID, rootDomain.Domain).First(&rootSubdomain).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Creating missing Subdomain entry for root domain host: %s", rootDomain.Domain)
		rootSubdomain = models.Subdomain{
			RootDomainID: rootDomainID,
			Hostname:     rootDomain.Domain,
			IsActive:     true, // Assume active
			DiscoveredAt: time.Now(),
			ScanID:       &scanID, // Associate with this scan
		}
		if err := tx.Create(&rootSubdomain).Error; err != nil {
			return fmt.Errorf("failed to create subdomain entry for root domain %s: %w", rootDomain.Domain, err)
		}
		subdomainIDMap[rootSubdomain.Hostname] = rootSubdomain.ID // Add to map
	} else if err != nil {
		return fmt.Errorf("failed to query subdomain entry for root domain %s: %w", rootDomain.Domain, err)
	} else {
		// Ensure it's in the map if fetched successfully
		subdomainIDMap[rootSubdomain.Hostname] = rootSubdomain.ID
	}

	// --- Process and Save Technologies ---
	var joinEntriesToCreate []models.SubdomainTechnology
	processedTechs := make(map[string]uint) // Cache found/created tech IDs: name -> ID
	now := time.Now()

	for urlStr, techs := range resultsByURL {
		// --- Extract Hostname from URL ---
		parsedURL, err := url.Parse(urlStr) // Use standard library url package
		if err != nil {
			log.Printf("Warning: Failed to parse URL '%s' to extract hostname. Skipping tech linking for this URL. Error: %v", urlStr, err)
			continue
		}
		host := parsedURL.Hostname()
		if host == "" {
			log.Printf("Warning: Could not extract hostname from URL '%s'. Skipping tech linking.", urlStr)
			continue
		}
		// --- End Hostname Extraction ---

		// Find the Subdomain ID for the extracted host
		subdomainID, ok := subdomainIDMap[host]
		if !ok {
			// This might happen if a subdomain was discovered *after* the initial fetch
			// or if it's a host not directly under the root domain (e.g., from URL scan)
			// Log a warning and skip linking techs for this host.
			log.Printf("Warning: Could not find Subdomain ID for host '%s' (from URL '%s') in map for RootDomainID %d. Skipping tech linking for this URL.", host, urlStr, rootDomainID)
			continue
		}

		for techName := range techs {
			normalizedTechName := strings.ToLower(techName)
			technologyID, techExists := processedTechs[normalizedTechName]

			if !techExists {
				// Try to find existing technology
				var technology models.Technology
				err := tx.Where("name = ?", normalizedTechName).First(&technology).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// Technology doesn't exist, create it
					technology = models.Technology{Name: normalizedTechName}
					// TODO: Add category lookup if possible/needed
					if err := tx.Create(&technology).Error; err != nil {
						log.Printf("Error creating technology '%s': %v. Skipping this tech for URL %s.", normalizedTechName, err, urlStr)
						continue // Skip this technology
					}
					log.Printf("Created new technology entry: %s (ID: %d)", normalizedTechName, technology.ID)
					technologyID = technology.ID
					processedTechs[normalizedTechName] = technologyID
				} else if err != nil {
					log.Printf("Error querying technology '%s': %v. Skipping this tech for URL %s.", normalizedTechName, err, urlStr)
					continue // Skip this technology
				} else {
					// Technology found
					technologyID = technology.ID
					processedTechs[normalizedTechName] = technologyID
				}
			}

			// Create the join table entry
			joinEntry := models.SubdomainTechnology{
				SubdomainID:  subdomainID,
				TechnologyID: technologyID,
				DetectedAt:   now,
				// ScanID: &scanID, // Add ScanID if the join table schema supports it
				// Confidence: // Add confidence if wappalyzergo provides it
			}
			joinEntriesToCreate = append(joinEntriesToCreate, joinEntry)
		}
	}

	if len(joinEntriesToCreate) == 0 {
		log.Printf("No valid technology relationships to save for scan %d.", scanID)
		// No need to commit if nothing was changed besides potentially creating the root subdomain entry
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit transaction after finding no tech relationships: %w", err)
		}
		return nil
	}

	log.Printf("Saving %d technology relationships for scan %d...", len(joinEntriesToCreate), scanID)

	// Batch insert join table entries, ignoring conflicts on (SubdomainID, TechnologyID)
	// This assumes a unique constraint exists on these two columns in SubdomainTechnology.
	// Use Clauses(clause.OnConflict{DoNothing: true}) for PostgreSQL/SQLite
	// Use Clauses(clause.Insert{Modifier: "IGNORE"}) or similar for MySQL - check GORM docs
	// Using DoNothing for broad compatibility assumption.
	// result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "subdomain_id"}, {Name: "technology_id"}}, DoNothing: true}).CreateInBatches(joinEntriesToCreate, 100)

	// Simpler approach without explicit conflict handling (relies on DB constraints or accepts potential duplicates if constraints are missing)
	result := tx.CreateInBatches(joinEntriesToCreate, 100)

	if result.Error != nil {
		// Rollback is handled by defer
		return fmt.Errorf("failed to save technology relationships: %w", result.Error)
	}

	log.Printf("Successfully saved %d technology relationships for scan %d.", result.RowsAffected, scanID)

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
