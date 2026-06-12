package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ddcrypto "github.com/knarayanareddy/DEADDROP-OS/client/crypto"
)

const powChallengeHeader = "X-Deaddrop-Pow-Challenge"
const powHeader = "X-Deaddrop-Pow"

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) Submit(hint, payload string) (string, error) {
	body, _ := json.Marshal(map[string]string{"hint": hint, "payload": payload})
	resp, err := c.doWithPow(http.MethodPost, c.BaseURL+"/drop", "application/json", body, "post")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("submit failed: %s", resp.Status)
	}
	var out struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b, &out)
	return out.ID, nil
}
func (c *Client) Drops(hint string) ([]struct{ ID, Payload string }, error) {
	resp, err := c.doWithPow(http.MethodGet, c.BaseURL+"/drops?hint="+url.QueryEscape(hint), "", nil, "get")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("retrieve failed: %s", resp.Status)
	}
	var out struct {
		Drops []struct{ ID, Payload string } `json:"drops"`
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out.Drops, err
}

func (c *Client) doWithPow(method, u, contentType string, body []byte, endpoint string) (*http.Response, error) {
	mkReq := func() (*http.Request, error) {
		var r io.Reader
		if body != nil {
			r = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, u, r)
		if err != nil {
			return nil, err
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		return req, nil
	}
	req, err := mkReq()
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil || resp.StatusCode != http.StatusTooManyRequests {
		return resp, err
	}
	challenge := resp.Header.Get(powChallengeHeader)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if challenge == "" {
		return resp, nil
	}
	token, ok := solvePow(endpoint, challenge)
	if !ok {
		return resp, nil
	}
	req, err = mkReq()
	if err != nil {
		return nil, err
	}
	req.Header.Set(powHeader, token)
	return c.HTTP.Do(req)
}

func EncryptMessage(publicPEM []byte, message string) (string, error) {
	pub, err := ddcrypto.ParsePublicKeyPEM(publicPEM)
	if err != nil {
		return "", err
	}
	return ddcrypto.Encrypt(pub, []byte(message))
}
