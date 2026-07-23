package positive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestOpenAPI_SpecValidity verifies swagger.json files are valid JSON with expected paths.
func TestOpenAPI_SpecValidity(t *testing.T) {
	dir := fixtureBookDir(t)

	t.Run("library_service swagger.json is valid JSON", func(t *testing.T) {
		path := filepath.Join(dir, "generated", "openapi", "library_service", "library_service.swagger.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read swagger.json: %v (did you run `apigen build`?)", err)
		}
		var spec map[string]any
		if err := json.Unmarshal(data, &spec); err != nil {
			t.Fatalf("swagger.json is not valid JSON: %v", err)
		}
		paths, ok := spec["paths"].(map[string]any)
		if !ok {
			t.Fatal("swagger.json missing or invalid 'paths' key")
		}
		if len(paths) == 0 {
			t.Error("swagger.json paths is empty")
		}
	})

	t.Run("library_service swagger.json has all routes", func(t *testing.T) {
		path := filepath.Join(dir, "generated", "openapi", "library_service", "library_service.swagger.json")
		data := mustReadFile(t, path)
		var spec map[string]any
		if err := json.Unmarshal([]byte(data), &spec); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		paths := spec["paths"].(map[string]any)

		expectedPaths := []string{
			"/library/LibraryService/book",                           // Create
			"/library/LibraryService/book/{key.id}",                  // Delete
			"/library/LibraryService/book/deleteSoft",                // DeleteSoft
			"/library/LibraryService/book/{key.id}/meta",             // Get/Update meta
			"/library/LibraryService/book/meta/batchGet",             // BatchGet
			"/library/LibraryService/book/meta/list",                 // List
			"/library/LibraryService/book/{key.id}/content",          // Get/Update content
			"/library/LibraryService/book/{book_id}:archive",         // custom method
		}
		for _, p := range expectedPaths {
			if _, ok := paths[p]; !ok {
				t.Errorf("swagger.json paths missing %q; got: %v", p, pathKeys(paths))
			}
		}
	})

	t.Run("library_service swagger.json HTTP verbs are correct", func(t *testing.T) {
		path := filepath.Join(dir, "generated", "openapi", "library_service", "library_service.swagger.json")
		data := mustReadFile(t, path)
		var spec map[string]any
		if err := json.Unmarshal([]byte(data), &spec); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		paths := spec["paths"].(map[string]any)

		// Verify CreateBook path has POST
		createPath, ok := paths["/library/LibraryService/book"].(map[string]any)
		if !ok {
			t.Fatal("Create path not found")
		}
		if _, ok := createPath["post"]; !ok {
			t.Error("CreateBook path missing 'post' verb")
		}

		// Verify DeleteBook path has DELETE
		deletePath, ok := paths["/library/LibraryService/book/{key.id}"].(map[string]any)
		if !ok {
			t.Fatal("Delete path not found")
		}
		if _, ok := deletePath["delete"]; !ok {
			t.Error("DeleteBook path missing 'delete' verb")
		}

		// Verify UpdateBookMeta path has PATCH
		updatePath, ok := paths["/library/LibraryService/book/{key.id}/meta"].(map[string]any)
		if !ok {
			t.Fatal("Update meta path not found")
		}
		if _, ok := updatePath["patch"]; !ok {
			t.Error("UpdateBookMeta path missing 'patch' verb")
		}
		if _, ok := updatePath["get"]; !ok {
			t.Error("GetBookMeta path missing 'get' verb")
		}
	})

	t.Run("admin_service swagger.json is valid", func(t *testing.T) {
		path := filepath.Join(dir, "generated", "openapi", "admin_service", "admin_service.swagger.json")
		data := mustReadFile(t, path)
		var spec map[string]any
		if err := json.Unmarshal([]byte(data), &spec); err != nil {
			t.Fatalf("admin swagger.json is not valid JSON: %v", err)
		}
		paths, ok := spec["paths"].(map[string]any)
		if !ok {
			t.Fatal("admin swagger.json missing 'paths'")
		}
		// AdminService should have ListBookMetas
		if _, ok := paths["/library/AdminService/book/meta/list"]; !ok {
			t.Errorf("admin swagger.json missing List path; got: %v", pathKeys(paths))
		}
		// AdminService should NOT have BatchGetBookMetas (narrowed)
		if _, ok := paths["/library/AdminService/book/meta/batchGet"]; ok {
			t.Error("admin swagger.json should NOT have BatchGet path (narrowed)")
		}
	})
}

func pathKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
