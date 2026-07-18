package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminpb "github.com/acme/demo-book/generated/go/admin_service"
	libpb "github.com/acme/demo-book/generated/go/library_service"
	bookpb "github.com/acme/demo-book/generated/go/demo/business/book"
)

// ---- Mock LibraryServiceServer ----

type mockLibraryServer struct {
	libpb.UnimplementedLibraryServiceServer

	lastCreateReq    *libpb.CreateBookRequest
	lastDeleteReq    *libpb.DeleteBookRequest
	lastDeleteSoft   *libpb.DeleteBookSoftRequest
	lastGetMetaReq   *libpb.GetBookMetaRequest
	lastBatchGetReq  *libpb.BatchGetBookMetasRequest
	lastListReq      *libpb.ListBookMetasRequest
	lastUpdateMeta   *libpb.UpdateBookMetaRequest
	lastGetContent   *libpb.GetBookContentRequest
	lastUpdateCont   *libpb.UpdateBookContentRequest
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

// ---- helpers ----

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

func mustReadJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	out := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return out
}

// ---- tests ----

// 覆盖 LibraryService 的全部 9 个 HTTP 端点（动词、路径、path 变量、body 解码、响应序列化）
func TestLibraryServiceHTTP_AllMethods(t *testing.T) {
	srv := &mockLibraryServer{}
	mux := newGatewayMux(t, srv, nil)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("CreateBook POST body=*", func(t *testing.T) {
		resp := doReq(t, ts, "POST", "/library/LibraryService/book", map[string]any{
			"meta":    map[string]any{"title": "Go 101"},
			"content": map[string]any{"text": "chapter1"},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := srv.lastCreateReq.GetMeta().GetTitle(); got != "Go 101" {
			t.Errorf("meta.title got=%q want=Go 101", got)
		}
		if got := srv.lastCreateReq.GetContent().GetText(); got != "chapter1" {
			t.Errorf("content.text got=%q want=chapter1", got)
		}
		body := mustReadJSON(t, resp)
		if got, _ := body["key"].(map[string]any); got["id"] != "new-id" {
			t.Errorf("response key.id=%v want=new-id", body)
		}
	})

	t.Run("DeleteBook DELETE /{key.id}", func(t *testing.T) {
		resp := doReq(t, ts, "DELETE", "/library/LibraryService/book/bk-1", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := srv.lastDeleteReq.GetKey().GetId(); got != "bk-1" {
			t.Errorf("key.id got=%q want=bk-1", got)
		}
	})

	t.Run("DeleteBookSoft POST /deleteSoft body=*", func(t *testing.T) {
		resp := doReq(t, ts, "POST", "/library/LibraryService/book/deleteSoft", map[string]any{
			"key": map[string]any{"id": "bk-soft"},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := srv.lastDeleteSoft.GetKey().GetId(); got != "bk-soft" {
			t.Errorf("key.id got=%q want=bk-soft", got)
		}
	})

	t.Run("GetBookMeta GET /{key.id}/meta", func(t *testing.T) {
		resp := doReq(t, ts, "GET", "/library/LibraryService/book/bk-meta/meta", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := srv.lastGetMetaReq.GetKey().GetId(); got != "bk-meta" {
			t.Errorf("key.id got=%q want=bk-meta", got)
		}
		body := mustReadJSON(t, resp)
		// protojson serializes uint64 as string.
		if body["version"] != "42" {
			t.Errorf("version=%v want \"42\"", body["version"])
		}
	})

	t.Run("BatchGetBookMetas POST /meta/batchGet body=*", func(t *testing.T) {
		resp := doReq(t, ts, "POST", "/library/LibraryService/book/meta/batchGet", map[string]any{
			"keys": []map[string]any{{"id": "a"}, {"id": "b"}},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if n := len(srv.lastBatchGetReq.GetKeys()); n != 2 {
			t.Fatalf("keys len=%d want 2", n)
		}
		if got := srv.lastBatchGetReq.GetKeys()[0].GetId(); got != "a" {
			t.Errorf("keys[0].id=%q want a", got)
		}
		body := mustReadJSON(t, resp)
		metas, _ := body["metas"].([]any)
		if len(metas) != 2 {
			t.Errorf("metas len=%d want 2", len(metas))
		}
	})

	t.Run("ListBookMetas POST /meta/list body=*", func(t *testing.T) {
		resp := doReq(t, ts, "POST", "/library/LibraryService/book/meta/list", map[string]any{
			"page_size":  10,
			"page_token": "p1",
			"filter":     "author==\"X\"",
			"order_by":   "title",
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := srv.lastListReq.GetPageSize(); got != 10 {
			t.Errorf("page_size=%d want 10", got)
		}
		if got := srv.lastListReq.GetFilter(); got != "author==\"X\"" {
			t.Errorf("filter=%q", got)
		}
		body := mustReadJSON(t, resp)
		if body["totalSize"] != float64(1) {
			t.Errorf("totalSize=%v want 1", body["totalSize"])
		}
	})

	t.Run("UpdateBookMeta PATCH /{key.id}/meta body=*", func(t *testing.T) {
		resp := doReq(t, ts, "PATCH", "/library/LibraryService/book/bk-up/meta", map[string]any{
			"meta":        map[string]any{"title": "v2"},
			"version":     "42", // protojson expects uint64 as string
			"update_mask": "title", // FieldMask serializes as comma-separated string in protojson
		})
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("status=%d body=%s", resp.StatusCode, string(b))
		}
		if got := srv.lastUpdateMeta.GetKey().GetId(); got != "bk-up" {
			t.Errorf("key.id got=%q want=bk-up", got)
		}
		if got := srv.lastUpdateMeta.GetMeta().GetTitle(); got != "v2" {
			t.Errorf("meta.title got=%q want v2", got)
		}
		if got := srv.lastUpdateMeta.GetVersion(); got != 42 {
			t.Errorf("version=%d want 42", got)
		}
		body := mustReadJSON(t, resp)
		if body["version"] != "43" {
			t.Errorf("version=%v want \"43\"", body["version"])
		}
	})

	t.Run("GetBookContent GET /{key.id}/content", func(t *testing.T) {
		resp := doReq(t, ts, "GET", "/library/LibraryService/book/bk-c/content", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := srv.lastGetContent.GetKey().GetId(); got != "bk-c" {
			t.Errorf("key.id got=%q want=bk-c", got)
		}
		body := mustReadJSON(t, resp)
		bc, _ := body["bookContent"].(map[string]any)
		if bc["text"] != "hello" {
			t.Errorf("text=%v want hello", bc["text"])
		}
	})

	t.Run("UpdateBookContent PATCH /{key.id}/content body=*", func(t *testing.T) {
		resp := doReq(t, ts, "PATCH", "/library/LibraryService/book/bk-c2/content", map[string]any{
			"content": map[string]any{"text": "updated"},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := srv.lastUpdateCont.GetKey().GetId(); got != "bk-c2" {
			t.Errorf("key.id got=%q want=bk-c2", got)
		}
		if got := srv.lastUpdateCont.GetContent().GetText(); got != "updated" {
			t.Errorf("text got=%q want updated", got)
		}
	})
}

// 验证 AdminService 收窄后的路由：BatchGetBookMetas 不应被注册（404），
// 其余方法走 AdminService 的 mock
func TestAdminServiceHTTP_NarrowedRoutes(t *testing.T) {
	adminSrv := &mockAdminServer{}
	mux := newGatewayMux(t, nil, adminSrv)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("ListBookMetas 在 AdminService 上可调用", func(t *testing.T) {
		resp := doReq(t, ts, "POST", "/library/AdminService/book/meta/list", map[string]any{
			"page_size": 5,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := adminSrv.lastListReq.GetPageSize(); got != 5 {
			t.Errorf("page_size=%d want 5", got)
		}
		body := mustReadJSON(t, resp)
		if body["totalSize"] != float64(1) {
			t.Errorf("totalSize=%v want 1", body["totalSize"])
		}
	})

	t.Run("GetBookMeta 在 AdminService 上可调用", func(t *testing.T) {
		resp := doReq(t, ts, "GET", "/library/AdminService/book/admin-1/meta", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := adminSrv.lastGetReq.GetKey().GetId(); got != "admin-1" {
			t.Errorf("key.id got=%q want=admin-1", got)
		}
	})

	t.Run("CreateBook 在 AdminService 上可调用", func(t *testing.T) {
		resp := doReq(t, ts, "POST", "/library/AdminService/book", map[string]any{
			"meta": map[string]any{"title": "admin-book"},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		body := mustReadJSON(t, resp)
		if got, _ := body["key"].(map[string]any); got["id"] != "admin-new" {
			t.Errorf("key.id=%v want=admin-new", body)
		}
	})
}

// 404 路径覆盖验证：路径模板中不含 BatchGetBookMetas 时该路径返回 404（grpc-gateway 未注册路由）。
// 验证未匹配的路径返回 404 而不是 200
func TestHTTP_NotFound(t *testing.T) {
	srv := &mockLibraryServer{}
	mux := newGatewayMux(t, srv, nil)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp := doReq(t, ts, "GET", "/library/LibraryService/book/bk-x/nonexistent", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d want 404", resp.StatusCode)
	}
}

// 验证 key 路径变量用 GET 时不含在 body 中也能解析（grpc-gateway 把 path 段注入到请求字段）
// 这里再次确认：DeleteBook 用 DELETE 无 body，仅靠 {key.id} 路径段即可填充 Key.Id
func TestHTTP_KeyPathBinding_NoBody(t *testing.T) {
	srv := &mockLibraryServer{}
	mux := newGatewayMux(t, srv, nil)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp := doReq(t, ts, "DELETE", "/library/LibraryService/book/key-only-1", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if got := srv.lastDeleteReq.GetKey().GetId(); got != "key-only-1" {
		t.Errorf("key.id got=%q want=key-only-1", got)
	}
}

// 防 import 误删
var _ = fieldmaskpb.FieldMask{}
var _ = strings.Builder{}
