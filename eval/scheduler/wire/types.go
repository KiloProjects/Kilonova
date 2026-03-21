// Package wire defines JSON-serializable types used to communicate between the
// knbox service and the monolith's remote BoxScheduler client.
//
// These types mirror the proto schema in eval/scheduler/proto/scheduler.proto
// and are intended to be replaced by generated protobuf code once buf becomes
// available in the build environment.
package wire

import (
	"io/fs"

	"github.com/KiloProjects/kilonova/domain/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
)

// BucketFile references a file in a named storage bucket on the shared filesystem.
type BucketFile struct {
	BucketType string `json:"bucket_type"`
	Filename   string `json:"filename"`
	Mode       uint32 `json:"mode"`
}

func ToBucketFile(bf *eval.BucketFile) *BucketFile {
	if bf == nil {
		return nil
	}
	return &BucketFile{
		BucketType: string(bf.Bucket),
		Filename:   bf.Filename,
		Mode:       uint32(bf.Mode),
	}
}

func FromBucketFile(wbf *BucketFile) *eval.BucketFile {
	if wbf == nil {
		return nil
	}
	return &eval.BucketFile{
		Bucket:   datastore.BucketType(wbf.BucketType),
		Filename: wbf.Filename,
		Mode:     fs.FileMode(wbf.Mode),
	}
}

// ByteFile is an inline file payload.
type ByteFile struct {
	Data []byte `json:"data"`
	Mode uint32 `json:"mode"`
}

func ToByteFile(bf *eval.ByteFile) *ByteFile {
	if bf == nil {
		return nil
	}
	return &ByteFile{Data: bf.Data, Mode: uint32(bf.Mode)}
}

func FromByteFile(wbf *ByteFile) *eval.ByteFile {
	if wbf == nil {
		return nil
	}
	return &eval.ByteFile{Data: wbf.Data, Mode: fs.FileMode(wbf.Mode)}
}

// Directory is a filesystem mount rule for the sandbox.
type Directory struct {
	In       string `json:"in"`
	Out      string `json:"out,omitempty"`
	Opts     string `json:"opts,omitempty"`
	Removes  bool   `json:"removes,omitempty"`
	Verbatim bool   `json:"verbatim,omitempty"`
}

func ToDirectory(d language.Directory) Directory {
	return Directory{In: d.In, Out: d.Out, Opts: d.Opts, Removes: d.Removes, Verbatim: d.Verbatim}
}

func FromDirectory(d Directory) language.Directory {
	return language.Directory{In: d.In, Out: d.Out, Opts: d.Opts, Removes: d.Removes, Verbatim: d.Verbatim}
}

// RunConfig controls the sandbox execution environment.
type RunConfig struct {
	StderrToStdout bool              `json:"stderr_to_stdout,omitempty"`
	InputPath      string            `json:"input_path,omitempty"`
	OutputPath     string            `json:"output_path,omitempty"`
	StderrPath     string            `json:"stderr_path,omitempty"`
	MemoryLimit    int               `json:"memory_limit,omitempty"`
	TimeLimit      float64           `json:"time_limit,omitempty"`
	WallTimeLimit  float64           `json:"wall_time_limit,omitempty"`
	InheritEnv     bool              `json:"inherit_env,omitempty"`
	EnvToInherit   []string          `json:"env_to_inherit,omitempty"`
	EnvToSet       map[string]string `json:"env_to_set,omitempty"`
	EnableInternet bool              `json:"enable_internet,omitempty"`
	Directories    []Directory       `json:"directories,omitempty"`
}

func ToRunConfig(rc *eval.RunConfig) *RunConfig {
	if rc == nil {
		return nil
	}
	dirs := make([]Directory, len(rc.Directories))
	for i, d := range rc.Directories {
		dirs[i] = ToDirectory(d)
	}
	return &RunConfig{
		StderrToStdout: rc.StderrToStdout,
		InputPath:      rc.InputPath,
		OutputPath:     rc.OutputPath,
		StderrPath:     rc.StderrPath,
		MemoryLimit:    rc.MemoryLimit,
		TimeLimit:      rc.TimeLimit,
		WallTimeLimit:  rc.WallTimeLimit,
		InheritEnv:     rc.InheritEnv,
		EnvToInherit:   rc.EnvToInherit,
		EnvToSet:       rc.EnvToSet,
		EnableInternet: rc.EnableInternet,
		Directories:    dirs,
	}
}

