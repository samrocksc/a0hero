package clients

import (
	"github.com/samrocksc/a0hero/modules/edit"
)

// ClientFields defines all editable fields for an Auth0 Client.
var ClientFields = []edit.FieldDef{
	// Section: Application Properties
	{
		Key:       "name",
		Label:     "Name",
		Type:      edit.FieldText,
		Required:  true,
		Placeholder: "My Application",
		Help:      "Human-readable identifier for your application",
	},
	{
		Key:       "description",
		Label:     "Description",
		Type:      edit.FieldTextarea,
		Placeholder: "Application description...",
		Help:      "Free-text field to describe the application's use cases",
	},
	{
		Key:       "app_type",
		Label:     "Application Type",
		Type:      edit.FieldSelect,
		Options:   []string{"spa", "regular_web", "native", "non_interactive", "machine"},
		Required:  true,
		Help:      "The type of application (affects token lifetime, allowed flows)",
	},
	{
		Key:       "logo_uri",
		Label:     "Application Logo URL",
		Type:      edit.FieldURL,
		Placeholder: "https://example.com/logo.png",
		Help:      "URL of the application logo (recommended: 150x150px)",
	},
	{
		Key:    "is_first_party",
		Label:  "First Party Application",
		Type:   edit.FieldBool,
		Help:   "Indicates if this is a first-party application (built-in)",
	},
	{
		Key:    "is_global",
		Label:  "Global Application",
		Type:   edit.FieldBool,
		Help:   "Indicates if this is a global application (e.g., Auth0 dashboard)",
	},

	// Section: Login URLs
	{
		Key:       "login_uri",
		Label:     "Custom Login Page URL",
		Type:      edit.FieldURL,
		Placeholder: "https://example.com/login",
		Help:      "URL of your custom login page (requires custom login page enabled)",
	},
	{
		Key:       "login_origin",
		Label:     "Custom Login Origin",
		Type:      edit.FieldURL,
		Placeholder: "https://example.com",
		Help:      "Origin of your custom login page server",
	},
	{
		Key:       "custom_login_page_preview",
		Label:     "Custom Login Page Preview URL",
		Type:      edit.FieldURL,
		Help:      "Preview URL for the custom login page",
	},

	// Section: Security URLs
	{
		Key:    "callbacks",
		Label:  "Allowed Callback URLs",
		Type:   edit.FieldTagArray,
		Help:   "URLs that Auth0 may redirect to after authentication",
	},
	{
		Key:    "logout_urls",
		Label:  "Allowed Logout URLs",
		Type:   edit.FieldTagArray,
		Help:   "URLs that Auth0 may redirect to after logout",
	},
	{
		Key:    "web_origins",
		Label:  "Web Origins",
		Type:   edit.FieldTagArray,
		Help:   "URLs for CORS requests from your application",
	},
	{
		Key:    "allowed_origins",
		Label:  "Allowed Origins (CORS)",
		Type:   edit.FieldTagArray,
		Help:   "Additional origins for CORS requests (includes web_origins)",
	},

	// Section: Mobile / Native
	{
		Key:    "allowed_clients",
		Label:  "Allowed Mobile Clients",
		Type:   edit.FieldTagArray,
		Help:   "Mobile SDKs that may use this application",
	},
	{
		Key:    "mobile",
		Label:  "Mobile Settings",
		Type:   edit.FieldTextarea,
		Help:   "JSON object for iOS/Android specific settings",
	},

	// Read-only fields (display only)
	{
		Key:      "client_id",
		Label:    "Client ID",
		Type:     edit.FieldText,
		ReadOnly: true,
	},
}

// ClientSections organizes fields into UI sections.
var ClientSections = []FieldSection{
	{
		Title: "Application Properties",
		Keys:  []string{"name", "description", "app_type", "logo_uri", "is_first_party", "is_global"},
	},
	{
		Title: "Login URLs",
		Keys:  []string{"login_uri", "login_origin", "custom_login_page_preview"},
	},
	{
		Title: "Security URLs",
		Keys:  []string{"callbacks", "logout_urls", "web_origins", "allowed_origins"},
	},
	{
		Title: "Mobile / Native",
		Keys:  []string{"allowed_clients", "mobile"},
	},
}

// FieldSection groups fields for UI display.
type FieldSection struct {
	Title string
	Keys  []string
}

// GetFieldsForSection returns fields for a specific section.
func GetFieldsForSection(sectionTitle string) []edit.FieldDef {
	for _, section := range ClientSections {
		if section.Title == sectionTitle {
			var fields []edit.FieldDef
			fieldMap := make(map[string]edit.FieldDef)
			for _, f := range ClientFields {
				fieldMap[f.Key] = f
			}
			for _, key := range section.Keys {
				if f, ok := fieldMap[key]; ok {
					fields = append(fields, f)
				}
			}
			return fields
		}
	}
	return nil
}
