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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaviate/weaviate/modules/text2vec-qwen/ent"
	"github.com/weaviate/weaviate/usecases/modulecomponents"
)

func TestClient(t *testing.T) {
	t.Run("when all is fine", func(t *testing.T) {
		logger, _ := test.NewNullLogger()
		server := httptest.NewServer(&fakeHandler{
			t: t,
			res: embeddingsResponse{
				Object: "list",
				Data: []embedding{
					{
						Object:    "embedding",
						Embedding: []float32{0.1, 0.2, 0.3},
						Index:     0,
					},
				},
				Model: "Qwen/Qwen3-Embedding-8B",
				Usage: usage{
					PromptTokens: 5,
					TotalTokens:  5,
				},
			},
		})
		defer server.Close()

		c := New(time.Second*30, logger)
		config := ent.VectorizationConfig{
			Model:   "Qwen/Qwen3-Embedding-8B",
			BaseURL: server.URL,
		}

		expected := &modulecomponents.VectorizationResult[[]float32]{
			Text:       []string{"This is my text"},
			Dimensions: 3,
			Vector:     [][]float32{{0.1, 0.2, 0.3}},
		}
		res, tokens, err := c.vectorize(context.Background(), []string{"This is my text"}, config)

		assert.Nil(t, err)
		assert.Equal(t, expected, res)
		assert.Equal(t, 5, tokens)
	})

	t.Run("when the context is expired", func(t *testing.T) {
		logger, _ := test.NewNullLogger()
		server := httptest.NewServer(&fakeHandler{
			t: t,
			res: embeddingsResponse{
				Data: []embedding{
					{
						Embedding: []float32{0.1, 0.2, 0.3},
					},
				},
			},
		})
		defer server.Close()

		c := New(time.Second*30, logger)
		config := ent.VectorizationConfig{
			Model:   "Qwen/Qwen3-Embedding-8B",
			BaseURL: server.URL,
		}

		ctx, cancel := context.WithDeadline(context.Background(), time.Now())
		defer cancel()

		_, _, err := c.vectorize(ctx, []string{"This is my text"}, config)

		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}

type fakeHandler struct {
	t   *testing.T
	res embeddingsResponse
}

func (f *fakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	assert.Equal(f.t, http.MethodPost, r.Method)
	assert.Equal(f.t, "/v1/embeddings", r.URL.String())
	assert.Equal(f.t, "application/json", r.Header.Get("Content-Type"))

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(f.res); err != nil {
		f.t.Errorf("failed to encode response: %s", err.Error())
	}
}