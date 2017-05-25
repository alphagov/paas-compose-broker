package compose

import "net/url"

// Client stores compose token and API URL
type Client struct {
	Token  string
	APIURL *url.URL
}

// Response stores compose API response json
type Response struct {
	Embedded struct {
		Deployments []Deployment `json:"deployments"`
		Accounts    []Account    `json:"accounts"`
		Recipes     []Recipe     `json:"recipes"`
	} `json:"_embedded"`
}

// Account stores information about account to which your user is a member.
type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

// Recipe is a process, or processes, which performs a task
type Recipe struct {
	ID           string `json:"id"`
	Template     string `json:"template"`
	Status       string `json:"status"`
	StatusDetail string `json:"status_detail"`
	AccountID    string `json:"account_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	DeploymentID string `json:"deployment_id"`
	Name         string `json:"name"`
}

// Deployment stores deployment data
type Deployment struct {
	ID                  string             `json:"id,omitempty"`
	AccountID           string             `json:"account_id,omitempty"`
	Name                string             `json:"name,omitempty"`
	CreatedAt           string             `json:"created_at,omitempty"`
	Type                string             `json:"type,omitempty"`
	Datacenter          string             `json:"datacenter,omitempty"`
	Version             string             `json:"version,omitempty"`
	Units               int                `json:"units,omitempty"`
	SSL                 bool               `json:"ssl,omitempty"`
	WiredTiger          bool               `json:"wired_tiger,omitempty"`
	Links               *Links             `json:"_links,omitempty"`
	ProvisionRecipeID   string             `json:"provision_recipe_id,omitempty"`
	CaCertificateBase64 string             `json:"ca_certificate_base64,omitempty"`
	ConnectionStrings   *ConnectionStrings `json:"connection_strings,omitempty"`
}

// ConnectionStrings stores connection strings returned by deployment
type ConnectionStrings struct {
	Health   []string `json:"health,omitempty"`
	SSH      []string `json:"ssh,omitempty"`
	Admin    []string `json:"admin,omitempty"`
	SSHAdmin []string `json:"ssh_admin,omitempty"`
	Cli      []string `json:"cli,omitempty"`
	Direct   []string `json:"direct,omitempty"`
}

// Links stores UI URLs
type Links struct {
	ComposeWebUI struct {
		Href      string `json:"href,omitempty"`
		Templated bool   `json:"templated,omitempty"`
	} `json:"compose_web_ui,omitempty"`
}
