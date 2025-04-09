package handlers

import (
	"encoding/json" // Added for parsing template config
	"errors"
	"fmt"
	"net/http"
	"rewrite-go/database"
	"rewrite-go/models"
	"rewrite-go/scanner" // Added scanner import
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- Response Structs ---

// ScanBasicResponse represents basic scan info for the list endpoint.
type ScanBasicResponse struct {
	ID             uint       `json:"id"`
	RootDomainID   uint       `json:"root_domain_id"`
	SubdomainID    *uint      `json:"subdomain_id,omitempty"` // Added
	ScanType       string     `json:"scan_type"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Status         string     `json:"status,omitempty"`
	ResultsSummary string     `json:"results_summary,omitempty"`
}

// ScanDetailResponse represents detailed scan info including discovered items.
// Reusing SubdomainBasicResponse and EndpointBasic from other handlers.
type ScanDetailResponse struct {
	ID                   uint                     `json:"id"`
	RootDomainID         uint                     `json:"root_domain_id"`
	SubdomainID          *uint                    `json:"subdomain_id,omitempty"` // Added
	ScanType             string                   `json:"scan_type"`
	StartedAt            time.Time                `json:"started_at"`
	CompletedAt          *time.Time               `json:"completed_at,omitempty"`
	Status               string                   `json:"status,omitempty"`
	ResultsSummary       string                   `json:"results_summary,omitempty"`
	DiscoveredSubdomains []SubdomainBasicResponse `json:"discovered_subdomains"`
	DiscoveredEndpoints  []EndpointBasic          `json:"discovered_endpoints"` // Using EndpointBasic for now
}

// --- Handler Functions ---

// GetScans handles GET requests to retrieve scans for a specific domain OR subdomain.
func GetScans(c *gin.Context) {
	db := database.GetDB()
	var scans []models.Scan

	// Allow filtering by root_domain_id OR subdomain_id
	rootDomainIDStr := c.Query("root_domain_id")
	subdomainIDStr := c.Query("subdomain_id")

	query := db.Order("started_at desc") // Start with ordering

	if rootDomainIDStr != "" {
		rootDomainID, err := strconv.ParseUint(rootDomainIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid root_domain_id format"})
			return
		}
		query = query.Where("root_domain_id = ?", uint(rootDomainID))
	} else if subdomainIDStr != "" {
		subdomainID, err := strconv.ParseUint(subdomainIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subdomain_id format"})
			return
		}
		// Find the root domain ID for the given subdomain ID first
		var sub models.Subdomain
		if res := db.Select("root_domain_id").First(&sub, uint(subdomainID)); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Subdomain with ID %d not found", subdomainID)})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find subdomain", "details": res.Error.Error()})
			}
			return
		}
		// Now filter scans by root domain AND specific subdomain
		query = query.Where("root_domain_id = ? AND subdomain_id = ?", sub.RootDomainID, uint(subdomainID))
	} else {
		// If neither is provided, maybe return all scans? Or require at least one?
		// For now, let's require at least root_domain_id for the general list.
		// If you want scans for a specific subdomain, use the subdomain_id query param.
		// If you want *all* scans, a different endpoint might be better.
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required query parameter: root_domain_id"})
		return
	}

	result := query.Find(&scans)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scans", "details": result.Error.Error()})
		return
	}

	// Build response
	response := make([]ScanBasicResponse, len(scans))
	for i, s := range scans {
		response[i] = ScanBasicResponse{
			ID:             s.ID,
			RootDomainID:   s.RootDomainID,
			SubdomainID:    s.SubdomainID, // Include SubdomainID
			ScanType:       s.ScanType,
			StartedAt:      s.StartedAt,
			CompletedAt:    s.CompletedAt,
			Status:         s.Status,
			ResultsSummary: s.ResultsSummary,
		}
	}
	c.JSON(http.StatusOK, response)
}

// GetScan handles GET requests for detailed information about a single scan.
func GetScan(c *gin.Context) {
	idStr := c.Param("id") // Get scan ID from path
	scanID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan ID format"})
		return
	}

	db := database.GetDB()
	var scan models.Scan

	// Query scan by ID, preloading discovered subdomains and endpoints
	result := db.Preload("DiscoveredSubdomains").
		Preload("DiscoveredEndpoints").
		First(&scan, uint(scanID))

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Scan with ID %d not found", scanID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scan details", "details": result.Error.Error()})
		}
		return
	}

	// Build response for discovered subdomains
	subdomainsData := make([]SubdomainBasicResponse, len(scan.DiscoveredSubdomains))
	for i, sub := range scan.DiscoveredSubdomains {
		subdomainsData[i] = SubdomainBasicResponse{
			ID:           sub.ID,
			RootDomainID: sub.RootDomainID,
			Hostname:     sub.Hostname,
			IPAddress:    sub.IPAddress,
			IsActive:     sub.IsActive,
			DiscoveredAt: sub.DiscoveredAt,
		}
	}

	// Build response for discovered endpoints
	endpointsData := make([]EndpointBasic, len(scan.DiscoveredEndpoints))
	for i, ep := range scan.DiscoveredEndpoints {
		endpointsData[i] = EndpointBasic{
			ID:           ep.ID,
			SubdomainID:  ep.SubdomainID,
			Path:         ep.Path,
			Method:       ep.Method,
			StatusCode:   ep.StatusCode,
			ContentType:  ep.ContentType,
			DiscoveredAt: ep.DiscoveredAt,
		}
	}

	// Construct the final detailed response
	response := ScanDetailResponse{
		ID:                   scan.ID,
		RootDomainID:         scan.RootDomainID,
		SubdomainID:          scan.SubdomainID, // Include SubdomainID
		ScanType:             scan.ScanType,
		StartedAt:            scan.StartedAt,
		CompletedAt:          scan.CompletedAt,
		Status:               scan.Status,
		ResultsSummary:       scan.ResultsSummary,
		DiscoveredSubdomains: subdomainsData,
		DiscoveredEndpoints:  endpointsData,
	}

	c.JSON(http.StatusOK, response)
}

// StartScan handles POST requests to initiate a new scan (root domain or subdomain).
func StartScan(c *gin.Context) {
	var input models.ScanStartRequest // Use model struct
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	db := database.GetDB()

	// --- Validate Root Domain ---
	var rootDomain models.RootDomain
	if err := db.First(&rootDomain, input.RootDomainID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Root domain with ID %d not found", input.RootDomainID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve root domain", "details": err.Error()})
		}
		return
	}

	// --- Validate Subdomain (if provided) ---
	var subdomain *models.Subdomain = nil // Use pointer
	targetHost := rootDomain.Domain       // Default target is the root domain
	scanType := "root_domain"             // Default scan type

	if input.SubdomainID != nil {
		var fetchedSubdomain models.Subdomain
		// Ensure the subdomain belongs to the specified root domain
		if err := db.Where("id = ? AND root_domain_id = ?", *input.SubdomainID, input.RootDomainID).First(&fetchedSubdomain).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Subdomain with ID %d not found or does not belong to root domain ID %d", *input.SubdomainID, input.RootDomainID)})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subdomain", "details": err.Error()})
			}
			return
		}
		subdomain = &fetchedSubdomain
		targetHost = subdomain.Hostname // Target is the specific subdomain
		scanType = "subdomain"          // Set scan type
	}

	// --- Scan Template Handling ---
	var scanTemplate *models.ScanTemplate = nil
	var scanConfig models.ScanConfig = models.ScanConfig{ // Use model struct and default values
		SubdomainScanConfig: make(map[string]interface{}),
		URLScanConfig:       make(map[string]interface{}),
		ParameterScanConfig: make(map[string]interface{}),
		TechDetectEnabled:   true,  // Default based on Python model
		ScreenshotEnabled:   false, // Default
	}
	var scanTemplateID *uint = input.ScanTemplateID

	if input.ScanTemplateID != nil {
		var fetchedTemplate models.ScanTemplate
		if err := db.First(&fetchedTemplate, *input.ScanTemplateID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Scan template with ID %d not found", *input.ScanTemplateID)})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scan template", "details": err.Error()})
			}
			return
		}
		scanTemplate = &fetchedTemplate

		// Parse JSON config strings (handle potential errors gracefully)
		_ = json.Unmarshal([]byte(scanTemplate.SubdomainScanConfig), &scanConfig.SubdomainScanConfig)
		_ = json.Unmarshal([]byte(scanTemplate.URLScanConfig), &scanConfig.URLScanConfig)
		_ = json.Unmarshal([]byte(scanTemplate.ParameterScanConfig), &scanConfig.ParameterScanConfig)
		scanConfig.TechDetectEnabled = scanTemplate.TechDetectEnabled
		scanConfig.ScreenshotEnabled = scanTemplate.ScreenshotEnabled // Use template setting
	}

	// --- Create Scan Record ---
	scan := models.Scan{
		RootDomainID:   input.RootDomainID,
		SubdomainID:    input.SubdomainID, // Assign subdomain ID (can be nil)
		ScanTemplateID: scanTemplateID,    // Assign template ID (can be nil)
		ScanType:       scanType,          // Set based on whether SubdomainID is present
		Status:         "pending",
		StartedAt:      time.Now(), // Set start time explicitly
	}

	result := db.Create(&scan)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scan record", "details": result.Error.Error()})
		return
	}

	// --- Start Scan Task (Asynchronously) ---
	// Start the appropriate scan type
	go scanner.ExecuteSubdomainScan(targetHost, scanType, rootDomain.ID, scan.ID, scanTemplate) // Pass targetHost and scanType

	// Respond immediately
	message := fmt.Sprintf("Scan started for %s", targetHost)
	if scanTemplateID != nil {
		message += fmt.Sprintf(" using template ID %d", *scanTemplateID)
	}

	c.JSON(http.StatusAccepted, gin.H{"message": message, "scan_id": scan.ID})
}
