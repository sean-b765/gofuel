package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awstypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	nswTasTokenURL = "https://api.onegov.nsw.gov.au/oauth/client_credential/accesstoken?grant_type=client_credentials"
	nswTasProvider = "nsw_tas"
)

type NswTasToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

type authItem struct {
	Provider    string `dynamodbav:"provider"`
	AccessToken string `dynamodbav:"access_token"`
	ExpiresIn   string `dynamodbav:"expires_in"`
	IssuedAt    string `dynamodbav:"issued_at"`
	TTL         int64  `dynamodbav:"ttl"`
}

func GetNswTasToken() (string, error) {
	tableName := os.Getenv("DDB_TABLE_AUTH")
	if tableName == "" {
		return "", fmt.Errorf("DDB_TABLE_AUTH not set")
	}

	cached, err := readCachedToken(tableName)
	if err == nil && !tokenExpired(cached) {
		return cached.AccessToken, nil
	}

	apiKey := os.Getenv("NSW_TAS_API_KEY")
	apiSecret := os.Getenv("NSW_TAS_API_SECRET")

	token, err := fetchNewToken(apiKey, apiSecret)
	if err != nil {
		return "", err
	}

	if err := writeCachedToken(tableName, token); err != nil {
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
	if err := json.Unmarshal(byteValue, &token); err != nil {
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

func readCachedToken(tableName string) (*NswTasToken, error) {
	out, err := client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]awstypes.AttributeValue{
			"provider": &awstypes.AttributeValueMemberS{Value: nswTasProvider},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Item) == 0 {
		return nil, fmt.Errorf("no cached token")
	}

	var item authItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("unmarshal auth item: %w", err)
	}

	return &NswTasToken{
		AccessToken: item.AccessToken,
		ExpiresIn:   item.ExpiresIn,
		IssuedAt:    item.IssuedAt,
	}, nil
}

func writeCachedToken(tableName string, token *NswTasToken) error {
	issuedAtMs, err := strconv.ParseInt(token.IssuedAt, 10, 64)
	if err != nil {
		return fmt.Errorf("parse issued_at: %w", err)
	}
	expiresInSec, err := strconv.ParseInt(token.ExpiresIn, 10, 64)
	if err != nil {
		return fmt.Errorf("parse expires_in: %w", err)
	}

	item := authItem{
		Provider:    nswTasProvider,
		AccessToken: token.AccessToken,
		ExpiresIn:   token.ExpiresIn,
		IssuedAt:    token.IssuedAt,
		TTL:         (issuedAtMs / 1000) + expiresInSec,
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("marshal auth item: %w", err)
	}

	_, err = client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	})
	return err
}
