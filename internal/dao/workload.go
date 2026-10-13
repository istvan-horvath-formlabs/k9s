// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/derailed/k9s/internal"
	"github.com/derailed/k9s/internal/client"
	"github.com/derailed/k9s/internal/render"
	"github.com/derailed/k9s/internal/slogs"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	StatusOK       = "OK"
	DegradedStatus = "DEGRADED"
)

var resList = []*client.GVR{
	client.PodGVR,
	client.SvcGVR,
	client.DsGVR,
	client.StsGVR,
	client.DpGVR,
	client.RsGVR,
}

// Workload tracks a select set of resources in a given namespace.
type Workload struct {
	Table
}

func (w *Workload) Delete(ctx context.Context, path string, propagation *metav1.DeletionPropagation, grace Grace) error {
	gvr, _ := ctx.Value(internal.KeyGVR).(*client.GVR)
	ns, n := client.Namespaced(path)
	auth, err := w.Client().CanI(ns, gvr, n, []string{client.DeleteVerb})
	if err != nil {
		return err
	}
	if !auth {
		return fmt.Errorf("user is not authorized to delete %s", path)
	}

	var gracePeriod *int64
	if grace != DefaultGrace {
		gracePeriod = (*int64)(&grace)
	}
	opts := metav1.DeleteOptions{
		PropagationPolicy:  propagation,
		GracePeriodSeconds: gracePeriod,
	}

	ctx, cancel := context.WithTimeout(ctx, w.Client().Config().CallTimeout())
	defer cancel()

	d, err := w.Client().DynDial()
	if err != nil {
		return err
	}
	dial := d.Resource(gvr.GVR())
	if client.IsClusterScoped(ns) {
		return dial.Delete(ctx, n, opts)
	}

	return dial.Namespace(ns).Delete(ctx, n, opts)
}

func (a *Workload) fetch(ctx context.Context, gvr *client.GVR, ns string) (*metav1.Table, error) {
	a.gvr = gvr
	oo, err := a.Table.List(ctx, ns)
	if err != nil {
		return nil, err
	}
	if len(oo) == 0 {
		return nil, fmt.Errorf("no table found for gvr: %s", gvr)
	}
	tt, ok := oo[0].(*metav1.Table)
	if !ok {
		return nil, errors.New("not a metav1.Table")
	}

	return tt, nil
}

// List fetch workloads.
func (a *Workload) List(ctx context.Context, ns string) ([]runtime.Object, error) {
	oo := make([]runtime.Object, 0, 100)
	for _, gvr := range resList {
		table, err := a.fetch(ctx, gvr, ns)
		if err != nil {
			return nil, err
		}
		var (
			ns string
			ts metav1.Time
		)
		for _, r := range table.Rows {
			if obj := r.Object.Object; obj != nil {
				if m, err := meta.Accessor(obj); err == nil {
					ns, ts = m.GetNamespace(), m.GetCreationTimestamp()
				}
			} else {
				var m metav1.PartialObjectMetadata
				if err := json.Unmarshal(r.Object.Raw, &m); err == nil {
					ns, ts = m.GetNamespace(), m.CreationTimestamp
				}
			}
			stat := status(gvr, &r, table.ColumnDefinitions)
			oo = append(oo, &render.WorkloadRes{Row: metav1.TableRow{Cells: []any{
				gvr.String(),
				ns,
				cellOrBlank(&r, "Name", table.ColumnDefinitions),
				stat,
				readiness(gvr, &r, table.ColumnDefinitions),
				validity(stat),
				ts,
			}}})
		}
	}

	return oo, nil
}

// Helpers...

func readiness(gvr *client.GVR, r *metav1.TableRow, h []metav1.TableColumnDefinition) string {
	switch gvr {
	case client.PodGVR, client.DpGVR, client.StsGVR, client.RsGVR, client.DsGVR:
		if ready, ok := readyCell(gvr, r, h); ok {
			return ready
		}
	case client.SvcGVR:
		return ""
	}

	return render.NAValue
}

func status(gvr *client.GVR, r *metav1.TableRow, h []metav1.TableColumnDefinition) string {
	switch gvr {
	case client.PodGVR:
		status, _ := cell(r, "Status", h)
		if status == render.PhaseCompleted {
			return StatusOK
		}
		if ready, ok := readyCell(gvr, r, h); !ok || !isReady(ready) || status != render.PhaseRunning {
			return DegradedStatus
		}
	case client.DpGVR, client.StsGVR, client.RsGVR, client.DsGVR:
		if ready, ok := readyCell(gvr, r, h); ok && !isReady(ready) {
			return DegradedStatus
		}
	case client.SvcGVR:
	default:
		return render.MissingValue
	}

	return StatusOK
}

// readyCell returns a ready/desired ratio for a workload row.
// Older API servers do not serve a Ready column for deployments and statefulsets.
func readyCell(gvr *client.GVR, r *metav1.TableRow, h []metav1.TableColumnDefinition) (string, bool) {
	switch gvr {
	case client.RsGVR, client.DsGVR:
		return ratio(r, h, "Ready", "Desired")
	}
	if ready, ok := cell(r, "Ready", h); ok {
		s, ok := ready.(string)
		return s, ok
	}
	switch gvr {
	case client.DpGVR:
		return ratio(r, h, "Available", "Desired")
	case client.StsGVR:
		return ratio(r, h, "Current", "Desired")
	}

	return "", false
}

func ratio(r *metav1.TableRow, h []metav1.TableColumnDefinition, num, den string) (string, bool) {
	n, ok1 := cell(r, num, h)
	d, ok2 := cell(r, den, h)
	if !ok1 || !ok2 {
		return "", false
	}

	return fmt.Sprintf("%v/%v", n, d), true
}

func cell(r *metav1.TableRow, n string, h []metav1.TableColumnDefinition) (any, bool) {
	idx := indexOf(n, h)
	if idx < 0 || idx >= len(r.Cells) {
		return nil, false
	}

	return r.Cells[idx], true
}

func cellOrBlank(r *metav1.TableRow, n string, h []metav1.TableColumnDefinition) any {
	if v, ok := cell(r, n, h); ok {
		return v
	}

	return ""
}

func validity(status string) string {
	if status != "DEGRADED" {
		return ""
	}

	return status
}

func isReady(s string) bool {
	tt := strings.Split(s, "/")
	if len(tt) != 2 {
		return false
	}
	r, err := strconv.Atoi(tt[0])
	if err != nil {
		slog.Error("Invalid ready count",
			slogs.Error, err,
			slogs.Count, tt[0],
		)
		return false
	}
	c, err := strconv.Atoi(tt[1])
	if err != nil {
		slog.Error("invalid expected count: %q",
			slogs.Error, err,
			slogs.Count, tt[1],
		)
		return false
	}

	if c == 0 {
		return true
	}
	return r == c
}

func indexOf(n string, defs []metav1.TableColumnDefinition) int {
	for i, d := range defs {
		if d.Name == n {
			return i
		}
	}

	return -1
}
