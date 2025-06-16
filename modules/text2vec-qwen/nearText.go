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
	"github.com/weaviate/weaviate/entities/modulecapabilities"
	"github.com/weaviate/weaviate/usecases/modulecomponents/arguments/nearText"
)

func (m *QwenModule) initNearText() error {
	m.searcher = nearText.NewSearcher(m.vectorizer)
	m.graphqlProvider = nearText.New(m.nearTextTransformer)
	return nil
}

func (m *QwenModule) Arguments() map[string]modulecapabilities.GraphQLArgument {
	return m.graphqlProvider.Arguments()
}

func (m *QwenModule) VectorSearches() map[string]modulecapabilities.VectorForParams[[]float32] {
	return m.searcher.VectorSearches()
}