func FromRunConfig(rc *RunConfig) *eval.RunConfig {
	if rc == nil {
		return nil
	}
	dirs := make([]language.Directory, len(rc.Directories))
	for i, d := range rc.Directories {
		dirs[i] = FromDirectory(d)
	}
	return &eval.RunConfig{
		StderrToStdout: rc.StderrToStdout,
		InputPath:      rc.InputPath,
		OutputPath:     rc.OutputPath,
		StderrPath:     rc.StderrPath,
		MemoryLimit:    rc.MemoryLimit,
		TimeLimit:      rc.TimeLimit,
		WallTimeLimit:  rc.WallTimeLimit,
		InheritEnv:     rc.InheritEnv,
		EnvToInherit:   rc.EnvToInherit,
		EnvToSet:       rc.EnvToSet,
		EnableInternet: rc.EnableInternet,
		Directories:    dirs,
	}
}

// RunStats contains the execution results from a sandbox run.
type RunStats struct {
	Memory              int     `json:"memory"`
	ExitCode            int     `json:"exit_code"`
	ExitSignal          int     `json:"exit_signal"`
	Killed              bool    `json:"killed,omitempty"`
	Message             string  `json:"message,omitempty"`
	Status              string  `json:"status,omitempty"`
	Time                float64 `json:"time"`
	InternalMessage     string  `json:"internal_message,omitempty"`
	MemoryLimitExceeded bool    `json:"memory_limit_exceeded,omitempty"`
}

func ToRunStats(rs *eval.RunStats) *RunStats {
	if rs == nil {
		return nil
	}
	return &RunStats{
		Memory:              rs.Memory,
		ExitCode:            rs.ExitCode,
		ExitSignal:          rs.ExitSignal,
		Killed:              rs.Killed,
		Message:             rs.Message,
		Status:              rs.Status,
		Time:                rs.Time,
		InternalMessage:     rs.InternalMessage,
		MemoryLimitExceeded: rs.MemoryLimitExceeded,
	}
}

func FromRunStats(rs *RunStats) *eval.RunStats {
	if rs == nil {
		return nil
	}
	return &eval.RunStats{
		Memory:              rs.Memory,
		ExitCode:            rs.ExitCode,
		ExitSignal:          rs.ExitSignal,
		Killed:              rs.Killed,
		Message:             rs.Message,
		Status:              rs.Status,
		Time:                rs.Time,
		InternalMessage:     rs.InternalMessage,
		MemoryLimitExceeded: rs.MemoryLimitExceeded,
	}
}

// Box2Request describes a single sandbox execution request.
type Box2Request struct {
	InputBucketFiles  map[string]*BucketFile `json:"input_bucket_files,omitempty"`
	InputByteFiles    map[string]*ByteFile   `json:"input_byte_files,omitempty"`
	Command           []string               `json:"command"`
	RunConfig         *RunConfig             `json:"run_config,omitempty"`
	OutputByteFiles   []string               `json:"output_byte_files,omitempty"`
	OutputBucketFiles map[string]*BucketFile `json:"output_bucket_files,omitempty"`
}

func ToBox2Request(req *eval.Box2Request) *Box2Request {
	if req == nil {
		return nil
	}
	w := &Box2Request{
		Command:         req.Command,
		RunConfig:       ToRunConfig(req.RunConfig),
		OutputByteFiles: req.OutputByteFiles,
	}
	if len(req.InputBucketFiles) > 0 {
		w.InputBucketFiles = make(map[string]*BucketFile, len(req.InputBucketFiles))
		for k, v := range req.InputBucketFiles {
			w.InputBucketFiles[k] = ToBucketFile(v)
		}
	}
	if len(req.InputByteFiles) > 0 {
		w.InputByteFiles = make(map[string]*ByteFile, len(req.InputByteFiles))
		for k, v := range req.InputByteFiles {
			w.InputByteFiles[k] = ToByteFile(v)
		}
	}
	if len(req.OutputBucketFiles) > 0 {
		w.OutputBucketFiles = make(map[string]*BucketFile, len(req.OutputBucketFiles))
		for k, v := range req.OutputBucketFiles {
			w.OutputBucketFiles[k] = ToBucketFile(v)
		}
	}
	return w
}

