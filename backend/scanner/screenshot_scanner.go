package scanner

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"rewrite-go/database"
	"rewrite-go/models"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// List of common user agents
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/109.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/109.0",
	"Mozilla/5.0 (X11; Linux i686; rv:109.0) Gecko/20100101 Firefox/109.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/109.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Mobile Safari/537.36",
}

// Seed the random number generator once
func init() {
	rand.Seed(time.Now().UnixNano())
}

// TakeScreenshot captures a screenshot of the given URL and saves it.
// It also records the screenshot metadata in the database.
func TakeScreenshot(ctx context.Context, targetURL string, scanID uint, subdomainID *uint, endpointID *uint) error {
	// Ensure the screenshots directory exists
	screenshotDir := filepath.Join(".", "data", "screenshots", fmt.Sprintf("scan_%d", scanID))
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		return fmt.Errorf("failed to create screenshot directory %s: %w", screenshotDir, err)
	}

	// Generate a unique filename based on the URL and timestamp
	safeFilename := strings.ReplaceAll(targetURL, "://", "_")
	safeFilename = strings.ReplaceAll(safeFilename, "/", "_")
	safeFilename = strings.ReplaceAll(safeFilename, ":", "_")
	safeFilename = strings.ReplaceAll(safeFilename, "?", "_")
	safeFilename = strings.ReplaceAll(safeFilename, "&", "_")
	if len(safeFilename) > 100 { // Limit filename length
		safeFilename = safeFilename[:100]
	}
	filename := fmt.Sprintf("%s_%d.png", safeFilename, time.Now().UnixNano())
	filePath := filepath.Join(screenshotDir, filename)

	// Select a random user agent
	randomUserAgent := userAgents[rand.Intn(len(userAgents))]
	log.Printf("Using User-Agent: %s for %s", randomUserAgent, targetURL)

	// Create a new chromedp context with random user agent
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("ignore-certificate-errors", true), // Ignore SSL errors
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true), // Often needed in containerized environments
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(randomUserAgent), // Set the random user agent
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancelTask()

	// Set a timeout for the screenshot task
	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, 120*time.Second) // 120-second timeout (increased from 60)
	defer cancelTimeout()

	var buf []byte
	log.Printf("Attempting to take screenshot of: %s", targetURL)
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(targetURL),
		// Wait for the page to load (adjust time as needed, or use other wait conditions)
		// chromedp.Sleep(5*time.Second), // Simple wait
		chromedp.WaitVisible(`body`, chromedp.ByQuery), // Wait for body element
		// Capture screenshot
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, err = page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatPng).
				WithQuality(80). // Adjust quality (0-100)
				Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to capture screenshot: %w", err)
			}
			return nil
		}),
	)

	if err != nil {
		// Don't treat screenshot failure as a fatal scan error, just log it
		log.Printf("Error taking screenshot for %s: %v", targetURL, err)
		return nil // Return nil to allow the scan to continue
	}

	// Save the screenshot buffer to a file
	if err := os.WriteFile(filePath, buf, 0644); err != nil {
		log.Printf("Error saving screenshot file %s: %v", filePath, err)
		return nil // Continue scan even if saving fails
	}

	log.Printf("Successfully saved screenshot for %s to %s", targetURL, filePath)

	// Save screenshot metadata to the database
	screenshot := models.Screenshot{
		SubdomainID: subdomainID,
		EndpointID:  endpointID,
		URL:         targetURL,
		FilePath:    filePath, // Store the relative path
		ScanID:      scanID,
		CapturedAt:  time.Now(),
	}

	db := database.GetDB()
	if result := db.Create(&screenshot); result.Error != nil {
		log.Printf("Error saving screenshot metadata for %s to database: %v", targetURL, result.Error)
		// Log the error but don't stop the scan
	}

	return nil // Screenshot taken (or failed non-fatally)
}

// ShouldScreenshot checks if a URL should be screenshotted based on its extension.
// It screenshots any URL unless it explicitly ends with one of the excludedExtensions.
func ShouldScreenshot(urlStr string) bool {
	lowerURL := strings.ToLower(urlStr)
	if strings.Contains(lowerURL, "?") {
		lowerURL = lowerURL[:strings.Index(lowerURL, "?")] // Ignore query parameters for extension check
	}

	// Check for extensions to exclude
	excludedExtensions := []string{
		".js", ".css", ".json", ".xml", ".txt", ".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".ppt", ".pptx", ".zip", ".rar", ".tar", ".gz", ".7z", ".jpg", ".jpeg", ".gif",
		".png", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".mp4", ".mp3", ".avi",
		".mov", ".csv", ".map", ".yaml", ".yml", ".md",
	}
	for _, ext := range excludedExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return false // Don't screenshot if it has an excluded extension
		}
	}

	// If it didn't match any excluded extension, screenshot it.
	return true
}
