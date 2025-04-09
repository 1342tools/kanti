package handlers

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"rewrite-go/database" // Correct module path
	"rewrite-go/models"   // Correct module path
	"strings"

	"strconv" // Need this to convert org_id string to uint

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HandleImportURLs processes the uploaded text file containing URLs/subdomains for a specific organization.
func HandleImportURLs(c *gin.Context) {
	db := database.GetDB() // Get DB instance

	// Get Organization ID from URL path parameter
	orgIDStr := c.Param("org_id")
	orgID64, err := strconv.ParseUint(orgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Organization ID format"})
		return
	}
	orgID := uint(orgID64) // Convert to uint

	// Check if organization exists (optional but good practice)
	var org models.Organization
	if err := db.First(&org, orgID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Organization with ID %d not found", orgID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking organization"})
		}
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from request: " + err.Error()})
		return
	}
	defer file.Close()

	log.Printf("Received file: %s, Size: %d", header.Filename, header.Size)

	// Basic validation (consider adding more robust checks)
	if header.Size == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Uploaded file is empty"})
		return
	}
	// Could also check Content-Type if needed, though frontend validates .txt

	scanner := bufio.NewScanner(file)
	var linesProcessed, domainsAdded, subdomainsAdded, endpointsAdded, paramsAdded int
	var errors []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}
		linesProcessed++

		// Attempt to parse the line as a URL
		parsedURL, err := url.Parse(line)
		if err != nil {
			// If parsing fails, treat it as a potential domain/subdomain string
			log.Printf("Line '%s' is not a valid URL, treating as domain/subdomain string for Org ID %d.", line, orgID)
			// Try to add as domain/subdomain directly (simplified logic)
			// Pass orgID to the processing function
			err = processDomainOrSubdomainString(db, line, orgID)
			if err != nil {
				errorMsg := fmt.Sprintf("Error processing '%s' for Org ID %d: %v", line, orgID, err)
				log.Println(errorMsg)
				errors = append(errors, errorMsg)
			} else {
				// We can't easily tell if a domain or subdomain was added here without more complex logic
				// For simplicity, we won't increment specific counters here.
			}
			continue
		}

		// If it has a scheme, prepend it for consistency if missing
		if parsedURL.Scheme == "" {
			// Default to http for parsing, but handle https later if needed
			parsedURL, err = url.Parse("http://" + line)
			if err != nil {
				errorMsg := fmt.Sprintf("Error re-parsing '%s' with scheme: %v", line, err)
				log.Println(errorMsg)
				errors = append(errors, errorMsg)
				continue
			}
		}

		// Process the parsed URL, passing orgID
		dAdded, sAdded, eAdded, pAdded, err := processParsedURL(db, parsedURL, orgID)
		if err != nil {
			errorMsg := fmt.Sprintf("Error processing URL '%s' for Org ID %d: %v", line, orgID, err)
			log.Println(errorMsg)
			errors = append(errors, errorMsg)
		} else {
			domainsAdded += dAdded
			subdomainsAdded += sAdded
			endpointsAdded += eAdded
			paramsAdded += pAdded
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading uploaded file: %v", err)
		// Decide if this is a fatal error or just add to the list
		errors = append(errors, "Error reading file stream: "+err.Error())
	}

	// Construct response message
	var responseMsg strings.Builder
	responseMsg.WriteString(fmt.Sprintf("Processed %d lines. ", linesProcessed))
	if domainsAdded > 0 {
		responseMsg.WriteString(fmt.Sprintf("Added %d new root domains. ", domainsAdded))
	}
	if subdomainsAdded > 0 {
		responseMsg.WriteString(fmt.Sprintf("Added %d new subdomains. ", subdomainsAdded))
	}
	if endpointsAdded > 0 {
		responseMsg.WriteString(fmt.Sprintf("Added %d new endpoints. ", endpointsAdded))
	}
	if paramsAdded > 0 {
		responseMsg.WriteString(fmt.Sprintf("Added %d new parameters. ", paramsAdded))
	}
	if len(errors) > 0 {
		responseMsg.WriteString(fmt.Sprintf("%d errors occurred.", len(errors)))
		// Optionally include detailed errors in response or just log them
		log.Printf("Import errors: %v", errors)
		// For security/simplicity, maybe don't return detailed errors to client
		// c.JSON(http.StatusMultiStatus, gin.H{"message": responseMsg.String(), "errors": errors})
		// return
	}

	if responseMsg.Len() == 0 { // Handle case where file was empty or only had blank lines
		responseMsg.WriteString("No processable content found in the file.")
	}

	c.JSON(http.StatusOK, gin.H{"message": strings.TrimSpace(responseMsg.String())})
}

