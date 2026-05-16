package clash

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type ProxyInfo struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Alive bool   `json:"alive"`
	Delay int    `json:"delay"`
}

type ProxiesResponse struct {
	Proxies map[string]ProxyDetail `json:"proxies"`
}

type ProxyDetail struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Alive   bool           `json:"alive"`
	Now     string         `json:"now"`
	All     []string       `json:"all"`
	History []DelayHistory `json:"history"`
}

type DelayHistory struct {
	Time  string `json:"time"`
	Delay int    `json:"delay"`
}

func (c *Client) GetAllProxies() ([]ProxyInfo, error) {
	data, err := c.Get("/proxies")
	if err != nil {
		return nil, err
	}

	var resp ProxiesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	var proxies []ProxyInfo
	for name, detail := range resp.Proxies {
		if isRealNode(name, detail.Type) {
			delay := 0
			if len(detail.History) > 0 {
				delay = detail.History[len(detail.History)-1].Delay
			}
			proxies = append(proxies, ProxyInfo{
				Name:  name,
				Type:  detail.Type,
				Alive: detail.Alive,
				Delay: delay,
			})
		}
	}

	return proxies, nil
}

func (c *Client) GetCurrentProxy() (string, error) {
	data, err := c.Get("/proxies")
	if err != nil {
		return "", err
	}

	var resp ProxiesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}

	groupNames := []string{"GLOBAL", "Proxy", "Auto", "代理", "自动选择"}

	for _, group := range groupNames {
		if detail, ok := resp.Proxies[group]; ok {
			if detail.Now == "DIRECT" || detail.Now == "REJECT" {
				continue
			}
			if detail.Now != "" {
				if subDetail, ok := resp.Proxies[detail.Now]; ok {
					if subDetail.Type == "URLTest" || subDetail.Type == "Selector" {
						if subDetail.Now != "" && subDetail.Now != "DIRECT" {
							return subDetail.Now, nil
						}
					}
				}
				return detail.Now, nil
			}
		}
	}

	return "", fmt.Errorf("no active proxy found")
}

func (c *Client) SwitchProxy(name string) error {
	// First, ensure GLOBAL points to Proxy (not DIRECT)
	// GLOBAL is the main entry point for traffic
	err := c.Put("/proxies/GLOBAL", map[string]string{"name": "Proxy"})
	if err != nil {
		// Try setting GLOBAL directly to the node
		c.Put("/proxies/GLOBAL", map[string]string{"name": name})
	}

	// Then switch in Proxy group (Selector type - allows manual selection)
	err = c.Put("/proxies/Proxy", map[string]string{"name": name})
	if err == nil {
		return nil
	}

	// Fallback: try other selector-type groups
	groupNames := []string{"GLOBAL", "代理"}
	for _, group := range groupNames {
		err = c.Put("/proxies/"+group, map[string]string{"name": name})
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to switch proxy")
}

func (c *Client) TestDelay(name string) (int, error) {
	encodedName := url.PathEscape(name)
	apiURL := fmt.Sprintf("/proxies/%s/delay?timeout=10000&url=https://cp.cloudflare.com", encodedName)
	data, err := c.Get(apiURL)
	if err != nil {
		return 0, err
	}

	var resp struct {
		Delay int `json:"delay"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, err
	}

	return resp.Delay, nil
}

func isRealNode(name, typ string) bool {
	skipTypes := []string{"Selector", "URLTest", "Fallback", "Direct", "Reject", "Pass", "Compatible"}
	for _, t := range skipTypes {
		if strings.ToLower(typ) == strings.ToLower(t) {
			return false
		}
	}
	return true
}
