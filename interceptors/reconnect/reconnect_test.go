package reconnect_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SKF/go-enlight-authorizer/interceptors/reconnect"
	"github.com/SKF/go-enlight-authorizer/tests/server"
	pb "github.com/SKF/go-enlight-authorizer/tests/server/helloworld"
	"github.com/SKF/go-utility/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	bufSize     = 1024 * 1024
	timeout     = time.Millisecond * 100
	timeoutWait = time.Millisecond * 150
)

func Test_ReconnectInterceptor_HappyCase(t *testing.T) {
	ctx := context.Background()
	s := server.New(bufSize)
	defer s.Stop()

	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithUnaryInterceptor(reconnect.UnaryInterceptor(
			reconnect.WithNewConnection(func(ctx context.Context, cc *grpc.ClientConn, opts ...grpc.CallOption) (context.Context, *grpc.ClientConn, []grpc.CallOption, error) {
				conn, err := grpc.DialContext(ctx, "bufnet",
					grpc.WithContextDialer(s.Dialer()),
					grpc.WithTransportCredentials(insecure.NewCredentials()),
				)

				if err != nil {
					log.Infof("Failed to dial bufnet: %v", err)
					return ctx, cc, opts, err
				}
				return ctx, conn, opts, nil
			}),
		)),
		grpc.WithContextDialer(s.Dialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "failed to dial bufnet")
	defer conn.Close()

	client := pb.NewGreeterClient(conn)

	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: "Lasse Kongo"})
	assert.NoError(t, err, "failed to call first SayHello")

	s.Restart()

	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: "Kalle Anka"})
	assert.NoError(t, err, "failed to call last SayHello")

	s.Restart()

	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: "Kalle Anka"})
	assert.NoError(t, err, "failed to call last SayHello")
}

func Test_ReconnectInterceptor_ConnectionClosed(t *testing.T) {
	ctx := context.Background()
	s := server.New(bufSize)
	defer s.Stop()

	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithUnaryInterceptor(reconnect.UnaryInterceptor()),
		grpc.WithContextDialer(s.Dialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "failed to dial bufnet")
	defer conn.Close()

	s.Stop()

	client := pb.NewGreeterClient(conn)
	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: "Kalle Anka"})
	assert.EqualError(t, err, `rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing closed"`)
}

func Test_ReconnectInterceptor_RepeatedReconnects(t *testing.T) {
	ctx := context.Background()
	s := server.New(bufSize)
	defer s.Stop()

	interceptorCalled := 0

	var client pb.GreeterClient
	var newClientConn reconnect.NewConnectionFunc
	newClientConn = func(ctx context.Context, cc *grpc.ClientConn, opts ...grpc.CallOption) (context.Context, *grpc.ClientConn, []grpc.CallOption, error) {
		interceptorCalled++

		conn, err := grpc.DialContext(ctx, "bufnet",
			grpc.WithUnaryInterceptor(reconnect.UnaryInterceptor(
				reconnect.WithNewConnection(newClientConn),
			)),
			grpc.WithContextDialer(s.Dialer()),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)

		if err != nil {
			err = fmt.Errorf("inside: %w", err)
			return ctx, cc, opts, err
		}
		_ = cc.Close()

		client = pb.NewGreeterClient(conn)
		return ctx, conn, opts, nil
	}
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithUnaryInterceptor(reconnect.UnaryInterceptor(
			reconnect.WithNewConnection(newClientConn),
		)),
		grpc.WithContextDialer(s.Dialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	require.NoError(t, err, "failed to dial bufnet")
	defer conn.Close()

	client = pb.NewGreeterClient(conn)

	// State: READY
	childCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err = client.SayHello(childCtx, &pb.HelloRequest{Name: "Call: 0"})
	assert.NoError(t, err, "failed to call first SayHello")

	loops := 5
	for i := 0; i < loops; i++ {
		msg := fmt.Sprintf("Loop no %d", i)
		s.Stop()

		// State: TRANSIENT_FAILURE
		childCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		_, err = client.SayHello(childCtx, &pb.HelloRequest{Name: fmt.Sprintf("Loop: %d, Call: 1", i)})
		assert.EqualError(t, err, "failed to reconnect: inside: context deadline exceeded", msg)

		s.Start()

		// State: TRANSIENT_FAILURE
		childCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		_, err = client.SayHello(childCtx, &pb.HelloRequest{Name: fmt.Sprintf("Loop: %d, Call: 2", i)})
		assert.NoError(t, err, "failed to call last SayHello", msg)

		time.Sleep(timeoutWait)

		// State: READY
		_, err = client.SayHello(childCtx, &pb.HelloRequest{Name: fmt.Sprintf("Loop: %d, Call: 3", i)})
		assert.Error(t, err, "context deadline exceeded", msg)
	}

	assert.Equal(t, loops*2, interceptorCalled)
}
