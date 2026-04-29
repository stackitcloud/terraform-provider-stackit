package testdestroy

import (
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// AccTestCheckDestroy is a helper function to destroy potential leftover resources from acceptance tests
func AccTestCheckDestroy(s *terraform.State) error {
	destroyFuncs := []func(state *terraform.State) error{
		testAccCheckSecretsManagerDestroy,
	}

	for _, fn := range destroyFuncs {
		err := fn(s)
		if err != nil {
			return err
		}
	}

	return nil
}
