package positive

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	adminpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/admin_service"
	libpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/library_service"
	bookpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/demo/business/book"
)

const bufSize = 1024 * 1024

// newGRPCServer starts an in-memory gRPC server with LibraryService + AdminService mocks.
func newGRPCServer(t *testing.T, libSrv libpb.LibraryServiceServer, adminSrv adminpb.AdminServiceServer) (libpb.LibraryServiceClient, adminpb.AdminServiceClient, func()) {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	if libSrv != nil {
		libpb.RegisterLibraryServiceServer(s, libSrv)
	}
	if adminSrv != nil {
		adminpb.RegisterAdminServiceServer(s, adminSrv)
	}
	go func() { _ = s.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	return libpb.NewLibraryServiceClient(conn), adminpb.NewAdminServiceClient(conn), func() { conn.Close(); s.Stop() }
}

// newGatewayMux creates a grpc-gateway ServeMux with LibraryService + AdminService handlers.
func newGatewayMux(t *testing.T, libSrv libpb.LibraryServiceServer, adminSrv adminpb.AdminServiceServer) *runtime.ServeMux {
	t.Helper()
	mux := runtime.NewServeMux()
	if err := libpb.RegisterLibraryServiceHandlerServer(context.Background(), mux, libSrv); err != nil {
		t.Fatalf("register library: %v", err)
	}
	if err := adminpb.RegisterAdminServiceHandlerServer(context.Background(), mux, adminSrv); err != nil {
		t.Fatalf("register admin: %v", err)
	}
	return mux
}

// doReq sends an HTTP request to the test server and returns the response.
func doReq(t *testing.T, ts *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.URL+path, r)
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	if r != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do req %s %s: %v", method, path, err)
	}
	return resp
}

// mustReadJSON decodes the response body as JSON.
func mustReadJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	out := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return out
}

// ---- Mock LibraryServiceServer ----

type mockLibraryServer struct {
	libpb.UnimplementedLibraryServiceServer

	lastCreateReq   *libpb.CreateBookRequest
	lastDeleteReq   *libpb.DeleteBookRequest
	lastDeleteSoft  *libpb.DeleteBookSoftRequest
	lastGetMetaReq  *libpb.GetBookMetaRequest
	lastBatchGetReq *libpb.BatchGetBookMetasRequest
	lastListReq     *libpb.ListBookMetasRequest
	lastUpdateMeta  *libpb.UpdateBookMetaRequest
	lastGetContent  *libpb.GetBookContentRequest
	lastUpdateCont  *libpb.UpdateBookContentRequest
	lastArchiveReq  *bookpb.ArchiveBookRequest
}

