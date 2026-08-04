package positive

import (
	"context"
	"io"
	"testing"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	libpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/library_service"
	adminpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/admin_service"
	bookpb "github.com/hellowzsg/api-gen/testcase/fixtures/book/generated/go/demo/business/book"
)

// TestGRPC_LibraryServiceAllMethods verifies all 10 LibraryService RPCs.
func TestGRPC_LibraryServiceAllMethods(t *testing.T) {
	srv := &mockLibraryServer{}
	libCli, _, cleanup := newGRPCServer(t, srv, nil)
	defer cleanup()
	ctx := context.Background()

	t.Run("CreateBook", func(t *testing.T) {
		resp, err := libCli.CreateBook(ctx, &libpb.CreateBookRequest{
			Meta:    &bookpb.BookMeta{Title: "Go 101", Author: "A"},
			Content: &bookpb.BookContent{Text: "ch1"},
		})
		if err != nil {
			t.Fatalf("CreateBook: %v", err)
		}
		if resp.GetKey().GetId() != "new-id" {
			t.Errorf("key.id=%q want new-id", resp.GetKey().GetId())
		}
		if srv.lastCreateReq.GetMeta().GetTitle() != "Go 101" {
			t.Errorf("meta.title=%q", srv.lastCreateReq.GetMeta().GetTitle())
		}
	})

	t.Run("BatchCreateBooks", func(t *testing.T) {
		resp, err := libCli.BatchCreateBooks(ctx, &libpb.BatchCreateBooksRequest{
			Requests: []*libpb.CreateBookRequest{
				{Meta: &bookpb.BookMeta{Title: "B1"}},
				{Meta: &bookpb.BookMeta{Title: "B2"}},
			},
		})
		if err != nil {
			t.Fatalf("BatchCreateBooks: %v", err)
		}
		if len(resp.GetKeys()) != 2 {
			t.Fatalf("keys len=%d want 2", len(resp.GetKeys()))
		}
		if len(srv.lastBatchCreate.GetRequests()) != 2 {
			t.Errorf("server received requests=%d want 2", len(srv.lastBatchCreate.GetRequests()))
		}
		if srv.lastBatchCreate.GetRequests()[0].GetMeta().GetTitle() != "B1" {
			t.Errorf("requests[0].meta.title=%q want B1", srv.lastBatchCreate.GetRequests()[0].GetMeta().GetTitle())
		}
	})

	t.Run("DeleteBook", func(t *testing.T) {
		_, err := libCli.DeleteBook(ctx, &libpb.DeleteBookRequest{
			Key: &bookpb.BookId{Id: "bk-1"},
		})
		if err != nil {
			t.Fatalf("DeleteBook: %v", err)
		}
		if srv.lastDeleteReq.GetKey().GetId() != "bk-1" {
			t.Errorf("key.id=%q want bk-1", srv.lastDeleteReq.GetKey().GetId())
		}
	})

	t.Run("DeleteBookSoft", func(t *testing.T) {
		_, err := libCli.DeleteBookSoft(ctx, &libpb.DeleteBookSoftRequest{
			Key: &bookpb.BookId{Id: "bk-soft"},
		})
		if err != nil {
			t.Fatalf("DeleteBookSoft: %v", err)
		}
		if srv.lastDeleteSoft.GetKey().GetId() != "bk-soft" {
			t.Errorf("key.id=%q want bk-soft", srv.lastDeleteSoft.GetKey().GetId())
		}
	})

	t.Run("GetBookMeta", func(t *testing.T) {
		resp, err := libCli.GetBookMeta(ctx, &libpb.GetBookMetaRequest{
			Key: &bookpb.BookId{Id: "bk-meta"},
		})
		if err != nil {
			t.Fatalf("GetBookMeta: %v", err)
		}
		if resp.GetBookMeta().GetTitle() != "T" {
			t.Errorf("title=%q want T", resp.GetBookMeta().GetTitle())
		}
		if resp.GetVersion() != 42 {
			t.Errorf("version=%d want 42", resp.GetVersion())
		}
	})

	t.Run("BatchGetBookMetas", func(t *testing.T) {
		resp, err := libCli.BatchGetBookMetas(ctx, &libpb.BatchGetBookMetasRequest{
			Keys: []*bookpb.BookId{{Id: "a"}, {Id: "b"}},
		})
		if err != nil {
			t.Fatalf("BatchGetBookMetas: %v", err)
		}
		if len(resp.GetMetas()) != 2 {
			t.Fatalf("metas len=%d want 2", len(resp.GetMetas()))
		}
		if len(srv.lastBatchGetReq.GetKeys()) != 2 {
			t.Errorf("server received keys=%d want 2", len(srv.lastBatchGetReq.GetKeys()))
		}
	})

	t.Run("ListBooks", func(t *testing.T) {
		resp, err := libCli.ListBooks(ctx, &libpb.ListBooksRequest{
			Limit:   10,
			Offset:  5,
			Filter:  &bookpb.BookMetaFilter{Author: "X"},
			OrderBy: "title",
		})
		if err != nil {
			t.Fatalf("ListBooks: %v", err)
		}
		if resp.GetTotalSize() != 1 {
			t.Errorf("totalSize=%d want 1", resp.GetTotalSize())
		}
		if len(resp.GetItems()) != 1 {
			t.Errorf("items len=%d want 1", len(resp.GetItems()))
		}
		if srv.lastListReq.GetLimit() != 10 {
			t.Errorf("limit=%d want 10", srv.lastListReq.GetLimit())
		}
		if srv.lastListReq.GetFilter().GetAuthor() != "X" {
			t.Errorf("filter.author=%q want X", srv.lastListReq.GetFilter().GetAuthor())
		}
	})

	t.Run("UpdateBookMeta", func(t *testing.T) {
		resp, err := libCli.UpdateBookMeta(ctx, &libpb.UpdateBookMetaRequest{
			Key:        &bookpb.BookId{Id: "bk-up"},
			Meta:       &bookpb.BookMeta{Title: "v2"},
			Version:    42,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		})
		if err != nil {
			t.Fatalf("UpdateBookMeta: %v", err)
		}
		if resp.GetVersion() != 43 {
			t.Errorf("version=%d want 43", resp.GetVersion())
		}
		if srv.lastUpdateMeta.GetKey().GetId() != "bk-up" {
			t.Errorf("key.id=%q want bk-up", srv.lastUpdateMeta.GetKey().GetId())
		}
		if srv.lastUpdateMeta.GetMeta().GetTitle() != "v2" {
			t.Errorf("meta.title=%q want v2", srv.lastUpdateMeta.GetMeta().GetTitle())
		}
		if len(srv.lastUpdateMeta.GetUpdateMask().GetPaths()) != 1 || srv.lastUpdateMeta.GetUpdateMask().GetPaths()[0] != "title" {
			t.Errorf("update_mask paths=%v", srv.lastUpdateMeta.GetUpdateMask().GetPaths())
		}
	})

	t.Run("GetBookContent", func(t *testing.T) {
		resp, err := libCli.GetBookContent(ctx, &libpb.GetBookContentRequest{
			Key: &bookpb.BookId{Id: "bk-c"},
		})
		if err != nil {
			t.Fatalf("GetBookContent: %v", err)
		}
		if resp.GetBookContent().GetText() != "hello" {
			t.Errorf("text=%q want hello", resp.GetBookContent().GetText())
		}
	})

	t.Run("UpdateBookContent", func(t *testing.T) {
		_, err := libCli.UpdateBookContent(ctx, &libpb.UpdateBookContentRequest{
			Key:     &bookpb.BookId{Id: "bk-c2"},
			Content: &bookpb.BookContent{Text: "updated"},
		})
		if err != nil {
			t.Fatalf("UpdateBookContent: %v", err)
		}
		if srv.lastUpdateCont.GetKey().GetId() != "bk-c2" {
			t.Errorf("key.id=%q want bk-c2", srv.lastUpdateCont.GetKey().GetId())
		}
		if srv.lastUpdateCont.GetContent().GetText() != "updated" {
			t.Errorf("text=%q want updated", srv.lastUpdateCont.GetContent().GetText())
		}
	})

	t.Run("ArchiveBook custom method", func(t *testing.T) {
		resp, err := libCli.ArchiveBook(ctx, &bookpb.ArchiveBookRequest{
			BookId: "bk-001",
		})
		if err != nil {
			t.Fatalf("ArchiveBook: %v", err)
		}
		if !resp.GetArchived() {
			t.Error("archived=false want true")
		}
		if srv.lastArchiveReq.GetBookId() != "bk-001" {
			t.Errorf("book_id=%q want bk-001", srv.lastArchiveReq.GetBookId())
		}
	})

	t.Run("StreamBookMetas server-streaming custom method", func(t *testing.T) {
		stream, err := libCli.StreamBookMetas(ctx, &bookpb.StreamBookMetasRequest{
			Author: "Alice",
		})
		if err != nil {
			t.Fatalf("StreamBookMetas: %v", err)
		}
		var received []*bookpb.StreamBookMetasResponse
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("stream.Recv: %v", err)
			}
			received = append(received, resp)
		}
		if len(received) != 2 {
			t.Fatalf("received %d items, want 2", len(received))
		}
		if received[0].GetMeta().GetTitle() != "stream-1" {
			t.Errorf("first item title=%q want stream-1", received[0].GetMeta().GetTitle())
		}
		if received[1].GetMeta().GetTitle() != "stream-2" {
			t.Errorf("second item title=%q want stream-2", received[1].GetMeta().GetTitle())
		}
		if srv.lastStreamMetasReq.GetAuthor() != "Alice" {
			t.Errorf("server received author=%q want Alice", srv.lastStreamMetasReq.GetAuthor())
		}
	})
}

