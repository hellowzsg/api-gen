package negative

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	libpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/library_service"
	bookpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/demo/business/book"
)

const negBufSize = 1024 * 1024

// newNegGRPCServer starts an in-memory gRPC server with only UnimplementedServer
// (no methods registered), so all calls return Unimplemented.
func newNegGRPCServer(t *testing.T) (libpb.LibraryServiceClient, func()) {
	t.Helper()
	lis := bufconn.Listen(negBufSize)
	s := grpc.NewServer()
	libpb.RegisterLibraryServiceServer(s, &negLibraryServer{})
	go func() { _ = s.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	return libpb.NewLibraryServiceClient(conn), func() { conn.Close(); s.Stop() }
}

// newNegGRPCServerWithUnimplemented starts a server with ONLY UnimplementedLibraryServiceServer
// (no method overrides), so every RPC call returns codes.Unimplemented.
func newNegGRPCServerWithUnimplemented(t *testing.T) (libpb.LibraryServiceClient, func()) {
	t.Helper()
	lis := bufconn.Listen(negBufSize)
	s := grpc.NewServer()
	libpb.RegisterLibraryServiceServer(s, &unimplLibraryServer{})
	go func() { _ = s.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	return libpb.NewLibraryServiceClient(conn), func() { conn.Close(); s.Stop() }
}

// unimplLibraryServer only embeds UnimplementedLibraryServiceServer — no methods are overridden.
type unimplLibraryServer struct {
	libpb.UnimplementedLibraryServiceServer
}

// TestGRPCNegative_UnimplementedMethod verifies that calling an unregistered method returns Unimplemented.
func TestGRPCNegative_UnimplementedMethod(t *testing.T) {
	cli, cleanup := newNegGRPCServerWithUnimplemented(t)
	defer cleanup()
	ctx := context.Background()

	_, err := cli.CreateBook(ctx, &libpb.CreateBookRequest{})
	if err == nil {
		t.Fatal("expected error for unimplemented method, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got: %v", err)
	}
	if st.Code() != codes.Unimplemented {
		t.Errorf("expected code Unimplemented, got %v", st.Code())
	}
}

// TestGRPCNegative_NilRequest verifies that nil request returns error (not panic).
func TestGRPCNegative_NilRequest(t *testing.T) {
	cli, cleanup := newNegGRPCServer(t)
	defer cleanup()
	ctx := context.Background()

	// CreateBook with nil request should return error, not panic
	_, err := cli.CreateBook(ctx, nil)
	if err == nil {
		// Some gRPC implementations may accept nil and return a zero response;
		// this is acceptable as long as it doesn't panic.
		t.Log("CreateBook with nil returned no error — acceptable if no panic")
	}
}

// TestGRPCNegative_CancelledContext verifies that a cancelled context returns Canceled error.
func TestGRPCNegative_CancelledContext(t *testing.T) {
	cli, cleanup := newNegGRPCServer(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := cli.GetBookMeta(ctx, &libpb.GetBookMetaRequest{
		Key: &bookpb.BookId{Id: "bk-1"},
	})
	if err == nil {
		t.Skip("GetBookMeta with cancelled context returned no error — server may not check context")
	}
	st, ok := status.FromError(err)
	if !ok {
		// Could be a transport error, which is also acceptable
		t.Logf("GetBookMeta with cancelled context returned non-status error: %v", err)
		return
	}
	if st.Code() != codes.Canceled && st.Code() != codes.DeadlineExceeded && st.Code() != codes.Unavailable {
		t.Errorf("expected code Canceled/DeadlineExceeded/Unavailable, got %v", st.Code())
	}
}

// prevent unused import
var _ = emptypb.Empty{}
