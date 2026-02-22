package ctxt

import "context"

func Value[T any, ctxType ~string](ctx context.Context, key ctxType) *T {
	switch v := ctx.Value(key).(type) {
	case *T:
		return v
	case T:
		return &v
	default:
		return nil
	}
}
