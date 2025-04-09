package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"rewrite-go/database"
	"rewrite-go/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- Request/Response Structs ---

// ScanToolConfig represents tool-specific configuration within a scan section.
type ScanToolConfig struct {
	Enabled bool     `json:"enabled"`
	Options []string `json:"options,omitempty"`
}

// ScanSectionConfig represents configuration for a scan section (e.g., subdomains, urls).
// Using map[string]interface{} for flexibility, similar to Python's dict.
// Could be map[string]ScanToolConfig if structure is strictly enforced.
type ScanSectionConfig struct {
	Enabled bool                      `json:"enabled"`
	Tools   map[string]ScanToolConfig `json:"tools,omitempty"`
}

// ScanTemplateCreate represents the request body for creating a scan template.
type ScanTemplateCreate struct {
	Name                string             `json:"name" binding:"required"`
	Description         *string            `json:"description"` // Use pointer for optional
	SubdomainScanConfig *ScanSectionConfig `json:"subdomain_scan_config"`
	URLScanConfig       *ScanSectionConfig `json:"url_scan_config"`
	ParameterScanConfig *ScanSectionConfig `json:"parameter_scan_config"`
	TechDetectEnabled   bool               `json:"tech_detect_enabled"` // Default handled by Go's bool default (false), adjust if needed
	ScreenshotEnabled   bool               `json:"screenshot_enabled"`  // Add screenshot enabled field
}

// ScanTemplateUpdate represents the request body for updating a scan template.
// Pointers are used to detect which fields are explicitly provided for update.
type ScanTemplateUpdate struct {
	Name                *string            `json:"name"`
	Description         *string            `json:"description"`
	SubdomainScanConfig *ScanSectionConfig `json:"subdomain_scan_config"`
	URLScanConfig       *ScanSectionConfig `json:"url_scan_config"`
	ParameterScanConfig *ScanSectionConfig `json:"parameter_scan_config"`
	TechDetectEnabled   *bool              `json:"tech_detect_enabled"`
	ScreenshotEnabled   *bool              `json:"screenshot_enabled"` // Add screenshot enabled field (pointer for update)
}

// ScanTemplateResponse represents the response structure for a scan template.
type ScanTemplateResponse struct {
	ID                  uint               `json:"id"`
	Name                string             `json:"name"`
	Description         *string            `json:"description,omitempty"`
	SubdomainScanConfig *ScanSectionConfig `json:"subdomain_scan_config,omitempty"`
	URLScanConfig       *ScanSectionConfig `json:"url_scan_config,omitempty"`
	ParameterScanConfig *ScanSectionConfig `json:"parameter_scan_config,omitempty"`
	TechDetectEnabled   bool               `json:"tech_detect_enabled"`
	ScreenshotEnabled   bool               `json:"screenshot_enabled"` // Add screenshot enabled field
	CreatedAt           *time.Time         `json:"created_at,omitempty"`
	UpdatedAt           *time.Time         `json:"updated_at,omitempty"`
}

// --- Helper Function ---

// mapScanTemplateToResponse converts a DB model to a response struct, handling JSON unmarshaling.
func mapScanTemplateToResponse(template *models.ScanTemplate) ScanTemplateResponse {
	resp := ScanTemplateResponse{
		ID:                template.ID,
		Name:              template.Name,
		Description:       &template.Description, // Assign directly if Description is string, handle if pointer
		TechDetectEnabled: template.TechDetectEnabled,
		ScreenshotEnabled: template.ScreenshotEnabled, // Add screenshot enabled
		CreatedAt:         &template.CreatedAt,        // Assign directly if CreatedAt is time.Time
		UpdatedAt:         template.UpdatedAt,         // UpdatedAt is already *time.Time
	}
	// Handle potential empty description
	if template.Description == "" {
		resp.Description = nil
	}

	// Unmarshal JSON config strings
	_ = json.Unmarshal([]byte(template.SubdomainScanConfig), &resp.SubdomainScanConfig)
	_ = json.Unmarshal([]byte(template.URLScanConfig), &resp.URLScanConfig)
	_ = json.Unmarshal([]byte(template.ParameterScanConfig), &resp.ParameterScanConfig)

	return resp
}

// --- Handler Functions ---

// GetScanTemplates handles GET requests to retrieve all scan templates.
func GetScanTemplates(c *gin.Context) {
	db := database.GetDB()
	var templates []models.ScanTemplate

	result := db.Order("name asc").Find(&templates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scan templates", "details": result.Error.Error()})
		return
	}

	response := make([]ScanTemplateResponse, len(templates))
	for i := range templates {
		response[i] = mapScanTemplateToResponse(&templates[i])
	}
	c.JSON(http.StatusOK, response)
}

