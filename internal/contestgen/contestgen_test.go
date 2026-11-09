package contestgen

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/sudoapi"
)

func TestRun(t *testing.T) {
	cfg, err := Parse([]byte(`{"language":"ro","write_bio":true,"teams":true,"profiles":[
		{"name":"Echipa Ștefan","contest_id":7,"member_names":["Pop Ion","Ionescu Ana"]},
		{"slug":"anon","contest_id":7,"member_names":["Pop Ion","Ionescu Ana"]},
		{"name":"Done","password":"x"},
		{"name":"NoContest"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profiles[0].Slug != "Echipa_Stefan" {
		t.Fatalf("slug = %q", cfg.Profiles[0].Slug)
	}
	backend := Backend{Generate: func(_ context.Context, req sudoapi.UserGenerationRequest) (string, error) {
		return "pw-" + req.Name, nil
	}}
	quiet := slog.New(slog.DiscardHandler)

	t.Run("dry run returns the requests", func(t *testing.T) {
		res, err := Run(t.Context(), cfg, backend, quiet, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Requests) != 2 || !strings.Contains(res.Requests[0].Bio, "Contestants: Pop Ion, Ionescu Ana") {
			t.Fatalf("requests = %+v", res.Requests)
		}
		// Member surnames are only the fallback for a nameless team.
		if *res.Requests[0].DisplayName != "Echipa Ștefan" || *res.Requests[1].DisplayName != "Pop, Ionescu" {
			t.Fatalf("requests = %+v", res.Requests)
		}
		files, _ := res.Files()
		if _, ok := files["requestsOut.json"]; !ok || len(files) != 1 {
			t.Fatalf("files = %v", files)
		}
	})

	t.Run("real run writes passwords into the csv", func(t *testing.T) {
		res, err := Run(t.Context(), cfg, backend, quiet, false)
		if err != nil {
			t.Fatal(err)
		}
		files, _ := res.Files()
		got := string(files["teamsOut.csv"])
		want := "Team Name,Members,Contact Email,Username,Password\nEchipa Ștefan,\"Pop Ion, Ionescu Ana\",email_Echipa_Stefan@kilonova.ro,Echipa_Stefan,pw-Echipa_Stefan\n,\"Pop Ion, Ionescu Ana\",email_anon@kilonova.ro,anon,pw-anon\n"
		if got != want {
			t.Fatalf("csv =\n%s\nwant\n%s", got, want)
		}
	})
}

func TestParseRejectsBadLanguage(t *testing.T) {
	if _, err := Parse([]byte(`{"language":"fr"}`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunRecordsFallbackContestID(t *testing.T) {
	cfg, err := Parse([]byte(`{"language":"en","contest_id":3,"profiles":[{"name":"aaa"},{"name":"bbb","contest_id":9}]}`))
	if err != nil {
		t.Fatal(err)
	}
	var sent []int
	backend := Backend{
		CheckContest: func(context.Context, int) error { return nil },
		Generate: func(_ context.Context, req sudoapi.UserGenerationRequest) (string, error) {
			sent = append(sent, *req.ContestID)
			return "pw", nil
		},
	}
	res, err := Run(t.Context(), cfg, backend, slog.New(slog.DiscardHandler), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sent) != 2 || sent[0] != 3 || sent[1] != 9 || res.Teams[0].ContestID != 3 || res.Teams[1].ContestID != 9 {
		t.Fatalf("sent = %v, teams = %+v", sent, res.Teams)
	}
}
