package positive

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHTTP_LibraryServiceAllRoutes verifies all 10 HTTP endpoints of LibraryService.
func TestHTTP_LibraryServiceAllRoutes(t *testing.T) {
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
			"filter":     map[string]any{"author": "X"},
			"order_by":   "title",
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := srv.lastListReq.GetPageSize(); got != 10 {
			t.Errorf("page_size=%d want 10", got)
		}
		if got := srv.lastListReq.GetFilter().GetAuthor(); got != "X" {
			t.Errorf("filter.author=%q want X", got)
		}
		body := mustReadJSON(t, resp)
		if body["totalSize"] != float64(1) {
			t.Errorf("totalSize=%v want 1", body["totalSize"])
		}
	})

	t.Run("UpdateBookMeta PATCH /{key.id}/meta body=*", func(t *testing.T) {
		resp := doReq(t, ts, "PATCH", "/library/LibraryService/book/bk-up/meta", map[string]any{
			"meta":        map[string]any{"title": "v2"},
			"version":     "42",
			"update_mask": "title",
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := srv.lastUpdateMeta.GetKey().GetId(); got != "bk-up" {
			t.Errorf("key.id got=%q want=bk-up", got)
		}
		if got := srv.lastUpdateMeta.GetMeta().GetTitle(); got != "v2" {
			t.Errorf("meta.title got=%q want v2", got)
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
		if got := srv.lastUpdateCont.GetContent().GetText(); got != "updated" {
			t.Errorf("text got=%q want updated", got)
		}
	})

	t.Run("ArchiveBook custom method POST /{book_id}:archive", func(t *testing.T) {
		resp := doReq(t, ts, "POST", "/library/LibraryService/book/bk-001:archive", map[string]any{})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if srv.lastArchiveReq == nil {
			t.Fatal("ArchiveBook not called")
		}
		if got := srv.lastArchiveReq.GetBookId(); got != "bk-001" {
			t.Errorf("book_id got=%q want=bk-001", got)
		}
	})
}

// TestHTTP_AdminServiceNarrowedRoutes verifies AdminService has narrowed routes.
func TestHTTP_AdminServiceNarrowedRoutes(t *testing.T) {
	adminSrv := &mockAdminServer{}
	mux := newGatewayMux(t, nil, adminSrv)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("ListBookMetas on AdminService", func(t *testing.T) {
		resp := doReq(t, ts, "POST", "/library/AdminService/book/meta/list", map[string]any{
			"page_size": 5,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := adminSrv.lastListReq.GetPageSize(); got != 5 {
			t.Errorf("page_size=%d want 5", got)
		}
	})

	t.Run("GetBookMeta on AdminService", func(t *testing.T) {
		resp := doReq(t, ts, "GET", "/library/AdminService/book/admin-1/meta", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if got := adminSrv.lastGetReq.GetKey().GetId(); got != "admin-1" {
			t.Errorf("key.id got=%q want=admin-1", got)
		}
	})

	t.Run("CreateBook on AdminService", func(t *testing.T) {
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

// TestHTTP_KeyPathBinding_NoBody verifies key path variable binding without body.
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
