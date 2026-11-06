package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/urfave/cli/v3"
)

// contestUtilsRemote is contest-utils without a database: it creates accounts through a running instance's API.
var contestUtilsRemote = &cli.Command{
	Name:  "contest-utils-remote",
	Usage: "Like contest-utils, but goes through the API of a running instance instead of the database",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "url",
			Usage:    "Base URL of the Kilonova instance (e.g. https://kilonova.ro)",
			Sources:  cli.EnvVars("KN_REMOTE_URL"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     "api-key",
			Usage:    "Session ID of an admin account (the kn-sessionid cookie)",
			Sources:  cli.EnvVars("KN_API_KEY"),
			Required: true,
		},
	}, contestUtils.Flags...),
	Action: func(ctx context.Context, command *cli.Command) error {
		base, err := url.Parse(strings.TrimSuffix(command.String("url"), "/"))
		if err != nil {
			return fmt.Errorf("invalid url: %w", err)
		}
		key := command.String("api-key")

		do := func(ctx context.Context, method, path string, body any, out any) error {
			var rd io.Reader
			if body != nil {
				b, err := json.Marshal(body)
				if err != nil {
					return err
				}
				rd = bytes.NewReader(b)
			}
			req, err := http.NewRequestWithContext(ctx, method, base.JoinPath(path).String(), rd)
			if err != nil {
				return err
			}
			// v1 reads Authorization, v2 reads X-Api-Key.
			req.Header.Set("Authorization", key)
			req.Header.Set("X-Api-Key", key)
			if method != http.MethodGet {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if resp.StatusCode/100 != 2 {
				return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(data)))
			}
			if out == nil {
				return nil
			}
			return json.Unmarshal(data, out)
		}

		return runContestUtils(ctx, command, contestBackend{
			checkContest: func(ctx context.Context, id int) error {
				return do(ctx, http.MethodGet, "/api/contest/"+strconv.Itoa(id)+"/", nil, nil)
			},
			generate: func(ctx context.Context, req sudoapi.UserGenerationRequest) (string, error) {
				var out struct {
					Password string `json:"password"`
				}
				if err := do(ctx, http.MethodPost, "/api/admin/v2/generateUser", req, &out); err != nil {
					return "", err
				}
				return out.Password, nil
			},
		})
	},
}
