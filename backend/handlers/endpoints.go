package handlers

import (
	"errors"
	"fmt"
	"log" // Add log import
	"net/http"
	"rewrite-go/database"
	"rewrite-go/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- Response Structs ---

// EndpointResponse represents the basic response structure for an endpoint.
type EndpointResponse struct {
	ID           uint      `json:"id"`
	SubdomainID  uint      `json:"subdomain_id"`
	Path         string    `json:"path"`
	Method       string    `json:"method"`
	StatusCode   int       `json:"status_code,omitempty"`
	ContentType  string    `json:"content_type,omitempty"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

// ParameterResponse represents the response structure for a parameter.
type ParameterResponse struct {
	ID           uint      `json:"id"`
	EndpointID   uint      `json:"endpoint_id"`
	Name         string    `json:"name"`
	ParamType    string    `json:"param_type"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

// RequestResponseResponse represents the response structure for a request/response pair.
type RequestResponseResponse struct {
	ID              uint      `json:"id"`
	EndpointID      uint      `json:"endpoint_id"`
	RequestHeaders  string    `json:"request_headers,omitempty"`
	RequestBody     string    `json:"request_body,omitempty"`
	ResponseHeaders string    `json:"response_headers,omitempty"`
	ResponseBody    string    `json:"response_body,omitempty"`
	CapturedAt      time.Time `json:"captured_at"`
}

// EndpointDetailResponse represents the detailed response for an endpoint.
type EndpointDetailResponse struct {
	ID                   uint                `json:"id"`
	SubdomainID          uint                `json:"subdomain_id"`
	Path                 string              `json:"path"`
	Method               string              `json:"method"`
	StatusCode           int                 `json:"status_code,omitempty"`
	ContentType          string              `json:"content_type,omitempty"`
	DiscoveredAt         time.Time           `json:"discovered_at"`
	Parameters           []ParameterResponse `json:"parameters"`                       // Use ParameterResponse
	Technologies         []TechnologyBasic   `json:"technologies"`                     // Reuse TechnologyBasic from subdomains.go
	LatestScreenshotPath *string             `json:"latest_screenshot_path,omitempty"` // Add field for screenshot path
}

// --- Handler Functions ---

// GetEndpoints handles GET requests to retrieve endpoints.
func GetEndpoints(c *gin.Context) {
	db := database.GetDB()
	var endpoints []models.Endpoint

	query := db.Model(&models.Endpoint{}) // Start query builder

	// Optional filtering by subdomain_id
	subdomainIDStr := c.Query("subdomain_id")
	if subdomainIDStr != "" {
		subdomainID, err := strconv.ParseUint(subdomainIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subdomain_id format"})
			return
		}
		query = query.Where("subdomain_id = ?", uint(subdomainID))
	}

	result := query.Find(&endpoints)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve endpoints", "details": result.Error.Error()})
		return
	}

	// Build response
	response := make([]EndpointResponse, len(endpoints))
	for i, ep := range endpoints {
		response[i] = EndpointResponse{
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

// GetEndpoint handles GET requests for a single endpoint by ID.
func GetEndpoint(c *gin.Context) {
	idStr := c.Param("endpoint_id")
	endpointID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endpoint ID format"})
		return
	}

	db := database.GetDB()
	var endpoint models.Endpoint

	// Query endpoint, preloading parameters and technologies
	result := db.Preload("Parameters").Preload("Technologies").First(&endpoint, uint(endpointID))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Endpoint with ID %d not found", endpointID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve endpoint", "details": result.Error.Error()})
		}
		return
	}

	// Build detailed response
	paramsResponse := make([]ParameterResponse, len(endpoint.Parameters))
	for i, p := range endpoint.Parameters {
		paramsResponse[i] = ParameterResponse{
			ID:           p.ID,
			EndpointID:   p.EndpointID,
			Name:         p.Name,
			ParamType:    p.ParamType,
			DiscoveredAt: p.DiscoveredAt,
		}
	}

	techsResponse := make([]TechnologyBasic, len(endpoint.Technologies))
	for i, t := range endpoint.Technologies {
		techsResponse[i] = TechnologyBasic{
			ID:       t.ID,
			Name:     t.Name,
			Category: t.Category,
		}
	}

	response := EndpointDetailResponse{
		ID:           endpoint.ID,
		SubdomainID:  endpoint.SubdomainID,
		Path:         endpoint.Path,
		Method:       endpoint.Method,
		StatusCode:   endpoint.StatusCode,
		ContentType:  endpoint.ContentType,
		DiscoveredAt: endpoint.DiscoveredAt,
		Parameters:   paramsResponse,
		Technologies: techsResponse,
	}

	// --- Fetch Latest Screenshot ---
	var latestScreenshot models.Screenshot
	screenshotResult := db.Where("endpoint_id = ?", endpointID).Order("captured_at desc").First(&latestScreenshot)

	if screenshotResult.Error == nil {
		// Found a screenshot, add its path to the response
		response.LatestScreenshotPath = &latestScreenshot.FilePath
	} else if !errors.Is(screenshotResult.Error, gorm.ErrRecordNotFound) {
		// Log error if it's something other than not found
		log.Printf("Error fetching latest screenshot for endpoint %d: %v", endpointID, screenshotResult.Error)
	}
	// If ErrRecordNotFound, LatestScreenshotPath remains nil, which is correct.
	// --- End Fetch Latest Screenshot ---

	c.JSON(http.StatusOK, response)
}

