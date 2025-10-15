package entities

// BrandingTemplate represents a pre-defined branding template
type BrandingTemplate struct {
	Name        string                `json:"name"`
	DisplayName string                `json:"display_name"`
	Description string                `json:"description"`
	Colors      CompanyBrandingEntity `json:"colors"`
}

// DefaultTemplates contains all pre-defined branding templates
var DefaultTemplates = []BrandingTemplate{
	{
		Name:        "modern_pink",
		DisplayName: "Rosa Moderno",
		Description: "Design vibrante e moderno em tons de rosa",
		Colors: CompanyBrandingEntity{
			TemplateName:       "modern_pink",
			HeaderBgColor:      "#ec4899",
			HeaderTextColor:    "#ffffff",
			PrimaryColor:       "#db2777",
			PrimaryTextColor:   "#ffffff",
			SecondaryColor:     "#f9a8d4",
			SecondaryTextColor: "#831843",
			TitleColor:         "#be185d",
			BodyTextColor:      "#404040",
			AccentColor:        "#a855f7",
			BorderColor:        "#fbcfe8",
			BackgroundColor:    "#ffffff",
			TableHeaderBgColor: "#fce7f3",
			TableRowAltBgColor: "#fdf2f8",
		},
	},
	{
		Name:        "corporate_blue",
		DisplayName: "Azul Corporativo",
		Description: "Profissional e confiável em tons de azul",
		Colors: CompanyBrandingEntity{
			TemplateName:       "corporate_blue",
			HeaderBgColor:      "#1e40af",
			HeaderTextColor:    "#ffffff",
			PrimaryColor:       "#3b82f6",
			PrimaryTextColor:   "#ffffff",
			SecondaryColor:     "#60a5fa",
			SecondaryTextColor: "#1e3a8a",
			TitleColor:         "#1e40af",
			BodyTextColor:      "#374151",
			AccentColor:        "#0ea5e9",
			BorderColor:        "#93c5fd",
			BackgroundColor:    "#ffffff",
			TableHeaderBgColor: "#dbeafe",
			TableRowAltBgColor: "#eff6ff",
		},
	},
	{
		Name:        "tech_green",
		DisplayName: "Verde Tecnologia",
		Description: "Sustentável e inovador em tons de verde",
		Colors: CompanyBrandingEntity{
			TemplateName:       "tech_green",
			HeaderBgColor:      "#059669",
			HeaderTextColor:    "#ffffff",
			PrimaryColor:       "#10b981",
			PrimaryTextColor:   "#ffffff",
			SecondaryColor:     "#34d399",
			SecondaryTextColor: "#064e3b",
			TitleColor:         "#047857",
			BodyTextColor:      "#374151",
			AccentColor:        "#14b8a6",
			BorderColor:        "#a7f3d0",
			BackgroundColor:    "#ffffff",
			TableHeaderBgColor: "#d1fae5",
			TableRowAltBgColor: "#ecfdf5",
		},
	},
	{
		Name:        "elegant_purple",
		DisplayName: "Roxo Elegante",
		Description: "Sofisticado e criativo em tons de roxo",
		Colors: CompanyBrandingEntity{
			TemplateName:       "elegant_purple",
			HeaderBgColor:      "#7c3aed",
			HeaderTextColor:    "#ffffff",
			PrimaryColor:       "#8b5cf6",
			PrimaryTextColor:   "#ffffff",
			SecondaryColor:     "#a78bfa",
			SecondaryTextColor: "#5b21b6",
			TitleColor:         "#6d28d9",
			BodyTextColor:      "#374151",
			AccentColor:        "#d946ef",
			BorderColor:        "#c4b5fd",
			BackgroundColor:    "#ffffff",
			TableHeaderBgColor: "#ede9fe",
			TableRowAltBgColor: "#f5f3ff",
		},
	},
}

// GetDefaultTemplate returns the default branding template (modern_pink)
func GetDefaultTemplate() *CompanyBrandingEntity {
	defaultTemplate := DefaultTemplates[0].Colors
	return &defaultTemplate
}

// GetTemplateByName returns a template by name, or default if not found
func GetTemplateByName(name string) *CompanyBrandingEntity {
	for _, template := range DefaultTemplates {
		if template.Name == name {
			colors := template.Colors
			return &colors
		}
	}
	return GetDefaultTemplate()
}
