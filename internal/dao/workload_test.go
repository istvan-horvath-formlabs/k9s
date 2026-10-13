// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package dao

import (
	"testing"

	"github.com/derailed/k9s/internal/client"
	"github.com/derailed/k9s/internal/render"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeCols(nn ...string) []metav1.TableColumnDefinition {
	cc := make([]metav1.TableColumnDefinition, 0, len(nn))
	for _, n := range nn {
		cc = append(cc, metav1.TableColumnDefinition{Name: n})
	}

	return cc
}

func TestWorkloadStatus(t *testing.T) {
	uu := map[string]struct {
		gvr       *client.GVR
		cols      []metav1.TableColumnDefinition
		cells     []any
		status    string
		readiness string
	}{
		"pod-ok": {
			gvr:       client.PodGVR,
			cols:      makeCols("Name", "Ready", "Status"),
			cells:     []any{"p1", "1/1", render.PhaseRunning},
			status:    StatusOK,
			readiness: "1/1",
		},
		"pod-not-ready": {
			gvr:       client.PodGVR,
			cols:      makeCols("Name", "Ready", "Status"),
			cells:     []any{"p1", "0/1", render.PhaseRunning},
			status:    DegradedStatus,
			readiness: "0/1",
		},
		"pod-no-ready-col": {
			gvr:       client.PodGVR,
			cols:      makeCols("Name", "Status"),
			cells:     []any{"p1", render.PhaseRunning},
			status:    DegradedStatus,
			readiness: render.NAValue,
		},
		"dp-ok": {
			gvr:       client.DpGVR,
			cols:      makeCols("Name", "Ready", "Up-to-date", "Available"),
			cells:     []any{"d1", "2/2", int64(2), int64(2)},
			status:    StatusOK,
			readiness: "2/2",
		},
		"dp-legacy": {
			gvr:       client.DpGVR,
			cols:      makeCols("Name", "Desired", "Current", "Up-to-date", "Available"),
			cells:     []any{"d1", int64(3), int64(3), int64(3), int64(1)},
			status:    DegradedStatus,
			readiness: "1/3",
		},
		"sts-legacy": {
			gvr:       client.StsGVR,
			cols:      makeCols("Name", "Desired", "Current"),
			cells:     []any{"s1", int64(2), int64(2)},
			status:    StatusOK,
			readiness: "2/2",
		},
		"rs-int": {
			gvr:       client.RsGVR,
			cols:      makeCols("Name", "Desired", "Current", "Ready"),
			cells:     []any{"r1", int64(2), int64(2), int64(1)},
			status:    DegradedStatus,
			readiness: "1/2",
		},
		"ds-string": {
			gvr:       client.DsGVR,
			cols:      makeCols("Name", "Desired", "Current", "Ready"),
			cells:     []any{"ds1", "2", "2", "2"},
			status:    StatusOK,
			readiness: "2/2",
		},
		"ds-missing-cols": {
			gvr:       client.DsGVR,
			cols:      makeCols("Name"),
			cells:     []any{"ds1"},
			status:    StatusOK,
			readiness: render.NAValue,
		},
		"svc": {
			gvr:    client.SvcGVR,
			cols:   makeCols("Name"),
			cells:  []any{"svc1"},
			status: StatusOK,
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			r := metav1.TableRow{Cells: u.cells}
			assert.Equal(t, u.status, status(u.gvr, &r, u.cols))
			assert.Equal(t, u.readiness, readiness(u.gvr, &r, u.cols))
			assert.Equal(t, u.cells[0], cellOrBlank(&r, "Name", u.cols))
		})
	}
}
