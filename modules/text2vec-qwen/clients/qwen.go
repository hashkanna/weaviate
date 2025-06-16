//                           _       _
// __      _____  __ ___   ___  __ _| |_ ___
// \ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
//  \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
//   \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
//
//  Copyright © 2016 - 2024 Weaviate B.V. All rights reserved.
//
//  CONTACT: hello@weaviate.io
//

package clients

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/weaviate/weaviate/entities/moduletools"
	"github.com/weaviate/weaviate/modules/text2vec-qwen/ent"
	"github.com/weaviate/weaviate/usecases/modulecomponents"
)

const (
	DefaultOrigin = "http://localhost:8000" // Default for local vLLM/SGLang deployment
	DefaultModel  = "Qwen/Qwen3-Embedding-8B"
)

type embeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingsResponse struct {
	Object string      `json:"object"`
	Data   []embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  usage       `json:"usage"`
}

type embedding struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type qwenApiError struct {
	Error   string `json:"error"`
	Type    string `json:"type,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type vectorizer struct {
	httpClient *http.Client
	logger     logrus.FieldLogger
}

func New(timeout time.Duration, logger logrus.FieldLogger) *vectorizer {
	return &vectorizer{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

func (v *vectorizer) Vectorize(ctx context.Context, input []string,
	cfg moduletools.ClassConfig,
) (*modulecomponents.VectorizationResult[[]float32], *modulecomponents.RateLimits, int, error) {
	config := v.getVectorizationConfig(cfg)
	res, tokens, err := v.vectorize(ctx, input, config)
	return res, nil, tokens, err
}

func (v *vectorizer) VectorizeQuery(ctx context.Context, input []string,
	cfg moduletools.ClassConfig,
) (*modulecomponents.VectorizationResult[[]float32], error) {
	config := v.getVectorizationConfig(cfg)
	res, _, err := v.vectorize(ctx, input, config)
	return res, err
}

func (v *vectorizer) vectorize(ctx context.Context, input []string,
	config ent.VectorizationConfig,
) (*modulecomponents.VectorizationResult[[]float32], int, error) {
	body, err := json.Marshal(embeddingsRequest{
		Model: config.Model,
		Input: input,
	})
	if err != nil {
		return nil, 0, errors.Wrapf(err, "marshal body")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", v.getURL(config),
		bytes.NewReader(body))
	if err != nil {
		return nil, 0, errors.Wrap(err, "create request")
	}
	req.Header.Add("Content-Type", "application/json")

	// Handle API key authentication
	apiKey := config.ApiKey
	if apiKey == "" {
		apiKey = os.Getenv("QWEN_APIKEY")
	}
	
	if apiKey != "" {
		// Use Bearer auth for all endpoints (confirmed DashScope compatible)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}

	res, err := v.httpClient.Do(req)
	if err != nil {
		return nil, 0, errors.Wrap(err, "send request")
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, errors.Wrap(err, "read response body")
	}

	if res.StatusCode != 200 {
		return nil, 0, v.getError(res.StatusCode, bodyBytes, config)
	}

	var resBody embeddingsResponse
	if err := json.Unmarshal(bodyBytes, &resBody); err != nil {
		return nil, 0, errors.Wrap(err, "unmarshal response body")
	}

	if len(resBody.Data) == 0 {
		return nil, 0, errors.Errorf("empty embeddings response")
	}

	embeddings := make([][]float32, len(resBody.Data))
	for i := range resBody.Data {
		embeddings[i] = resBody.Data[i].Embedding
	}

	return &modulecomponents.VectorizationResult[[]float32]{
		Text:       input,
		Dimensions: len(embeddings[0]),
		Vector:     embeddings,
	}, resBody.Usage.TotalTokens, nil
}

func (v *vectorizer) getURL(config ent.VectorizationConfig) string {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultOrigin
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	
	// Handle DashScope API structure (both international and domestic)
	if strings.Contains(baseURL, "dashscope") && strings.Contains(baseURL, "aliyuncs.com") {
		return fmt.Sprintf("%s/embeddings", baseURL)
	}
	
	return fmt.Sprintf("%s/v1/embeddings", baseURL)
}

func (v *vectorizer) getVectorizationConfig(cfg moduletools.ClassConfig) ent.VectorizationConfig {
	settings := ent.NewClassSettings(cfg)
	return ent.VectorizationConfig{
		Model:   settings.Model(),
		BaseURL: settings.BaseURL(),
		ApiKey:  settings.ApiKey(),
	}
}

func (v *vectorizer) getError(statusCode int, bodyBytes []byte, config ent.VectorizationConfig) error {
	var qwenErr qwenApiError
	if err := json.Unmarshal(bodyBytes, &qwenErr); err != nil {
		return errors.Errorf("connection to Qwen API failed with status: %d error: %v", statusCode, string(bodyBytes))
	}
	return errors.Errorf("connection to Qwen API failed with status: %d error: %v", statusCode, qwenErr.Error)
}

func (v *vectorizer) MetaInfo() (map[string]interface{}, error) {
	return map[string]interface{}{
		"name":              "Qwen3 Embedding Module",
		"description":       "Vectorizer module for Qwen3-Embedding models",
		"documentationHref": "https://github.com/QwenLM/Qwen3-Embedding",
	}, nil
}

// GetApiKeyHash returns a hash of the API key for caching purposes
func (v *vectorizer) GetApiKeyHash(ctx context.Context, config moduletools.ClassConfig) [32]byte {
	settings := ent.NewClassSettings(config)
	apiKey := settings.ApiKey()
	if apiKey == "" {
		apiKey = os.Getenv("QWEN_APIKEY")
	}
	return sha256.Sum256([]byte(apiKey))
}

// GetVectorizerRateLimit returns the rate limit for the vectorizer
func (v *vectorizer) GetVectorizerRateLimit(ctx context.Context, cfg moduletools.ClassConfig) *modulecomponents.RateLimits {
	rpm, _ := strconv.Atoi(os.Getenv("QWEN_RPM"))
	if rpm == 0 {
		rpm = 100 // Default rate limit
	}

	tpm, _ := strconv.Atoi(os.Getenv("QWEN_TPM"))
	if tpm == 0 {
		tpm = 100000 // Default token limit
	}

	return &modulecomponents.RateLimits{
		LimitRequests:     rpm,
		LimitTokens:       tpm,
		RemainingRequests: rpm,
		RemainingTokens:   tpm,
		ResetRequests:     time.Now().Add(time.Minute),
		ResetTokens:       time.Now().Add(time.Minute),
	}
}

// Additional methods required for batch client interface
func (v *vectorizer) HasTokenLimit() bool {
	return true
}

func (v *vectorizer) ReturnsRateLimit() bool {
	return true
}