// TestGRPC_AdminServiceNarrowedMethods verifies AdminService has narrowed methods.
func TestGRPC_AdminServiceNarrowedMethods(t *testing.T) {
	adminSrv := &mockAdminServer{}
	_, adminCli, cleanup := newGRPCServer(t, nil, adminSrv)
	defer cleanup()
	ctx := context.Background()

	t.Run("CreateBook", func(t *testing.T) {
		resp, err := adminCli.CreateBook(ctx, &adminpb.CreateBookRequest{
			Meta: &bookpb.BookMeta{Title: "admin-book"},
		})
		if err != nil {
			t.Fatalf("CreateBook: %v", err)
		}
		if resp.GetKey().GetId() != "admin-new" {
			t.Errorf("key.id=%q want admin-new", resp.GetKey().GetId())
		}
	})

	t.Run("DeleteBook", func(t *testing.T) {
		_, err := adminCli.DeleteBook(ctx, &adminpb.DeleteBookRequest{
			Key: &bookpb.BookId{Id: "bk-del"},
		})
		if err != nil {
			t.Fatalf("DeleteBook: %v", err)
		}
	})

	t.Run("DeleteBookSoft", func(t *testing.T) {
		_, err := adminCli.DeleteBookSoft(ctx, &adminpb.DeleteBookSoftRequest{
			Key: &bookpb.BookId{Id: "bk-soft"},
		})
		if err != nil {
			t.Fatalf("DeleteBookSoft: %v", err)
		}
	})

	t.Run("GetBookMeta", func(t *testing.T) {
		resp, err := adminCli.GetBookMeta(ctx, &adminpb.GetBookMetaRequest{
			Key: &bookpb.BookId{Id: "admin-1"},
		})
		if err != nil {
			t.Fatalf("GetBookMeta: %v", err)
		}
		if resp.GetBookMeta().GetTitle() != "admin-meta" {
			t.Errorf("title=%q want admin-meta", resp.GetBookMeta().GetTitle())
		}
		if resp.GetVersion() != 7 {
			t.Errorf("version=%d want 7", resp.GetVersion())
		}
	})

	t.Run("ListBooks", func(t *testing.T) {
		resp, err := adminCli.ListBooks(ctx, &adminpb.ListBooksRequest{
			Limit: 5,
		})
		if err != nil {
			t.Fatalf("ListBooks: %v", err)
		}
		if resp.GetTotalSize() != 1 {
			t.Errorf("totalSize=%d want 1", resp.GetTotalSize())
		}
		if adminSrv.lastListReq.GetLimit() != 5 {
			t.Errorf("limit=%d want 5", adminSrv.lastListReq.GetLimit())
		}
	})

	t.Run("UpdateBookMeta", func(t *testing.T) {
		resp, err := adminCli.UpdateBookMeta(ctx, &adminpb.UpdateBookMetaRequest{
			Key:  &bookpb.BookId{Id: "admin-up"},
			Meta: &bookpb.BookMeta{Title: "v2"},
		})
		if err != nil {
			t.Fatalf("UpdateBookMeta: %v", err)
		}
		if resp.GetVersion() != 8 {
			t.Errorf("version=%d want 8", resp.GetVersion())
		}
	})
}

// prevent unused import errors
var (
	_ = emptypb.Empty{}
	_ = fieldmaskpb.FieldMask{}
)
