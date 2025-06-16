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

package modqwen

import (
	"context"

	"github.com/weaviate/weaviate/entities/models"
	"github.com/weaviate/weaviate/entities/moduletools"
	"github.com/weaviate/weaviate/entities/schema"
	"github.com/weaviate/weaviate/modules/text2vec-qwen/ent"
	basesettings "github.com/weaviate/weaviate/usecases/modulecomponents/settings"
)

func (m *QwenModule) ClassConfigDefaults() map[string]interface{} {
	return map[string]interface{}{
		"model":   ent.DefaultModel,
		"baseURL": ent.DefaultBaseURL,
	}
}

func (m *QwenModule) PropertyConfigDefaults(
	dt *schema.DataType,
) map[string]interface{} {
	return map[string]interface{}{}
}

func (m *QwenModule) ValidateClass(ctx context.Context,
	class *models.Class, cfg moduletools.ClassConfig,
) error {
	settings := ent.NewClassSettings(cfg)
	return settings.Validate(basesettings.NewBaseClassSettings(cfg, false))
}