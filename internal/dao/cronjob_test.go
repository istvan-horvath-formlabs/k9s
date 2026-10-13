// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package dao

import (
	"context"
	"testing"

	"github.com/derailed/k9s/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

func TestCronJobToggleSuspend(t *testing.T) {
	uu := map[string]struct {
		gvr     *client.GVR
		suspend any
		e       bool
	}{
		"v1-unset": {
			gvr: client.CjGVR,
			e:   true,
		},
		"v1-suspended": {
			gvr:     client.CjGVR,
			suspend: true,
			e:       false,
		},
		"v1beta1-active": {
			gvr:     client.NewGVR("batch/v1beta1/cronjobs"),
			suspend: false,
			e:       true,
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			spec := map[string]any{"schedule": "* * * * *"}
			if u.suspend != nil {
				spec["suspend"] = u.suspend
			}
			o := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": u.gvr.GV().String(),
				"kind":       "CronJob",
				"metadata":   map[string]any{"name": "cj1", "namespace": "ns1"},
				"spec":       spec,
			}}
			dial := fake.NewSimpleDynamicClientWithCustomListKinds(
				runtime.NewScheme(),
				map[schema.GroupVersionResource]string{u.gvr.GVR(): "CronJobList"},
				o,
			)

			var cj CronJob
			cj.Init(&cjFactory{conn: &cjConn{dial: dial}}, u.gvr)
			require.NoError(t, cj.ToggleSuspend(context.Background(), "ns1/cj1"))

			res, err := dial.Resource(u.gvr.GVR()).Namespace("ns1").Get(context.Background(), "cj1", metav1.GetOptions{})
			require.NoError(t, err)
			suspended, ok, err := unstructured.NestedBool(res.Object, "spec", "suspend")
			require.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, u.e, suspended)
		})
	}
}

// Helpers...

type cjFactory struct {
	pfFactory
	conn client.Connection
}

func (f *cjFactory) Client() client.Connection { return f.conn }

type cjConn struct {
	pfConn
	dial dynamic.Interface
}

func (c *cjConn) DynDial() (dynamic.Interface, error) { return c.dial, nil }

func (*cjConn) CanI(string, *client.GVR, string, []string) (bool, error) { return true, nil }
