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

package ent

import (
	"os"

	"github.com/weaviate/weaviate/entities/moduletools"
	basesettings "github.com/weaviate/weaviate/usecases/modulecomponents/settings"
)

const (
	DefaultModel   = "Qwen/Qwen3-Embedding-8B"
	DefaultBaseURL = "http://localhost:8000"
	LowerCaseInput = false
)

type classSettings struct {
	basesettings.BaseClassSettings
	cfg moduletools.ClassConfig
}

func NewClassSettings(cfg moduletools.ClassConfig) *classSettings {
	return &classSettings{cfg: cfg, BaseClassSettings: *basesettings.NewBaseClassSettings(cfg, LowerCaseInput)}
}

func (s *classSettings) Model() string {
	return s.BaseClassSettings.GetPropertyAsString("model", DefaultModel)
}

func (s *classSettings) BaseURL() string {
	return s.BaseClassSettings.GetPropertyAsString("baseURL", DefaultBaseURL)
}

func (s *classSettings) ApiKey() string {
	// First check class config
	apiKey := s.BaseClassSettings.GetPropertyAsString("apiKey", "")
	if apiKey != "" {
		return apiKey
	}
	
	// Then check environment variables
	if envKey := os.Getenv("QWEN_APIKEY"); envKey != "" {
		return envKey
	}
	
	// Also check DASHSCOPE_API_KEY for compatibility
	if envKey := os.Getenv("DASHSCOPE_API_KEY"); envKey != "" {
		return envKey
	}
	
	return ""
}

func (s *classSettings) Validate(class *basesettings.BaseClassSettings) error {
	return nil
}

type VectorizationConfig struct {
	Model   string `json:"model"`
	BaseURL string `json:"baseURL"`
	ApiKey  string `json:"apiKey"`
}

