package validate

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestUUID(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"cae27bba-c43d-498a-861e-d11d241c4ff8",
			true,
		},
		{
			"too short",
			"a-b-c-d",
			false,
		},
		{
			"Empty",
			"",
			false,
		},
		{
			"not UUID",
			"www-541-%",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			UUID().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestIP(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok IP4",
			"111.222.111.222",
			true,
		},
		{
			"ok IP6",
			"2001:0db8:85a3:08d3::0370:7344",
			true,
		},
		{
			"too short",
			"0.1.2",
			false,
		},
		{
			"Empty",
			"",
			false,
		},
		{
			"Not an IP",
			"for-sure-not-an-IP",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			IP().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestRecordSet(t *testing.T) {
	tests := []struct {
		description string
		record      string
		recordType  string
		isValid     bool
	}{
		{
			"A record ok IP4",
			"111.222.111.222",
			"A",
			true,
		},
		{
			"A record fail IP6",
			"2001:0db8:85a3:08d3::0370:7344",
			"A",
			false,
		},
		{
			"A record too short",
			"0.1.2",
			"A",
			false,
		},
		{
			"A record Empty",
			"",
			"A",
			false,
		},
		{
			"A record Not an IP",
			"for-sure-not-an-IP",
			"A",
			false,
		},
		{
			"AAAA record fail IP4",
			"111.222.111.222",
			"AAAA",
			false,
		},
		{
			"AAAA record ok IP6",
			"2001:0db8:85a3:08d3::0370:7344",
			"AAAA",
			true,
		},
		{
			"AAAA record too short",
			"0.1.2",
			"AAAA",
			false,
		},
		{
			"AAAA record Empty",
			"",
			"AAAA",
			false,
		},
		{
			"AAAA record Not an IP",
			"for-sure-not-an-IP",
			"AAAA",
			false,
		},
		{
			"CNAME record",
			"some-record",
			"CNAME",
			true,
		},
		{
			"NS record",
			"some-record",
			"NS",
			true,
		},
		{
			"MX record",
			"some-record",
			"MX",
			true,
		},
		{
			"TXT record",
			"some-record",
			"TXT",
			true,
		},
		{
			"ALIAS record",
			"some-record",
			"ALIAS",
			true,
		},
		{
			"DNAME record",
			"some-record",
			"DNAME",
			true,
		},
		{
			"CAA record",
			"some-record",
			"CAA",
			true,
		},
		{
			"random record",
			"some-record",
			"random",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			scheme := tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"type": tftypes.String,
				},
			}
			value := map[string]tftypes.Value{
				"type": tftypes.NewValue(tftypes.String, tt.recordType),
			}
			record := tftypes.NewValue(scheme, value)

			RecordSet().ValidateString(context.Background(), validator.StringRequest{
				Config: tfsdk.Config{
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{},
						},
					},
					Raw: record,
				},
				ConfigValue: types.StringValue(tt.record),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestNoSeparator(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"ABCD",
			true,
		},
		{
			"ok-2",
			"#$%&/()=.;-",
			true,
		},
		{
			"Empty",
			"",
			true,
		},
		{
			"not ok",
			"ab,",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			NoSeparator().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestNonLegacyProjectRole(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"owner",
			true,
		},
		{
			"ok-2",
			"reader",
			true,
		},
		{
			"leagcy-role",
			"project.owner",
			false,
		},
		{
			"leagcy-role-2",
			"project.admin",
			false,
		},
		{
			"leagcy-role-3",
			"project.member",
			false,
		},
		{
			"leagcy-role-4",
			"project.auditor",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			NonLegacyProjectRole().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestMinorVersionNumber(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"1.20",
			true,
		},
		{
			"ok-2",
			"1.3",
			true,
		},
		{
			"ok-3",
			"10.1",
			true,
		},
		{
			"Empty",
			"",
			false,
		},
		{
			"not ok",
			"afssfdfs",
			false,
		},
		{
			"not ok-major-version",
			"1",
			false,
		},
		{
			"not ok-patch-version",
			"1.20.1",
			false,
		},
		{
			"not ok-version",
			"v1.20.1",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			MinorVersionNumber().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestVersionNumber(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"1.20",
			true,
		},
		{
			"ok-2",
			"1.3",
			true,
		},
		{
			"ok-3",
			"10.1",
			true,
		},
		{
			"ok-patch-version",
			"1.20.1",
			true,
		},
		{
			"ok-patch-version-2",
			"1.20.10",
			true,
		},
		{
			"ok-patch-version-3",
			"10.20.10",
			true,
		},
		{
			"Empty",
			"",
			false,
		},
		{
			"not ok",
			"afssfdfs",
			false,
		},
		{
			"not ok-major-version",
			"1",
			false,
		},
		{
			"not ok-version",
			"v1.20.1",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			VersionNumber().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestRFC3339SecondsOnly(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"9999-01-02T03:04:05Z",
			true,
		},
		{
			"ok_2",
			"9999-01-02T03:04:05+06:00",
			true,
		},
		{
			"empty",
			"",
			false,
		},
		{
			"not_ok",
			"foo-bar",
			false,
		},
		{
			"with_sub_seconds",
			"9999-01-02T03:04:05.678Z",
			false,
		},
		{
			"with_sub_seconds_2",
			"9999-01-02T03:04:05.678+06:00",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			RFC3339SecondsOnly().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestCIDR(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"IPv4_block",
			"198.51.100.14/24",
			true,
		},
		{
			"IPv4_block_2",
			"111.222.111.222/22",
			true,
		},
		{
			"IPv4_single",
			"198.51.100.14/32",
			true,
		},
		{
			"IPv4_entire_internet",
			"0.0.0.0/0",
			true,
		},
		{
			"IPv4_block_invalid",
			"198.51.100.14/33",
			false,
		},
		{
			"IPv4_no_block",
			"111.222.111.222",
			false,
		},
		{
			"IPv6_block",
			"2001:db8::/48",
			true,
		},
		{
			"IPv6_single",
			"2001:0db8:85a3:08d3::0370:7344/128",
			true,
		},
		{
			"IPv6_all",
			"::/0",
			true,
		},
		{
			"IPv6_block_invalid",
			"2001:0db8:85a3:08d3::0370:7344/129",
			false,
		},
		{
			"IPv6_no_block",
			"2001:0db8:85a3:08d3::0370:7344",
			false,
		},
		{
			"empty",
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			CIDR().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestRrule(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1",
			true,
		},
		{
			"ok-2",
			"DTSTART;TZID=Europe/Sofia:20200803T023000\nRULE:FREQ=DAILY;INTERVAL=1",
			true,
		},
		{
			"Empty",
			"",
			false,
		},
		{
			"not ok",
			"afssfdfs",
			false,
		},
		{
			"not ok-missing-space-before-rrule",
			"DTSTART;TZID=Europe/Sofia:20200803T023000RRULE:FREQ=DAILY;INTERVAL=1",
			false,
		},
		{
			"not ok-missing-interval",
			"DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			Rrule().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"ok",
			"testdata/file.txt",
			true,
		},
		{
			"not ok",
			"testdata/non-existing-file.txt",
			false,
		},
		{
			"empty",
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			FileExists().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Should not have failed: %v", r.Diagnostics.Errors())
			}
		})
	}
}
