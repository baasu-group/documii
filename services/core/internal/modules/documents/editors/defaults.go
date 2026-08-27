package editors

type ProviderConfiguration struct {
	AirGapped                   bool
	CollaboraEnabled            bool
	CollaboraReady              bool
	CollaboraUnavailableReason  string
	OnlyOfficeEnabled           bool
	OnlyOfficeReady             bool
	OnlyOfficeUnavailableReason string
}

func DefaultProviders(configuration ProviderConfiguration) []Provider {
	officeFamilies := []DocumentFamily{FamilyText, FamilySpreadsheet, FamilyPresentation, FamilyPDF}
	browserOfficeCapabilities := []Capability{
		CapabilityCreate,
		CapabilityEdit,
		CapabilityView,
		CapabilityExport,
		CapabilityCollaborative,
		CapabilityBrowser,
	}

	collaboraAvailability := configuredAvailability(configuration.CollaboraEnabled, configuration.CollaboraReady, configuration.CollaboraUnavailableReason)
	onlyOfficeAvailability := configuredAvailability(configuration.OnlyOfficeEnabled, configuration.OnlyOfficeReady, configuration.OnlyOfficeUnavailableReason)
	libreOfficeAvailability := Availability{Available: false, Reason: "desktop_detection_required"}

	googleAvailability := Availability{Available: false, Reason: "connector_not_configured"}
	if configuration.AirGapped {
		googleAvailability.Reason = "air_gapped"
	}

	return []Provider{
		NewNativeProvider(),
		NewStaticProvider(ProviderCollabora, Capabilities{Operations: browserOfficeCapabilities, Families: officeFamilies}, collaboraAvailability),
		NewStaticProvider(ProviderOnlyOffice, Capabilities{Operations: browserOfficeCapabilities, Families: officeFamilies}, onlyOfficeAvailability),
		NewStaticProvider(
			ProviderLibreOfficeDesktop,
			Capabilities{
				Operations: []Capability{CapabilityCreate, CapabilityEdit, CapabilityView, CapabilityConvert, CapabilityExport, CapabilityDesktop, CapabilityOffline, CapabilityExternalProcess},
				Families:   officeFamilies,
			},
			libreOfficeAvailability,
		),
		NewStaticProvider(
			ProviderGoogleDocs,
			Capabilities{Operations: []Capability{CapabilityCreate, CapabilityEdit, CapabilityView, CapabilityExport, CapabilityCollaborative, CapabilityBrowser, CapabilityExternal}, Families: []DocumentFamily{FamilyText}},
			googleAvailability,
		),
		NewStaticProvider(
			ProviderGoogleSheets,
			Capabilities{Operations: []Capability{CapabilityCreate, CapabilityEdit, CapabilityView, CapabilityExport, CapabilityCollaborative, CapabilityBrowser, CapabilityExternal}, Families: []DocumentFamily{FamilySpreadsheet}},
			googleAvailability,
		),
		NewStaticProvider(
			ProviderGoogleSlides,
			Capabilities{Operations: []Capability{CapabilityCreate, CapabilityEdit, CapabilityView, CapabilityExport, CapabilityCollaborative, CapabilityBrowser, CapabilityExternal}, Families: []DocumentFamily{FamilyPresentation}},
			googleAvailability,
		),
	}
}

func configuredAvailability(enabled, ready bool, unavailableReason string) Availability {
	if !enabled {
		return Availability{Available: false, Reason: "disabled"}
	}
	if !ready {
		if unavailableReason == "" {
			unavailableReason = "invalid_configuration"
		}
		return Availability{Available: false, Reason: unavailableReason}
	}
	return Availability{Available: true}
}
