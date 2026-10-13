// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/watchlist"
)

func TestNoWatchList(t *testing.T) {
	dial, err := dynamic.NewForConfig(&rest.Config{Host: "https://localhost"})
	require.NoError(t, err)

	assert.False(t, watchlist.DoesClientNotSupportWatchListSemantics(dial))
	assert.True(t, watchlist.DoesClientNotSupportWatchListSemantics(noWatchList{Interface: dial}))
}
