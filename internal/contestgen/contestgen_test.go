package contestgen

import (
	"context"
	"log/slog"
	"testing"

	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	cfg, err := Parse([]byte(`{"language":"ro","write_bio":true,"teams":true,"profiles":[
		{"name":"Echipa Ștefan","contest_id":7,"member_names":["Pop Ion","Ionescu Ana"]},
		{"slug":"anon","contest_id":7,"member_names":["Pop Ion","Ionescu Ana"]},
		{"name":"Done","password":"x"},
		{"name":"NoContest"}]}`))
	require.NoError(t, err)
	require.Equal(t, "Echipa_Stefan", cfg.Profiles[0].Slug)
	backend := Backend{Generate: func(_ context.Context, req sudoapi.UserGenerationRequest) (string, error) {
		return "pw-" + req.Name, nil
	}}
	quiet := slog.New(slog.DiscardHandler)

	t.Run("dry run returns the requests", func(t *testing.T) {
		res, err := Run(t.Context(), cfg, backend, quiet, true)
		require.NoError(t, err)
		require.Len(t, res.Requests, 2)
		require.Contains(t, res.Requests[0].Bio, "Contestants: Pop Ion, Ionescu Ana")
		// Member surnames are only the fallback for a nameless team.
		require.Equal(t, "Echipa Ștefan", *res.Requests[0].DisplayName)
		require.Equal(t, "Pop, Ionescu", *res.Requests[1].DisplayName)
		files, _ := res.Files()
		require.Len(t, files, 1)
		require.Contains(t, files, "requestsOut.json")
	})

	t.Run("real run writes passwords into the csv", func(t *testing.T) {
		res, err := Run(t.Context(), cfg, backend, quiet, false)
		require.NoError(t, err)
		files, _ := res.Files()
		got := string(files["teamsOut.csv"])
		want := "Team Name,Members,Contact Email,Username,Password\nEchipa Ștefan,\"Pop Ion, Ionescu Ana\",email_Echipa_Stefan@kilonova.ro,Echipa_Stefan,pw-Echipa_Stefan\n,\"Pop Ion, Ionescu Ana\",email_anon@kilonova.ro,anon,pw-anon\n"
		require.Equal(t, want, got)
	})
}

func TestParseRejectsBadLanguage(t *testing.T) {
	_, err := Parse([]byte(`{"language":"fr"}`))
	require.Error(t, err)
}

func TestRunRecordsFallbackContestID(t *testing.T) {
	cfg, err := Parse([]byte(`{"language":"en","contest_id":3,"profiles":[{"name":"aaa"},{"name":"bbb","contest_id":9}]}`))
	require.NoError(t, err)
	var sent []int
	backend := Backend{
		CheckContest: func(context.Context, int) error { return nil },
		Generate: func(_ context.Context, req sudoapi.UserGenerationRequest) (string, error) {
			sent = append(sent, *req.ContestID)
			return "pw", nil
		},
	}
	res, err := Run(t.Context(), cfg, backend, slog.New(slog.DiscardHandler), false)
	require.NoError(t, err)
	require.Equal(t, []int{3, 9}, sent)
	require.Len(t, res.Teams, 2)
	require.Equal(t, 3, res.Teams[0].ContestID)
	require.Equal(t, 9, res.Teams[1].ContestID)
}
