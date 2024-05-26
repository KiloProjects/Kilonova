package tasks

import (
	"context"
	"maps"
	"slices"

	"github.com/KiloProjects/kilonova/eval"
)

func VersionTask(ctx context.Context, mgr eval.BoxScheduler, lang eval.Language) (string, error) {
	resp, err := mgr.RunBox2(ctx, &eval.Box2Request{
		RunConfig: &eval.RunConfig{
			EnvToSet:    maps.Clone(lang.BuildEnv),
			Directories: slices.Clone(lang.Mounts),

			WallTimeLimit:  5, // seconds
			OutputPath:     "/box/version.out",
			StderrToStdout: true,
		},

		OutputByteFiles: []string{"/box/version.out"},

		Command: slices.Clone(lang.VersionCommand),
	}, 0)
	if err != nil {
		return "", err
	}

	data, ok := resp.ByteFiles["/box/version.out"]
	if !ok {
		return "???", nil
	}

	s := string(data)
	if lang.VersionParser != nil {
		s = lang.VersionParser(s)
	}

	return s, nil
}
