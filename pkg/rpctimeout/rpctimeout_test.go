package rpctimeout

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/kitex/internal/test"
	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
)

var panicMsg = "hello world"

func block(ctx context.Context, request, response interface{}) (err error) {
	time.Sleep(1 * time.Second)
	return nil
}

func pass(ctx context.Context, request, response interface{}) (err error) {
	time.Sleep(200 * time.Millisecond)
	return nil
}

func ache(ctx context.Context, request, response interface{}) (err error) {
	panic(panicMsg)
}

func TestNewRPCTimeoutMW(t *testing.T) {
	t.Parallel()

	s := rpcinfo.NewEndpointInfo("mockService", "mockMethod", nil, nil)
	c := rpcinfo.NewRPCConfig()
	r := rpcinfo.NewRPCInfo(nil, s, nil, c, rpcinfo.NewRPCStats())
	m := rpcinfo.AsMutableRPCConfig(c)
	m.SetRPCTimeout(time.Millisecond * 500)

	ctx := rpcinfo.NewCtxWithRPCInfo(context.Background(), r)
	mwCtx := context.Background()
	mwCtx = context.WithValue(mwCtx, endpoint.CtxLoggerKey, klog.DefaultLogger())

	var err error
	// 1. normal
	err = MiddlewareBuilder(0)(mwCtx)(pass)(ctx, nil, nil)
	test.Assert(t, err == nil)

	// 2. block to mock timeout
	err = MiddlewareBuilder(0)(mwCtx)(block)(ctx, nil, nil)
	test.Assert(t, err != nil, err)
	test.Assert(t, err.(*kerrors.DetailedError).ErrorType() == kerrors.ErrRPCTimeout)

	// 3. block, pass more timeout, timeout won't happen
	err = MiddlewareBuilder(510*time.Millisecond)(mwCtx)(block)(ctx, nil, nil)
	test.Assert(t, err == nil)

	// 4. mock panic happen
	// < v1.1.* panic happen, >=v1.1* wrap panic to error
	err = MiddlewareBuilder(0)(mwCtx)(ache)(ctx, nil, nil)
	test.Assert(t, strings.Contains(err.Error(), panicMsg))

	// 5. cancel
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	time.AfterFunc(100*time.Millisecond, func() {
		cancelFunc()
	})
	err = MiddlewareBuilder(0)(mwCtx)(block)(cancelCtx, nil, nil)
	test.Assert(t, errors.Is(err, context.Canceled), err)
}