// GetEndpointParameters handles GET requests for parameters of a specific endpoint.
func GetEndpointParameters(c *gin.Context) {
	idStr := c.Param("endpoint_id")
	endpointID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endpoint ID format"})
		return
	}

	db := database.GetDB()

	// Check if endpoint exists first
	var endpoint models.Endpoint
	if err := db.First(&endpoint, uint(endpointID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Endpoint with ID %d not found", endpointID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check endpoint existence", "details": err.Error()})
		}
		return
	}

	// Find parameters
	var parameters []models.Parameter
	result := db.Where("endpoint_id = ?", uint(endpointID)).Find(&parameters)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve parameters", "details": result.Error.Error()})
		return
	}

	// Build response
	response := make([]ParameterResponse, len(parameters))
	for i, p := range parameters {
		response[i] = ParameterResponse{
			ID:           p.ID,
			EndpointID:   p.EndpointID,
			Name:         p.Name,
			ParamType:    p.ParamType,
			DiscoveredAt: p.DiscoveredAt,
		}
	}
	c.JSON(http.StatusOK, response)
}

// GetEndpointRequestResponses handles GET requests for request/response pairs of a specific endpoint.
func GetEndpointRequestResponses(c *gin.Context) {
	idStr := c.Param("endpoint_id")
	endpointID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endpoint ID format"})
		return
	}

	db := database.GetDB()

	// Check if endpoint exists first
	var endpoint models.Endpoint
	if err := db.First(&endpoint, uint(endpointID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Endpoint with ID %d not found", endpointID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check endpoint existence", "details": err.Error()})
		}
		return
	}

	// Find request/responses
	var reqResps []models.RequestResponse
	result := db.Where("endpoint_id = ?", uint(endpointID)).Find(&reqResps)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve request/responses", "details": result.Error.Error()})
		return
	}

	// Build response
	response := make([]RequestResponseResponse, len(reqResps))
	for i, rr := range reqResps {
		response[i] = RequestResponseResponse{
			ID:              rr.ID,
			EndpointID:      rr.EndpointID,
			RequestHeaders:  rr.RequestHeaders,
			RequestBody:     rr.RequestBody,
			ResponseHeaders: rr.ResponseHeaders,
			ResponseBody:    rr.ResponseBody,
			CapturedAt:      rr.CapturedAt,
		}
	}
	c.JSON(http.StatusOK, response)
}
