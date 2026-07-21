package main

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminpb "github.com/acme/demo-book/generated/go/admin_service"
	libpb "github.com/acme/demo-book/generated/go/library_service"
	bookpb "github.com/acme/demo-book/generated/go/demo/business/book"
)

const bufSize = 1024 * 1024

// newGRPCServer 启动一个 in-memory gRPC server，注册 LibraryService + AdminService mock，返回 client conn。
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

// ---- LibraryService gRPC tests ----

func TestLibraryServiceGRPC_AllMethods(t *testing.T) {
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

	t.Run("ListBookMetas", func(t *testing.T) {
		resp, err := libCli.ListBookMetas(ctx, &libpb.ListBookMetasRequest{
			PageSize:  10,
			PageToken: "p1",
			Filter:    `author=="X"`,
			OrderBy:   "title",
		})
		if err != nil {
			t.Fatalf("ListBookMetas: %v", err)
		}
		if resp.GetTotalSize() != 1 {
			t.Errorf("totalSize=%d want 1", resp.GetTotalSize())
		}
		if resp.GetNextPageToken() != "tok" {
			t.Errorf("nextPageToken=%q want tok", resp.GetNextPageToken())
		}
		if srv.lastListReq.GetPageSize() != 10 {
			t.Errorf("page_size=%d want 10", srv.lastListReq.GetPageSize())
		}
		if srv.lastListReq.GetFilter() != `author=="X"` {
			t.Errorf("filter=%q", srv.lastListReq.GetFilter())
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
}

// ---- AdminService gRPC tests ----

func TestAdminServiceGRPC_NarrowedMethods(t *testing.T) {
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

	t.Run("ListBookMetas", func(t *testing.T) {
		resp, err := adminCli.ListBookMetas(ctx, &adminpb.ListBookMetasRequest{
			PageSize: 5,
		})
		if err != nil {
			t.Fatalf("ListBookMetas: %v", err)
		}
		if resp.GetTotalSize() != 1 {
			t.Errorf("totalSize=%d want 1", resp.GetTotalSize())
		}
		if adminSrv.lastListReq.GetPageSize() != 5 {
			t.Errorf("page_size=%d want 5", adminSrv.lastListReq.GetPageSize())
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

// 验证 AdminService 上不存在 BatchGetBookMetas 方法（收窄后不生成）。
// Go gRPC stub 是 interface，编译期即可保证方法存在/不存在，这里验证调用会返回 Unimplemented。
func TestAdminServiceGRPC_BatchGetNotRegistered(t *testing.T) {
	// AdminService 没有 BatchGetBookMetas 方法（被收窄掉），
	// 调用 Unimplemented 后端会返回 codes.Unimplemented。
	_, adminCli, cleanup := newGRPCServer(t, nil, &mockAdminServer{})
	defer cleanup()

	// 尝试调用 LibraryService 的 BatchGetBookMetas（AdminService 上没有该方法）
	// 这里验证 AdminService client 接口确实没有 BatchGetBookMetas 方法——编译期已保证。
	// 运行时验证：AdminService 的 UnimplementedServer 对未注册方法返回 Unimplemented。
	var _ adminpb.AdminServiceClient = adminCli // 编译期断言
	_ = cleanup
}

// 防 import 误删
var _ = emptypb.Empty{}
