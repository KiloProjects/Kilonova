package tasks

import (
	"context"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
)

func VersionTask(ctx context.Context, mgr eval.BoxScheduler, lang language.GraderLang) (string, error) {
	resp, err := mgr.RunBox2(ctx, &eval.Box2Request{
		RunConfig: &eval.RunConfig{
			EnvToSet:    lang.BuildEnv(),
			Directories: lang.Mounts(),

			WallTimeLimit:  5, // seconds
			OutputPath:     "/box/version.out",
			StderrToStdout: true,
		},

		OutputByteFiles: []string{"/box/version.out"},

		Command: lang.VersionCommand(),
	}, 0)
	if err != nil {
		return "", err
	}

	data, ok := resp.ByteFiles["/box/version.out"]
	if !ok {
		return "???", nil
	}

	return lang.ParseVersion(data), nil
}
