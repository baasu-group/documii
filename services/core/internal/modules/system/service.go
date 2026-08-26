package system

type Info struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	AirGapped   bool   `json:"airGapped"`
}

type Service struct {
	info Info
}

func NewService(version, environment string, airGapped bool) *Service {
	return &Service{info: Info{Name: "Documi", Version: version, Environment: environment, AirGapped: airGapped}}
}

func (service *Service) Info() Info {
	return service.info
}
