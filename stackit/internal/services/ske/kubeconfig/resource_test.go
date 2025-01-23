package ske

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *ske.Kubeconfig
		expected    Model
		isValid     bool
	}{
		{
			"simple_values",
			&ske.Kubeconfig{
				ExpirationTimestamp: utils.Ptr(time.Date(2024, 2, 7, 16, 42, 12, 0, time.UTC)),
				Kubeconfig:          utils.Ptr("kubeconfig"),
			},
			Model{
				ClusterName:  types.StringValue("name"),
				ProjectId:    types.StringValue("pid"),
				Kubeconfig:   types.StringValue("kubeconfig"),
				Expiration:   types.Int64Null(),
				Refresh:      types.BoolNull(),
				ExpiresAt:    types.StringValue("2024-02-07T16:42:12Z"),
				CreationTime: types.StringValue("2024-02-05T14:40:12Z"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"empty_kubeconfig",
			&ske.Kubeconfig{},
			Model{},
			false,
		},
		{
			"no_kubeconfig_field",
			&ske.Kubeconfig{
				ExpirationTimestamp: utils.Ptr(time.Date(2024, 2, 7, 16, 42, 12, 0, time.UTC)),
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:   tt.expected.ProjectId,
				ClusterName: tt.expected.ClusterName,
			}
			creationTime, _ := time.Parse(time.RFC3339, tt.expected.CreationTime.ValueString())
			err := mapFields(tt.input, state, creationTime)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected, cmpopts.IgnoreFields(Model{}, "Id")) // Id includes a random uuid
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *ske.CreateKubeconfigPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&ske.CreateKubeconfigPayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Expiration: types.Int64Value(3600),
			},
			&ske.CreateKubeconfigPayload{
				ExpirationSeconds: utils.Ptr("3600"),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestCheckHasExpired(t *testing.T) {
	tests := []struct {
		description   string
		inputModel    *Model
		currentTime   time.Time
		expected      bool
		expectedError bool
	}{
		{
			description: "has expired",
			inputModel: &Model{
				Refresh:   types.BoolValue(true),
				ExpiresAt: types.StringValue(time.Now().Add(-1 * time.Hour).Format(time.RFC3339)), // one hour ago
			},
			currentTime:   time.Now(),
			expected:      true,
			expectedError: false,
		},
		{
			description: "not expired",
			inputModel: &Model{
				Refresh:   types.BoolValue(true),
				ExpiresAt: types.StringValue(time.Now().Add(1 * time.Hour).Format(time.RFC3339)), // in one hour
			},
			currentTime:   time.Now(),
			expected:      false,
			expectedError: false,
		},
		{
			description: "refresh is false, expired won't be checked",
			inputModel: &Model{
				Refresh:   types.BoolValue(false),
				ExpiresAt: types.StringValue(time.Now().Add(-1 * time.Hour).Format(time.RFC3339)), // one hour ago
			},
			currentTime:   time.Now(),
			expected:      false,
			expectedError: false,
		},
		{
			description: "invalid time",
			inputModel: &Model{
				Refresh:   types.BoolValue(true),
				ExpiresAt: types.StringValue("invalid time"),
			},
			currentTime:   time.Now(),
			expected:      false,
			expectedError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := checkHasExpired(tt.inputModel, tt.currentTime)
			if (err != nil) != tt.expectedError {
				t.Errorf("checkHasExpired() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if got != tt.expected {
				t.Errorf("checkHasExpired() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCheckCredentialsRotation(t *testing.T) {
	tests := []struct {
		description   string
		inputCluster  *ske.Cluster
		inputModel    *Model
		expected      bool
		expectedError bool
	}{
		{
			description: "creation time after credentials rotation",
			inputCluster: &ske.Cluster{
				Status: &ske.ClusterStatus{
					CredentialsRotation: &ske.CredentialsRotationState{
						LastCompletionTime: utils.Ptr(time.Now().Add(-1 * time.Hour)), // one hour ago
					},
				},
			},
			inputModel: &Model{
				CreationTime: types.StringValue(time.Now().Format(time.RFC3339)),
			},
			expected:      false,
			expectedError: false,
		},
		{
			description: "creation time before credentials rotation",
			inputCluster: &ske.Cluster{
				Status: &ske.ClusterStatus{
					CredentialsRotation: &ske.CredentialsRotationState{
						LastCompletionTime: utils.Ptr(time.Now().Add(1 * time.Hour)),
					},
				},
			},
			inputModel: &Model{
				CreationTime: types.StringValue(time.Now().Format(time.RFC3339)),
			},
			expected:      true,
			expectedError: false,
		},
		{
			description: "last completion time not set",
			inputCluster: &ske.Cluster{
				Status: &ske.ClusterStatus{
					CredentialsRotation: &ske.CredentialsRotationState{
						LastCompletionTime: nil,
					},
				},
			},
			inputModel: &Model{
				CreationTime: types.StringValue(time.Now().Format(time.RFC3339)),
			},
			expected:      false,
			expectedError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := checkCredentialsRotation(tt.inputCluster, tt.inputModel)
			if (err != nil) != tt.expectedError {
				t.Errorf("checkCredentialsRotation() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if got != tt.expected {
				t.Errorf("checkCredentialsRotation() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCheckClusterRecreation(t *testing.T) {
	tests := []struct {
		description   string
		inputCluster  *ske.Cluster
		inputModel    *Model
		expected      bool
		expectedError bool
	}{
		{
			description: "cluster creation time after kubeconfig creation time",
			inputCluster: &ske.Cluster{
				Status: &ske.ClusterStatus{
					CreationTime: utils.Ptr(time.Now().Add(-1 * time.Hour)),
				},
			},
			inputModel: &Model{
				CreationTime: types.StringValue(time.Now().Format(time.RFC3339)),
			},
			expected:      false,
			expectedError: false,
		},
		{
			description: "cluster creation time before kubeconfig creation time",
			inputCluster: &ske.Cluster{
				Status: &ske.ClusterStatus{
					CreationTime: utils.Ptr(time.Now().Add(1 * time.Hour)),
				},
			},
			inputModel: &Model{
				CreationTime: types.StringValue(time.Now().Format(time.RFC3339)),
			},
			expected:      true,
			expectedError: false,
		},
		{
			description: "cluster creation time not set",
			inputCluster: &ske.Cluster{
				Status: &ske.ClusterStatus{
					CreationTime: nil,
				},
			},
			inputModel: &Model{
				CreationTime: types.StringValue(time.Now().Format(time.RFC3339)),
			},
			expected:      false,
			expectedError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := checkClusterRecreation(tt.inputCluster, tt.inputModel)
			if (err != nil) != tt.expectedError {
				t.Errorf("checkClusterRecreation() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if got != tt.expected {
				t.Errorf("checkClusterRecreation() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
