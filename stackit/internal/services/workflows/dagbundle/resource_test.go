package dagbundle

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"
)

func ptrString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func mustGitAuth(t *testing.T, am gitAuthModel) basetypes.ObjectValue {
	t.Helper()
	o, diags := types.ObjectValueFrom(context.Background(), gitAuthTypes, am)
	if diags.HasError() {
		t.Fatalf("building git auth fixture: %v", diags.Errors())
	}
	return o
}

func mustS3Auth(t *testing.T, am s3AuthModel) basetypes.ObjectValue {
	t.Helper()
	o, diags := types.ObjectValueFrom(context.Background(), s3AuthTypes, am)
	if diags.HasError() {
		t.Fatalf("building s3 auth fixture: %v", diags.Errors())
	}
	return o
}

func mustGit(t *testing.T, gm gitModel) basetypes.ObjectValue {
	t.Helper()
	o, diags := types.ObjectValueFrom(context.Background(), gitTypes, gm)
	if diags.HasError() {
		t.Fatalf("building git fixture: %v", diags.Errors())
	}
	return o
}

func mustS3(t *testing.T, sm s3Model) basetypes.ObjectValue {
	t.Helper()
	o, diags := types.ObjectValueFrom(context.Background(), s3Types, sm)
	if diags.HasError() {
		t.Fatalf("building s3 fixture: %v", diags.Errors())
	}
	return o
}

func gitBundleResponse() *workflows.DagBundleResponse {
	resp := workflows.GitDagBundleResponseAsDagBundleResponse(&workflows.GitDagBundleResponse{
		Type:            workflows.GITDAGBUNDLERESPONSETYPE_GIT,
		Name:            "main-dags",
		Url:             "https://git.example.com/repo.git",
		Branch:          "main",
		Subdir:          sdkUtils.Ptr("dags/"),
		RefreshInterval: sdkUtils.Ptr(int32(60)),
		Auth: workflows.BasicAuthResponseAsGitAuthResponse(&workflows.BasicAuthResponse{
			Type:     sdkUtils.Ptr("basic"),
			Username: sdkUtils.Ptr("git-user"),
		}),
	})
	return &resp
}

func s3BundleResponse() *workflows.DagBundleResponse {
	resp := workflows.S3DagBundleResponseAsDagBundleResponse(&workflows.S3DagBundleResponse{
		Type:            workflows.S3DAGBUNDLERESPONSETYPE_S3,
		Name:            "backup-dags",
		BucketName:      "my-bucket",
		Endpoint:        sdkUtils.Ptr("https://object.storage.eu01.onstackit.cloud"),
		Prefix:          sdkUtils.Ptr("dags/"),
		RefreshInterval: sdkUtils.Ptr(int32(120)),
		S3Auth: workflows.S3AccessKeyAuthResponseAsS3AuthResponse(&workflows.S3AccessKeyAuthResponse{
			Type:        workflows.S3ACCESSKEYAUTHRESPONSETYPE_ACCESS_KEY,
			AccessKeyId: "AKIA...",
		}),
	})
	return &resp
}

func priorGitModel(t *testing.T) *Model {
	t.Helper()
	auth := mustGitAuth(t, gitAuthModel{
		Type:     types.StringValue("basic"),
		Username: types.StringValue("git-user"),
		Password: types.StringValue("PLANNED-PASSWORD"),
	})
	git := mustGit(t, gitModel{
		URL:             types.StringValue("https://git.example.com/repo.git"),
		Branch:          types.StringValue("main"),
		Subdir:          types.StringNull(),
		RefreshInterval: types.Int32Null(),
		Auth:            auth,
	})
	return &Model{
		ProjectID:  types.StringValue("pid"),
		InstanceID: types.StringValue("iid"),
		Name:       types.StringValue("main-dags"),
		Git:        git,
		S3:         types.ObjectNull(s3Types),
	}
}

