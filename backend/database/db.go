package database

import (
	"encoding/json"
	"log"
	"os"
	"rewrite-go/models" // Import the models package

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectDatabase initializes the database connection using GORM.
func ConnectDatabase() {
	var err error
	// Use a database file within the 'new' directory.
	// This path assumes the executable is run from within the 'new' directory.
	dbPath := "./asm_go.db" // Path relative to the 'new' directory

	// Configure GORM logger (optional, similar to echo=True)
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             0,           // Log all SQL
			LogLevel:                  logger.Info, // LogLevel
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)

	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: newLogger, // Use configured logger
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connection successfully opened")
}

// MigrateDatabase runs GORM's auto-migration feature.
func MigrateDatabase() {
	if DB == nil {
		log.Fatal("Database connection is not initialized. Call ConnectDatabase first.")
	}
	log.Println("Running database migrations...")
	// GORM needs pointers to the structs for migration
	err := DB.AutoMigrate(
		&models.Organization{},
		&models.RootDomain{},
		&models.Subdomain{},
		&models.Endpoint{},
		&models.Parameter{},
		&models.Technology{},
		&models.SubdomainTechnology{}, // Join table
		&models.EndpointTechnology{},  // Join table
		&models.RequestResponse{},
		&models.Scan{},
		&models.ScanTemplate{},
		&models.Screenshot{}, // Add the new Screenshot model
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	log.Println("Database migration completed.")

	// Seed default scan templates
	seedDefaultScanTemplates(DB)
}

// seedDefaultScanTemplates inserts default scan templates if they don't exist.
func seedDefaultScanTemplates(db *gorm.DB) {
	log.Println("Seeding default scan templates...")

	// --- Define Default Configurations using the nested structure ---

	// Default Subdomain Config (Enabled Section, Enabled Tools)
	defaultSubdomainSection := models.ScanSectionConfig{
		Enabled: true,
		Tools: map[string]models.ScanToolConfig{
			"subfinder": {
				Enabled: true,
				Options: []string{"--threads=10", "--timeout=30", "--maxEnumerationTime=10"}, // Use string options
			},
			"crtsh": {
				Enabled: true,
				Options: []string{}, // No specific options for crt.sh fetcher
			},
		},
	}
	subdomainConfigJSON, _ := json.Marshal(defaultSubdomainSection)

	// Default URL Config (Enabled Section, Enabled Tool)
	defaultURLSection := models.ScanSectionConfig{
		Enabled: true,
		Tools: map[string]models.ScanToolConfig{
			"katana": { // Assuming 'katana' is the key used in the scanner
				Enabled: true,
				// Add "outputFile" to enable file output by default for this template
				Options: []string{"--max-depth=2", "--concurrency=25", "--parallelism=10", "--rate-limit=150", "--timeout=10", "outputFile"},
			},
		},
	}
	urlConfigJSON, _ := json.Marshal(defaultURLSection)

	// Disabled Subdomain Config
	disabledSubdomainSection := models.ScanSectionConfig{
		Enabled: false,                              // Section disabled
		Tools:   map[string]models.ScanToolConfig{}, // Empty tools map
	}
	emptySubdomainConfigJSON, _ := json.Marshal(disabledSubdomainSection)

	// Disabled URL Config
	disabledURLSection := models.ScanSectionConfig{
		Enabled: false,                              // Section disabled
		Tools:   map[string]models.ScanToolConfig{}, // Empty tools map
	}
	emptyURLConfigJSON, _ := json.Marshal(disabledURLSection)

	// --- Define Templates ---
	templates := []models.ScanTemplate{
		{
			Name:                "Default Subdomain Scan",
			Description:         "Scans for subdomains using standard tools.",
			SubdomainScanConfig: string(subdomainConfigJSON),
			URLScanConfig:       string(emptyURLConfigJSON), // Disable URL scanning
			TechDetectEnabled:   false,
			ScreenshotEnabled:   false, // Add ScreenshotEnabled
		},
		{
			Name:                "Default URL Scan",
			Description:         "Scans for URLs/endpoints on known subdomains.",
			SubdomainScanConfig: string(emptySubdomainConfigJSON), // Disable subdomain scanning
			URLScanConfig:       string(urlConfigJSON),
			TechDetectEnabled:   false,
			ScreenshotEnabled:   false, // Add ScreenshotEnabled
		},
		{
			Name:                "Default Technology Detection",
			Description:         "Identifies technologies on known subdomains/endpoints.",
			SubdomainScanConfig: string(emptySubdomainConfigJSON), // Disable subdomain scanning
			URLScanConfig:       string(emptyURLConfigJSON),       // Disable URL scanning
			TechDetectEnabled:   true,
			ScreenshotEnabled:   false, // Add ScreenshotEnabled
		},
		// Optional: A full scan template
		{
			Name:                "Default Full Scan",
			Description:         "Performs subdomain discovery, URL scanning, and technology detection.",
			SubdomainScanConfig: string(subdomainConfigJSON),
			URLScanConfig:       string(urlConfigJSON),
			TechDetectEnabled:   true,
			ScreenshotEnabled:   true, // Add ScreenshotEnabled
		},
	}

	for _, tmpl := range templates {
		// Check if a template with the same name already exists
		var existing models.ScanTemplate
		result := db.Where("name = ?", tmpl.Name).First(&existing)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// Template doesn't exist, create it
				if err := db.Create(&tmpl).Error; err != nil {
					log.Printf("Failed to create default template '%s': %v\n", tmpl.Name, err)
				} else {
					log.Printf("Created default template: '%s'\n", tmpl.Name)
				}
			} else {
				// Other database error
				log.Printf("Error checking for template '%s': %v\n", tmpl.Name, result.Error)
			}
		} else {
			log.Printf("Default template '%s' already exists, skipping.\n", tmpl.Name)
		}
	}
	log.Println("Finished seeding default scan templates.")
}

// GetDB returns the initialized GORM DB instance.
// In a real app, you might manage sessions differently (e.g., per request).
func GetDB() *gorm.DB {
	if DB == nil {
		log.Fatal("Database connection is not initialized.")
	}
	return DB
}
