package iaasalpha

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func VpcCheckDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := iaas.NewAPIClient(testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildClientOptions(testutil.IaaSCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type vpcIds struct {
		projectID string
		vpcID     string
	}

	vpcsToDestroy := []vpcIds{}
	var errs []error
	// vpc
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_vpc" {
			continue
		}
		projectId, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", rs.Primary))
			continue
		}
		vpcId, ok := rs.Primary.Attributes["vpc_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no vpc_id found in %s", rs.Primary))
			continue
		}
		vpcsToDestroy = append(vpcsToDestroy, vpcIds{
			projectID: projectId,
			vpcID:     vpcId,
		})
	}

	for _, vpc := range vpcsToDestroy {
		_, err := client.DefaultAPI.GetVPC(ctx, vpc.projectID, vpc.vpcID).Execute()
		if err == nil {
			err := client.DefaultAPI.DeleteVPC(ctx, vpc.projectID, vpc.vpcID).Execute()
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting vpc with ID %q: %w", vpc.vpcID, err))
			}
			continue
		}
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if oapiErr.StatusCode == 404 || oapiErr.StatusCode == 403 {
				continue
			}
		}
		errs = append(errs, fmt.Errorf("deleting vpc: %w", err))
	}

	return errors.Join(errs...)
}

func VpcRegionDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := iaas.NewAPIClient(testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildClientOptions(testutil.IaaSCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	type regionIds struct {
		projectId, vpcId, region string
	}

	var toDestroy []regionIds
	var errs []error
	for _, r := range s.RootModule().Resources {
		if r.Type != "stackit_vpc_region" {
			continue
		}
		projectId, ok := r.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", r.Primary))
			continue
		}
		vpcId, ok := r.Primary.Attributes["vpc_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no vpc_id found in %s", r.Primary))
			continue
		}
		region, ok := r.Primary.Attributes["region"]
		if !ok {
			errs = append(errs, fmt.Errorf("no region found in %s", r.Primary))
			continue
		}
		toDestroy = append(toDestroy, regionIds{
			projectId: projectId,
			vpcId:     vpcId,
			region:    region,
		})
	}
	for _, id := range toDestroy {
		_, err := client.DefaultAPI.GetVPCRegion(ctx, id.projectId, id.vpcId, id.region).Execute()
		if err == nil {
			err := client.DefaultAPI.DeleteVPCRegion(ctx, id.projectId, id.vpcId, id.region).Execute()
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting region with ID %q in project %q, vpc %q: %w", id.region, id.projectId, id.vpcId, err))
			}
			continue
		}
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if oapiErr.StatusCode == 404 || oapiErr.StatusCode == 403 {
				continue
			}
		}
		errs = append(errs, fmt.Errorf("deleting region: %w", err))
	}
	return errors.Join(errs...)
}

func VpcNetworkRangeDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := iaas.NewAPIClient(testutil.NewConfigBuilder().Experiments(testutil.ExperimentVPC).BuildClientOptions(testutil.IaaSCustomEndpoint, false)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	type networkRangeIds struct {
		projectId      string
		vpcId          string
		region         string
		networkRangeId string
	}

	networkRangesToDestroy := []networkRangeIds{}
	var errs []error
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_vpc_network_range" {
			continue
		}
		projectId, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no project_id found in %s", rs.Primary))
			continue
		}
		vpcId, ok := rs.Primary.Attributes["vpc_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no vpc_id found in %s", rs.Primary))
			continue
		}
		region, ok := rs.Primary.Attributes["region"]
		if !ok {
			errs = append(errs, fmt.Errorf("no region found in %s", rs.Primary))
			continue
		}
		networkRangeId, ok := rs.Primary.Attributes["network_range_id"]
		if !ok {
			errs = append(errs, fmt.Errorf("no network_range_id found in %s", rs.Primary))
			continue
		}
		networkRangesToDestroy = append(networkRangesToDestroy, networkRangeIds{
			projectId:      projectId,
			vpcId:          vpcId,
			region:         region,
			networkRangeId: networkRangeId,
		})
	}

	for _, networkRange := range networkRangesToDestroy {
		_, err := client.DefaultAPI.GetVPCNetworkRange(ctx, networkRange.projectId, networkRange.vpcId, networkRange.region, networkRange.networkRangeId).Execute()
		if err == nil {
			err := client.DefaultAPI.DeleteVPCNetworkRange(ctx, networkRange.projectId, networkRange.vpcId, networkRange.region, networkRange.networkRangeId).Execute()
			if err != nil {
				errs = append(errs, fmt.Errorf("deleting network range with ID %q in project %q, vpc %q, region %q : %w", networkRange.networkRangeId, networkRange.projectId, networkRange.vpcId, networkRange.region, err))
			}
			continue
		}

		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if oapiErr.StatusCode == 404 || oapiErr.StatusCode == 403 {
				continue
			}
		}
		errs = append(errs, fmt.Errorf("deleting static route: %w", err))
	}

	return errors.Join(errs...)
}
