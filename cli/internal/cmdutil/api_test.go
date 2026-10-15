package cmdutil

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/auth"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newAPITestFactory(t *testing.T, vars map[string]string, rt roundTripFunc) *Factory {
	t.Helper()

	ios, _, _, _ := iostreams.Test()
	vars["XDG_CONFIG_HOME"] = t.TempDir()
	vars[auth.EnvTokenStore] = auth.StoreFile

	return &Factory{
		IOStreams:  ios,
		Localizer:  i18n.NewLocalizer(i18n.LangJa),
		Version:    "1.2.3",
		Getenv:     func(key string) string { return vars[key] },
		HTTPClient: &http.Client{Transport: rt},
	}
}

func newAPITestCommand(f *Factory, args ...string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	config.AddFlags(cmd.Flags(), f.Localizer)
	_ = cmd.Flags().Parse(args)

	return cmd
}

func TestAPIClient_SavedCredential(t *testing.T) {
	t.Parallel()

	var gotAuth, gotPath string
	f := newAPITestFactory(t, map[string]string{}, func(req *http.Request) (*http.Response, error) {
		gotAuth, gotPath = req.Header.Get("Authorization"), req.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"11111111-1111-1111-1111-111111111111","atname":"alice","name":"Alice"}`))),
		}, nil
	})

	// ログインした状態を、設定と0600のファイルのトークンで用意する。
	dir, err := f.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	cred := config.Credential{Host: "https://wikino.example.com", Space: "team", MemberID: "11111111-1111-1111-1111-111111111111", MemberAtname: "alice"}
	store, err := f.TokenStore(dir).Save(&cred, auth.Secret{AccessToken: "wikino_pat_saved-secret"})
	if err != nil {
		t.Fatal(err)
	}
	cred.Store = store
	cfg := &config.Config{}
	cfg.AddCredential(cred)
	if err := cfg.Save(dir); err != nil {
		t.Fatal(err)
	}

	client, space, err := f.APIClient(newAPITestCommand(f))
	if err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if space != "team" {
		t.Errorf("スペース = %q、期待値 = team", space)
	}
	if _, err := client.GetCurrentSpaceMemberWithResponse(t.Context(), space); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer wikino_pat_saved-secret" || gotPath != "/api/v1/spaces/team/members/me" {
		t.Errorf("Authorization = %q、パス = %q", gotAuth, gotPath)
	}
}

func TestAPIClient_NotLoggedIn(t *testing.T) {
	t.Parallel()

	f := newAPITestFactory(t, map[string]string{}, func(req *http.Request) (*http.Response, error) {
		t.Errorf("要求を送らないことを期待したが送った: %s", req.URL)
		return nil, errors.New("送らない")
	})

	_, _, err := f.APIClient(newAPITestCommand(f))

	var cliErr *output.Error
	if !errors.As(err, &cliErr) || cliErr.Code != output.CodeNotLoggedIn || cliErr.ExitCode != output.ExitAuth {
		t.Errorf("未ログインのエラーを期待したが %v だった", err)
	}
}
