package clash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const apiBase = "http://127.0.0.1:9090"
const timeout = 10 * time.Second

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL: apiBase,
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *Client) Get(path string) ([]byte, error) {
	url := c.baseURL + path
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) Put(path string, body interface{}) error {
	url := c.baseURL + path
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) IsConnected() bool {
	_, err := c.Get("/version")
	return err == nil
}