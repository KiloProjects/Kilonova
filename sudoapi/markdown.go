package sudoapi

import (
	"context"

	"go.uber.org/zap"
)

func (s *BaseAPI) RenderMarkdown(src []byte) ([]byte, *StatusError) {
	out, err := s.rd.Render(src)
	if err != nil {
		return nil, WrapError(err, "Couldn't render markdown")
	}
	return out, nil
}

// WarmupCache loads all statements and attempts to render them.
// It's stupid, but it should do the trick to improve initial loads.
func (s *BaseAPI) warmupMarkdownCache(ctx context.Context) {
	zap.S().Info("Warming up markdown cache...")
	atts, err := s.db.MarkdownAttachments(ctx, 0, 0)
	if err != nil {
		zap.S().Warn("Couldn't fetch attachments for warmup: ", err)
		return
	}
	zap.S().Info(len(atts))
	workQueue := make(chan []byte, 5)
	defer close(workQueue)
	for i := 0; i < 5; i++ {
		go func() {
			for {
				select {
				case val, ok := <-workQueue:
					if !ok {
						return
					}
					if _, err := s.RenderMarkdown(val); err != nil {
						zap.S().Warn("Couldn't render markdown: ", err)
					}
				}
			}
		}()
	}
	for _, att := range atts {
		att := att
		workQueue <- att
	}
	zap.S().Info("Finished warming up")
}
