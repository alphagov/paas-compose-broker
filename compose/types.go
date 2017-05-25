package compose

import "net/url"

// Client will define some parameters and be used to communicate with the
// Compose API.
type Client struct {
	Token  string
	APIURL *url.URL
}

// Response will be responsible struct for whatever comes back from the API at
// a time.
type Response struct {
	Embedded struct {
		Deployments []Deployment `json:"deployments"`
		Accounts    []Account    `json:"accounts"`
		Recipes     []Recipe     `json:"recipes"`
	} `json:"_embedded"`
}

// Account stores information of your user belonging.
type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

// Recipe is a process, which performs a job on Compose backend.
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

// Deployment will consist of some useful data, that will comeback from Compose
// regarding our specific deployment.
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

// ConnectionStrings are part of the deployment. These are ready to use strings,
// which we could pass onto our app.
type ConnectionStrings struct {
	Health   []string `json:"health,omitempty"`
	SSH      []string `json:"ssh,omitempty"`
	Admin    []string `json:"admin,omitempty"`
	SSHAdmin []string `json:"ssh_admin,omitempty"`
	Cli      []string `json:"cli,omitempty"`
	Direct   []string `json:"direct,omitempty"`
}

// Links to different UI components returned as part of the deployment.
type Links struct {
	ComposeWebUI struct {
		Href      string `json:"href,omitempty"`
		Templated bool   `json:"templated,omitempty"`
	} `json:"compose_web_ui,omitempty"`
}