func FromBox2Request(req *Box2Request) *eval.Box2Request {
	if req == nil {
		return nil
	}
	r := &eval.Box2Request{
		Command:         req.Command,
		RunConfig:       FromRunConfig(req.RunConfig),
		OutputByteFiles: req.OutputByteFiles,
	}
	if len(req.InputBucketFiles) > 0 {
		r.InputBucketFiles = make(map[string]*eval.BucketFile, len(req.InputBucketFiles))
		for k, v := range req.InputBucketFiles {
			r.InputBucketFiles[k] = FromBucketFile(v)
		}
	}
	if len(req.InputByteFiles) > 0 {
		r.InputByteFiles = make(map[string]*eval.ByteFile, len(req.InputByteFiles))
		for k, v := range req.InputByteFiles {
			r.InputByteFiles[k] = FromByteFile(v)
		}
	}
	if len(req.OutputBucketFiles) > 0 {
		r.OutputBucketFiles = make(map[string]*eval.BucketFile, len(req.OutputBucketFiles))
		for k, v := range req.OutputBucketFiles {
			r.OutputBucketFiles[k] = FromBucketFile(v)
		}
	}
	return r
}

// Box2Response contains the results of a single sandbox execution.
type Box2Response struct {
	Stats       *RunStats              `json:"stats,omitempty"`
	ByteFiles   map[string][]byte      `json:"byte_files,omitempty"`
	BucketFiles map[string]*BucketFile `json:"bucket_files,omitempty"`
}

func ToBox2Response(resp *eval.Box2Response) *Box2Response {
	if resp == nil {
		return nil
	}
	w := &Box2Response{
		Stats:     ToRunStats(resp.Stats),
		ByteFiles: resp.ByteFiles,
	}
	if len(resp.BucketFiles) > 0 {
		w.BucketFiles = make(map[string]*BucketFile, len(resp.BucketFiles))
		for k, v := range resp.BucketFiles {
			w.BucketFiles[k] = ToBucketFile(v)
		}
	}
	return w
}

func FromBox2Response(resp *Box2Response) *eval.Box2Response {
	if resp == nil {
		return nil
	}
	r := &eval.Box2Response{
		Stats:     FromRunStats(resp.Stats),
		ByteFiles: resp.ByteFiles,
	}
	if len(resp.BucketFiles) > 0 {
		r.BucketFiles = make(map[string]*eval.BucketFile, len(resp.BucketFiles))
		for k, v := range resp.BucketFiles {
			r.BucketFiles[k] = FromBucketFile(v)
		}
	}
	return r
}

// RunBoxRequest is the body for the RunBox endpoint.
type RunBoxRequest struct {
	Request  *Box2Request `json:"request"`
	MemQuota int64        `json:"mem_quota"`
}

// RunBoxResponse is the response body for the RunBox endpoint.
type RunBoxResponse struct {
	Response *Box2Response `json:"response"`
}

// RunMultiboxRequest is the body for the RunMultibox endpoint.
type RunMultiboxRequest struct {
	ManagerSandbox     *Box2Request   `json:"manager_sandbox"`
	UserSandboxConfigs []*Box2Request `json:"user_sandbox_configs"`
	UseStdin           bool           `json:"use_stdin,omitempty"`
	ManagerMemQuota    int64          `json:"manager_mem_quota"`
	IndividualMemQuota int64          `json:"individual_mem_quota"`
}

func ToRunMultiboxRequest(req *eval.MultiboxRequest, managerMemQuota, individualMemQuota int64) *RunMultiboxRequest {
	userConfigs := make([]*Box2Request, len(req.UserSandboxConfigs))
	for i, uc := range req.UserSandboxConfigs {
		userConfigs[i] = ToBox2Request(uc)
	}
	return &RunMultiboxRequest{
		ManagerSandbox:     ToBox2Request(req.ManagerSandbox),
		UserSandboxConfigs: userConfigs,
		UseStdin:           req.UseStdin,
		ManagerMemQuota:    managerMemQuota,
		IndividualMemQuota: individualMemQuota,
	}
}

func FromRunMultiboxRequest(req *RunMultiboxRequest) (*eval.MultiboxRequest, int64, int64) {
	userConfigs := make([]*eval.Box2Request, len(req.UserSandboxConfigs))
	for i, uc := range req.UserSandboxConfigs {
		userConfigs[i] = FromBox2Request(uc)
	}
	return &eval.MultiboxRequest{
		ManagerSandbox:     FromBox2Request(req.ManagerSandbox),
		UserSandboxConfigs: userConfigs,
		UseStdin:           req.UseStdin,
	}, req.ManagerMemQuota, req.IndividualMemQuota
}

// RunMultiboxResponse is the response body for the RunMultibox endpoint.
type RunMultiboxResponse struct {
	ManagerResponse *Box2Response `json:"manager_response"`
	UserStats       []*RunStats   `json:"user_stats"`
}

// GetLanguageVersionsResponse is the response body for GetLanguageVersions.
type GetLanguageVersionsResponse struct {
	Versions map[string]string `json:"versions"`
}

// PingResponse is the response body for Ping.
type PingResponse struct {
	Version string `json:"version"`
}