// processDomainOrSubdomainString handles lines that couldn't be parsed as full URLs for a specific organization.
// This is a simplified approach: it assumes the string is either a root domain or a subdomain.
// TODO: Enhance root domain extraction (e.g., using publicsuffix-go).
func processDomainOrSubdomainString(db *gorm.DB, input string, orgID uint) error {
	// Basic check: Does it look like a domain name? (Contains dots, no path characters)
	if !strings.Contains(input, ".") || strings.ContainsAny(input, "/?#") {
		return fmt.Errorf("invalid format for domain/subdomain string")
	}

	// Attempt to find/create as a RootDomain first (assuming no org context for now)
	// This is problematic without an Organization ID. We might just skip root domain creation here.
	// For now, let's just try adding it as a subdomain, assuming the root domain might exist.
	// A better approach needs Organization context.

	// Try adding as a Subdomain (will fail if RootDomain doesn't exist)
	// We need to extract the potential root domain part. This is non-trivial.
	// Example: If input is "sub.example.com", root is "example.com".
	// Using a library like publicsuffix-go is the robust way.
	// Simplified: Assume last two parts are the root domain (e.g., example.com, example.co.uk)
	parts := strings.Split(input, ".")
	if len(parts) < 2 {
		return fmt.Errorf("cannot determine root domain from '%s'", input)
	}

	// Simplified root domain extraction (adjust for TLDs like .co.uk if needed)
	rootDomainName := strings.Join(parts[len(parts)-2:], ".")

	var rootDomain models.RootDomain
	// Find the root domain for the specific organization
	err := db.Where("domain = ? AND organization_id = ?", rootDomainName, orgID).First(&rootDomain).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Root domain doesn't exist for this org, skip this line silently
			log.Printf("Skipping '%s': Root domain '%s' not found for Org ID %d", input, rootDomainName, orgID)
			return nil // Return nil error to indicate skipping, not failure
		} else {
			// Actual database error occurred during lookup
			return fmt.Errorf("error finding root domain '%s': %w", rootDomainName, err)
		}
	}

	// If we reach here, the root domain exists for the org. Proceed to check/add subdomain.

	// If the input is *not* the same as the found root domain, try adding it as a subdomain
	if input != rootDomainName {
		subdomain := models.Subdomain{
			Hostname:     input, // Correct field name
			RootDomainID: rootDomain.ID,
		}
		// Use FirstOrCreate to avoid duplicates
		result := db.FirstOrCreate(&subdomain, models.Subdomain{Hostname: input, RootDomainID: rootDomain.ID}) // Correct field name
		if result.Error != nil {
			return fmt.Errorf("failed to create subdomain '%s': %w", input, result.Error)
		}
		// if result.RowsAffected > 0 {
		//     // Increment subdomain counter if needed (can't easily return counts from here)
		// }
	} else {
		// Input was just the root domain itself, which already exists. Do nothing.
	}

	return nil
}

