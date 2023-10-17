package secretsmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/secretsmanager"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *secretsmanager.Instance
		aclList     *secretsmanager.AclList
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&secretsmanager.Instance{},
			&secretsmanager.AclList{},
			Model{
				Id:         types.StringValue("pid,iid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringNull(),
				ACLs:       types.SetNull(types.StringType),
			},
			true,
		},
		{
			"simple_values",
			&secretsmanager.Instance{
				Name: utils.Ptr("name"),
			},
			&secretsmanager.AclList{
				Acls: &[]secretsmanager.Acl{
					{
						Cidr: utils.Ptr("cidr-1"),
						Id:   utils.Ptr("id-cidr-1"),
					},
					{
						Cidr: utils.Ptr("cidr-2"),
						Id:   utils.Ptr("id-cidr-2"),
					},
					{
						Cidr: utils.Ptr("cidr-3"),
						Id:   utils.Ptr("id-cidr-3"),
					},
				},
			},
			Model{
				Id:         types.StringValue("pid,iid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				ACLs: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("cidr-1"),
					types.StringValue("cidr-2"),
					types.StringValue("cidr-3"),
				}),
			},
			true,
		},
		{
			"nil_response",
			nil,
			&secretsmanager.AclList{},
			Model{},
			false,
		},
		{
			"nil_acli_list",
			&secretsmanager.Instance{},
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&secretsmanager.Instance{},
			&secretsmanager.AclList{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(tt.input, tt.aclList, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
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
		expected    *secretsmanager.CreateInstancePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&secretsmanager.CreateInstancePayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Name: types.StringValue("name"),
			},
			&secretsmanager.CreateInstancePayload{
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Name: types.StringValue(""),
			},
			&secretsmanager.CreateInstancePayload{
				Name: utils.Ptr(""),
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

func TestUpdateACLs(t *testing.T) {
	// This is the response used when getting all ACLs currently, across all tests
	getAllACLsResp := secretsmanager.AclList{
		Acls: &[]secretsmanager.Acl{
			{
				Cidr: utils.Ptr("acl-1"),
				Id:   utils.Ptr("id-acl-1"),
			},
			{
				Cidr: utils.Ptr("acl-2"),
				Id:   utils.Ptr("id-acl-2"),
			},
			{
				Cidr: utils.Ptr("acl-3"),
				Id:   utils.Ptr("id-acl-3"),
			},
		},
	}
	getAllACLsRespBytes, err := json.Marshal(getAllACLsResp)
	if err != nil {
		t.Fatalf("Failed to marshal get all ACLs response: %v", err)
	}

	// This is the response used whenever an API returns a failure response
	failureRespBytes := []byte("{\"message\": \"Something bad happened\"")

	tests := []struct {
		description        string
		acls               []string
		getAllACLsFails    bool
		createACLFails     bool
		deleteACLFails     bool
		isValid            bool
		expectedACLsStates map[string]bool // Keys are CIDR; value is true if CIDR should exist at the end, false if should be deleted
	}{
		{
			description: "no_changes",
			acls:        []string{"acl-3", "acl-2", "acl-1"},
			expectedACLsStates: map[string]bool{
				"acl-1": true,
				"acl-2": true,
				"acl-3": true,
			},
			isValid: true,
		},
		{
			description: "create_acl",
			acls:        []string{"acl-1", "acl-2", "acl-3", "acl-4"},
			expectedACLsStates: map[string]bool{
				"acl-1": true,
				"acl-2": true,
				"acl-3": true,
				"acl-4": true,
			},
			isValid: true,
		},
		{
			description: "delete_acl",
			acls:        []string{"acl-1", "acl-3"},
			expectedACLsStates: map[string]bool{
				"acl-1": true,
				"acl-2": false,
				"acl-3": true,
			},
			isValid: true,
		},
		{
			description: "multiple_changes",
			acls:        []string{"acl-4", "acl-3", "acl-1", "acl-5"},
			expectedACLsStates: map[string]bool{
				"acl-1": true,
				"acl-2": false,
				"acl-3": true,
				"acl-4": true,
				"acl-5": true,
			},
			isValid: true,
		},
		{
			description: "multiple_changes_2",
			acls:        []string{"acl-4", "acl-5"},
			expectedACLsStates: map[string]bool{
				"acl-1": false,
				"acl-2": false,
				"acl-3": false,
				"acl-4": true,
				"acl-5": true,
			},
			isValid: true,
		},
		{
			description: "multiple_changes_3",
			acls:        []string{},
			expectedACLsStates: map[string]bool{
				"acl-1": false,
				"acl-2": false,
				"acl-3": false,
			},
			isValid: true,
		},
		{
			description:     "get_fails",
			acls:            []string{"acl-1", "acl-2", "acl-3"},
			getAllACLsFails: true,
			isValid:         false,
		},
		{
			description:    "create_fails_1",
			acls:           []string{"acl-1", "acl-2", "acl-3", "acl-4"},
			createACLFails: true,
			isValid:        false,
		},
		{
			description:    "create_fails_2",
			acls:           []string{"acl-1", "acl-2"},
			createACLFails: true,
			expectedACLsStates: map[string]bool{
				"acl-1": true,
				"acl-2": true,
				"acl-3": false,
			},
			isValid: true,
		},
		{
			description:    "delete_fails_1",
			acls:           []string{"acl-1", "acl-2"},
			deleteACLFails: true,
			isValid:        false,
		},
		{
			description:    "delete_fails_2",
			acls:           []string{"acl-1", "acl-2", "acl-3", "acl-4"},
			deleteACLFails: true,
			expectedACLsStates: map[string]bool{
				"acl-1": true,
				"acl-2": true,
				"acl-3": true,
				"acl-4": true,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// Will be compared to tt.expectedACLsStates at the end
			aclsStates := make(map[string]bool)
			aclsStates["acl-1"] = true
			aclsStates["acl-2"] = true
			aclsStates["acl-3"] = true

			// Handler for getting all ACLs
			getAllACLsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tt.getAllACLsFails {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write(failureRespBytes)
					if err != nil {
						t.Errorf("Get all ACLs handler: failed to write bad response: %v", err)
					}
					return
				}

				_, err := w.Write(getAllACLsRespBytes)
				if err != nil {
					t.Errorf("Get all ACLs handler: failed to write response: %v", err)
				}
			})

			// Handler for creating ACL
			createACLHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				decoder := json.NewDecoder(r.Body)
				var payload secretsmanager.CreateAclPayload
				err := decoder.Decode(&payload)
				if err != nil {
					t.Errorf("Create ACL handler: failed to parse payload")
					return
				}
				if payload.Cidr == nil {
					t.Errorf("Create ACL handler: nil CIDR")
					return
				}
				cidr := *payload.Cidr
				if cidrExists, cidrWasCreated := aclsStates[cidr]; cidrWasCreated && cidrExists {
					t.Errorf("Create ACL handler: attempted to create CIDR '%v' that already exists", *payload.Cidr)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				if tt.createACLFails {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write(failureRespBytes)
					if err != nil {
						t.Errorf("Create ACL handler: failed to write bad response: %v", err)
					}
					return
				}

				resp := secretsmanager.Acl{
					Cidr: utils.Ptr(cidr),
					Id:   utils.Ptr(fmt.Sprintf("id-%s", cidr)),
				}
				respBytes, err := json.Marshal(resp)
				if err != nil {
					t.Errorf("Create ACL handler: failed to marshal response: %v", err)
					return
				}
				_, err = w.Write(respBytes)
				if err != nil {
					t.Errorf("Create ACL handler: failed to write response: %v", err)
				}
				aclsStates[cidr] = true
			})

			// Handler for deleting ACL
			deleteACLHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				aclId, ok := vars["aclId"]
				if !ok {
					t.Errorf("Delete ACL handler: no ACL ID")
					return
				}
				if aclId[:3] != "id-" {
					t.Errorf("Delete ACL handler: got unexpected ACL ID '%v'", aclId)
					return
				}
				cidr := aclId[3:]
				cidrExists, cidrWasCreated := aclsStates[cidr]
				if !cidrWasCreated {
					t.Errorf("Delete ACL handler: attempted to delete CIDR '%v' that wasn't created", cidr)
					return
				}
				if cidrWasCreated && !cidrExists {
					t.Errorf("Delete ACL handler: attempted to delete CIDR '%v' that was already deleted", cidr)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				if tt.deleteACLFails {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write(failureRespBytes)
					if err != nil {
						t.Errorf("Delete ACL handler: failed to write bad response: %v", err)
					}
					return
				}

				_, err = w.Write([]byte("{}"))
				if err != nil {
					t.Errorf("Delete ACL handler: failed to write response: %v", err)
				}
				aclsStates[cidr] = false
			})

			// Setup server and client
			router := mux.NewRouter()
			router.HandleFunc("/v1/projects/{projectId}/instances/{instanceId}/acls", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					getAllACLsHandler(w, r)
				} else if r.Method == "POST" {
					createACLHandler(w, r)
				}
			})
			router.HandleFunc("/v1/projects/{projectId}/instances/{instanceId}/acls/{aclId}", deleteACLHandler)
			mockedServer := httptest.NewServer(router)
			defer mockedServer.Close()
			client, err := secretsmanager.NewAPIClient(
				config.WithEndpoint(mockedServer.URL),
				config.WithoutAuthentication(),
				config.WithRetryTimeout(time.Millisecond),
			)
			if err != nil {
				t.Fatalf("Failed to initialize client: %v", err)
			}

			// Run test
			err = updateACLs(context.Background(), "pid", "iid", tt.acls, client)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(aclsStates, tt.expectedACLsStates)
				if diff != "" {
					t.Fatalf("ACL states do not match: %s", diff)
				}
			}
		})
	}
}
