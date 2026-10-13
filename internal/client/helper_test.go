// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package client_test

import (
	"testing"

	"github.com/derailed/k9s/internal/client"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
)

func TestMetaFQN(t *testing.T) {
	uu := map[string]struct {
		meta metav1.ObjectMeta
		e    string
	}{
		"empty": {
			e: "-/",
		},
		"full": {
			meta: metav1.ObjectMeta{Name: "blee", Namespace: "ns1"},
			e:    "ns1/blee",
		},
		"no-ns": {
			meta: metav1.ObjectMeta{Name: "blee"},
			e:    "-/blee",
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, client.MetaFQN(&u.meta))
		})
	}
}

func TestCoFQN(t *testing.T) {
	uu := map[string]struct {
		meta metav1.ObjectMeta
		co   string
		e    string
	}{
		"empty": {
			e: "-/:",
		},
		"full": {
			meta: metav1.ObjectMeta{Name: "blee", Namespace: "ns1"},
			co:   "fred",
			e:    "ns1/blee:fred",
		},
		"no-co": {
			meta: metav1.ObjectMeta{Name: "blee", Namespace: "ns1"},
			e:    "ns1/blee:",
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, client.CoFQN(&u.meta, u.co))
		})
	}
}

func TestIsClusterScoped(t *testing.T) {
	uu := map[string]struct {
		ns string
		e  bool
	}{
		"empty": {},
		"all": {
			ns: client.NamespaceAll,
		},
		"none": {
			ns: client.BlankNamespace,
		},
		"custom": {
			ns: "fred",
		},
		"scoped": {
			ns: "-",
			e:  true,
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, client.IsClusterScoped(u.ns))
		})
	}
}

func TestIsNamespaced(t *testing.T) {
	uu := map[string]struct {
		ns string
		e  bool
	}{
		"empty": {},
		"all": {
			ns: client.NamespaceAll,
		},
		"cluster": {
			ns: client.ClusterScope,
		},
		"none": {
			ns: client.BlankNamespace,
		},
		"custom": {
			ns: "fred",
			e:  true,
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, client.IsNamespaced(u.ns))
		})
	}
}

func TestIsAllNamespaces(t *testing.T) {
	uu := map[string]struct {
		ns string
		e  bool
	}{
		"empty": {
			e: true,
		},
		"all": {
			ns: client.NamespaceAll,
			e:  true,
		},
		"none": {
			ns: client.BlankNamespace,
			e:  true,
		},
		"custom": {
			ns: "fred",
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, client.IsAllNamespaces(u.ns))
		})
	}
}

func TestIsAllNamespace(t *testing.T) {
	uu := map[string]struct {
		ns string
		e  bool
	}{
		"empty": {},
		"all": {
			ns: client.NamespaceAll,
			e:  true,
		},
		"custom": {
			ns: "fred",
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, client.IsAllNamespace(u.ns))
		})
	}
}

func TestCleanseNamespace(t *testing.T) {
	uu := map[string]struct {
		ns, e string
	}{
		"empty": {},
		"all": {
			ns: client.NamespaceAll,
			e:  client.BlankNamespace,
		},
		"custom": {
			ns: "fred",
			e:  "fred",
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, client.CleanseNamespace(u.ns))
		})
	}
}

func TestNamespaced(t *testing.T) {
	uu := []struct {
		p, ns, n string
	}{
		{"fred/blee", "fred", "blee"},
		{"blee", "", "blee"},
	}

	for _, u := range uu {
		ns, n := client.Namespaced(u.p)
		assert.Equal(t, u.ns, ns)
		assert.Equal(t, u.n, n)
	}
}

func TestFQN(t *testing.T) {
	uu := []struct {
		ns, n string
		e     string
	}{
		{"fred", "blee", "fred/blee"},
		{"", "blee", "blee"},
	}

	for _, u := range uu {
		assert.Equal(t, u.e, client.FQN(u.ns, u.n))
	}
}

func TestSupportsWatchList(t *testing.T) {
	uu := map[string]struct {
		info *version.Info
		e    bool
	}{
		"nil": {
			e: true,
		},
		"garbage": {
			info: &version.Info{GitVersion: "blee"},
			e:    true,
		},
		"1.10": {
			info: &version.Info{GitVersion: "v1.10.11"},
		},
		"1.26": {
			info: &version.Info{GitVersion: "v1.26.0"},
		},
		"1.27": {
			info: &version.Info{GitVersion: "v1.27.0"},
			e:    true,
		},
		"1.27-gke": {
			info: &version.Info{GitVersion: "v1.27.3-gke.100"},
			e:    true,
		},
		"1.37": {
			info: &version.Info{GitVersion: "v1.37.1"},
			e:    true,
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, client.SupportsWatchList(u.info))
		})
	}
}
