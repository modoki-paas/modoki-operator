package k8sclientutil

import (
	"encoding/json"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type refreshPatchType struct{}

// RefreshPatch is a patch to trigger reconciliation
var RefreshPatch client.Patch = &refreshPatchType{}

// Type is the PatchType of the patch.
func (*refreshPatchType) Type() types.PatchType {
	return types.MergePatchType
}

// Data is the raw data representing the patch.
func (*refreshPatchType) Data(obj runtime.Object) ([]byte, error) {
	m := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				"modoki.tsuzu.dev/refreshed-at": time.Now().Format(time.RFC3339),
			},
		},
	}

	return json.Marshal(m)
}
