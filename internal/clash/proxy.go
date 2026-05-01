package clash

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ProxyInfo struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Alive  bool   `json:"alive"`
	Delay  int    `json:"delay"`
}

type ProxiesResponse struct {
	Proxies map[string]ProxyDetail `json:"proxies"`
}

type ProxyDetail struct {
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	Alive   bool          `json:"alive"`
	Now     string        `json:"now"`
	All     []string      `json:"all"`
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
	// Try multiple proxy group names (different subscriptions use different names)
	groupNames := []string{"Auto", "自动选择", "GLOBAL", "Proxy", "代理"}

	for _, name := range groupNames {
		data, err := c.Get("/proxies/" + name)
		if err != nil {
			continue
		}

		var detail ProxyDetail
		if err := json.Unmarshal(data, &detail); err != nil {
			continue
		}

		if detail.Now != "" {
			// If it's another group, drill down to find actual node
			for _, subName := range groupNames {
				if detail.Now == subName {
					// Recursively get the actual node
					subData, subErr := c.Get("/proxies/" + detail.Now)
					if subErr == nil {
						var subDetail ProxyDetail
						if json.Unmarshal(subData, &subDetail) == nil && subDetail.Now != "" {
							return subDetail.Now, nil
						}
					}
				}
			}
			return detail.Now, nil
		}
	}

	return "", fmt.Errorf("no active proxy found")
}

func (c *Client) SwitchProxy(name string) error {
	// Try multiple proxy group names
	groupNames := []string{"Auto", "自动选择", "GLOBAL", "Proxy", "代理"}

	for _, group := range groupNames {
		err := c.Put("/proxies/"+group, map[string]string{"name": name})
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to switch proxy in any group")
}

func (c *Client) TestDelay(name string) (int, error) {
	url := fmt.Sprintf("/proxies/%s/delay?timeout=5000&url=http://www.gstatic.com/generate_204", name)
	data, err := c.Get(url)
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
	if strings.Contains(name, "流量") || strings.Contains(name, "到期") ||
		strings.Contains(name, "重置") || strings.Contains(name, "建议") {
		return false
	}
	skipTypes := []string{"Selector", "URLTest", "Fallback", "Direct", "Reject", "Pass", "Compatible"}
	for _, t := range skipTypes {
		if strings.ToLower(typ) == strings.ToLower(t) {
			return false
		}
	}
	return true
}