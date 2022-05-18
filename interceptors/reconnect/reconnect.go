package reconnect

import (
	"context"
	"fmt"

	"github.com/SKF/go-utility/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

func UnaryInterceptor(opts ...CallOption) func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	options := evaluateCallOptions(opts)
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		switch cc.GetState() {
		case connectivity.Idle, connectivity.Connecting, connectivity.Ready:
			return invoker(ctx, method, req, reply, cc, opts...)
		default:
			log.WithTracing(ctx).
				WithField("state", cc.GetState().String()).
				Info("Calling reconnect function")

			newCtx, newCC, newOpts, err := options.newClientConn(ctx, cc, opts...)
			if err != nil {
				return fmt.Errorf("failed to reconnect: %w", err)
			}
			return invoker(newCtx, method, req, reply, newCC, newOpts...)
		}
	}
}
