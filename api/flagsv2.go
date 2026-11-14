package api

import (
	"context"
	"encoding/json"
	"maps"
	"slices"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/danielgtaylor/huma/v2"
)

// flagExportVersion is the envelope format version; imports of any other version are refused.
const flagExportVersion = 1

type flagEnvelope struct {
	Version int                        `json:"version" doc:"Format version, currently 1"`
	Flags   map[string]json.RawMessage `json:"flags" doc:"Flag values keyed by internal name"`
}

type flagUpdateBody struct {
	BoolFlags    map[string]bool     `json:"bool_flags,omitempty"`
	StringFlags  map[string]string   `json:"string_flags,omitempty"`
	IntFlags     map[string]int      `json:"int_flags,omitempty"`
	IntListFlags map[string][]int    `json:"int_array_flags,omitempty"`
	StrListFlags map[string][]string `json:"str_array_flags,omitempty"`
}

type flagUpdateResult struct {
	Updated []string `json:"updated"`
	// Errors names flags that were unknown, had the wrong type, or failed to persist.
	Errors []string `json:"errors"`
}

// registerFlagOps adds the flag update, export and import operations. They read the
// instance's flagset from the request context, so callers must sit behind the
// instance middleware; access control is the router's job (MustBeAdminV2).
func registerFlagOps(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "update-flags",
		Method:      "PATCH",
		Path:        "/flags",
		Summary:     "Update runtime flags",
		Security:    []map[string][]string{},
	}, updateFlags)

	huma.Register(api, huma.Operation{
		OperationID: "export-flags",
		Method:      "GET",
		Path:        "/flags/export",
		Summary:     "Export the flags set on this instance",
		Security:    []map[string][]string{},
	}, exportFlags)

	huma.Register(api, huma.Operation{
		OperationID:  "import-flags",
		Method:       "POST",
		Path:         "/flags/import",
		Summary:      "Import flags exported from another instance",
		MaxBodyBytes: 1 << 20,
		Security:     []map[string][]string{},
	}, importFlags)
}

func flagsetOrError(ctx context.Context) (*config.Flagset, error) {
	if fs := config.FlagsetFromContext(ctx); fs != nil {
		return fs, nil
	}
	return nil, huma.Error500InternalServerError("no flagset for this request")
}

func updateFlags(ctx context.Context, args *struct{ Body flagUpdateBody }) (*struct{ Body flagUpdateResult }, error) {
	if _, err := flagsetOrError(ctx); err != nil {
		return nil, err
	}
	res := flagUpdateResult{Updated: []string{}, Errors: []string{}}
	updateFlagMap(ctx, args.Body.BoolFlags, &res)
	updateFlagMap(ctx, args.Body.StringFlags, &res)
	updateFlagMap(ctx, args.Body.IntFlags, &res)
	updateFlagMap(ctx, args.Body.IntListFlags, &res)
	updateFlagMap(ctx, args.Body.StrListFlags, &res)
	return &struct{ Body flagUpdateResult }{res}, nil
}

// updateFlagMap applies every flag in vals, recording unknown or mistyped names and
// persistence failures in res instead of stopping.
func updateFlagMap[T any](ctx context.Context, vals map[string]T, res *flagUpdateResult) {
	for _, name := range slices.Sorted(maps.Keys(vals)) {
		flg, ok := config.GetFlag[T](name)
		if !ok {
			res.Errors = append(res.Errors, name+": unknown flag or wrong type")
			continue
		}
		if err := flg.Update(ctx, vals[name]); err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		res.Updated = append(res.Updated, name)
	}
}

func exportFlags(ctx context.Context, args *struct {
	Secrets bool `query:"secrets" doc:"Include flags marked secret (OIDC keys, webhook secrets)"`
}) (*struct{ Body flagEnvelope }, error) {
	fs, err := flagsetOrError(ctx)
	if err != nil {
		return nil, err
	}
	vals, err := fs.Export(args.Secrets)
	if err != nil {
		return nil, huma.Error500InternalServerError("couldn't export flags", err)
	}
	return &struct{ Body flagEnvelope }{flagEnvelope{Version: flagExportVersion, Flags: vals}}, nil
}

func importFlags(ctx context.Context, args *struct{ Body flagEnvelope }) (*struct{ Body config.ImportResult }, error) {
	fs, err := flagsetOrError(ctx)
	if err != nil {
		return nil, err
	}
	if args.Body.Version != flagExportVersion {
		return nil, huma.Error400BadRequest("unsupported flag export version")
	}
	return &struct{ Body config.ImportResult }{fs.Import(ctx, args.Body.Flags)}, nil
}
