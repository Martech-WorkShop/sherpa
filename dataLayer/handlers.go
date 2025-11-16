// In file: handlers.go
package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

// renderTemplate is a helper function to parse and execute templates.
func renderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	// We parse the layout and the specific template file together.
	t, err := template.ParseFiles("templates/layout.html", "templates/"+tmplName)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Execute the template. Since layout.html is the first file parsed, it's the one that will be executed.
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
	}
}

// DashboardData holds all the data needed for the main dashboard template.
type DashboardData struct {
	Pieces   []ContentPiece
	Contlets []Contlet
	Tags     []Tag
}

// dashboardHandler renders the main dashboard page.
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	pieces, err := getAllContentPieces()
	if err != nil {
		http.Error(w, "Failed to retrieve content pieces: "+err.Error(), http.StatusInternalServerError)
		return
	}

	contlets, err := getAllContlets()
	if err != nil {
		http.Error(w, "Failed to retrieve contlets: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tags, err := getAllTags()
	if err != nil {
		http.Error(w, "Failed to retrieve tags: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := DashboardData{
		Pieces:   pieces,
		Contlets: contlets,
		Tags:     tags,
	}

	renderTemplate(w, "dashboard.html", data)
}

// piecesHandler displays a list of all content pieces.
func piecesHandler(w http.ResponseWriter, r *http.Request) {
	pieces, err := getAllContentPieces()
	if err != nil {
		http.Error(w, "Failed to retrieve content pieces: "+err.Error(), http.StatusInternalServerError)
		return
	}
	renderTemplate(w, "pieces.html", pieces)
}

// contletsHandler displays a list of all contlets.
func contletsHandler(w http.ResponseWriter, r *http.Request) {
	contlets, err := getAllContlets()
	if err != nil {
		http.Error(w, "Failed to retrieve contlets: "+err.Error(), http.StatusInternalServerError)
		return
	}
	renderTemplate(w, "contlets.html", contlets)
}

// tagsHandler displays a list of all tags.
func tagsHandler(w http.ResponseWriter, r *http.Request) {
	tags, err := getAllTags()
	if err != nil {
		http.Error(w, "Failed to retrieve tags: "+err.Error(), http.StatusInternalServerError)
		return
	}
	renderTemplate(w, "tags.html", tags)
}

// schemaHandler displays the database schema.
func schemaHandler(w http.ResponseWriter, r *http.Request) {
	schema, err := getSchemaDetails()
	if err != nil {
		http.Error(w, "Failed to retrieve schema: "+err.Error(), http.StatusInternalServerError)
		return
	}
	renderTemplate(w, "schema.html", schema)
}

// updateSchemaHandler handles the submission of the schema editor form.
func updateSchemaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tableName := strings.TrimPrefix(r.URL.Path, "/schema/")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	var columns []ColumnDetail
	for i := 0; ; i++ {
		fieldKey := fmt.Sprintf("col_%d_field", i)
		if _, ok := r.Form[fieldKey]; !ok {
			break // No more columns
		}

		col := ColumnDetail{
			Field: r.FormValue(fmt.Sprintf("col_%d_field", i)),
			Type:  r.FormValue(fmt.Sprintf("col_%d_type", i)),
			Null:  r.FormValue(fmt.Sprintf("col_%d_null", i)),
			Key:   r.FormValue(fmt.Sprintf("col_%d_key", i)),
			Extra: r.FormValue(fmt.Sprintf("col_%d_extra", i)),
		}
		defaultVal := r.FormValue(fmt.Sprintf("col_%d_default", i))
		if defaultVal != "" {
			col.Default = sql.NullString{String: defaultVal, Valid: true}
		}
		columns = append(columns, col)
	}

	if err := updateTableSchema(tableName, columns); err != nil {
		http.Error(w, "Failed to update schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to the schema page to show the changes
	http.Redirect(w, r, "/schema", http.StatusFound)
}

// pieceDetailHandler displays the full details for a single content piece.
func pieceDetailHandler(w http.ResponseWriter, r *http.Request, id int) {
	piece, err := getPieceByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Failed to retrieve piece details: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	renderTemplate(w, "piece_form.html", piece)
}
// piecesRouter is a custom router that handles all requests under /pieces/.
func piecesRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/pieces/")
	parts := strings.Split(path, "/")

	// This is a simple router. A more robust solution might use a regex-based router.
	switch {
	case len(parts) == 1 && parts[0] == "new" && r.Method == http.MethodGet:
		newPieceHandler(w, r)
	case len(parts) == 1 && parts[0] == "create" && r.Method == http.MethodPost:
		createPieceHandler(w, r)
	case len(parts) == 2 && parts[1] == "edit" && r.Method == http.MethodGet:
		// e.g., /pieces/123/edit
		id, err := strconv.Atoi(parts[0])
		if err == nil {
			editPieceHandler(w, r, id)
			return
		}
	case len(parts) == 1 && parts[0] == "update" && r.Method == http.MethodPost:
		updatePieceHandler(w, r)
	case len(parts) == 1 && parts[0] == "delete" && r.Method == http.MethodPost:
		deletePieceHandler(w, r)
	case len(parts) == 1 && parts[0] != "":
		// e.g., /pieces/123
		id, err := strconv.Atoi(parts[0])
		if err == nil {
			pieceDetailHandler(w, r, id)
			return
		}
	default:
		// Default case or more complex routes will be added here.
		http.NotFound(w, r)
	}
}
// newPieceHandler displays a form to create a new content piece object.
func newPieceHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "piece_form.html", nil)
}

// createPieceHandler handles the submission of the new piece form.
func createPieceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	class := r.FormValue("class")

	id, err := createContentPiece(title, class)
	if err != nil {
		http.Error(w, "Failed to create piece: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/pieces/%d", id), http.StatusFound)
}
// editPieceHandler displays a form to edit an existing content piece object.
func editPieceHandler(w http.ResponseWriter, r *http.Request, id int) {
	piece, err := getPieceByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Failed to retrieve piece for editing: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	renderTemplate(w, "piece_form.html", piece)
}

// updatePieceHandler handles the submission of the edit piece form.
func updatePieceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid piece ID for update", http.StatusBadRequest)
		return
	}
	title := r.FormValue("title")
	class := r.FormValue("class")

	if err := updateContentPiece(id, title, class); err != nil {
		http.Error(w, "Failed to update piece: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/pieces/%d", id), http.StatusFound)
}
// deletePieceHandler handles the deletion of a content piece object.
func deletePieceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form for delete: "+err.Error(), http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid piece ID for delete", http.StatusBadRequest)
		return
	}

	if err := deleteContentPiece(id); err != nil {
		http.Error(w, "Failed to delete piece: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to the main pieces list after deletion.
	http.Redirect(w, r, "/pieces", http.StatusFound)
}
// contletsRouter is a custom router for all /contlets/ paths.
func contletsRouter(w http.ResponseWriter, r *http.Request) {
	// Logic to be added
	http.NotFound(w, r)
}