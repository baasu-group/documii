package collabora

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type discoveryDocument struct {
	Zones []struct {
		Apps []struct {
			Name    string `xml:"name,attr"`
			Actions []struct {
				Extension string `xml:"ext,attr"`
				Name      string `xml:"name,attr"`
				URL       string `xml:"urlsrc,attr"`
			} `xml:"action"`
		} `xml:"app"`
	} `xml:"net-zone"`
}
type actionKey struct{ extension, mode string }
type Discovery struct {
	base    *url.URL
	client  *http.Client
	mu      sync.Mutex
	expires time.Time
	actions map[actionKey]string
}

func NewDiscovery(rawURL string) (*Discovery, error) {
	base, err := url.Parse(rawURL)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil, fmt.Errorf("invalid Collabora URL")
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 2 || !strings.EqualFold(req.URL.Host, base.Host) || req.URL.Scheme != base.Scheme {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	return &Discovery{base: base, client: client}, nil
}
func (discovery *Discovery) Action(ctx context.Context, extension, mode string) (string, error) {
	discovery.mu.Lock()
	defer discovery.mu.Unlock()
	if time.Now().After(discovery.expires) {
		if err := discovery.refresh(ctx); err != nil {
			return "", err
		}
	}
	value, ok := discovery.actions[actionKey{strings.TrimPrefix(strings.ToLower(extension), "."), mode}]
	if !ok {
		return "", fmt.Errorf("Collabora action unavailable")
	}
	return value, nil
}

func (discovery *Discovery) Healthy(ctx context.Context) error {
	discovery.mu.Lock()
	defer discovery.mu.Unlock()
	if time.Now().Before(discovery.expires) && len(discovery.actions) > 0 {
		return nil
	}
	return discovery.refresh(ctx)
}
func (discovery *Discovery) refresh(ctx context.Context) error {
	endpoint := discovery.base.ResolveReference(&url.URL{Path: "/hosting/discovery"})
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	response, err := discovery.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Collabora discovery returned %d", response.StatusCode)
	}
	actions, err := parseDiscovery(response.Body)
	if err != nil {
		return err
	}
	discovery.actions = actions
	discovery.expires = time.Now().Add(5 * time.Minute)
	return nil
}

func parseDiscovery(reader io.Reader) (map[actionKey]string, error) {
	var document discoveryDocument
	decoder := xml.NewDecoder(io.LimitReader(reader, 2<<20))
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode Collabora discovery: %w", err)
	}
	actions := map[actionKey]string{}
	for _, zone := range document.Zones {
		for _, app := range zone.Apps {
			for _, action := range app.Actions {
				if action.Extension != "" && action.URL != "" && (action.Name == "edit" || action.Name == "view") {
					parsed, err := url.Parse(action.URL)
					if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
						continue
					}
					actions[actionKey{strings.ToLower(action.Extension), action.Name}] = action.URL
				}
			}
		}
	}
	if len(actions) == 0 {
		return nil, fmt.Errorf("Collabora discovery contains no supported actions")
	}
	return actions, nil
}