func (m *mockLibraryServer) CreateBook(_ context.Context, req *libpb.CreateBookRequest) (*libpb.CreateBookResponse, error) {
	m.lastCreateReq = req
	return &libpb.CreateBookResponse{Key: &bookpb.BookId{Id: "new-id"}}, nil
}
func (m *mockLibraryServer) DeleteBook(_ context.Context, req *libpb.DeleteBookRequest) (*emptypb.Empty, error) {
	m.lastDeleteReq = req
	return &emptypb.Empty{}, nil
}
func (m *mockLibraryServer) DeleteBookSoft(_ context.Context, req *libpb.DeleteBookSoftRequest) (*emptypb.Empty, error) {
	m.lastDeleteSoft = req
	return &emptypb.Empty{}, nil
}
func (m *mockLibraryServer) GetBookMeta(_ context.Context, req *libpb.GetBookMetaRequest) (*libpb.GetBookMetaResponse, error) {
	m.lastGetMetaReq = req
	return &libpb.GetBookMetaResponse{
		BookMeta: &bookpb.BookMeta{Title: "T", Author: "A", Isbn: "I"},
		Version:  42,
	}, nil
}
func (m *mockLibraryServer) BatchGetBookMetas(_ context.Context, req *libpb.BatchGetBookMetasRequest) (*libpb.BatchGetBookMetasResponse, error) {
	m.lastBatchGetReq = req
	metas := make([]*bookpb.BookMeta, 0, len(req.GetKeys()))
	for range req.GetKeys() {
		metas = append(metas, &bookpb.BookMeta{Title: "batch"})
	}
	return &libpb.BatchGetBookMetasResponse{Metas: metas}, nil
}
func (m *mockLibraryServer) ListBookMetas(_ context.Context, req *libpb.ListBookMetasRequest) (*libpb.ListBookMetasResponse, error) {
	m.lastListReq = req
	return &libpb.ListBookMetasResponse{
		Metas:         []*bookpb.BookMeta{{Title: "list-item"}},
		NextPageToken: "tok",
		TotalSize:     1,
	}, nil
}
func (m *mockLibraryServer) UpdateBookMeta(_ context.Context, req *libpb.UpdateBookMetaRequest) (*libpb.UpdateBookMetaResponse, error) {
	m.lastUpdateMeta = req
	return &libpb.UpdateBookMetaResponse{Version: 43}, nil
}
func (m *mockLibraryServer) GetBookContent(_ context.Context, req *libpb.GetBookContentRequest) (*libpb.GetBookContentResponse, error) {
	m.lastGetContent = req
	return &libpb.GetBookContentResponse{BookContent: &bookpb.BookContent{Text: "hello"}}, nil
}
func (m *mockLibraryServer) UpdateBookContent(_ context.Context, req *libpb.UpdateBookContentRequest) (*emptypb.Empty, error) {
	m.lastUpdateCont = req
	return &emptypb.Empty{}, nil
}
func (m *mockLibraryServer) ArchiveBook(_ context.Context, req *bookpb.ArchiveBookRequest) (*bookpb.ArchiveBookResponse, error) {
	m.lastArchiveReq = req
	return &bookpb.ArchiveBookResponse{Archived: true}, nil
}

// ---- Mock AdminServiceServer ----

type mockAdminServer struct {
	adminpb.UnimplementedAdminServiceServer

	lastListReq *adminpb.ListBookMetasRequest
	lastGetReq  *adminpb.GetBookMetaRequest
}

func (m *mockAdminServer) CreateBook(context.Context, *adminpb.CreateBookRequest) (*adminpb.CreateBookResponse, error) {
	return &adminpb.CreateBookResponse{Key: &bookpb.BookId{Id: "admin-new"}}, nil
}
func (m *mockAdminServer) DeleteBook(context.Context, *adminpb.DeleteBookRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (m *mockAdminServer) DeleteBookSoft(context.Context, *adminpb.DeleteBookSoftRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (m *mockAdminServer) GetBookMeta(_ context.Context, req *adminpb.GetBookMetaRequest) (*adminpb.GetBookMetaResponse, error) {
	m.lastGetReq = req
	return &adminpb.GetBookMetaResponse{BookMeta: &bookpb.BookMeta{Title: "admin-meta"}, Version: 7}, nil
}
func (m *mockAdminServer) ListBookMetas(_ context.Context, req *adminpb.ListBookMetasRequest) (*adminpb.ListBookMetasResponse, error) {
	m.lastListReq = req
	return &adminpb.ListBookMetasResponse{
		Metas:     []*bookpb.BookMeta{{Title: "admin-list"}},
		TotalSize: 1,
	}, nil
}
func (m *mockAdminServer) UpdateBookMeta(context.Context, *adminpb.UpdateBookMetaRequest) (*adminpb.UpdateBookMetaResponse, error) {
	return &adminpb.UpdateBookMetaResponse{Version: 8}, nil
}

// Prevent unused import errors when mocks are compiled.
var (
	_ = fieldmaskpb.FieldMask{}
	_ = wrapperspb.StringValue{}
)

// ---- Helper function tests ----

// TestHelperGRPCServer verifies that newGRPCServer returns working clients and cleanup.
func TestHelperGRPCServer(t *testing.T) {
	libCli, adminCli, cleanup := newGRPCServer(t, nil, nil)
	defer cleanup()
	if libCli == nil {
		t.Fatal("libCli is nil")
	}
	if adminCli == nil {
		t.Fatal("adminCli is nil")
	}
}

// TestHelperGatewayMux verifies that newGatewayMux returns a usable mux.
func TestHelperGatewayMux(t *testing.T) {
	mux := newGatewayMux(t, nil, nil)
	if mux == nil {
		t.Fatal("mux is nil")
	}
}