// GetScanTemplate handles GET requests for a single scan template by ID.
func GetScanTemplate(c *gin.Context) {
	idStr := c.Param("template_id")
	templateID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID format"})
		return
	}

	db := database.GetDB()
	var template models.ScanTemplate

	result := db.First(&template, uint(templateID))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Scan template with ID %d not found", templateID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scan template", "details": result.Error.Error()})
		}
		return
	}

	response := mapScanTemplateToResponse(&template)
	c.JSON(http.StatusOK, response)
}

// CreateScanTemplate handles POST requests to create a new scan template.
func CreateScanTemplate(c *gin.Context) {
	var input ScanTemplateCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	// Check if name already exists
	var existing models.ScanTemplate
	if err := db.Where("name = ?", input.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Scan template with name '%s' already exists", input.Name)})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for existing template name", "details": err.Error()})
		return
	}

	// Marshal config structs to JSON strings
	subdomainCfgJSON, _ := json.Marshal(input.SubdomainScanConfig)
	urlCfgJSON, _ := json.Marshal(input.URLScanConfig)
	paramCfgJSON, _ := json.Marshal(input.ParameterScanConfig)

	newTemplate := models.ScanTemplate{
		Name:                input.Name,
		Description:         *input.Description, // Dereference pointer
		SubdomainScanConfig: string(subdomainCfgJSON),
		URLScanConfig:       string(urlCfgJSON),
		ParameterScanConfig: string(paramCfgJSON),
		TechDetectEnabled:   input.TechDetectEnabled,
		ScreenshotEnabled:   input.ScreenshotEnabled, // Set screenshot enabled
	}
	// Handle nil description
	if input.Description == nil {
		newTemplate.Description = ""
	}

	result := db.Create(&newTemplate)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scan template", "details": result.Error.Error()})
		return
	}

	response := mapScanTemplateToResponse(&newTemplate)
	c.JSON(http.StatusCreated, response)
}

// UpdateScanTemplate handles PUT requests to update an existing scan template.
func UpdateScanTemplate(c *gin.Context) {
	idStr := c.Param("template_id")
	templateID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID format"})
		return
	}

	var input ScanTemplateUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	var template models.ScanTemplate

	// Find existing template
	if err := db.First(&template, uint(templateID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Scan template with ID %d not found", templateID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scan template for update", "details": err.Error()})
		}
		return
	}

	// Check for name conflict if name is being updated
	if input.Name != nil && *input.Name != template.Name {
		var existing models.ScanTemplate
		if err := db.Where("name = ? AND id != ?", *input.Name, templateID).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Scan template with name '%s' already exists", *input.Name)})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for name conflict during update", "details": err.Error()})
			return
		}
		template.Name = *input.Name // Update name
	}

	// Update fields only if they are provided in the input
	if input.Description != nil {
		template.Description = *input.Description
	}
	if input.SubdomainScanConfig != nil {
		subdomainCfgJSON, _ := json.Marshal(input.SubdomainScanConfig)
		template.SubdomainScanConfig = string(subdomainCfgJSON)
	}
	if input.URLScanConfig != nil {
		urlCfgJSON, _ := json.Marshal(input.URLScanConfig)
		template.URLScanConfig = string(urlCfgJSON)
	}
	if input.ParameterScanConfig != nil {
		paramCfgJSON, _ := json.Marshal(input.ParameterScanConfig)
		template.ParameterScanConfig = string(paramCfgJSON)
	}
	if input.TechDetectEnabled != nil {
		template.TechDetectEnabled = *input.TechDetectEnabled
	}
	if input.ScreenshotEnabled != nil {
		template.ScreenshotEnabled = *input.ScreenshotEnabled // Update screenshot enabled
	}

	// Save updates
	// GORM's Save updates all fields, including associations.
	// Use Updates for partial updates if only changing specific columns.
	result := db.Save(&template)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scan template", "details": result.Error.Error()})
		return
	}

	response := mapScanTemplateToResponse(&template)
	c.JSON(http.StatusOK, response)
}

// DeleteScanTemplate handles DELETE requests to remove a scan template.
func DeleteScanTemplate(c *gin.Context) {
	idStr := c.Param("template_id")
	templateID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID format"})
		return
	}

	db := database.GetDB()

	// Find the template first to ensure it exists
	var template models.ScanTemplate
	if err := db.First(&template, uint(templateID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Scan template with ID %d not found", templateID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scan template for deletion", "details": err.Error()})
		}
		return
	}

	// Delete the template
	// Consider foreign key constraints: If scans reference this template,
	// deletion might fail unless the foreign key allows SET NULL or CASCADE.
	result := db.Delete(&template)
	if result.Error != nil {
		// Check for foreign key constraint error (specific error varies by DB)
		// if strings.Contains(result.Error.Error(), "FOREIGN KEY constraint failed") { ... }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete scan template", "details": result.Error.Error()})
		return
	}

	c.Status(http.StatusNoContent) // Return 204 No Content on successful deletion
}
