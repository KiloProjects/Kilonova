package scheduler

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"slices"

	"github.com/KiloProjects/kilonova/eval"
)

// Box2Wrapper is the platform-side convenience layer that presents the
// datastore-aware Box2 interface on top of a (local or remote) Box3Scheduler.
// It owns moving bytes in/out of the scratch — tasks/ never touch scratch
// directly — and cleans up scratch identifiers it creates. In remote mode mgr
// and scratch are the RPC client + SFTP scratch; nothing else here changes.
type Box2Wrapper struct {
	scratch eval.Scratch
	store   eval.Store
	mgr     eval.Box3Scheduler
}

func NewBox2Wrapper(scratch eval.Scratch, store eval.Store, mgr eval.Box3Scheduler) eval.BoxScheduler {
	return &Box2Wrapper{
		scratch: scratch,
		store:   store,
		mgr:     mgr,
	}
}

func (b *Box2Wrapper) RunBox2(ctx context.Context, b2Req *eval.Box2Request, memQuota int64) (*eval.Box2Response, error) {
	b3Req, err := b.convertRequest(ctx, b2Req)
	if b3Req != nil {
		// Setup the defer here for cleaning up identifiers
		defer b.clearInput(ctx, b3Req)
	}

	if err != nil {
		return nil, err
	}

	result, err := b.mgr.RunBox3(ctx, b3Req, memQuota)
	if err != nil {
		return nil, err
	}

	return b.convertResponse(ctx, b2Req, result)
}

func (b *Box2Wrapper) RunMultibox2(ctx context.Context, b2Req *eval.Multibox2Request, managerMemQuota int64, individualMemQuota int64) (*eval.Box2Response, []*eval.RunStats, error) {
	managerRequest, err := b.convertRequest(ctx, b2Req.ManagerSandbox)
	if err != nil {
		return nil, nil, err
	}
	defer b.clearInput(ctx, managerRequest)

	userConfigs := make([]*eval.Box3Request, 0, len(b2Req.UserSandboxConfigs))
	for _, userReq := range b2Req.UserSandboxConfigs {
		userConfig, err := b.convertRequest(ctx, userReq)
		if err != nil {
			return nil, nil, err
		}
		defer b.clearInput(ctx, userConfig)
		userConfigs = append(userConfigs, userConfig)
	}

	result, stats, err := b.mgr.RunMultibox3(ctx, &eval.Multibox3Request{
		ManagerSandbox:     managerRequest,
		UserSandboxConfigs: userConfigs,
		UseStdin:           b2Req.UseStdin,
	}, managerMemQuota, individualMemQuota)
	if err != nil {
		return nil, stats, err
	}

	resp2, err := b.convertResponse(ctx, b2Req.ManagerSandbox, result)
	return resp2, stats, err
}

func (b *Box2Wrapper) convertRequest(ctx context.Context, b2Req *eval.Box2Request) (*eval.Box3Request, error) {
	b3Req := &eval.Box3Request{
		InputFiles:      make([]eval.ScratchFile, 0, len(b2Req.InputBucketFiles)+len(b2Req.InputByteFiles)),
		Command:         b2Req.Command,
		RunConfig:       b2Req.RunConfig,
		OutputFilePaths: make([]string, 0, len(b2Req.OutputByteFiles)+len(b2Req.OutputBucketFiles)),
	}

	// First convert input files from byte files / bucket files
	// We do this by saving the files to the scratch space first.

	for boxPath, byteFile := range b2Req.InputByteFiles {
		identifier, err := b.scratch.SaveFile(bytes.NewReader(byteFile.Data))
		if err != nil {
			return nil, err
		}
		b3Req.InputFiles = append(b3Req.InputFiles, eval.ScratchFile{
			Identifier: identifier,
			BoxPath:    boxPath,
			Mode:       byteFile.Mode,
		})
	}

	for boxPath, bucketFile := range b2Req.InputBucketFiles {
		rc, err := b.store.Reader(bucketFile.Bucket, bucketFile.Filename)
		if err != nil {
			return nil, err
		}

		identifier, err := b.scratch.SaveFile(rc)
		if err != nil {
			return nil, err
		}
		if err := rc.Close(); err != nil {
			slog.WarnContext(ctx, "Could not clean up bucket file read", slog.Any("err", err))
		}
		b3Req.InputFiles = append(b3Req.InputFiles, eval.ScratchFile{
			Identifier: identifier,
			BoxPath:    boxPath,
			Mode:       bucketFile.Mode,
		})
	}

	// Then set up the mappings
	for _, val := range b2Req.OutputByteFiles {
		b3Req.OutputFilePaths = append(b3Req.OutputFilePaths, val)
	}

	for path := range b2Req.OutputBucketFiles {
		b3Req.OutputFilePaths = append(b3Req.OutputFilePaths, path)
	}

	return b3Req, nil
}

