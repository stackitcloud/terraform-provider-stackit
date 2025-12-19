// Copyright (c) STACKIT

package utils

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type attributeGetterFunc func(ctx context.Context, attributePath path.Path, target interface{}) diag.Diagnostics

func (a attributeGetterFunc) GetAttribute(ctx context.Context, attributePath path.Path, target interface{}) diag.Diagnostics {
	return a(ctx, attributePath, target)
}

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		log.Panicf("cannot load location %s: %v", name, err)
	}
	return loc
}

func TestGetTimeFromString(t *testing.T) {
	type args struct {
		path       path.Path
		source     attributeGetterFunc
		dateFormat string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    time.Time
	}{
		{
			name: "simple string",
			args: args{
				path: path.Root("foo"),
				source: func(_ context.Context, _ path.Path, target interface{}) diag.Diagnostics {
					t, ok := target.(*types.String)
					if !ok {
						log.Panicf("wrong type %T", target)
					}
					*t = types.StringValue("2025-02-06T09:41:00+01:00")
					return nil
				},
				dateFormat: time.RFC3339,
			},
			want: time.Date(2025, 2, 6, 9, 41, 0, 0, mustLocation("Europe/Berlin")),
		},
		{
			name: "invalid type",
			args: args{
				path: path.Root("foo"),
				source: func(_ context.Context, p path.Path, _ interface{}) (diags diag.Diagnostics) {
					diags.AddAttributeError(p, "kapow", "kapow")
					return diags
				},
				dateFormat: time.RFC3339,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target time.Time
			gotDiags := GetTimeFromStringAttribute(context.Background(), tt.args.path, tt.args.source, tt.args.dateFormat, &target)
			if tt.wantErr {
				if !gotDiags.HasError() {
					t.Errorf("expected error")
				}
			} else {
				if gotDiags.HasError() {
					t.Errorf("expected no errors, but got %v", gotDiags)
				} else {
					if want, got := tt.want, target; !want.Equal(got) {
						t.Errorf("got wrong date, want %s but got %s", want, got)
					}
				}
			}
		})
	}
}
