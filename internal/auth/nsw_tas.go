package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	nswTasTokenURL = "https://api.onegov.nsw.gov.au/oauth/client_credential/accesstoken?grant_type=client_credentials"
	cacheFilePath  = "./auth.json"
	providerID     = "nsw_tas"
)

type NswTasToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

type authCache struct {
	Providers map[string]providerEntry `json:"providers"`
}

type providerEntry struct {
	Body NswTasToken `json:"body"`
}

func GetNswTasToken() (string, error) {
	cached, err := readCachedToken()
	if err == nil && !tokenExpired(cached) {
		return cached.AccessToken, nil
	}

	apiKey := os.Getenv("NSW_TAS_API_KEY")
	apiSecret := os.Getenv("NSW_TAS_API_SECRET")

	token, err := fetchNewToken(apiKey, apiSecret)
	if err != nil {
		return "", err
	}

	err = writeCachedToken(token)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

func fetchNewToken(apiKey, apiSecret string) (*NswTasToken, error) {
	req, err := http.NewRequest("GET", nswTasTokenURL, strings.NewReader(""))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(apiKey+":"+apiSecret)))
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed: %d", resp.StatusCode)
	}

	byteValue, _ := io.ReadAll(resp.Body)

	var token NswTasToken
	err = json.Unmarshal(byteValue, &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func tokenExpired(token *NswTasToken) bool {
	issuedAtMs, err := strconv.ParseInt(token.IssuedAt, 10, 64)
	if err != nil {
		return true
	}

	expiresInSec, err := strconv.ParseInt(token.ExpiresIn, 10, 64)
	if err != nil {
		return true
	}

	return time.Now().Unix() >= (issuedAtMs/1000)+expiresInSec
}

func readCachedToken() (*NswTasToken, error) {
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil, err
	}

	var cache authCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	entry, ok := cache.Providers[providerID]
	if !ok {
		return nil, fmt.Errorf("no cached token")
	}

	return &entry.Body, nil
}

func writeCachedToken(token *NswTasToken) error {
	var cache authCache

	data, err := os.ReadFile(cacheFilePath)
	if err == nil {
		json.Unmarshal(data, &cache)
	}

	if cache.Providers == nil {
		cache.Providers = make(map[string]providerEntry)
	}

	cache.Providers[providerID] = providerEntry{Body: *token}

	data, err = json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFilePath, data, 0644)
}