func priorS3Model(t *testing.T) *Model {
	t.Helper()
	auth := mustS3Auth(t, s3AuthModel{
		Type:            types.StringValue("access_key"),
		AccessKeyID:     types.StringValue("AKIA..."),
		SecretAccessKey: types.StringValue("PLANNED-S3-SECRET"),
	})
	s3 := mustS3(t, s3Model{
		BucketName:      types.StringValue("my-bucket"),
		Endpoint:        types.StringNull(),
		Prefix:          types.StringNull(),
		RefreshInterval: types.Int32Null(),
		Auth:            auth,
	})
	return &Model{
		ProjectID:  types.StringValue("pid"),
		InstanceID: types.StringValue("iid"),
		Name:       types.StringValue("backup-dags"),
		S3:         s3,
		Git:        types.ObjectNull(gitTypes),
	}
}

func TestMapFields_Git(t *testing.T) {
	model := priorGitModel(t)
	err := mapFields(context.Background(), gitBundleResponse(), model, "eu01")
	if err != nil {
		t.Fatalf("mapFields error: %v", err)
	}

	if got, want := model.ID.ValueString(), "pid,eu01,iid,main-dags"; got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if model.Git.IsNull() {
		t.Fatalf("Git block should be set")
	}
	if !model.S3.IsNull() {
		t.Errorf("S3 block should be null")
	}
	var gm gitModel
	if d := model.Git.As(context.Background(), &gm, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("reading git: %v", d.Errors())
	}
	if got, want := gm.URL.ValueString(), "https://git.example.com/repo.git"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
	if got, want := gm.Branch.ValueString(), "main"; got != want {
		t.Errorf("Branch = %q, want %q", got, want)
	}
	if got, want := gm.RefreshInterval.ValueInt32(), int32(60); got != want {
		t.Errorf("RefreshInterval = %d, want %d", got, want)
	}
	var auth gitAuthModel
	if d := gm.Auth.As(context.Background(), &auth, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("reading auth: %v", d.Errors())
	}
	if auth.Password.ValueString() != "PLANNED-PASSWORD" {
		t.Errorf("Password should be preserved as %q, got %q", "PLANNED-PASSWORD", auth.Password.ValueString())
	}
	if auth.Type.ValueString() != "basic" {
		t.Errorf("auth.Type = %q, want basic", auth.Type.ValueString())
	}
	if auth.Username.ValueString() != "git-user" {
		t.Errorf("auth.Username = %q, want git-user", auth.Username.ValueString())
	}
}

// TestMapFields_Git_SubdirAndRefreshIntervalEdgeCases pins behavior the
// production code relies on: mapFields writes the server's verbatim subdir
// (including any trailing slash) and tolerates a missing refresh_interval.
func TestMapFields_Git_SubdirAndRefreshIntervalEdgeCases(t *testing.T) {
	tests := []struct {
		desc                string
		subdir              *string
		refreshInterval     *int32
		wantSubdir          types.String
		wantRefreshInterval types.Int32
	}{
		{"server returns trailing slash subdir verbatim", sdkUtils.Ptr("dags/"), sdkUtils.Ptr(int32(60)), types.StringValue("dags/"), types.Int32Value(60)},
		{"server returns null subdir + null refresh_interval", nil, nil, types.StringNull(), types.Int32Null()},
		{"server returns empty subdir literal", sdkUtils.Ptr(""), sdkUtils.Ptr(int32(0)), types.StringValue(""), types.Int32Value(0)},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			resp := workflows.GitDagBundleResponseAsDagBundleResponse(&workflows.GitDagBundleResponse{
				Type:            workflows.GITDAGBUNDLERESPONSETYPE_GIT,
				Name:            "main-dags",
				Url:             "https://git.example.com/repo.git",
				Branch:          "main",
				Subdir:          tt.subdir,
				RefreshInterval: tt.refreshInterval,
				Auth: workflows.BasicAuthResponseAsGitAuthResponse(&workflows.BasicAuthResponse{
					Type:     sdkUtils.Ptr("basic"),
					Username: sdkUtils.Ptr("git-user"),
				}),
			})
			model := priorGitModel(t)
			if err := mapFields(context.Background(), &resp, model, "eu01"); err != nil {
				t.Fatalf("mapFields: %v", err)
			}
			var gm gitModel
			if d := model.Git.As(context.Background(), &gm, basetypes.ObjectAsOptions{}); d.HasError() {
				t.Fatalf("reading git: %v", d.Errors())
			}
			if !gm.Subdir.Equal(tt.wantSubdir) {
				t.Errorf("Subdir = %v, want %v", gm.Subdir, tt.wantSubdir)
			}
			if !gm.RefreshInterval.Equal(tt.wantRefreshInterval) {
				t.Errorf("RefreshInterval = %v, want %v", gm.RefreshInterval, tt.wantRefreshInterval)
			}
		})
	}
}

