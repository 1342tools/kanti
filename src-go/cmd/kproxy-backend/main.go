package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/1342tools/kproxy/backend/internal/ipc"
	"github.com/1342tools/kproxy/backend/internal/proxy"
	"github.com/1342tools/kproxy/backend/pkg/models"
)

func main() {
	// Parse command-line flags
	var (
		dataDir   = flag.String("data", getDefaultDataDir(), "Data directory for certificates and cache")
		ipcPort   = flag.Int("ipc-port", 9090, "IPC server port")
		proxyPort = flag.Int("proxy-port", 8080, "Proxy server port")
	)
	flag.Parse()

	log.Println("KProxy Backend starting...")
	log.Printf("Data directory: %s\n", *dataDir)
	log.Printf("IPC port: %d\n", *ipcPort)
	log.Printf("Proxy port: %d\n", *proxyPort)

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v\n", err)
	}

	// Create proxy configuration
	config := &models.ProxyConfig{
		Port:            *proxyPort,
		SSLInterception: true,
		CustomHeaders:   make(map[string]string),
		SaveOnlyInScope: false,
		InScope:         []string{},
		OutOfScope:      []string{},
	}

	// Initialize proxy server
	proxyServer, err := proxy.NewProxyServer(*dataDir, config)
	if err != nil {
		log.Fatalf("Failed to create proxy server: %v\n", err)
	}

	log.Printf("Proxy server initialized (CA cert: %s)\n", proxyServer.GetCertificatePath())

	// Initialize IPC server
	ipcServer := ipc.NewServer(proxyServer, *ipcPort)

	// Start IPC server in a goroutine
	go func() {
		log.Printf("Starting IPC server on port %d...\n", *ipcPort)
		if err := ipcServer.Start(); err != nil {
			log.Printf("IPC server stopped: %v\n", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("KProxy Backend ready. Waiting for commands via IPC...")
	log.Println("Press Ctrl+C to exit")

	// Wait for interrupt signal
	<-sigChan

	log.Println("\nShutting down...")

	// Stop proxy if running
	if proxyServer.IsRunning() {
		log.Println("Stopping proxy server...")
		if err := proxyServer.Stop(); err != nil {
			log.Printf("Error stopping proxy server: %v\n", err)
		}
	}

	// Stop IPC server
	log.Println("Stopping IPC server...")
	if err := ipcServer.Stop(); err != nil {
		log.Printf("Error stopping IPC server: %v\n", err)
	}

	log.Println("Shutdown complete")
}

// getDefaultDataDir returns the default data directory based on OS
func getDefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./data"
	}

	return filepath.Join(home, ".kproxy")
}
