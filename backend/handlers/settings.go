package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"rewrite-go/config" // Use the correct module path from go.mod
)

// GetSettingsHandler handles GET requests to /api/settings
func GetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	settings := config.GetAll() // Get all current settings

	// Filter out sensitive keys if necessary before sending to frontend
	// For now, sending all keys. Consider security implications.
	// Example filtering:
	// safeSettings := make(map[string]string)
	// allowedKeys := []string{"SOME_SAFE_KEY"} // Define keys safe to expose
	// for _, key := range allowedKeys {
	//     if val, ok := settings[key]; ok {
	//         safeSettings[key] = val
	//     }
	// }

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(settings); err != nil {
		log.Printf("Error encoding settings response: %v", err)
		http.Error(w, "Failed to encode settings", http.StatusInternalServerError)
	}
}

// SaveSettingsHandler handles POST requests to /api/settings
func SaveSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var newSettings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		log.Printf("Error decoding settings request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Basic validation (optional): Check if keys are expected, etc.
	// ...

	if err := config.Save(newSettings); err != nil {
		log.Printf("Error saving settings: %v", err)
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Or http.StatusNoContent if no body is returned
	json.NewEncoder(w).Encode(map[string]string{"message": "Settings saved successfully"})
}
