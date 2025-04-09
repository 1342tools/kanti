package handlers

import (
	"errors"
	"fmt"
	"log" // Ensure log package is imported
	"net/http"
	"rewrite-go/database"
	"rewrite-go/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- Response Structs ---

// TechnologyBasic represents basic technology info for responses.
type TechnologyBasic struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category,omitempty"`
}

// SubdomainResponse represents the response structure for a subdomain.
type SubdomainResponse struct {
	ID                   uint              `json:"id"`
	RootDomainID         uint              `json:"root_domain_id"`
	Hostname             string            `json:"hostname"`
	IPAddress            string            `json:"ip_address,omitempty"`
	IsActive             bool              `json:"is_active"`
	DiscoveredAt         time.Time         `json:"discovered_at"`
	Technologies         []TechnologyBasic `json:"technologies,omitempty"`           // Use slice of TechnologyBasic
	LatestScreenshotPath *string           `json:"latest_screenshot_path,omitempty"` // Add field for screenshot path
}

// EndpointBasic represents basic endpoint info for responses.
type EndpointBasic struct {
	ID           uint      `json:"id"`
	SubdomainID  uint      `json:"subdomain_id"`
	Path         string    `json:"path"`
	Method       string    `json:"method"`
	StatusCode   int       `json:"status_code,omitempty"`
	ContentType  string    `json:"content_type,omitempty"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

// --- Handler Functions ---

// GetSubdomains handles GET requests to retrieve subdomains.
func GetSubdomains(c *gin.Context) {
	db := database.GetDB()
	var subdomains []models.Subdomain

	// Base query with preloading
	query := db.Preload("Technologies") // GORM handles many-to-many preload

	// Optional filtering by root_domain_id
	domainIDStr := c.Query("domain_id") // Get query parameter
	if domainIDStr != "" {
		domainID, err := strconv.ParseUint(domainIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain_id format"})
			return
		}
		query = query.Where("root_domain_id = ?", uint(domainID))
	}

	// Execute query
	result := query.Find(&subdomains)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subdomains", "details": result.Error.Error()})
		return
	}

	// Build response with deduplicated technologies
	response := make([]SubdomainResponse, len(subdomains))
	for i, sub := range subdomains {
		uniqueTechs := make([]TechnologyBasic, 0, len(sub.Technologies))
		seenTechIDs := make(map[uint]struct{}) // Set to track seen IDs

		for _, tech := range sub.Technologies {
			if _, seen := seenTechIDs[tech.ID]; !seen {
				uniqueTechs = append(uniqueTechs, TechnologyBasic{
					ID:       tech.ID,
					Name:     tech.Name,
					Category: tech.Category,
				})
				seenTechIDs[tech.ID] = struct{}{} // Mark as seen
			}
		}

		response[i] = SubdomainResponse{
			ID:           sub.ID,
			RootDomainID: sub.RootDomainID,
			Hostname:     sub.Hostname,
			IPAddress:    sub.IPAddress,
			IsActive:     sub.IsActive,
			DiscoveredAt: sub.DiscoveredAt,
			Technologies: uniqueTechs, // Use the deduplicated slice
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetSubdomain handles GET requests for a single subdomain by ID.
func GetSubdomain(c *gin.Context) {
	idStr := c.Param("subdomain_id")
	subdomainID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subdomain ID format"})
		return
	}

	db := database.GetDB()
	var subdomain models.Subdomain

	// Query with preload
	result := db.Preload("Technologies").First(&subdomain, uint(subdomainID))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Subdomain with ID %d not found", subdomainID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subdomain", "details": result.Error.Error()})
		}
		return
	}

	// Build response with deduplicated technologies
	uniqueTechs := make([]TechnologyBasic, 0, len(subdomain.Technologies))
	seenTechIDs := make(map[uint]struct{}) // Set to track seen IDs

	for _, tech := range subdomain.Technologies {
		if _, seen := seenTechIDs[tech.ID]; !seen {
			uniqueTechs = append(uniqueTechs, TechnologyBasic{
				ID:       tech.ID,
				Name:     tech.Name,
				Category: tech.Category,
			})
			seenTechIDs[tech.ID] = struct{}{} // Mark as seen
		}
	}

	response := SubdomainResponse{
		ID:           subdomain.ID,
		RootDomainID: subdomain.RootDomainID,
		Hostname:     subdomain.Hostname,
		IPAddress:    subdomain.IPAddress,
		IsActive:     subdomain.IsActive,
		DiscoveredAt: subdomain.DiscoveredAt,
		Technologies: uniqueTechs, // Use the deduplicated slice
	}

	// --- Fetch Latest Screenshot ---
	var latestScreenshot models.Screenshot
	screenshotResult := db.Where("subdomain_id = ?", subdomainID).Order("captured_at desc").First(&latestScreenshot)

	if screenshotResult.Error == nil {
		// Found a screenshot, add its path to the response
		response.LatestScreenshotPath = &latestScreenshot.FilePath
	} else if !errors.Is(screenshotResult.Error, gorm.ErrRecordNotFound) {
		// Log error if it's something other than not found
		log.Printf("Error fetching latest screenshot for subdomain %d: %v", subdomainID, screenshotResult.Error)
	}
	// If ErrRecordNotFound, LatestScreenshotPath remains nil, which is correct.
	// --- End Fetch Latest Screenshot ---

	c.JSON(http.StatusOK, response)
}

// GetSubdomainEndpoints handles GET requests for endpoints of a specific subdomain.
func GetSubdomainEndpoints(c *gin.Context) {
	idStr := c.Param("subdomain_id")
	subdomainID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subdomain ID format"})
		return
	}

	db := database.GetDB()

	// First, check if subdomain exists (optional, but good practice)
	var subdomain models.Subdomain
	if err := db.First(&subdomain, uint(subdomainID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Subdomain with ID %d not found", subdomainID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check subdomain existence", "details": err.Error()})
		}
		return
	}

	// Find endpoints associated with the subdomain
	var endpoints []models.Endpoint
	result := db.Where("subdomain_id = ?", uint(subdomainID)).Find(&endpoints)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve endpoints", "details": result.Error.Error()})
		return
	}

	// Build response
	response := make([]EndpointBasic, len(endpoints))
	for i, ep := range endpoints {
		response[i] = EndpointBasic{
			ID:           ep.ID,
			SubdomainID:  ep.SubdomainID,
			Path:         ep.Path,
			Method:       ep.Method,
			StatusCode:   ep.StatusCode,
			ContentType:  ep.ContentType,
			DiscoveredAt: ep.DiscoveredAt,
		}
	}

	c.JSON(http.StatusOK, response)
}
