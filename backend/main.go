package main

import (
	"log"
	"net/http"
	"os"                  // Import os package
	"path/filepath"       // Import filepath package
	"rewrite-go/config"   // Import the config package
	"rewrite-go/database" // Import the database package
	"rewrite-go/handlers" // Import the handlers package
	"strings"             // Import strings package

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// ServeScreenshot serves a specific screenshot file.
func ServeScreenshot(c *gin.Context) {
	// Get the requested file path from the URL parameter
	// The *filepath captures everything after /api/screenshots/
	requestedPath := c.Param("filepath")
	if requestedPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filepath parameter is missing"})
		return
	}

	// Construct the full path to the file on the server
	// IMPORTANT: Sanitize the path to prevent directory traversal attacks
	// Base directory where screenshots are stored, relative to project root
	// Assumes the executable is run from the project root directory.
	// The path stored in the DB (and thus requested in the URL) is relative to the project root,
	// e.g., "data/screenshots/scan_1/image.png".
	// Therefore, the base directory for serving should be the project root itself,
	// or we adjust how the final path is constructed.

	// Clean the requested path to remove any potentially malicious elements like '..'
	// The requestedPath already contains the "scan_X/image.png" part if the DB path is correct.
	// We need to ensure the join doesn't duplicate parts of the path.
	// Let's rethink the join logic based on the expected requestedPath format.
	// If requestedPath is "scan_1/image.png", then filepath.Join(baseDir, requestedPath) works.
	// If requestedPath is "data/screenshots/scan_1/image.png", we need to strip the prefix.

	// Let's assume the frontend requests `/api/screenshots/scan_1/image.png`
	// by taking the DB path `data/screenshots/scan_1/image.png` and stripping `data/screenshots/`
	// If that's the case, the current baseDir and join logic might be okay IF the DB path was different.
	// BUT, the DB path IS `data/screenshots/...`.

	// Revised approach: Assume requestedPath *is* the full relative path from the DB.
	// We need to construct the absolute path from the project root.
	// The baseDir should just be "." if the executable runs from the project root.
	// fullPath := filepath.Join(".", filepath.Clean("/"+requestedPath)) // Path relative to project root

	// Let's stick to the original logic but fix the baseDir:
	// baseDir is where the screenshot *types* are stored.
	// requestedPath is the specific scan/file part.

	// Re-evaluating: The DB stores `data/screenshots/scan_X/file.png`.
	// The API handler `GetEndpoint` returns this full path.
	// The frontend likely requests `/api/screenshots/data/screenshots/scan_X/file.png`.
	// So, `requestedPath` in `ServeScreenshot` will be `data/screenshots/scan_X/file.png`.
	// The original `baseDir` was `./backend/data/screenshots`. Joining resulted in `./backend/data/screenshots/data/screenshots/...` (WRONG).
	// The corrected `baseDir` is `./data/screenshots`. Joining results in `./data/screenshots/data/screenshots/...` (STILL WRONG).

	// The actual file path on disk is `./data/screenshots/scan_X/file.png` (relative to project root).
	// The `requestedPath` parameter contains `data/screenshots/scan_X/file.png`.
	// We need `filepath.Join(".", requestedPath)` but need to ensure security.

	// Revised Logic based on feedback:
	// The requestedPath from the URL seems to be relative *within* the screenshots dir,
	// e.g., "scan_1/image.png".
	// Define the base directory on the server where screenshots are stored.
	serverSideBaseDir := filepath.Join(".", "data", "screenshots")

	// Clean the user-provided path segment to prevent traversal like "../.." within it.
	// Prepending "/" ensures Clean treats it like an absolute path segment for cleaning purposes,
	// preventing it from potentially escaping the intended subdirectory if it starts with "..".
	cleanedRelativePath := filepath.Clean("/" + requestedPath)
	if strings.HasPrefix(cleanedRelativePath, "/..") || cleanedRelativePath == "/.." {
		// If cleaning results in trying to go above the root of the relative path, deny.
		log.Printf("Attempted directory traversal within relative path: %s", requestedPath)
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid path"})
		return
	}
	// Remove the leading "/" added for cleaning, as Join expects relative paths.
	cleanedRelativePath = strings.TrimPrefix(cleanedRelativePath, "/")

	// Construct the full path by joining the server's base screenshot directory
	// with the cleaned relative path provided in the request.
	fullPath := filepath.Join(serverSideBaseDir, cleanedRelativePath)

	// Security Check: Ensure the final resolved path is still prefixed by the server's base directory.
	// This is a crucial check against more complex traversal attacks.
	if !strings.HasPrefix(fullPath, serverSideBaseDir+string(filepath.Separator)) && fullPath != serverSideBaseDir {
		// Check prefix + separator to avoid matching "/base/dir" with "/base/directory"
		// Also allow exact match if requesting the base directory itself (though unlikely here).
		log.Printf("Security check failed: Path %s resolved outside base directory %s", fullPath, serverSideBaseDir)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if the file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Screenshot not found"})
		return
	} else if err != nil {
		log.Printf("Error checking screenshot file %s: %v", fullPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error accessing screenshot file"})
		return
	}

	// Serve the file
	// Set appropriate Content-Type header (optional but good practice)
	// c.Header("Content-Type", "image/png") // Assuming all screenshots are PNG
	c.File(fullPath)
}

func main() {
	// Initialize Database
	database.ConnectDatabase()
	database.MigrateDatabase()

	// Load Config (Load it early, e.g., after DB init)
	config.LoadConfig()

	// Create Gin router
	router := gin.Default()

	// Configure CORS
	// Mimics the FastAPI CORS settings
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"} // Allow SvelteKit dev server
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	// config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"} // Original
	config.AllowHeaders = []string{"*"} // Allow all headers for local dev testing
	config.AllowCredentials = true
	router.Use(cors.New(config))

	// Define root route
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Attack Surface Management API (Go Version)"})
	})

	// API Route Group
	api := router.Group("/api")
	{
		// Organization routes
		orgRoutes := api.Group("/organizations")
		{
			orgRoutes.POST("", handlers.CreateOrganization) // Also handle POST without trailing slash
			orgRoutes.GET("", handlers.GetOrganizations)    // Handle GET without trailing slash
			orgRoutes.GET("/:org_id", handlers.GetOrganization)
			// Add the organization-specific import route here
			orgRoutes.POST("/:org_id/import/urls", handlers.HandleImportURLs)
		}

		// Domain routes
		domainRoutes := api.Group("/domains")
		{
			domainRoutes.POST("", handlers.CreateDomain) // Handle POST without trailing slash
			domainRoutes.GET("", handlers.GetDomains)    // Handle GET without trailing slash
			domainRoutes.GET("/:domain_id", handlers.GetDomain)
			// Removed deprecated domain-specific scan route: POST /:domain_id/scan
		}

		// Subdomain routes
		subdomainRoutes := api.Group("/subdomains")
		{
			subdomainRoutes.GET("", handlers.GetSubdomains) // Handle GET without trailing slash
			subdomainRoutes.GET("/:subdomain_id", handlers.GetSubdomain)
			subdomainRoutes.GET("/:subdomain_id/endpoints", handlers.GetSubdomainEndpoints)
		}

		// Endpoint routes
		endpointRoutes := api.Group("/endpoints")
		{
			endpointRoutes.GET("", handlers.GetEndpoints) // Handle GET without trailing slash
			endpointRoutes.GET("/:endpoint_id", handlers.GetEndpoint)
			endpointRoutes.GET("/:endpoint_id/parameters", handlers.GetEndpointParameters)
			endpointRoutes.GET("/:endpoint_id/request-responses", handlers.GetEndpointRequestResponses)
		}

		// Technology routes
		techRoutes := api.Group("/technologies")
		{
			techRoutes.GET("", handlers.GetTechnologies) // Handle GET without trailing slash
			techRoutes.GET("/:technology_id", handlers.GetTechnology)
			techRoutes.GET("/:technology_id/domains", handlers.GetDomainsWithTechnology)
			techRoutes.GET("/:technology_id/subdomains", handlers.GetSubdomainsWithTechnology)
			techRoutes.GET("/:technology_id/endpoints", handlers.GetEndpointsWithTechnology)
		}

		// Scan routes
		scanRoutes := api.Group("/scans")
		{
			scanRoutes.POST("", handlers.StartScan) // Add route for starting scans (root or subdomain)
			scanRoutes.GET("", handlers.GetScans)   // Handle GET without trailing slash
			scanRoutes.GET("/:id", handlers.GetScan)
		}

		// Scan Template routes
		scanTemplateRoutes := api.Group("/scan-templates")
		{
			scanTemplateRoutes.POST("", handlers.CreateScanTemplate)
			scanTemplateRoutes.GET("", handlers.GetScanTemplates)
			scanTemplateRoutes.GET("/:template_id", handlers.GetScanTemplate)
			scanTemplateRoutes.PUT("/:template_id", handlers.UpdateScanTemplate)
			scanTemplateRoutes.DELETE("/:template_id", handlers.DeleteScanTemplate)
		}

		// Graph routes
		graphRoutes := api.Group("/graph")
		{
			graphRoutes.GET("", handlers.GetGraphData) // Handle GET without trailing slash
		}

		// Settings routes
		settingsRoutes := api.Group("/settings")
		{
			// Wrap standard http handlers for Gin
			settingsRoutes.GET("", gin.WrapF(handlers.GetSettingsHandler))
			settingsRoutes.POST("", gin.WrapF(handlers.SaveSettingsHandler))
		}

		// Screenshot serving route (outside specific resource groups)
		api.GET("/screenshots/*filepath", ServeScreenshot)

		// Import routes are now nested under organizations
		// Remove the old top-level import route group
	}

	// Remove the duplicated orgRoutes group below

	// Start server
	port := "8080" // Use a different port than the Python version (8000)
	log.Printf("Starting Go server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to run server:", err)
	}
}