func TestMapFields_S3(t *testing.T) {
	model := priorS3Model(t)
	err := mapFields(context.Background(), s3BundleResponse(), model, "eu01")
	if err != nil {
		t.Fatalf("mapFields error: %v", err)
	}

	if model.S3.IsNull() {
		t.Fatalf("S3 block should be set")
	}
	if !model.Git.IsNull() {
		t.Errorf("Git block should be null")
	}
	var sm s3Model
	if d := model.S3.As(context.Background(), &sm, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("reading s3: %v", d.Errors())
	}
	if got, want := sm.BucketName.ValueString(), "my-bucket"; got != want {
		t.Errorf("BucketName = %q, want %q", got, want)
	}
	if got, want := sm.RefreshInterval.ValueInt32(), int32(120); got != want {
		t.Errorf("RefreshInterval = %d, want %d", got, want)
	}
	var auth s3AuthModel
	if d := sm.Auth.As(context.Background(), &auth, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("reading s3 auth: %v", d.Errors())
	}
	if auth.SecretAccessKey.ValueString() != "PLANNED-S3-SECRET" {
		t.Errorf("SecretAccessKey should be preserved as %q, got %q", "PLANNED-S3-SECRET", auth.SecretAccessKey.ValueString())
	}
	if auth.AccessKeyID.ValueString() != "AKIA..." {
		t.Errorf("AccessKeyID = %q, want AKIA...", auth.AccessKeyID.ValueString())
	}
}