// processParsedURL handles lines that were successfully parsed as URLs for a specific organization.
// It attempts to add the root domain, subdomain, endpoint, and parameters.
// Returns counts of added items and any error.
func processParsedURL(db *gorm.DB, u *url.URL, orgID uint) (domainsAdded, subdomainsAdded, endpointsAdded, paramsAdded int, err error) {
	host := u.Hostname()
	path := u.Path
	queryParams := u.Query()

	// --- 1. Find Root Domain (MUST exist for this Org) ---
	// Extract root domain (requires proper TLD handling, using simplified approach here)
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		err = fmt.Errorf("cannot determine root domain from host '%s'", host)
		return
	}
	rootDomainName := strings.Join(parts[len(parts)-2:], ".") // Simplified

	// Use the provided orgID
	var rootDomain models.RootDomain
	// Find the RootDomain, DO NOT create it if missing.
	err = db.Where("domain = ? AND organization_id = ?", rootDomainName, orgID).First(&rootDomain).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Root domain doesn't exist for this org, skip this line silently
			log.Printf("Skipping URL '%s': Root domain '%s' not found for Org ID %d", u.String(), rootDomainName, orgID)
			err = nil // Clear the error, as skipping is not a failure
			return    // Return 0 counts and nil error
		} else {
			// Actual database error occurred during lookup
			err = fmt.Errorf("error finding root domain '%s': %w", rootDomainName, err)
			return // Return the actual error
		}
	}

	// If we reach here, the root domain exists for the org. Proceed.
	// Root domain was found, not created, so domainsAdded remains 0.

	// --- 2. Find or Create Subdomain ---
	var subdomain models.Subdomain
	// Only create subdomain if host is different from root domain
	if host != rootDomainName {
		result := db.FirstOrCreate(&subdomain, models.Subdomain{Hostname: host, RootDomainID: rootDomain.ID}) // Correct field name
		if result.Error != nil {
			err = fmt.Errorf("failed to find/create subdomain '%s': %w", host, result.Error)
			return
		}
		if result.RowsAffected > 0 {
			log.Printf("Created new subdomain: %s for root %s", host, rootDomainName)
			subdomainsAdded = 1
		}
	} else {
		// If host is the root domain, we might still have endpoints/params for it.
		// We need a Subdomain record to link endpoints to.
		// Let's find the "subdomain" record that represents the root domain itself.
		// This assumes a convention where a Subdomain record exists even for the root.
		// If not, the schema/logic needs adjustment.
		err = db.Where("hostname = ? AND root_domain_id = ?", host, rootDomain.ID).First(&subdomain).Error // Use Hostname here too
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create the "root" subdomain entry if it doesn't exist
				subdomain = models.Subdomain{Hostname: host, RootDomainID: rootDomain.ID} // Correct field name
				if res := db.Create(&subdomain); res.Error != nil {
					err = fmt.Errorf("failed to create root-level subdomain entry '%s': %w", host, res.Error)
					return
				}
				log.Printf("Created root-level subdomain entry: %s", host)
				// Don't count this as a "new subdomain" in the user message? Or maybe do? Let's count it.
				subdomainsAdded = 1
			} else {
				err = fmt.Errorf("failed to find root-level subdomain entry '%s': %w", host, err)
				return
			}
		}
	}

	// We must have a valid subdomain ID to proceed
	if subdomain.ID == 0 {
		err = fmt.Errorf("failed to obtain a valid Subdomain ID for host '%s'", host)
		return
	}

	// --- 3. Find or Create Endpoint ---
	// Only create endpoint if path is not empty or "/"
	if path != "" && path != "/" {
		var endpoint models.Endpoint
		// Normalize path? e.g., remove trailing slash? Depends on desired behavior.
		normalizedPath := strings.TrimSuffix(path, "/")
		if normalizedPath == "" {
			normalizedPath = "/"
		} // Handle root path explicitly if needed

		// TODO: Endpoint model needs Method. How to determine from URL? Default to GET?
		// For now, let's assume GET or leave it blank if the model allows.
		// Assuming Method is nullable or defaults appropriately in the model/DB.
		result := db.FirstOrCreate(&endpoint, models.Endpoint{Path: normalizedPath, SubdomainID: subdomain.ID, Method: "GET"}) // Assuming GET
		if result.Error != nil {
			err = fmt.Errorf("failed to find/create endpoint '%s' for subdomain '%s': %w", normalizedPath, host, result.Error)
			return
		}
		if result.RowsAffected > 0 {
			log.Printf("Created new endpoint: %s for subdomain %s", normalizedPath, host)
			endpointsAdded = 1
		}

		// --- 4. Find or Create Parameters ---
		if len(queryParams) > 0 && endpoint.ID != 0 {
			for key := range queryParams { // Iterate only over keys since values are unused
				// Store each value? Or just the key? Current model likely just stores the key.
				// Assuming Parameter model just has Name and EndpointID.
				// TODO: Parameter model needs ParamType. Assume 'query' for now.
				param := models.Parameter{
					Name:       key,
					EndpointID: endpoint.ID,
					ParamType:  "query", // Assuming query param
				}
				result = db.FirstOrCreate(&param, models.Parameter{Name: key, EndpointID: endpoint.ID, ParamType: "query"})
				if result.Error != nil {
					// Log error but continue processing other params
					log.Printf("Failed to find/create parameter '%s' for endpoint '%s': %v", key, normalizedPath, result.Error)
					// Optionally add to a list of parameter errors
				} else if result.RowsAffected > 0 {
					log.Printf("Created new parameter: %s for endpoint %s", key, normalizedPath)
					paramsAdded++
				}
			}
		}
	} // End if path exists

	return // Return collected counts and nil error if successful so far
}
