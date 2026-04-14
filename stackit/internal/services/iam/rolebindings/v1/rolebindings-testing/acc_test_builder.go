package rolebindings_testing

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testdestroy"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func NewRoleBindingAccTestBuilder(tfProviderConfig, apiName, resourceType, resourceID string) *RoleBindingAccTestBuilder {
	return &RoleBindingAccTestBuilder{
		providerConfig:     tfProviderConfig,
		resourceIdentifier: "stackit_" + apiName + "_" + resourceType + "_role_binding_v1." + resourceID,
	}
}

// RoleBindingAccTestBuilder helps to implement acceptance tests for role binding resources and is used to prevent the boilerplate code needed for that type of tests.
type RoleBindingAccTestBuilder struct {
	providerConfig string

	resourceIdentifier string // e.g. "stackit_secretsmanager_instance_role_binding.role_binding"

	// Note: Keep these steps here in the order they are executed later
	createStep *resource.TestStep
	importStep *resource.TestStep
	updateStep *resource.TestStep
}

// CreateStep is the first step in your acceptance test and creates the resources initially
func (b *RoleBindingAccTestBuilder) CreateStep(tfConfig string, variables config.Variables, resourceIdResourceID, resourceIdField string) *RoleBindingAccTestBuilder {
	b.createStep = &resource.TestStep{
		Config:          b.providerConfig + "\n" + tfConfig,
		ConfigVariables: variables,
		Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrPair(
				b.resourceIdentifier, "resource_id",
				resourceIdResourceID, resourceIdField,
			),
			resource.TestCheckResourceAttr(b.resourceIdentifier, "role", testutil.ConvertConfigVariable(variables["role"])),
			resource.TestCheckResourceAttr(b.resourceIdentifier, "subject", testutil.ConvertConfigVariable(variables["subject"])),
		),
	}
	return b
}

// ImportStep adds a terraform import test to your acceptance test case
func (b *RoleBindingAccTestBuilder) ImportStep(variables config.Variables) *RoleBindingAccTestBuilder {
	b.importStep = &resource.TestStep{
		ConfigVariables: variables,
		ResourceName:    b.resourceIdentifier,
		ImportStateIdFunc: func(s *terraform.State) (string, error) {
			r, ok := s.RootModule().Resources[b.resourceIdentifier]
			if !ok {
				return "", fmt.Errorf("couldn't find resource %s", b.resourceIdentifier)
			}

			resourceId, ok := r.Primary.Attributes["resource_id"]
			if !ok {
				return "", fmt.Errorf("couldn't find attribute resource_id")
			}

			subject, ok := r.Primary.Attributes["subject"]
			if !ok {
				return "", fmt.Errorf("couldn't find attribute subject")
			}

			role, ok := r.Primary.Attributes["role"]
			if !ok {
				return "", fmt.Errorf("couldn't find attribute role")
			}

			return fmt.Sprintf("%s,%s,%s,%s", testutil.Region, resourceId, role, subject), nil
		},
		ImportState:       true,
		ImportStateVerify: true,
	}
	return b
}

// UpdateStep is the first step in your acceptance test and updates the resources
func (b *RoleBindingAccTestBuilder) UpdateStep(tfConfig string, variables config.Variables, resourceIdResourceID, resourceIdField string) *RoleBindingAccTestBuilder {
	b.createStep = &resource.TestStep{
		Config:          b.providerConfig + "\n" + tfConfig,
		ConfigVariables: variables,
		Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrPair(
				b.resourceIdentifier, "resource_id",
				resourceIdResourceID, resourceIdField,
			),
			resource.TestCheckResourceAttr(b.resourceIdentifier, "role", testutil.ConvertConfigVariable(variables["role"])),
			resource.TestCheckResourceAttr(b.resourceIdentifier, "subject", testutil.ConvertConfigVariable(variables["subject"])),
		),
	}
	return b
}

func (b *RoleBindingAccTestBuilder) Build() resource.TestCase {
	tc := resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testdestroy.AccTestCheckDestroy,
		Steps:                    []resource.TestStep{},
	}

	if b.createStep != nil {
		tc.Steps = append(tc.Steps, *b.createStep)
	}

	if b.importStep != nil {
		tc.Steps = append(tc.Steps, *b.importStep)
	}

	if b.updateStep != nil {
		tc.Steps = append(tc.Steps, *b.updateStep)
	}

	return tc
}
