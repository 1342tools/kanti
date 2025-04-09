package handlers

import (
	"errors"
	"net/http"
	"rewrite-go/database"
	"rewrite-go/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- Request/Response Structs ---

// OrganizationCreate represents the request body for creating an organization.
type OrganizationCreate struct {
	Name string `json:"name" binding:"required,min=1"` // Use Gin binding for validation
}

// OrganizationResponse represents the basic response for an organization.
type OrganizationResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// RootDomainBasic represents basic info for a root domain in responses.
type RootDomainBasic struct {
	ID     uint   `json:"id"`
	Domain string `json:"domain"`
}

// --- Handler Functions ---

// CreateOrganization handles POST requests to create a new organization.
func CreateOrganization(c *gin.Context) {
	var input OrganizationCreate
	// Bind JSON request body to the input struct, handling validation errors
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trimmedName := strings.TrimSpace(input.Name)
	if trimmedName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization name cannot be empty"})
		return
	}

	org := models.Organization{Name: trimmedName}
	db := database.GetDB()

	// Attempt to create the organization
	result := db.Create(&org)
	if result.Error != nil {
		// Check for unique constraint violation (specific error might depend on DB driver)
		// A simple check for existing name before creating might be more reliable across DBs
		var existingOrg models.Organization
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) || db.Where("name = ?", trimmedName).First(&existingOrg).Error == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Organization with name '" + trimmedName + "' already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization", "details": result.Error.Error()})
		}
		return
	}

	// Return the created organization details
	response := OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
	}
	c.JSON(http.StatusCreated, response)
}

// GetOrganizations handles GET requests to retrieve all organizations.
func GetOrganizations(c *gin.Context) {
	var organizations []models.Organization
	db := database.GetDB()

	// Retrieve all organizations, ordered by name
	result := db.Order("name asc").Find(&organizations)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve organizations", "details": result.Error.Error()})
		return
	}

	// Convert to response format
	response := make([]OrganizationResponse, len(organizations))
	for i, org := range organizations {
		response[i] = OrganizationResponse{
			ID:        org.ID,
			Name:      org.Name,
			CreatedAt: org.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetOrganization handles GET requests to retrieve a single organization by ID.
func GetOrganization(c *gin.Context) {
	idStr := c.Param("org_id")                     // Gin uses :param_name syntax in route definition
	orgID, err := strconv.ParseUint(idStr, 10, 32) // Parse ID from URL param
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	var organization models.Organization
	db := database.GetDB()

	// Retrieve organization by ID, preloading associated RootDomains
	result := db.Preload("RootDomains").First(&organization, uint(orgID)) // Preload RootDomains here
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve organization", "details": result.Error.Error()})
		}
		return
	}

	// Calculate counts
	// Total Root Domains
	db.Model(&models.RootDomain{}).Where("organization_id = ?", orgID).Count(&organization.TotalRootDomains)

	// Total Subdomains
	db.Model(&models.Subdomain{}).
		Joins("join root_domains on root_domains.id = subdomains.root_domain_id").
		Where("root_domains.organization_id = ?", orgID).
		Count(&organization.TotalSubdomains)

	// Total Endpoints
	db.Model(&models.Endpoint{}).
		Joins("join subdomains on subdomains.id = endpoints.subdomain_id").
		Joins("join root_domains on root_domains.id = subdomains.root_domain_id").
		Where("root_domains.organization_id = ?", orgID).
		Count(&organization.TotalEndpoints)

	// Return the organization object which now includes the counts AND the preloaded RootDomains
	c.JSON(http.StatusOK, organization)
}
