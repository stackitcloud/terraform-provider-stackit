package core

import (
	"reflect"
	"testing"
)

func Test_initClientCollection(t *testing.T) {
	type args struct {
		clientFactory ClientFactory
	}
	tests := []struct {
		name    string
		args    args
		want    *clientCollection
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := initClientCollection(tt.args.clientFactory)
			if (err != nil) != tt.wantErr {
				t.Errorf("initClientCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("initClientCollection() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitClientCollection_NoNilFields(t *testing.T) {
	// This test case makes sure that all fields in the client collection are initialized after initializing it
	// using a client factory. It uses reflection to make sure all fields are covered, also new ones which got added.
	// A regular unit test without reflection couldn't cover this.

	mockFactory := &MockClientFactory{}

	cc, err := initClientCollection(mockFactory)
	if err != nil {
		t.Fatalf("unexpected error during initialization: %v", err)
	}

	if cc == nil {
		t.Fatal("expected clientCollection pointer to be non-nil")
	}

	val := reflect.ValueOf(*cc)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldName := typ.Field(i).Name

		if fieldVal.IsZero() {
			t.Errorf("field %q is nil or uninitialized", fieldName)
		}
	}
}
