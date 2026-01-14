package kubeconfig

import (
	"encoding/json"
	"testing"
)

func TestMarshalKubeconfig(t *testing.T) {
	// Valid kubeconfig data
	validMapData := map[string]any{
		"apiVersion": "v1",
		"kind":       "Config",
		"clusters":   []any{},
	}
	// We marshal this here to establish the "expected" string
	validKubeconfigJSON, _ := json.Marshal(validMapData)

	// Data that triggers a JSON Marshal error
	unmarshalableMap := map[string]any{
		"a": make(chan int),
	}

	tests := []struct {
		name           string
		kubeconfigData map[string]any
		wantResult     string
		wantErr        bool
	}{
		{
			name:           "Successful marshaling",
			kubeconfigData: validMapData,
			wantResult:     string(validKubeconfigJSON),
			wantErr:        false,
		},
		{
			name:           "Nil kubeconfig data",
			kubeconfigData: nil,
			wantResult:     "", // Expect empty string on error
			wantErr:        true,
		},
		{
			name:           "Empty kubeconfig data",
			kubeconfigData: map[string]any{},
			wantResult:     "", // Expect empty string on error
			wantErr:        true,
		},
		{
			name:           "JSON marshal error",
			kubeconfigData: unmarshalableMap,
			wantResult:     "", // Expect empty string on error
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := marshalKubeconfig(tt.kubeconfigData)
			if (err != nil) != tt.wantErr {
				t.Errorf("marshalKubeconfig() error = %v, wantErr %v", err, tt.wantErr)
				return // Stop if error status is wrong
			}
			if gotResult != tt.wantResult {
				t.Errorf("marshalKubeconfig() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
