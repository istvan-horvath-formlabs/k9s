// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package render_test

import (
	"testing"

	"github.com/derailed/k9s/internal/model1"
	"github.com/derailed/k9s/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomResourceDefinitionRender(t *testing.T) {
	c := render.CustomResourceDefinition{}
	r := model1.NewRow(2)

	require.NoError(t, c.Render(load(t, "crd"), "", &r))
	assert.Equal(t, "-/adapters.config.istio.io", r.ID)
	assert.Equal(t, "adapters", r.Fields[0])
	assert.Equal(t, "config.istio.io", r.Fields[1])
	// LABELS column (index 6) must render the CRD labels, not be empty.
	assert.Equal(
		t,
		"addonmanager.kubernetes.io/mode=Reconcile,app=mixer,istio=mixer-adapter,k8s-app=istio,package=adapter",
		r.Fields[6],
	)
}

func TestCustomResourceDefinitionRenderV1beta1(t *testing.T) {
	c := render.CustomResourceDefinition{}
	r := model1.NewRow(2)

	require.NoError(t, c.Render(load(t, "crd_v1beta1"), "", &r))
	assert.Equal(t, "-/certificates.certmanager.k8s.io", r.ID)
	assert.Equal(t, "certificates", r.Fields[0])
	assert.Equal(t, "certmanager.k8s.io", r.Fields[1])
	assert.Equal(t, "Certificate", r.Fields[2])
	assert.Equal(t, "v1alpha1", r.Fields[3])
	assert.Empty(t, r.Fields[7])
}
