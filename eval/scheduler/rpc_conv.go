package scheduler

import (
	"io/fs"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
	graderv1 "github.com/KiloProjects/kilonova/eval/scheduler/proto/kilonova/grader/v1"
)

// This file maps between the authoritative eval.* structs and the generated
// grader.v1 proto messages. eval.* stays the source of truth; proto is wire-only.

func dirToProto(d language.Directory) *graderv1.Directory {
	return &graderv1.Directory{
		In: d.In, Out: d.Out, Opts: d.Opts, Removes: d.Removes, Verbatim: d.Verbatim,
	}
}

func dirFromProto(d *graderv1.Directory) language.Directory {
	return language.Directory{
		In: d.GetIn(), Out: d.GetOut(), Opts: d.GetOpts(),
		Removes: d.GetRemoves(), Verbatim: d.GetVerbatim(),
	}
}

func runConfigToProto(c *eval.RunConfig) *graderv1.RunConfig {
	if c == nil {
		return nil
	}
	dirs := make([]*graderv1.Directory, len(c.Directories))
	for i := range c.Directories {
		dirs[i] = dirToProto(c.Directories[i])
	}
	return &graderv1.RunConfig{
		StderrToStdout: c.StderrToStdout,
		InputPath:      c.InputPath,
		OutputPath:     c.OutputPath,
		StderrPath:     c.StderrPath,
		MemoryLimit:    int32(c.MemoryLimit),
		TimeLimit:      c.TimeLimit,
		WallTimeLimit:  c.WallTimeLimit,
		InheritEnv:     c.InheritEnv,
		EnvToInherit:   c.EnvToInherit,
		EnvToSet:       c.EnvToSet,
		EnableInternet: c.EnableInternet,
		Directories:    dirs,
	}
}

func runConfigFromProto(c *graderv1.RunConfig) *eval.RunConfig {
	if c == nil {
		return nil
	}
	dirs := make([]language.Directory, len(c.GetDirectories()))
	for i, d := range c.GetDirectories() {
		dirs[i] = dirFromProto(d)
	}
	return &eval.RunConfig{
		StderrToStdout: c.GetStderrToStdout(),
		InputPath:      c.GetInputPath(),
		OutputPath:     c.GetOutputPath(),
		StderrPath:     c.GetStderrPath(),
		MemoryLimit:    int(c.GetMemoryLimit()),
		TimeLimit:      c.GetTimeLimit(),
		WallTimeLimit:  c.GetWallTimeLimit(),
		InheritEnv:     c.GetInheritEnv(),
		EnvToInherit:   c.GetEnvToInherit(),
		EnvToSet:       c.GetEnvToSet(),
		EnableInternet: c.GetEnableInternet(),
		Directories:    dirs,
	}
}

func statsToProto(s *eval.RunStats) *graderv1.RunStats {
	if s == nil {
		return nil
	}
	return &graderv1.RunStats{
		Memory:              int32(s.Memory),
		ExitCode:            int32(s.ExitCode),
		ExitSignal:          int32(s.ExitSignal),
		Killed:              s.Killed,
		Message:             s.Message,
		Status:              s.Status,
		Time:                s.Time,
		InternalMessage:     s.InternalMessage,
		MemoryLimitExceeded: s.MemoryLimitExceeded,
	}
}

func statsFromProto(s *graderv1.RunStats) *eval.RunStats {
	if s == nil {
		return nil
	}
	return &eval.RunStats{
		Memory:              int(s.GetMemory()),
		ExitCode:            int(s.GetExitCode()),
		ExitSignal:          int(s.GetExitSignal()),
		Killed:              s.GetKilled(),
		Message:             s.GetMessage(),
		Status:              s.GetStatus(),
		Time:                s.GetTime(),
		InternalMessage:     s.GetInternalMessage(),
		MemoryLimitExceeded: s.GetMemoryLimitExceeded(),
	}
}

func scratchFileToProto(f eval.ScratchFile) *graderv1.ScratchFile {
	return &graderv1.ScratchFile{
		Identifier: f.Identifier, BoxPath: f.BoxPath, Mode: uint32(f.Mode),
	}
}

func scratchFileFromProto(f *graderv1.ScratchFile) eval.ScratchFile {
	return eval.ScratchFile{
		Identifier: f.GetIdentifier(), BoxPath: f.GetBoxPath(), Mode: fs.FileMode(f.GetMode()),
	}
}

func box3RequestToProto(r *eval.Box3Request) *graderv1.Box3Request {
	if r == nil {
		return nil
	}
	files := make([]*graderv1.ScratchFile, len(r.InputFiles))
	for i := range r.InputFiles {
		files[i] = scratchFileToProto(r.InputFiles[i])
	}
	return &graderv1.Box3Request{
		InputFiles:      files,
		Command:         r.Command,
		RunConfig:       runConfigToProto(r.RunConfig),
		OutputFilePaths: r.OutputFilePaths,
	}
}

func box3RequestFromProto(r *graderv1.Box3Request) *eval.Box3Request {
	if r == nil {
		return nil
	}
	files := make([]eval.ScratchFile, len(r.GetInputFiles()))
	for i, f := range r.GetInputFiles() {
		files[i] = scratchFileFromProto(f)
	}
	return &eval.Box3Request{
		InputFiles:      files,
		Command:         r.GetCommand(),
		RunConfig:       runConfigFromProto(r.GetRunConfig()),
		OutputFilePaths: r.GetOutputFilePaths(),
	}
}

func box3ResponseToProto(r *eval.Box3Response) *graderv1.Box3Response {
	if r == nil {
		return nil
	}
	return &graderv1.Box3Response{Stats: statsToProto(r.Stats), Files: r.Files}
}

func box3ResponseFromProto(r *graderv1.Box3Response) *eval.Box3Response {
	if r == nil {
		return nil
	}
	return &eval.Box3Response{Stats: statsFromProto(r.GetStats()), Files: r.GetFiles()}
}

func multibox3RequestToProto(r *eval.Multibox3Request) *graderv1.Multibox3Request {
	if r == nil {
		return nil
	}
	users := make([]*graderv1.Box3Request, len(r.UserSandboxConfigs))
	for i := range r.UserSandboxConfigs {
		users[i] = box3RequestToProto(r.UserSandboxConfigs[i])
	}
	return &graderv1.Multibox3Request{
		ManagerSandbox:     box3RequestToProto(r.ManagerSandbox),
		UserSandboxConfigs: users,
		UseStdin:           r.UseStdin,
	}
}

func multibox3RequestFromProto(r *graderv1.Multibox3Request) *eval.Multibox3Request {
	if r == nil {
		return nil
	}
	users := make([]*eval.Box3Request, len(r.GetUserSandboxConfigs()))
	for i, u := range r.GetUserSandboxConfigs() {
		users[i] = box3RequestFromProto(u)
	}
	return &eval.Multibox3Request{
		ManagerSandbox:     box3RequestFromProto(r.GetManagerSandbox()),
		UserSandboxConfigs: users,
		UseStdin:           r.GetUseStdin(),
	}
}

func statsSliceToProto(stats []*eval.RunStats) []*graderv1.RunStats {
	out := make([]*graderv1.RunStats, len(stats))
	for i := range stats {
		out[i] = statsToProto(stats[i])
	}
	return out
}

func statsSliceFromProto(stats []*graderv1.RunStats) []*eval.RunStats {
	out := make([]*eval.RunStats, len(stats))
	for i, s := range stats {
		out[i] = statsFromProto(s)
	}
	return out
}
