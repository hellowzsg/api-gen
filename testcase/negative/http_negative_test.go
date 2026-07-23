package negative

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	adminpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/admin_service"
	libpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/library_service"
	bookpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/demo/business/book"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// Mock servers for negative HTTP tests — minimal implementations.

type negLibraryServer struct {
	libpb.UnimplementedLibraryServiceServer
}

func (m *negLibraryServer) CreateBook(context.Context, *libpb.CreateBookRequest) (*libpb.CreateBookResponse, error) {
	return &libpb.CreateBookResponse{Key: &bookpb.BookId{Id: "new"}}, nil
}
func (m *negLibraryServer) DeleteBook(context.Context, *libpb.DeleteBookRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (m *negLibraryServer) GetBookMeta(context.Context, *libpb.GetBookMetaRequest) (*libpb.GetBookMetaResponse, error) {
	return &libpb.GetBookMetaResponse{BookMeta: &bookpb.BookMeta{Title: "T"}}, nil
}

type negAdminServer struct {
	adminpb.UnimplementedAdminServiceServer
}

// newNegGatewayMux creates a grpc-gateway mux for negative tests.
func newNegGatewayMux(t *testing.T, libSrv libpb.LibraryServiceServer, adminSrv adminpb.AdminServiceServer) *runtime.ServeMux {
	t.Helper()
	mux := runtime.NewServeMux()
	if libSrv != nil {
		if err := libpb.RegisterLibraryServiceHandlerServer(context.Background(), mux, libSrv); err != nil {
			t.Fatalf("register library: %v", err)
		}
	}
	if adminSrv != nil {
		if err := adminpb.RegisterAdminServiceHandlerServer(context.Background(), mux, adminSrv); err != nil {
			t.Fatalf("register admin: %v", err)
		}
	}
	return mux
}

func negDoReq(t *testing.T, ts *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
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
		t.Fatalf("do req: %v", err)
	}
	return resp
}

// TestHTTPNegative_NotFoundRoute verifies that unregistered routes return 404.
func TestHTTPNegative_NotFoundRoute(t *testing.T) {
	mux := newNegGatewayMux(t, &negLibraryServer{}, nil)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp := negDoReq(t, ts, "GET", "/library/LibraryService/book/bk-x/nonexistent", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d want 404", resp.StatusCode)
	}
}

// TestHTTPNegative_WrongMethod verifies that wrong HTTP method returns error.
func TestHTTPNegative_WrongMethod(t *testing.T) {
	mux := newNegGatewayMux(t, &negLibraryServer{}, nil)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// CreateBook is POST, not GET
	resp := negDoReq(t, ts, "GET", "/library/LibraryService/book", nil)
	if resp.StatusCode == http.StatusOK {
		t.Errorf("GET on POST-only endpoint should not return 200, got %d", resp.StatusCode)
	}
}

// TestHTTPNegative_InvalidJSONBody verifies that invalid JSON body returns error.
func TestHTTPNegative_InvalidJSONBody(t *testing.T) {
	mux := newNegGatewayMux(t, &negLibraryServer{}, nil)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Send invalid JSON to CreateBook (POST body=*)
	req, _ := http.NewRequest("POST", ts.URL+"/library/LibraryService/book", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do req: %v", err)
	}
	if resp.StatusCode == http.StatusOK {
		t.Errorf("invalid JSON should not return 200, got %d", resp.StatusCode)
	}
}

// TestHTTPNegative_AdminServiceNarrowedRoute verifies BatchGet is 404 on AdminService.
func TestHTTPNegative_AdminServiceNarrowedRoute(t *testing.T) {
	mux := newNegGatewayMux(t, nil, &negAdminServer{})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// BatchGetBookMetas should not be registered on AdminService
	resp := negDoReq(t, ts, "POST", "/library/AdminService/book/meta/batchGet", map[string]any{"keys": []map[string]any{{"id": "a"}}})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("BatchGetBookMetas on AdminService should be 404, got %d", resp.StatusCode)
	}
}
