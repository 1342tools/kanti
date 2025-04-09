package handlers

import (
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

// --- Response Structs ---

// TechnologyResponse represents the response structure for a technology.
// Reusing TechnologyBasic from subdomains.go for consistency
// type TechnologyResponse struct {
// 	ID       uint   `json:"id"`
// 	Name     string `json:"name"`
// 	Category string `json:"category,omitempty"`
// }

// DomainBasicResponse represents basic domain info for responses.
type DomainBasicResponse struct {
	ID            uint       `json:"id"`
	Domain        string     `json:"domain"`
	CreatedAt     time.Time  `json:"created_at"`
	LastScannedAt *time.Time `json:"last_scanned_at,omitempty"`
}

// SubdomainBasicResponse represents basic subdomain info for responses.
type SubdomainBasicResponse struct {
	ID           uint      `json:"id"`
	RootDomainID uint      `json:"root_domain_id"`
	Hostname     string    `json:"hostname"`
	IPAddress    string    `json:"ip_address,omitempty"`
	IsActive     bool      `json:"is_active"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

// Reusing EndpointBasic from subdomains.go

// --- Helper Function ---

// checkTechnologyExists checks if a technology exists and returns it or an error.
func checkTechnologyExists(db *gorm.DB, technologyID uint) (*models.Technology, error) {
	var technology models.Technology
	if err := db.First(&technology, technologyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("technology with ID %d not found", technologyID)
		}
		return nil, fmt.Errorf("failed to check technology existence: %w", err)
	}
	return &technology, nil
}

// --- Handler Functions ---

// GetTechnologies handles GET requests to retrieve all technologies.
func GetTechnologies(c *gin.Context) {
	db := database.GetDB()
	var technologies []models.Technology

	result := db.Find(&technologies)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve technologies", "details": result.Error.Error()})
		return
	}

	// Reuse TechnologyBasic for response
	response := make([]TechnologyBasic, len(technologies))
	for i, t := range technologies {
		response[i] = TechnologyBasic{
			ID:       t.ID,
			Name:     t.Name,
			Category: t.Category,
		}
	}
	c.JSON(http.StatusOK, response)
}

// GetTechnology handles GET requests for a single technology by ID.
func GetTechnology(c *gin.Context) {
	idStr := c.Param("technology_id")
	technologyID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technology ID format"})
		return
	}

	db := database.GetDB()
	technology, err := checkTechnologyExists(db, uint(technologyID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || err.Error() == fmt.Sprintf("technology with ID %d not found", technologyID) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve technology", "details": err.Error()})
		}
		return
	}

	response := TechnologyBasic{
		ID:       technology.ID,
		Name:     technology.Name,
		Category: technology.Category,
	}
	c.JSON(http.StatusOK, response)
}

// GetDomainsWithTechnology handles GET requests for domains associated with a technology.
func GetDomainsWithTechnology(c *gin.Context) {
	idStr := c.Param("technology_id")
	technologyID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technology ID format"})
		return
	}

	db := database.GetDB()
	_, err = checkTechnologyExists(db, uint(technologyID)) // Just check existence
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Find subdomains with this technology
	var subdomains []models.Subdomain
	// GORM handles joins implicitly for many-to-many through Preload or explicit Join
	// We need the root_domain_id from subdomains that have the technology.
	// A subquery or join might be more efficient, but this is simpler for now.
	resultSubdomains := db.Joins("JOIN subdomain_technologies ON subdomains.id = subdomain_technologies.subdomain_id").
		Where("subdomain_technologies.technology_id = ?", uint(technologyID)).
		Distinct("subdomains.root_domain_id"). // Get distinct root domain IDs
		Find(&subdomains)

	if resultSubdomains.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subdomains for technology", "details": resultSubdomains.Error.Error()})
		return
	}

	// Extract unique root domain IDs
	rootDomainIDs := make([]uint, 0, len(subdomains))
	seenIDs := make(map[uint]bool)
	for _, sub := range subdomains {
		if !seenIDs[sub.RootDomainID] {
			rootDomainIDs = append(rootDomainIDs, sub.RootDomainID)
			seenIDs[sub.RootDomainID] = true
		}
	}

	if len(rootDomainIDs) == 0 {
		c.JSON(http.StatusOK, []DomainBasicResponse{}) // Return empty list
		return
	}

	// Find the corresponding root domains
	var rootDomains []models.RootDomain
	resultDomains := db.Where("id IN ?", rootDomainIDs).Find(&rootDomains)
	if resultDomains.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve root domains", "details": resultDomains.Error.Error()})
		return
	}

	// Build response
	response := make([]DomainBasicResponse, len(rootDomains))
	for i, rd := range rootDomains {
		response[i] = DomainBasicResponse{
			ID:            rd.ID,
			Domain:        rd.Domain,
			CreatedAt:     rd.CreatedAt,
			LastScannedAt: rd.LastScannedAt,
		}
	}
	c.JSON(http.StatusOK, response)
}

// GetSubdomainsWithTechnology handles GET requests for subdomains associated with a technology.
func GetSubdomainsWithTechnology(c *gin.Context) {
	idStr := c.Param("technology_id")
	technologyID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technology ID format"})
		return
	}

	db := database.GetDB()
	_, err = checkTechnologyExists(db, uint(technologyID)) // Just check existence
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Find subdomains with this technology using join
	var subdomains []models.Subdomain
	result := db.Joins("JOIN subdomain_technologies ON subdomains.id = subdomain_technologies.subdomain_id").
		Where("subdomain_technologies.technology_id = ?", uint(technologyID)).
		Find(&subdomains)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subdomains", "details": result.Error.Error()})
		return
	}

	// Build response
	response := make([]SubdomainBasicResponse, len(subdomains))
	for i, sub := range subdomains {
		response[i] = SubdomainBasicResponse{
			ID:           sub.ID,
			RootDomainID: sub.RootDomainID,
			Hostname:     sub.Hostname,
			IPAddress:    sub.IPAddress,
			IsActive:     sub.IsActive,
			DiscoveredAt: sub.DiscoveredAt,
		}
	}
	c.JSON(http.StatusOK, response)
}

// GetEndpointsWithTechnology handles GET requests for endpoints associated with a technology.
func GetEndpointsWithTechnology(c *gin.Context) {
	idStr := c.Param("technology_id")
	technologyID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technology ID format"})
		return
	}

	db := database.GetDB()
	_, err = checkTechnologyExists(db, uint(technologyID)) // Just check existence
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Find endpoints with this technology using join
	var endpoints []models.Endpoint
	result := db.Joins("JOIN endpoint_technologies ON endpoints.id = endpoint_technologies.endpoint_id").
		Where("endpoint_technologies.technology_id = ?", uint(technologyID)).
		Find(&endpoints)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve endpoints", "details": result.Error.Error()})
		return
	}

	// Build response using EndpointBasic
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
