package config

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

const configFilePath = "config.json" // Relative path from where the binary is run (should be project root)

var (
	cfg  map[string]string
	once sync.Once
	mu   sync.RWMutex
)

// LoadConfig loads the configuration from the JSON file.
// It's safe for concurrent use due to sync.Once.
func LoadConfig() {
	once.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		cfg = make(map[string]string) // Initialize the map

		data, err := os.ReadFile(configFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("Config file '%s' not found, using empty configuration.", configFilePath)
				// Create an empty file if it doesn't exist
				if err := os.WriteFile(configFilePath, []byte("{}"), 0644); err != nil {
					log.Printf("Warning: Could not create empty config file '%s': %v", configFilePath, err)
				}
				return // Return with empty cfg map
			}
			log.Printf("Error reading config file '%s': %v. Using empty configuration.", configFilePath, err)
			return // Return with empty cfg map
		}

		// Ensure data is not empty before trying to unmarshal
		if len(data) == 0 || string(data) == "{}" {
			log.Printf("Config file '%s' is empty or just '{}', using empty configuration.", configFilePath)
			return // Return with empty cfg map
		}

		err = json.Unmarshal(data, &cfg)
		if err != nil {
			log.Printf("Error unmarshalling config file '%s': %v. Using empty configuration.", configFilePath, err)
			cfg = make(map[string]string) // Reset to empty map on error
			return
		}
		log.Printf("Configuration loaded successfully from %s", configFilePath)
	})
}

// Get returns the value for a given key from the configuration.
// It ensures the config is loaded before accessing.
func Get(key string) string {
	LoadConfig() // Ensure config is loaded
	mu.RLock()
	defer mu.RUnlock()
	return cfg[key] // Returns empty string if key doesn't exist
}

// GetAll returns a copy of the entire configuration map.
func GetAll() map[string]string {
	LoadConfig() // Ensure config is loaded
	mu.RLock()
	defer mu.RUnlock()
	// Return a copy to prevent external modification
	copyCfg := make(map[string]string, len(cfg))
	for k, v := range cfg {
		copyCfg[k] = v
	}
	return copyCfg
}

// Save saves the current configuration map back to the JSON file.
func Save(newCfg map[string]string) error {
	LoadConfig() // Ensure config is loaded initially (though we overwrite)
	mu.Lock()
	defer mu.Unlock()

	// Update the global cfg variable
	cfg = make(map[string]string, len(newCfg))
	for k, v := range newCfg {
		// Optionally filter out empty keys before saving
		// if v != "" {
		//  cfg[k] = v
		// }
		cfg[k] = v // Saving all keys for now, including potentially empty ones
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("Error marshalling config to JSON: %v", err)
		return err
	}

	err = os.WriteFile(configFilePath, data, 0644)
	if err != nil {
		log.Printf("Error writing config file '%s': %v", configFilePath, err)
		return err
	}
	log.Printf("Configuration saved successfully to %s", configFilePath)
	return nil
}
