package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"rewrite-go/database"
	"rewrite-go/models"
	"rewrite-go/scanner" // Import the scanner package
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/weppos/publicsuffix-go/publicsuffix"
	"gorm.io/gorm"
)

// --- Request/Response Structs ---

// DomainCreate represents the request body for creating a root domain.
type DomainCreate struct {
	Domain         string `json:"domain" binding:"required"`
	OrganizationID uint   `json:"organization_id" binding:"required"`
}

// DomainResponse represents the response structure for a root domain.
type DomainResponse struct {
	ID             uint       `json:"id"`
	Domain         string     `json:"domain"`
	OrganizationID uint       `json:"organization_id"`
	CreatedAt      time.Time  `json:"created_at"`
	LastScannedAt  *time.Time `json:"last_scanned_at,omitempty"`
	// Note: TotalSubdomains and TotalEndpoints are added to models.RootDomain
}

// Note: ScanStartRequest and ScanConfig structs are now defined in models/models.go

// --- Handler Functions ---

// CreateDomain handles POST requests to create a new root domain.
func CreateDomain(c *gin.Context) {
	var input DomainCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract root domain using publicsuffix-go
	// Note: This library focuses on eTLD+1, similar to tldextract's domain+suffix
	parsedDomain, err := publicsuffix.Parse(input.Domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain format", "details": err.Error()})
		return
	}
	// Reconstruct the root domain (e.g., "google.com" from "www.google.com")
	rootDomain := fmt.Sprintf("%s.%s", parsedDomain.SLD, parsedDomain.TLD) // Combine SLD and TLD

	db := database.GetDB()

	// Verify organization exists
	var organization models.Organization
	if err := db.First(&organization, input.OrganizationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Organization with ID %d not found", input.OrganizationID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify organization", "details": err.Error()})
		}
		return
	}

	// Check if domain already exists within this organization
	var existingDomain models.RootDomain
	errCheck := db.Where("domain = ? AND organization_id = ?", rootDomain, input.OrganizationID).First(&existingDomain).Error
	if errCheck == nil {
		// Domain already exists for this organization
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Domain '%s' already exists in organization ID %d", rootDomain, input.OrganizationID)})
		return
	} else if !errors.Is(errCheck, gorm.ErrRecordNotFound) {
		// Handle potential database errors during the check
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for existing domain", "details": errCheck.Error()})
		return
	}
	// If errCheck is gorm.ErrRecordNotFound, the domain does not exist, proceed.

	// Create new domain
	domain := models.RootDomain{
		Domain:         rootDomain,
		OrganizationID: input.OrganizationID,
	}

	result := db.Create(&domain)
	if result.Error != nil {
		// The duplicate check is now handled above.
		// Handle other potential creation errors.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create domain", "details": result.Error.Error()})
		return
	}

	response := DomainResponse{
		ID:             domain.ID,
		Domain:         domain.Domain,
		OrganizationID: domain.OrganizationID,
		CreatedAt:      domain.CreatedAt,
		LastScannedAt:  domain.LastScannedAt,
	}
	c.JSON(http.StatusCreated, response)
}

// GetDomains handles GET requests to retrieve all root domains.
func GetDomains(c *gin.Context) {
	var domains []models.RootDomain
	db := database.GetDB()

	result := db.Find(&domains)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve domains", "details": result.Error.Error()})
		return
	}

	response := make([]DomainResponse, len(domains))
	for i, d := range domains {
		response[i] = DomainResponse{
			ID:             d.ID,
			Domain:         d.Domain,
			OrganizationID: d.OrganizationID,
			CreatedAt:      d.CreatedAt,
			LastScannedAt:  d.LastScannedAt,
		}
	}
	c.JSON(http.StatusOK, response)
}

// GetDomain handles GET requests to retrieve a single root domain by ID.
func GetDomain(c *gin.Context) {
	idStr := c.Param("domain_id")
	domainID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain ID format"})
		return
	}

	var domain models.RootDomain
	db := database.GetDB()

	result := db.First(&domain, uint(domainID))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Domain with ID %d not found", domainID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve domain", "details": result.Error.Error()})
		}
		return
	}

	// Calculate counts
	// Total Subdomains for this RootDomain
	db.Model(&models.Subdomain{}).Where("root_domain_id = ?", domainID).Count(&domain.TotalSubdomains)

	// Total Endpoints for this RootDomain
	db.Model(&models.Endpoint{}).
		Joins("join subdomains on subdomains.id = endpoints.subdomain_id").
		Where("subdomains.root_domain_id = ?", domainID).
		Count(&domain.TotalEndpoints)

	// Return the domain object which now includes the counts
	c.JSON(http.StatusOK, domain)
}

