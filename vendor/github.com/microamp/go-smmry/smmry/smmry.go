package smmry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	smmryBaseURL   = "http://api.smmry.com/"
	smmryAPIKeyEnv = "SMMRY_API_KEY"
)

// SmmryClient ...
type SmmryClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// SmmryResult ...
type SmmryResult struct {
	SmAPICharacterCount string `json:"sm_api_character_count"`
	SmAPITitle          string `json:"sm_api_title"`
	SmAPIContent        string `json:"sm_api_content"`
	SmAPILimitation     string `json:"sm_api_limitation"`
}

// NewSmmryClient ...
func NewSmmryClient() (*SmmryClient, error) {
	apiKey := os.Getenv(smmryAPIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("Invalid env value for %s", smmryAPIKeyEnv)
	}
	return &SmmryClient{
		httpClient: &http.Client{},
		baseURL:    smmryBaseURL,
		apiKey:     apiKey,
	}, nil
}

// SummaryByWebsite ...
func (client *SmmryClient) SummaryByWebsite(websiteURL, length string) (*SmmryResult, error) {
	// NOTE: URL encode?
	url := fmt.Sprintf(
		"%s&SM_API_KEY=%s&SM_LENGTH=%s&SM_URL=%s",
		client.baseURL,
		client.apiKey,
		length,
		websiteURL,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// TODO: Set headers?

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Printf("Error closing body: %+v", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result SmmryResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// SummaryByText ...
func (client *SmmryClient) SummaryByText(text, length string) (*SmmryResult, error) {
	// NOTE: URL encode?
	url := fmt.Sprintf(
		"%s&SM_API_KEY=%s&SM_LENGTH=%s", // No [BREAK] in the summarised text
		client.baseURL,
		client.apiKey,
		length,
	)

	reqBody := []byte("sm_api_input=" + text)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Expect", "100-continue")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Printf("Error closing body: %+v", err)
		}
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result SmmryResult
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