func (b *Box2Wrapper) convertResponse(ctx context.Context, req *eval.Box2Request, result *eval.Box3Response) (*eval.Box2Response, error) {
	if result == nil || req == nil {
		return nil, nil
	}
	resp := &eval.Box2Response{
		Stats:       result.Stats,
		ByteFiles:   make(map[string][]byte),
		BucketFiles: make([]string, 0, len(req.OutputBucketFiles)),
	}

	for fPath, identifier := range result.Files {
		if slices.Contains(req.OutputByteFiles, fPath) {
			val, err := b.readDeleteScratch(ctx, identifier)
			if err != nil {
				slog.WarnContext(ctx, "Could not read byte scratch file", slog.Any("err", err), slog.Any("identifier", identifier))
				return nil, err
			}
			resp.ByteFiles[fPath] = val
		} else if _, ok := req.OutputBucketFiles[fPath]; ok {
			if err := b.copyDeleteScratch(ctx, identifier, req.OutputBucketFiles[fPath]); err != nil {
				slog.WarnContext(ctx, "Could not read bucket scratch file", slog.Any("err", err), slog.Any("identifier", identifier))
				return nil, err
			}
			resp.BucketFiles = append(resp.BucketFiles, fPath)
		} else {
			slog.WarnContext(ctx, "Could not recognize scratch file", slog.Any("fname", fPath), slog.Any("identifier", identifier))
			b.scratch.DeleteFile(identifier)
		}
	}

	return resp, nil
}

func (b *Box2Wrapper) readDeleteScratch(ctx context.Context, identifier string) ([]byte, error) {
	defer func(identifier string) {
		err := b.scratch.DeleteFile(identifier)
		if err != nil {
			slog.WarnContext(ctx, "Could not clean up scratch file", slog.Any("err", err), slog.Any("identifier", identifier))
		}
	}(identifier)

	rc, err := b.scratch.ReadFile(identifier)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

func (b *Box2Wrapper) copyDeleteScratch(ctx context.Context, identifier string, file *eval.BucketFile) error {
	defer func(identifier string) {
		err := b.scratch.DeleteFile(identifier)
		if err != nil {
			slog.WarnContext(ctx, "Could not clean up scratch file", slog.Any("err", err), slog.Any("identifier", identifier))
		}
	}(identifier)

	rc, err := b.scratch.ReadFile(identifier)
	if err != nil {
		return err
	}
	defer rc.Close()

	return b.store.WriteFile(file.Bucket, file.Filename, rc, file.Mode)
}

func (b *Box2Wrapper) clearInput(ctx context.Context, req *eval.Box3Request) {
	for _, file := range req.InputFiles {
		err := b.scratch.DeleteFile(file.Identifier)
		if err != nil {
			slog.WarnContext(ctx, "Could not clean up scratch file", slog.Any("err", err), slog.Any("identifier", file.Identifier))
		}
	}
}

func (b *Box2Wrapper) Close(ctx context.Context) error {
	return b.mgr.Close(ctx)
}
