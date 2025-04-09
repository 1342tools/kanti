package handlers

import (
	"fmt"
	"net/http"
	"rewrite-go/database"
	"rewrite-go/models"

	"github.com/gin-gonic/gin"
)

// --- Response Structs ---

// NodeData represents a node in the graph visualization.
// Omitting X, Y as layout will be handled by frontend.
type NodeData struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Size  int    `json:"size"`
	Color string `json:"color"`
	// X     float64 `json:"x,omitempty"` // Layout handled by frontend
	// Y     float64 `json:"y,omitempty"` // Layout handled by frontend
}

// LinkData represents a link (edge) between two nodes.
// Using 'from'/'to' convention often used by vis.js.
type LinkData struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// NodeProperties defines visual attributes for different node types.
var NodeProperties = map[string]map[string]interface{}{
	"domain":    {"size": 15, "color": "#ff6b6b"},
	"subdomain": {"size": 12, "color": "#48dbfb"},
	"endpoint":  {"size": 8, "color": "#1dd1a1"},
	"parameter": {"size": 5, "color": "#f368e0"},
}

// --- Handler Function ---

// GetGraphData handles GET requests to retrieve graph data.
func GetGraphData(c *gin.Context) {
	db := database.GetDB()
	var domains []models.RootDomain

	// Fetch all domains, eagerly loading all nested relationships needed for the graph
	result := db.Preload("Subdomains.Endpoints.Parameters").Find(&domains)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve graph data", "details": result.Error.Error()})
		return
	}

	nodesMap := make(map[string]NodeData) // Use map to easily check for existing nodes
	var links []LinkData

	// Helper to add node if it doesn't exist
	addNodeIfNotExists := func(nodeID, nodeType, label string) {
		if _, exists := nodesMap[nodeID]; !exists {
			props, ok := NodeProperties[nodeType]
			if !ok {
				props = map[string]interface{}{"size": 5, "color": "#cccccc"} // Default props
			}
			nodesMap[nodeID] = NodeData{
				ID:    nodeID,
				Label: label,
				Type:  nodeType,
				Size:  props["size"].(int), // Type assertion
				Color: props["color"].(string),
			}
		}
	}

	// Helper to add link
	addLink := func(sourceID, targetID string) {
		// Ensure both nodes exist before adding link (should always be true with this logic)
		if _, existsSrc := nodesMap[sourceID]; existsSrc {
			if _, existsTgt := nodesMap[targetID]; existsTgt {
				links = append(links, LinkData{From: sourceID, To: targetID})
			}
		}
	}

	// Process data and build node/link structures
	for _, domain := range domains {
		domainID := fmt.Sprintf("domain_%d", domain.ID)
		addNodeIfNotExists(domainID, "domain", domain.Domain)

		for _, subdomain := range domain.Subdomains {
			subdomainID := fmt.Sprintf("subdomain_%d", subdomain.ID)
			addNodeIfNotExists(subdomainID, "subdomain", subdomain.Hostname)
			addLink(domainID, subdomainID)

			for _, endpoint := range subdomain.Endpoints {
				endpointLabel := fmt.Sprintf("%s %s", endpoint.Method, endpoint.Path)
				endpointID := fmt.Sprintf("endpoint_%d", endpoint.ID)
				addNodeIfNotExists(endpointID, "endpoint", endpointLabel)
				addLink(subdomainID, endpointID)

				for _, parameter := range endpoint.Parameters {
					paramID := fmt.Sprintf("param_%d", parameter.ID)
					addNodeIfNotExists(paramID, "parameter", parameter.Name)
					addLink(endpointID, paramID)
				}
			}
		}
	}

	// Convert map values to slice for response
	nodes := make([]NodeData, 0, len(nodesMap))
	for _, node := range nodesMap {
		nodes = append(nodes, node)
	}

	c.JSON(http.StatusOK, gin.H{"nodes": nodes, "links": links})
}