func TestMapFields_Errors(t *testing.T) {
	tests := []struct {
		desc string
		in   *workflows.DagBundleResponse
	}{
		{"nil bundle", nil},
		{"empty discriminator", &workflows.DagBundleResponse{}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			model := priorGitModel(t)
			if err := mapFields(context.Background(), tt.in, model, "eu01"); err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestBuildCreatePayload_Git(t *testing.T) {
	auth := mustGitAuth(t, gitAuthModel{
		Type:     types.StringValue("basic"),
		Username: types.StringValue("u"),
		Password: types.StringValue("p"),
	})
	git := mustGit(t, gitModel{
		URL:             types.StringValue("https://git.example.com/repo.git"),
		Branch:          types.StringValue("main"),
		Subdir:          types.StringValue("dags/"),
		RefreshInterval: types.Int32Value(60),
		Auth:            auth,
	})
	model := &Model{
		Name: types.StringValue("main-dags"),
		Git:  git,
		S3:   types.ObjectNull(s3Types),
	}

	payload, err := toCreatePayload(context.Background(), model)
	if err != nil {
		t.Fatalf("toCreatePayload: %v", err)
	}
	if payload.GitDagBundle == nil {
		t.Fatalf("expected GitDagBundle variant")
	}
	g := payload.GitDagBundle
	if g.Type != workflows.GITDAGBUNDLETYPE_GIT {
		t.Errorf("Type = %v", g.Type)
	}
	if g.Url != "https://git.example.com/repo.git" {
		t.Errorf("Url = %q", g.Url)
	}
	if g.Branch != "main" {
		t.Errorf("Branch = %q", g.Branch)
	}
	if ptrString(g.Subdir) != "dags/" {
		t.Errorf("Subdir = %q", ptrString(g.Subdir))
	}
	if g.RefreshInterval == nil || *g.RefreshInterval != 60 {
		t.Errorf("RefreshInterval = %v", g.RefreshInterval)
	}
	if g.Auth.BasicAuth == nil {
		t.Fatalf("expected BasicAuth variant")
	}
	if ptrString(g.Auth.BasicAuth.Username) != "u" {
		t.Errorf("Username = %q", ptrString(g.Auth.BasicAuth.Username))
	}
}

func TestBuildCreatePayload_S3(t *testing.T) {
	auth := mustS3Auth(t, s3AuthModel{
		Type:            types.StringValue("access_key"),
		AccessKeyID:     types.StringValue("AKIA"),
		SecretAccessKey: types.StringValue("SECRET"),
	})
	s3 := mustS3(t, s3Model{
		BucketName:      types.StringValue("my-bucket"),
		Endpoint:        types.StringValue("https://object.example.com"),
		Prefix:          types.StringValue("dags/"),
		RefreshInterval: types.Int32Null(),
		Auth:            auth,
	})
	model := &Model{
		Name: types.StringValue("backup-dags"),
		S3:   s3,
		Git:  types.ObjectNull(gitTypes),
	}

	payload, err := toCreatePayload(context.Background(), model)
	if err != nil {
		t.Fatalf("toCreatePayload: %v", err)
	}
	if payload.S3DagBundle == nil {
		t.Fatalf("expected S3DagBundle variant")
	}
	s := payload.S3DagBundle
	if s.BucketName != "my-bucket" {
		t.Errorf("BucketName = %q", s.BucketName)
	}
	if s.S3Auth.S3AccessKeyAuth == nil {
		t.Fatalf("expected S3AccessKeyAuth variant")
	}
	if s.S3Auth.S3AccessKeyAuth.AccessKeyId != "AKIA" {
		t.Errorf("AccessKeyId = %q", s.S3Auth.S3AccessKeyAuth.AccessKeyId)
	}
}

func TestBuildCreatePayload_NeitherSet(t *testing.T) {
	model := &Model{
		Git: types.ObjectNull(gitTypes),
		S3:  types.ObjectNull(s3Types),
	}
	if _, err := toCreatePayload(context.Background(), model); err == nil {
		t.Errorf("expected error when neither git nor s3 is set")
	}
}

func TestBuildUpdatePayload_Git(t *testing.T) {
	plan := &Model{
		Name: types.StringValue("main-dags"),
		Git: mustGit(t, gitModel{
			URL:             types.StringValue("https://git.example.com/repo.git"),
			Branch:          types.StringValue("dev"),
			Subdir:          types.StringValue("dags"),
			RefreshInterval: types.Int32Value(120),
			Auth:            types.ObjectNull(gitAuthTypes),
		}),
		S3: types.ObjectNull(s3Types),
	}
	state := &Model{
		Git: mustGit(t, gitModel{
			URL:             types.StringValue("https://git.example.com/repo.git"),
			Branch:          types.StringValue("main"),
			Subdir:          types.StringValue("dags"),
			RefreshInterval: types.Int32Null(),
			Auth:            types.ObjectNull(gitAuthTypes),
		}),
		S3: types.ObjectNull(s3Types),
	}
	payload, err := toUpdatePayload(context.Background(), plan, state)
	if err != nil {
		t.Fatalf("toUpdatePayload: %v", err)
	}
	if payload.UpdateGitDagBundlePayload == nil {
		t.Fatalf("expected UpdateGitDagBundlePayload variant")
	}
	g := payload.UpdateGitDagBundlePayload
	if ptrString(g.Url) != "https://git.example.com/repo.git" {
		t.Errorf("Url = %q", ptrString(g.Url))
	}
	if ptrString(g.Branch) != "dev" {
		t.Errorf("Branch = %q", ptrString(g.Branch))
	}
	if g.Auth != nil {
		t.Errorf("Auth should be nil when not set (server-side: leaves credentials untouched), got %+v", g.Auth)
	}
}

func TestBuildUpdatePayload_GitWithAuth(t *testing.T) {
	auth := mustGitAuth(t, gitAuthModel{
		Type:     types.StringValue("basic"),
		Username: types.StringValue("u"),
		Password: types.StringValue("p"),
	})
	plan := &Model{
		Git: mustGit(t, gitModel{
			URL:             types.StringValue("https://git.example.com/repo.git"),
			Branch:          types.StringValue("main"),
			Subdir:          types.StringNull(),
			RefreshInterval: types.Int32Null(),
			Auth:            auth,
		}),
		S3: types.ObjectNull(s3Types),
	}
	state := &Model{Git: types.ObjectNull(gitTypes), S3: types.ObjectNull(s3Types)}
	payload, err := toUpdatePayload(context.Background(), plan, state)
	if err != nil {
		t.Fatalf("toUpdatePayload: %v", err)
	}
	if payload.UpdateGitDagBundlePayload.Auth == nil {
		t.Fatalf("Auth should be set")
	}
	if payload.UpdateGitDagBundlePayload.Auth.BasicAuth == nil {
		t.Errorf("expected BasicAuth variant")
	}
}

func TestBuildUpdatePayload_S3(t *testing.T) {
	plan := &Model{
		S3: mustS3(t, s3Model{
			BucketName:      types.StringValue("my-bucket"),
			Endpoint:        types.StringValue("https://example.com"),
			Prefix:          types.StringValue("dags"),
			RefreshInterval: types.Int32Null(),
			Auth:            types.ObjectNull(s3AuthTypes),
		}),
		Git: types.ObjectNull(gitTypes),
	}
	state := &Model{Git: types.ObjectNull(gitTypes), S3: types.ObjectNull(s3Types)}
	payload, err := toUpdatePayload(context.Background(), plan, state)
	if err != nil {
		t.Fatalf("toUpdatePayload: %v", err)
	}
	if payload.UpdateS3DagBundlePayload == nil {
		t.Fatalf("expected UpdateS3DagBundlePayload variant")
	}
	if payload.UpdateS3DagBundlePayload.S3Auth != nil {
		t.Errorf("S3Auth should be nil when not set")
	}
}

func TestBuildUpdatePayload_NeitherSet(t *testing.T) {
	plan := &Model{Git: types.ObjectNull(gitTypes), S3: types.ObjectNull(s3Types)}
	state := &Model{Git: types.ObjectNull(gitTypes), S3: types.ObjectNull(s3Types)}
	if _, err := toUpdatePayload(context.Background(), plan, state); err == nil {
		t.Errorf("expected error when neither git nor s3 is set")
	}
}

// TestBuildUpdatePayload_ClearsGitSubdir verifies that removing subdir from the
// config translates to "" so the server clears it (server uses "" as the clear
// signal — see workflows_service.py).
func TestBuildUpdatePayload_ClearsGitSubdir(t *testing.T) {
	plan := &Model{
		Git: mustGit(t, gitModel{
			URL:             types.StringValue("https://git.example.com/repo.git"),
			Branch:          types.StringValue("main"),
			Subdir:          types.StringNull(),
			RefreshInterval: types.Int32Null(),
			Auth:            types.ObjectNull(gitAuthTypes),
		}),
		S3: types.ObjectNull(s3Types),
	}
	state := &Model{
		Git: mustGit(t, gitModel{
			URL:             types.StringValue("https://git.example.com/repo.git"),
			Branch:          types.StringValue("main"),
			Subdir:          types.StringValue("had-subdir"),
			RefreshInterval: types.Int32Null(),
			Auth:            types.ObjectNull(gitAuthTypes),
		}),
		S3: types.ObjectNull(s3Types),
	}
	payload, err := toUpdatePayload(context.Background(), plan, state)
	if err != nil {
		t.Fatalf("toUpdatePayload: %v", err)
	}
	g := payload.UpdateGitDagBundlePayload
	if g.Subdir == nil || *g.Subdir != "" {
		t.Errorf("Subdir = %v, want pointer to \"\"", g.Subdir)
	}
}

// TestBuildUpdatePayload_ClearsS3PrefixAndEndpoint verifies "" clearing for s3
// fields with the same server contract.
func TestBuildUpdatePayload_ClearsS3PrefixAndEndpoint(t *testing.T) {
	plan := &Model{
		S3: mustS3(t, s3Model{
			BucketName:      types.StringValue("my-bucket"),
			Endpoint:        types.StringNull(),
			Prefix:          types.StringNull(),
			RefreshInterval: types.Int32Null(),
			Auth:            types.ObjectNull(s3AuthTypes),
		}),
		Git: types.ObjectNull(gitTypes),
	}
	state := &Model{
		S3: mustS3(t, s3Model{
			BucketName:      types.StringValue("my-bucket"),
			Endpoint:        types.StringValue("had-endpoint"),
			Prefix:          types.StringValue("had-prefix"),
			RefreshInterval: types.Int32Null(),
			Auth:            types.ObjectNull(s3AuthTypes),
		}),
		Git: types.ObjectNull(gitTypes),
	}
	payload, err := toUpdatePayload(context.Background(), plan, state)
	if err != nil {
		t.Fatalf("toUpdatePayload: %v", err)
	}
	s := payload.UpdateS3DagBundlePayload
	if s.Endpoint == nil || *s.Endpoint != "" {
		t.Errorf("Endpoint = %v, want pointer to \"\"", s.Endpoint)
	}
	if s.Prefix == nil || *s.Prefix != "" {
		t.Errorf("Prefix = %v, want pointer to \"\"", s.Prefix)
	}
}

// TestBuildUpdatePayload_OmitsUnsetSubdir verifies an always-unset subdir is
// omitted, not sent as "". Otherwise every update would clear it.
func TestBuildUpdatePayload_OmitsUnsetSubdir(t *testing.T) {
	plan := &Model{
		Git: mustGit(t, gitModel{
			URL:             types.StringValue("https://git.example.com/repo.git"),
			Branch:          types.StringValue("main"),
			Subdir:          types.StringNull(),
			RefreshInterval: types.Int32Null(),
			Auth:            types.ObjectNull(gitAuthTypes),
		}),
		S3: types.ObjectNull(s3Types),
	}
	state := &Model{
		Git: mustGit(t, gitModel{
			URL:             types.StringValue("https://git.example.com/repo.git"),
			Branch:          types.StringValue("main"),
			Subdir:          types.StringNull(),
			RefreshInterval: types.Int32Null(),
			Auth:            types.ObjectNull(gitAuthTypes),
		}),
		S3: types.ObjectNull(s3Types),
	}
	payload, err := toUpdatePayload(context.Background(), plan, state)
	if err != nil {
		t.Fatalf("toUpdatePayload: %v", err)
	}
	if payload.UpdateGitDagBundlePayload.Subdir != nil {
		t.Errorf("Subdir should be nil when never set, got %q", *payload.UpdateGitDagBundlePayload.Subdir)
	}
}

// TestIsEmptyStr pins the non-trivial semantic: the empty literal is treated as
// "absent" inside auth blocks (so a server-rejected empty credential surfaces at
// plan time), while Unknown defers.
func TestIsEmptyStr(t *testing.T) {
	cases := []struct {
		in   types.String
		want bool
	}{
		{types.StringNull(), true},
		{types.StringValue(""), true},
		{types.StringUnknown(), false}, // defer
		{types.StringValue("x"), false},
	}
	for _, c := range cases {
		if got := isEmptyStr(c.in); got != c.want {
			t.Errorf("isEmptyStr(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestValidateGitAuth(t *testing.T) {
	mkAuth := func(typ string, user, pw types.String) basetypes.ObjectValue {
		return mustGitAuth(t, gitAuthModel{
			Type:     types.StringValue(typ),
			Username: user,
			Password: pw,
		})
	}
	tests := []struct {
		desc    string
		auth    basetypes.ObjectValue
		wantErr bool
	}{
		{"basic with creds OK", mkAuth("basic", types.StringValue("u"), types.StringValue("p")), false},
		{"basic missing username", mkAuth("basic", types.StringNull(), types.StringValue("p")), true},
		{"basic missing password", mkAuth("basic", types.StringValue("u"), types.StringNull()), true},
		{"basic empty username literal", mkAuth("basic", types.StringValue(""), types.StringValue("p")), true},
		{"basic unknown username defers", mkAuth("basic", types.StringUnknown(), types.StringValue("p")), false},
		{"none + no creds OK", mkAuth("none", types.StringNull(), types.StringNull()), false},
		{"none + username forbidden", mkAuth("none", types.StringValue("u"), types.StringNull()), true},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var diags diag.Diagnostics
			validateGitAuth(context.Background(), tt.auth, &diags)
			if tt.wantErr && !diags.HasError() {
				t.Errorf("expected diagnostics, got none")
			}
			if !tt.wantErr && diags.HasError() {
				t.Errorf("expected no diagnostics, got %v", diags.Errors())
			}
		})
	}
}

func TestValidateS3Auth(t *testing.T) {
	mkAuth := func(typ string, id, key types.String) basetypes.ObjectValue {
		return mustS3Auth(t, s3AuthModel{
			Type:            types.StringValue(typ),
			AccessKeyID:     id,
			SecretAccessKey: key,
		})
	}
	tests := []struct {
		desc    string
		auth    basetypes.ObjectValue
		wantErr bool
	}{
		{"access_key with creds OK", mkAuth("access_key", types.StringValue("AKIA"), types.StringValue("secret")), false},
		{"access_key missing id", mkAuth("access_key", types.StringNull(), types.StringValue("secret")), true},
		{"access_key empty literal id", mkAuth("access_key", types.StringValue(""), types.StringValue("secret")), true},
		{"access_key unknown defers", mkAuth("access_key", types.StringUnknown(), types.StringValue("secret")), false},
		{"none + no creds OK", mkAuth("none", types.StringNull(), types.StringNull()), false},
		{"none + id forbidden", mkAuth("none", types.StringValue("AKIA"), types.StringNull()), true},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var diags diag.Diagnostics
			validateS3Auth(context.Background(), tt.auth, &diags)
			if tt.wantErr && !diags.HasError() {
				t.Errorf("expected diagnostics, got none")
			}
			if !tt.wantErr && diags.HasError() {
				t.Errorf("expected no diagnostics, got %v", diags.Errors())
			}
		})
	}
}

// TestValidateBundleConfig covers the top-level "exactly one of git/s3" gate.
// Per-block content checks are exercised separately by TestValidateGitAuth /
// TestValidateS3Auth above.
func TestValidateBundleConfig(t *testing.T) {
	ctx := context.Background()

	gitOK := mustGit(t, gitModel{
		URL:    types.StringValue("https://example.com/r.git"),
		Branch: types.StringValue("main"),
		Auth: mustGitAuth(t, gitAuthModel{
			Type:     types.StringValue("none"),
			Username: types.StringNull(),
			Password: types.StringNull(),
		}),
	})
	s3OK := mustS3(t, s3Model{
		BucketName: types.StringValue("b"),
		Auth: mustS3Auth(t, s3AuthModel{
			Type:            types.StringValue("none"),
			AccessKeyID:     types.StringNull(),
			SecretAccessKey: types.StringNull(),
		}),
	})

	tests := []struct {
		desc    string
		model   Model
		wantErr string // substring; "" → no error expected
	}{
		{
			desc:  "git only OK",
			model: Model{Git: gitOK, S3: types.ObjectNull(s3Types)},
		},
		{
			desc:  "s3 only OK",
			model: Model{Git: types.ObjectNull(gitTypes), S3: s3OK},
		},
		{
			desc:    "neither set → error",
			model:   Model{Git: types.ObjectNull(gitTypes), S3: types.ObjectNull(s3Types)},
			wantErr: "Exactly one of `git` or `s3` must be set",
		},
		{
			desc:    "both set → error",
			model:   Model{Git: gitOK, S3: s3OK},
			wantErr: "Only one of `git` or `s3` may be set",
		},
		{
			desc:  "git unknown defers (no error)",
			model: Model{Git: types.ObjectUnknown(gitTypes), S3: types.ObjectNull(s3Types)},
		},
		{
			desc:  "s3 unknown defers (no error)",
			model: Model{Git: types.ObjectNull(gitTypes), S3: types.ObjectUnknown(s3Types)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var diags diag.Diagnostics
			validateBundleConfig(ctx, &tt.model, &diags)
			if tt.wantErr == "" {
				if diags.HasError() {
					t.Fatalf("expected no errors, got: %v", diags.Errors())
				}
				return
			}
			if !diags.HasError() {
				t.Fatalf("expected an error containing %q, got none", tt.wantErr)
			}
			found := false
			for _, d := range diags.Errors() {
				if strings.Contains(d.Detail(), tt.wantErr) {
					found = true
					break
				}
			}
			if !found {
				msgs := make([]string, 0, len(diags.Errors()))
				for _, d := range diags.Errors() {
					msgs = append(msgs, d.Detail())
				}
				t.Errorf("expected an error containing %q, got: %v", tt.wantErr, msgs)
			}
		})
	}
}
