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

func TestNoUUID(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"UUID",
			"cae27bba-c43d-498a-861e-d11d241c4ff8",
			false,
		},
		{
			"no UUID",
			"a-b-c-d",
			true,
		},
		{
			"Empty",
			"",
			true,
		},
		{
			"domain name",
			"www.test.de",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			NoUUID().ValidateString(context.Background(), validator.StringRequest{
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
		invalidZero bool
		input       string
		isValid     bool
	}{
		{
			"ok IP4",
			false,
			"111.222.111.222",
			true,
		},
		{
			"ok IP6",
			false,
			"2001:0db8:85a3:08d3::0370:7344",
			true,
		},
		{
			"too short",
			false,
			"0.1.2",
			false,
		},
		{
			"Empty",
			false,
			"",
			false,
		},
		{
			"Not an IP",
			false,
			"for-sure-not-an-IP",
			false,
		},
		{
			"valid ipv4 zero",
			true,
			"0.0.0.0",
			true,
		},
		{
			"invalid ipv4 zero",
			false,
			"0.0.0.0",
			false,
		},
		{
			"valid ipv6 zero",
			true,
			"::",
			true,
		},
		{
			"valid ipv6 zero short notation",
			true,
			"::0",
			true,
		},
		{
			"valid ipv6 zero long notation",
			true,
			"0000:0000:0000:0000:0000:0000:0000:0000",
			true,
		},
		{
			"invalid ipv6 zero short notation",
			false,
			"::",
			false,
		},
		{
			"invalid ipv6 zero long notation",
			false,
			"0000:0000:0000:0000:0000:0000:0000:0000",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			IP(tt.invalidZero).ValidateString(context.Background(), validator.StringRequest{
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
			"CNAME record Not a Fully Qualified Domain Name",
			"stackit.de",
			"CNAME",
			false,
		},
		{
			"CNAME record ok Fully Qualified Domain Name",
			"stackit.de.",
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

func TestValidTtlDuration(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"valid duration with hours, minutes, and seconds",
			"5h30m40s",
			true,
		},
		{
			"valid duration with hours only",
			"5h",
			true,
		},
		{
			"valid duration with hours and minutes",
			"5h30m",
			true,
		},
		{
			"valid duration with minutes only",
			"30m",
			true,
		},
		{
			"valid duration with seconds only",
			"30s",
			true,
		},
		{
			"invalid duration with incorrect unit",
			"30o",
			false,
		},
		{
			"invalid duration without unit",
			"30",
			false,
		},
		{
			"invalid duration with invalid letters",
			"30e",
			false,
		},
		{
			"invalid duration with letters in middle",
			"1h30x",
			false,
		},
		{
			"empty string",
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			va := ValidDurationString()
			va.ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Expected validation to fail for input: %v", tt.input)
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Expected validation to succeed for input: %v, but got errors: %v", tt.input, r.Diagnostics.Errors())
			}
		})
	}
}

func TestValidNoTrailingNewline(t *testing.T) {
	tests := []struct {
		description string
		input       string
		isValid     bool
	}{
		{
			"string with no trailing newline",
			"abc",
			true,
		},
		{
			"string with trailing \\n",
			"abc\n",
			false,
		},
		{
			"string with trailing \\r\\n",
			"abc\r\n",
			false,
		},
		{
			"string with internal newlines but not trailing",
			"abc\ndef\nghi",
			true,
		},
		{
			"empty string",
			"",
			true,
		},
		{
			"string that is just \\n",
			"\n",
			false,
		},
		{
			"string that is just \\r\\n",
			"\r\n",
			false,
		},
		{
			"string with multiple newlines, trailing",
			"abc\n\n",
			false,
		},
		{
			"string with newlines but ends with character",
			"abc\ndef\n",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := validator.StringResponse{}
			va := ValidNoTrailingNewline()
			va.ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}, &r)

			if !tt.isValid && !r.Diagnostics.HasError() {
				t.Fatalf("Expected validation to fail for input: %q", tt.input)
			}
			if tt.isValid && r.Diagnostics.HasError() {
				t.Fatalf("Expected validation to succeed for input: %q, but got errors: %v", tt.input, r.Diagnostics.Errors())
			}
		})
	}
}