// ScanDomain handles POST requests to initiate a scan for a domain.
// DEPRECATED: Use POST /api/scans instead. This function remains for potential backward compatibility or reference.
// It's recommended to remove or refactor this in the future.
func ScanDomain(c *gin.Context) {
	idStr := c.Param("domain_id")
	domainID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain ID format"})
		return
	}

	// This struct is now defined in models, but we need a local one for binding here
	// if we keep this deprecated function. Let's define a minimal local version.
	type ScanDomainRequest struct {
		ScanTemplateID *uint `json:"scan_template_id"`
	}
	var localInput ScanDomainRequest

	// Bind JSON request body, allowing empty body if template ID is not provided
	if err := c.ShouldBindJSON(&localInput); err != nil && !errors.Is(err, errors.New("EOF")) { // Ignore EOF which means empty body
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	db := database.GetDB()

	// Check if domain exists
	var domain models.RootDomain
	if err := db.First(&domain, uint(domainID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Domain with ID %d not found", domainID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve domain", "details": err.Error()})
		}
		return
	}

	// --- Scan Template Handling ---
	var scanTemplate *models.ScanTemplate = nil
	var scanConfig models.ScanConfig = models.ScanConfig{ // Use model struct
		SubdomainScanConfig: make(map[string]interface{}),
		URLScanConfig:       make(map[string]interface{}),
		ParameterScanConfig: make(map[string]interface{}),
		TechDetectEnabled:   true,
		ScreenshotEnabled:   false, // Add screenshot default
	}
	var scanTemplateID *uint = localInput.ScanTemplateID // Use localInput

	if localInput.ScanTemplateID != nil {
		var fetchedTemplate models.ScanTemplate
		if err := db.First(&fetchedTemplate, *localInput.ScanTemplateID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Scan template with ID %d not found", *localInput.ScanTemplateID)})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scan template", "details": err.Error()})
			}
			return
		}
		scanTemplate = &fetchedTemplate

		// Parse JSON config strings from the fetched template
		_ = json.Unmarshal([]byte(scanTemplate.SubdomainScanConfig), &scanConfig.SubdomainScanConfig)
		_ = json.Unmarshal([]byte(scanTemplate.URLScanConfig), &scanConfig.URLScanConfig)
		_ = json.Unmarshal([]byte(scanTemplate.ParameterScanConfig), &scanConfig.ParameterScanConfig)
		scanConfig.TechDetectEnabled = scanTemplate.TechDetectEnabled
		scanConfig.ScreenshotEnabled = scanTemplate.ScreenshotEnabled // Use template setting
	}

	// --- Create Scan Record ---
	// Note: This creates a root_domain scan only. Subdomain scans use the new StartScan handler.
	scan := models.Scan{
		RootDomainID:   uint(domainID),
		SubdomainID:    nil, // Explicitly nil for root domain scan
		ScanTemplateID: scanTemplateID,
		ScanType:       "root_domain", // Set type explicitly
		Status:         "pending",
		StartedAt:      time.Now(), // Set start time
	}

	result := db.Create(&scan)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scan record", "details": result.Error.Error()})
		return
	}

	// --- Start Scan Task (Asynchronously) ---
	// Run the subdomain scan (always root_domain type for this deprecated function)
	go scanner.ExecuteSubdomainScan(domain.Domain, "root_domain", domain.ID, scan.ID, scanTemplate) // Pass scanType="root_domain"

	// Respond immediately that the scan has been initiated
	message := fmt.Sprintf("Scan started for domain %s", domain.Domain)
	if scanTemplateID != nil {
		message += fmt.Sprintf(" using template ID %d", *scanTemplateID)
	}

	c.JSON(http.StatusAccepted, gin.H{"message": message, "scan_id": scan.ID})
}
