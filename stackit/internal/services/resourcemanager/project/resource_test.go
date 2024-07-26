package project

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
)

func TestMapFields(t *testing.T) {
	testUUID := uuid.New().String()
	tests := []struct {
		description           string
		uuidContainerParentId bool
		projectResp           *resourcemanager.GetProjectResponse
		membersResp           *[]authorization.Member
		expected              Model
		expectedLabels        *map[string]string
		isValid               bool
	}{
		{
			"default_ok",
			false,
			&resourcemanager.GetProjectResponse{
				ContainerId: utils.Ptr("cid"),
				ProjectId:   utils.Ptr("pid"),
			},
			&[]authorization.Member{
				{
					Subject: utils.Ptr("owner_email"),
					Role:    utils.Ptr("owner"),
				},
				{
					Subject: utils.Ptr("reader_email"),
					Role:    utils.Ptr("reader"),
				},
			},
			Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ProjectId:         types.StringValue("pid"),
				ContainerParentId: types.StringNull(),
				Name:              types.StringNull(),
				Members: types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("owner_email"),
							"role":    types.StringValue("owner"),
						},
					),
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("reader_email"),
							"role":    types.StringValue("reader"),
						},
					),
				}),
			},
			nil,
			true,
		},
		{
			"container_parent_id_ok",
			false,
			&resourcemanager.GetProjectResponse{
				ContainerId: utils.Ptr("cid"),
				ProjectId:   utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "ref1",
					"label2": "ref2",
				},
				Parent: &resourcemanager.Parent{
					ContainerId: utils.Ptr("parent_cid"),
					Id:          utils.Ptr("parent_pid"),
				},
				Name: utils.Ptr("name"),
			},
			nil,
			Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ProjectId:         types.StringValue("pid"),
				ContainerParentId: types.StringValue("parent_cid"),
				Name:              types.StringValue("name"),
				Members:           types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
			},
			&map[string]string{
				"label1": "ref1",
				"label2": "ref2",
			},
			true,
		},
		{
			"uuid_parent_id_ok",
			true,
			&resourcemanager.GetProjectResponse{
				ContainerId: utils.Ptr("cid"),
				ProjectId:   utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "ref1",
					"label2": "ref2",
				},
				Parent: &resourcemanager.Parent{
					ContainerId: utils.Ptr("parent_cid"),
					Id:          utils.Ptr(testUUID),
				},
				Name: utils.Ptr("name"),
			},
			nil,
			Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ProjectId:         types.StringValue("pid"),
				ContainerParentId: types.StringValue(testUUID),
				Name:              types.StringValue("name"),
				Members:           types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
			},
			&map[string]string{
				"label1": "ref1",
				"label2": "ref2",
			},
			true,
		},
		{
			"response_nil_fail",
			false,
			nil,
			nil,
			Model{},
			nil,
			false,
		},
		{
			"no_resource_id",
			false,
			&resourcemanager.GetProjectResponse{},
			nil,
			Model{},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if tt.expectedLabels == nil {
				tt.expected.Labels = types.MapNull(types.StringType)
			} else {
				convertedLabels, err := conversion.ToTerraformStringMap(context.Background(), *tt.expectedLabels)
				if err != nil {
					t.Fatalf("Error converting to terraform string map: %v", err)
				}
				tt.expected.Labels = convertedLabels
			}
			var containerParentId = types.StringNull()
			if tt.uuidContainerParentId {
				containerParentId = types.StringValue(testUUID)
			}
			state := &Model{
				ContainerId:       tt.expected.ContainerId,
				ContainerParentId: containerParentId,
			}

			err := mapFields(context.Background(), tt.projectResp, tt.membersResp, state)
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
		inputLabels *map[string]string
		expected    *resourcemanager.CreateProjectPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{},
			nil,
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: nil,
				Labels:            nil,
				Members:           nil,
				Name:              nil,
			},
			true,
		},
		{
			"mapping_with_conversions_single_member",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
				Members: types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("owner_email"),
							"role":    types.StringValue("owner"),
						},
					),
				}),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Members: &[]resourcemanager.Member{
					{
						Subject: utils.Ptr("owner_email"),
						Role:    utils.Ptr("owner"),
					},
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"mapping_with_conversions_ok_multiple_members",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
				Members: types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("owner_email"),
							"role":    types.StringValue("owner"),
						},
					),
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("reader_email"),
							"role":    types.StringValue("reader"),
						},
					),
				}),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Members: &[]resourcemanager.Member{
					{
						Subject: utils.Ptr("owner_email"),
						Role:    utils.Ptr("owner"),
					},
					{
						Subject: utils.Ptr("reader_email"),
						Role:    utils.Ptr("reader"),
					},
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"new members field takes precedence over deprecated owner_email field",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
				OwnerEmail:        types.StringValue("some_email_deprecated"),
				Members: types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("owner_email"),
							"role":    types.StringValue("owner"),
						},
					),
				}),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Members: &[]resourcemanager.Member{
					{
						Subject: utils.Ptr("owner_email"),
						Role:    utils.Ptr("owner"),
					},
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if tt.input != nil {
				if tt.inputLabels == nil {
					tt.input.Labels = types.MapNull(types.StringType)
				} else {
					convertedLabels, err := conversion.ToTerraformStringMap(context.Background(), *tt.inputLabels)
					if err != nil {
						t.Fatalf("Error converting to terraform string map: %v", err)
					}
					tt.input.Labels = convertedLabels
				}
			}
			output, err := toCreatePayload(context.Background(), tt.input)
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

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		inputLabels *map[string]string
		expected    *resourcemanager.PartialUpdateProjectPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{},
			nil,
			&resourcemanager.PartialUpdateProjectPayload{
				ContainerParentId: nil,
				Labels:            nil,
				Name:              nil,
			},
			true,
		},
		{
			"mapping_with_conversions_ok",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
				OwnerEmail:        types.StringValue("owner_email"),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.PartialUpdateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if tt.input != nil {
				if tt.inputLabels == nil {
					tt.input.Labels = types.MapNull(types.StringType)
				} else {
					convertedLabels, err := conversion.ToTerraformStringMap(context.Background(), *tt.inputLabels)
					if err != nil {
						t.Fatalf("Error converting to terraform string map: %v", err)
					}
					tt.input.Labels = convertedLabels
				}
			}
			output, err := toUpdatePayload(tt.input)
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

var fixtureMembers = []authorization.Member{
	{
		Subject: utils.Ptr("email_owner"),
		Role:    utils.Ptr("owner"),
	},
	{
		Subject: utils.Ptr("email_owner_2"),
		Role:    utils.Ptr("owner"),
	},
	{
		Subject: utils.Ptr("email_reader"),
		Role:    utils.Ptr("reader"),
	},
}

func TestUpdateMembers(t *testing.T) {
	// This is the response used when getting all members currently, across all tests
	getAllMembersResp := authorization.MembersResponse{
		Members: &fixtureMembers,
	}
	getAllMembersRespBytes, err := json.Marshal(getAllMembersResp)
	if err != nil {
		t.Fatalf("Failed to marshal get all Members response: %v", err)
	}

	// This is the response used whenever an API returns a failure response
	failureRespBytes := []byte("{\"message\": \"Something bad happened\"")

	tests := []struct {
		description           string
		modelMembers          []authorization.Member
		getAllMembersFails    bool
		addMembersFails       bool
		removeMembersFails    bool
		isValid               bool
		expectedMembersStates map[string]bool // Keys are member; value is true if member should exist at the end, false otherwise
	}{
		{
			description:  "no changes",
			modelMembers: fixtureMembers,
			expectedMembersStates: map[string]bool{
				memberId(fixtureMembers[0]): true,
				memberId(fixtureMembers[1]): true,
				memberId(fixtureMembers[2]): true,
			},
			isValid: true,
		},
		{
			description: "add one member",
			modelMembers: append(
				fixtureMembers,
				authorization.Member{Subject: utils.Ptr("email_reader_2"), Role: utils.Ptr("reader")},
			),
			expectedMembersStates: map[string]bool{
				memberId(fixtureMembers[0]): true,
				memberId(fixtureMembers[1]): true,
				memberId(fixtureMembers[2]): true,
				"email_reader_2,reader":     true,
			},
			isValid: true,
		},
		{
			description: "add multiple members",
			modelMembers: append(
				fixtureMembers,
				authorization.Member{Subject: utils.Ptr("email_reader_2"), Role: utils.Ptr("reader")},
				authorization.Member{Subject: utils.Ptr("email_reader_3"), Role: utils.Ptr("reader")},
			),
			expectedMembersStates: map[string]bool{
				memberId(fixtureMembers[0]): true,
				memberId(fixtureMembers[1]): true,
				memberId(fixtureMembers[2]): true,
				"email_reader_2,reader":     true,
				"email_reader_3,reader":     true,
			},
			isValid: true,
		},
		{
			description:  "removing member",
			modelMembers: fixtureMembers[:2],
			expectedMembersStates: map[string]bool{
				memberId(fixtureMembers[0]): true,
				memberId(fixtureMembers[1]): true,
				memberId(fixtureMembers[2]): false,
			},
			isValid: true,
		},
		{
			description:  "removing multiple members",
			modelMembers: fixtureMembers[:1],
			expectedMembersStates: map[string]bool{
				memberId(fixtureMembers[0]): true,
				memberId(fixtureMembers[1]): false,
				memberId(fixtureMembers[2]): false,
			},
			isValid: true,
		},
		{
			description: "multiple changes (add and remove)",
			modelMembers: append(
				fixtureMembers[:2],
				authorization.Member{Subject: utils.Ptr("email_reader_2"), Role: utils.Ptr("reader")},
				authorization.Member{Subject: utils.Ptr("email_reader_3"), Role: utils.Ptr("reader")},
			),
			expectedMembersStates: map[string]bool{
				memberId(fixtureMembers[0]): true,
				memberId(fixtureMembers[1]): true,
				memberId(fixtureMembers[2]): false,
				"email_reader_2,reader":     true,
				"email_reader_3,reader":     true,
			},
			isValid: true,
		},
		{
			description: "multiple changes 2 (add and remove)",
			modelMembers: []authorization.Member{
				{Subject: utils.Ptr("email_reader_2"), Role: utils.Ptr("reader")},
				{Subject: utils.Ptr("email_reader_3"), Role: utils.Ptr("reader")},
			},
			expectedMembersStates: map[string]bool{
				memberId(fixtureMembers[0]): false,
				memberId(fixtureMembers[1]): false,
				memberId(fixtureMembers[2]): false,
				"email_reader_2,reader":     true,
				"email_reader_3,reader":     true,
			},
			isValid: true,
		},
		{
			description:        "get fails",
			modelMembers:       fixtureMembers,
			getAllMembersFails: true,
			isValid:            false,
		},
		{
			description: "add fails",
			modelMembers: append(
				fixtureMembers,
				authorization.Member{Subject: utils.Ptr("email_reader_2"), Role: utils.Ptr("reader")},
			),
			addMembersFails: true,
			isValid:         false,
		},
		{
			description:        "remove fails",
			modelMembers:       fixtureMembers[:1],
			removeMembersFails: true,
			isValid:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// Will be compared to tt.expectedMembersStates at the end
			membersStates := map[string]bool{
				memberId(fixtureMembers[0]): true,
				memberId(fixtureMembers[1]): true,
				memberId(fixtureMembers[2]): true,
			}

			// Handler for getting all Members
			getAllMembersHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tt.getAllMembersFails {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write(failureRespBytes)
					if err != nil {
						t.Errorf("Get all Members handler: failed to write bad response: %v", err)
					}
					return
				}

				_, err := w.Write(getAllMembersRespBytes)
				if err != nil {
					t.Errorf("Get all Members handler: failed to write response: %v", err)
				}
			})

			// Handler for adding members
			addMembersHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				decoder := json.NewDecoder(r.Body)
				var payload authorization.AddMembersPayload
				err := decoder.Decode(&payload)
				if err != nil {
					t.Errorf("Add members handler: failed to parse payload")
					return
				}
				if payload.Members == nil {
					t.Errorf("Add members handler: nil members")
					return
				}
				members := *payload.Members
				for _, m := range members {
					if memberExists, memberWasAdded := membersStates[memberId(m)]; memberWasAdded && memberExists {
						t.Errorf("Add members handler: attempted to add member '%v' that already exists", memberId(m))
						return
					}
				}

				w.Header().Set("Content-Type", "application/json")
				if tt.addMembersFails {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write(failureRespBytes)
					if err != nil {
						t.Errorf("Add members handler: failed to write bad response: %v", err)
					}
					return
				}

				for _, m := range members {
					membersStates[memberId(m)] = true
				}
			})

			// Handler for removing members
			removeMembersHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				decoder := json.NewDecoder(r.Body)
				var payload authorization.RemoveMembersPayload
				err := decoder.Decode(&payload)
				if err != nil {
					t.Errorf("Remove members handler: failed to parse payload")
					return
				}
				if payload.Members == nil {
					t.Errorf("Remove members handler: nil members")
					return
				}
				members := *payload.Members
				for _, m := range members {
					memberExists, memberWasCreated := membersStates[memberId(m)]
					if !memberWasCreated {
						t.Errorf("Remove members handler: attempted to remove member '%v' that wasn't created", memberId(m))
						return
					}
					if memberWasCreated && !memberExists {
						t.Errorf("Remove members handler: attempted to remove member '%v' that was already removed", memberId(m))
						return
					}
				}

				w.Header().Set("Content-Type", "application/json")
				if tt.removeMembersFails {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write(failureRespBytes)
					if err != nil {
						t.Errorf("Remove members handler: failed to write bad response: %v", err)
					}
					return
				}

				for _, m := range members {
					membersStates[memberId(m)] = false
				}
			})

			// Setup server and client
			router := mux.NewRouter()
			router.HandleFunc("/v2/{resourceType}/{resourceId}/members", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					getAllMembersHandler(w, r)
				} else {
					t.Fatalf("Unexpected method: %v", r.Method)
				}
			})
			router.HandleFunc("/v2/{resourceId}/members", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPatch {
					addMembersHandler(w, r)
				} else {
					t.Fatalf("Unexpected method: %v", r.Method)
				}
			})
			router.HandleFunc("/v2/{resourceId}/members/remove", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					removeMembersHandler(w, r)
				} else {
					t.Fatalf("Unexpected method: %v", r.Method)
				}
			})
			mockedServer := httptest.NewServer(router)
			defer mockedServer.Close()
			client, err := authorization.NewAPIClient(
				config.WithEndpoint(mockedServer.URL),
				config.WithoutAuthentication(),
			)
			if err != nil {
				t.Fatalf("Failed to initialize client: %v", err)
			}

			// Run test
			err = updateMembers(context.Background(), "pid", &tt.modelMembers, client)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(membersStates, tt.expectedMembersStates)
				if diff != "" {
					t.Fatalf("Member states do not match: %s", diff)
				}
			}
		})
	}
}
