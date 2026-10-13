// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package dao

import (
	"testing"

	"github.com/derailed/k9s/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTableCodec(t *testing.T) {
	uu := map[string]struct {
		raw string
	}{
		"v1": {
			raw: `{"kind":"Table","apiVersion":"meta.k8s.io/v1","columnDefinitions":[{"name":"Name","type":"string"}],"rows":[{"cells":["fred"]}]}`,
		},
		"v1beta1": {
			raw: `{"kind":"Table","apiVersion":"meta.k8s.io/v1beta1","columnDefinitions":[{"name":"Name","type":"string"}],"rows":[{"cells":["fred"]}]}`,
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			var ta Table
			ta.gvr = client.NewGVR("apps/v1/statefulsets")
			f, _ := ta.codec()
			o, _, err := f.UniversalDeserializer().Decode([]byte(u.raw), nil, nil)
			require.NoError(t, err)

			tt, ok := o.(*metav1.Table)
			require.True(t, ok)
			assert.Equal(t, "Name", tt.ColumnDefinitions[0].Name)
			assert.Equal(t, []any{"fred"}, tt.Rows[0].Cells)
		})
	}
}
