package server

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/SKF/go-enlight-authorizer/tests/server/helloworld"
)

type server struct {
	bufSize  int
	listener *bufconn.Listener
	opts     []grpc.ServerOption

	Server      *grpc.Server
	TestService *service
}

type service struct {
	calls int
	pb.UnimplementedGreeterServer
}

func (s *service) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	s.calls++
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func (s *server) Start() {
	s.Server = grpc.NewServer(s.opts...)
	s.TestService = &service{}
	pb.RegisterGreeterServer(s.Server, s.TestService)

	listening := make(chan int)

	go func() {
		s.listener = bufconn.Listen(s.bufSize)
		close(listening)
		s.Server.Serve(s.listener) //nolint:errcheck
	}()

	<-listening
}

func New(bufferSize int, opts ...grpc.ServerOption) *server {
	s := &server{
		bufSize: bufferSize,
		opts:    opts,
	}

	s.Start()

	return s
}

func (s *server) Dialer() func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return s.listener.Dial()
	}
}

func (s *server) NumberOfClientCalls() int {
	return s.TestService.calls
}

func (s *server) Restart() {
	s.Stop()
	s.Start()
}

func (s *server) RestartWithWaiting(d time.Duration) {
	s.Stop()

	go func() {
		time.Sleep(d)
		s.Start()
	}()
}

func (s *server) Stop() {
	s.Server.GracefulStop()
	time.Sleep(time.Millisecond * 100)
}
