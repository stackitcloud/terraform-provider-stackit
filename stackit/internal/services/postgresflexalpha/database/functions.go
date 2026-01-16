package postgresflexalpha

import (
	"context"
	"fmt"

	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
)

// databaseClientReader represents the contract to listing databases from postgresflex.APIClient.
type databaseClientReader interface {
	ListDatabasesRequest(
		ctx context.Context,
		projectId string,
		region string,
		instanceId string,
	) postgresflex.ApiListDatabasesRequestRequest
}

// getDatabaseById gets a database by its ID.
func getDatabaseById(
	ctx context.Context,
	client databaseClientReader,
	projectId, region, instanceId string,
	databaseId int64,
) (*postgresflex.ListDatabase, error) {
	filter := func(db postgresflex.ListDatabase) bool {
		return db.Id != nil && *db.Id == databaseId
	}
	return getDatabase(ctx, client, projectId, region, instanceId, filter)
}

// getDatabaseByName gets a database by its name.
func getDatabaseByName(
	ctx context.Context,
	client databaseClientReader,
	projectId, region, instanceId, databaseName string,
) (*postgresflex.ListDatabase, error) {
	filter := func(db postgresflex.ListDatabase) bool {
		return db.Name != nil && *db.Name == databaseName
	}
	return getDatabase(ctx, client, projectId, region, instanceId, filter)
}

// getDatabase is a helper function to retrieve a database using a filter function.
// Hint: The API does not have a GetDatabase endpoint, only ListDatabases
func getDatabase(
	ctx context.Context,
	client databaseClientReader,
	projectId, region, instanceId string,
	filter func(db postgresflex.ListDatabase) bool,
) (*postgresflex.ListDatabase, error) {
	if projectId == "" || region == "" || instanceId == "" {
		return nil, fmt.Errorf("all parameters (project, region, instance) are required")
	}

	const pageSize = 25

	for page := int64(1); ; page++ {
		res, err := client.ListDatabasesRequest(ctx, projectId, region, instanceId).
			Page(page).Size(pageSize).Sort(postgresflex.DATABASESORT_INDEX_ASC).Execute()
		if err != nil {
			return nil, fmt.Errorf("requesting database list (page %d): %w", page, err)
		}

		// If the API returns no databases, we have reached the end of the list.
		if res.Databases == nil || len(*res.Databases) == 0 {
			break
		}

		// Iterate over databases to find a match
		for _, db := range *res.Databases {
			if filter(db) {
				foundDb := db
				return &foundDb, nil
			}
		}
	}

	return nil, fmt.Errorf("database not found for instance %s", instanceId)
}
