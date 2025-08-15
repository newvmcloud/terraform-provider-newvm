package newvm

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// HostURL - Default NewVM URL
const HostURL string = "https://api.newvm.com"

// Client -
type Client struct {
	HostURL    string
	HTTPClient *http.Client
	Token      string
	Auth       AuthStruct
}

// AuthStruct -
type AuthStruct struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Totp     string `json:"totp"`
}

// AuthResponse -
type AuthResponse struct {
	Token string `json:"token"`
}

// NewClient -
func NewClient(host, username, password *string, totp *string) (*Client, error) {
	c := Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		// Default NewVM URL
		HostURL: HostURL,
	}

	if host != nil {
		c.HostURL = *host
	}

	// If username or password not provided, return empty client
	if username == nil || password == nil {
		return &c, nil
	}

	if totp == nil {
		totp = new(string)
	}

	c.Auth = AuthStruct{
		Username: *username,
		Password: *password,
		Totp:     *totp,
	}

	ar, err := c.Login()
	if err != nil {
		return nil, err
	}

	c.Token = ar.Token

	// make a request to verify token, this updates the roles and privileges
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/token", c.HostURL), nil)
	if err != nil {
		return nil, err
	}

	_, err2 := c.doRequest(req)
	if err2 != nil {
		return nil, err2
	}

	return &c, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	token := c.Token

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}
