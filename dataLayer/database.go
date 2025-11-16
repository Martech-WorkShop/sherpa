// In file: database.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// connectToDB establishes a connection to the MariaDB database.
func connectToDB() {
	var err error
	// Use the hardcoded DSN as requested, enabling multi-statement support for schema creation.
	const dsn_app = "dataLayer_admin:password@tcp(127.0.0.1:3306)/content_db?parseTime=true&multiStatements=true"
	db, err = sql.Open("mysql", dsn_app)
	if err != nil {
		log.Fatal("Database connection string is invalid:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("Could not connect to database 'content_db'. Please ensure it exists and the credentials are correct.", err)
	}
	log.Println("✅ Successfully connected to the database.")
}

// setupDatabase ensures the database exists.
func setupDatabase() {
	const dsn_setup = "dataLayer_admin:password@tcp(127.0.0.1:3306)/"
	tempDB, err := sql.Open("mysql", dsn_setup)
	if err != nil {
		log.Fatal("Failed to connect for DB setup:", err)
	}
	defer tempDB.Close()

	_, err = tempDB.Exec("CREATE DATABASE IF NOT EXISTS content_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	if err != nil {
		log.Fatal("Failed to execute 'CREATE DATABASE':", err)
	}
}

// resetDB drops and recreates the database.
func resetDB() {
	const dsn_setup = "dataLayer_admin:password@tcp(127.0.0.1:3306)/"
	tempDB, err := sql.Open("mysql", dsn_setup)
	if err != nil {
		log.Fatal("Failed to connect for DB reset:", err)
	}
	defer tempDB.Close()

	log.Println("⚠️ --reset-db flag detected. Dropping database...")
	_, err = tempDB.Exec("DROP DATABASE IF EXISTS content_db")
	if err != nil {
		log.Fatal("Failed to drop database:", err)
	}
	log.Println("✅ Database dropped successfully.")
}

// createSchemaFromArchitecture builds the database tables according to the architecture.
func createSchemaFromArchitecture() {
	schemaSQL, err := os.ReadFile("architecture.md")
	if err != nil {
		log.Fatalf("Failed to read architecture.md file: %v", err)
	}

	// Find the start of the actual SQL schema in the markdown file.
	sqlStartIndex := strings.Index(string(schemaSQL), "-- LAYER 0:")
	if sqlStartIndex == -1 {
		log.Fatal("Could not find the start of the SQL schema in architecture.md")
	}
	sqlScriptWithComments := string(schemaSQL)[sqlStartIndex:]

	// Find the end of the SQL script.
	sqlEndIndex := strings.Index(sqlScriptWithComments, "## 5. Application & UI Design")
	if sqlEndIndex == -1 {
		log.Fatal("Could not find the end of the SQL schema in architecture.md")
	}
	sqlScript := sqlScriptWithComments[:sqlEndIndex]

	// Execute the extracted SQL script.
	if _, err := db.Exec(sqlScript); err != nil {
		log.Fatalf("Failed to execute schema script: %v", err)
	}

	log.Println("✅ Database schema created successfully.")
}

// seedSampleData populates the database with high-quality sample data.
func seedSampleData(enabled bool) {
	if !enabled {
		log.Println("Skipping sample data insertion.")
		return
	}

	// Check if data already exists to prevent duplicate seeding
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM entity").Scan(&count)
	if err != nil || count > 1 { // >1 because the entity table might be created but empty
		log.Println("Database already contains data, skipping seed.")
		return
	}

	log.Println("🌱 Seeding database with sample data...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Failed to begin transaction for seeding:", err)
	}

	// Helper to create an entity and return its ID
	createEntity := func() int64 {
		res, err := tx.Exec("INSERT INTO entity () VALUES ()")
		if err != nil {
			tx.Rollback()
			log.Fatal("Failed to create entity:", err)
		}
		id, _ := res.LastInsertId()
		return id
	}

	// -- Create Taxonomies and Tags --
	taxonomyTechID := createEntity()
	_, err = tx.Exec("INSERT INTO taxonomy (id, name, description) VALUES (?, ?, ?)", taxonomyTechID, "Technology", "Programming languages, frameworks, and other tech.")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	tagGoID := createEntity()
	_, err = tx.Exec("INSERT INTO tag (id, taxonomy_id, value) VALUES (?, ?, ?)", tagGoID, taxonomyTechID, "Go")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	// -- Create Content Piece 1: "About This System" --
	piece1ID := createEntity()
	_, err = tx.Exec("INSERT INTO content_piece (id, class, title) VALUES (?, ?, ?)", piece1ID, "blog_post", "About This System")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	// Tag the piece itself
	_, err = tx.Exec("INSERT INTO entity_tags (entity_id, tag_id) VALUES (?, ?)", piece1ID, tagGoID)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	// Create and add contlets to Piece 1
	heading1ID := createEntity()
	_, err = tx.Exec("INSERT INTO contlet_heading (id, text_content, level) VALUES (?, ?, ?)", heading1ID, "Core Philosophy", 1)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	_, err = tx.Exec("INSERT INTO content_piece_contlets (content_piece_id, contlet_id, sort_order) VALUES (?, ?, ?)", piece1ID, heading1ID, 100)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	para1ID := createEntity()
	_, err = tx.Exec("INSERT INTO contlet_paragraph (id, text_content) VALUES (?, ?)", para1ID, "This system is built on MariaDB and Go, following a pragmatic design.")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	_, err = tx.Exec("INSERT INTO content_piece_contlets (content_piece_id, contlet_id, sort_order) VALUES (?, ?, ?)", piece1ID, para1ID, 200)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal("Failed to commit seed data transaction:", err)
	}

	log.Println("✅ Sample data seeded successfully.")
}

// validIdentifier checks for a safe table/column name.
func validIdentifier(s string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, s)
	return matched
}

// ContentPiece defines the structure for a single content piece record.
type ContentPiece struct {
	ID    int
	Class string
	Title string
}

// getAllContentPieces retrieves all content pieces from the database.
func getAllContentPieces() ([]ContentPiece, error) {
	rows, err := db.Query("SELECT id, class, title FROM content_piece ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pieces []ContentPiece
	for rows.Next() {
		var p ContentPiece
		if err := rows.Scan(&p.ID, &p.Class, &p.Title); err != nil {
			return nil, err
		}
		pieces = append(pieces, p)
	}
	return pieces, nil
}

// Contlet defines the structure for a contlet record (of any class).
// Note: This is a simplified view. We will need more detailed structs later.
type Contlet struct {
	ID    int
	Class string
	// We need a way to represent the specific data, like text or src.
	// For a simple list, we can try to coalesce them in the query.
	Content string
}

// getAllContlets retrieves a summary of all contlets.
func getAllContlets() ([]Contlet, error) {
	// This query is a bit complex because it needs to pull data from multiple tables.
	// We use LEFT JOINs and COALESCE to get a single "content" string.
	query := `
	SELECT
		e.id,
		COALESCE(cp.text_content, ci.src, ch.text_content) as content
	FROM entity e
	LEFT JOIN contlet_paragraph cp ON e.id = cp.id
	LEFT JOIN contlet_image ci ON e.id = ci.id
	LEFT JOIN contlet_heading ch ON e.id = ch.id
	WHERE cp.id IS NOT NULL OR ci.id IS NOT NULL OR ch.id IS NOT NULL
	ORDER BY e.id DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contlets []Contlet
	for rows.Next() {
		var c Contlet
		// The class is not easily available in this query. We'll omit it for now for simplicity.
		if err := rows.Scan(&c.ID, &c.Content); err != nil {
			return nil, err
		}
		contlets = append(contlets, c)
	}
	return contlets, nil
}

// Tag defines the structure for a single tag record.
type Tag struct {
	ID    int
	Value string
	// We'll need Taxonomy info later for a more detailed view.
	TaxonomyName string
}

// getAllTags retrieves all tags from the database.
func getAllTags() ([]Tag, error) {
	query := `
	SELECT t.id, t.value, tx.name
	FROM tag t
	JOIN taxonomy tx ON t.taxonomy_id = tx.id
	ORDER BY tx.name, t.value`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Value, &t.TaxonomyName); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// ColumnDetail struct holds schema information for a table column.
type ColumnDetail struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default sql.NullString
	Extra   string
}

// getSchemaDetails retrieves the full schema for all tables.
func getSchemaDetails() (map[string][]ColumnDetail, error) {
	schema := make(map[string][]ColumnDetail)
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	for _, table := range tables {
		descRows, err := db.Query("DESCRIBE `" + table + "`")
		if err != nil {
			return nil, err
		}
		defer descRows.Close()

		var columns []ColumnDetail
		for descRows.Next() {
			var col ColumnDetail
			if err := descRows.Scan(&col.Field, &col.Type, &col.Null, &col.Key, &col.Default, &col.Extra); err != nil {
				return nil, err
			}
			columns = append(columns, col)
		}
		schema[table] = columns
	}
	return schema, nil
}

// updateTableSchema modifies an existing table to match the provided schema details.
// WARNING: This is a simplistic implementation and can be destructive.
func updateTableSchema(tableName string, columns []ColumnDetail) error {
	if !validIdentifier(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	// For simplicity, we'll build a series of ALTER TABLE statements.
	// A more robust solution would compare old and new schemas to generate precise changes.
	var alterClauses []string
	for _, col := range columns {
		if !validIdentifier(col.Field) {
			return fmt.Errorf("invalid column name: %s", col.Field)
		}
		// Basic validation for type - very simplistic
		if strings.ContainsAny(col.Type, ";)'\"") {
			return fmt.Errorf("invalid characters in column type: %s", col.Type)
		}

		clause := fmt.Sprintf("MODIFY COLUMN `%s` %s", col.Field, col.Type)
		if col.Null == "NO" {
			clause += " NOT NULL"
		} else {
			clause += " NULL"
		}
		if col.Default.Valid && col.Default.String != "" {
			clause += fmt.Sprintf(" DEFAULT '%s'", col.Default.String) // simplistic quoting
		}
		if col.Extra != "" {
			clause += " " + col.Extra // e.g., AUTO_INCREMENT
		}
		alterClauses = append(alterClauses, clause)
	}

	if len(alterClauses) == 0 {
		return nil // No changes to make
	}

	query := fmt.Sprintf("ALTER TABLE `%s` %s", tableName, strings.Join(alterClauses, ", "))

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to alter table %s: %w. Query: %s", tableName, err, query)
	}

	return nil
}

// PieceDetail defines the structure for a full content piece with its contlets.
type PieceDetail struct {
	ID       int
	Class    string
	Title    string
	Contlets []ContletDetail
}

// ContletDetail holds the full data for a single contlet.
type ContletDetail struct {
	ID          int
	Class       string // e.g., 'paragraph', 'image', 'heading'
	TextContent string
	Src         string
	AltText     string
	Level       int
}

// getPieceByID retrieves a single content piece and all its constituent contlets.
func getPieceByID(id int) (PieceDetail, error) {
	var piece PieceDetail
	row := db.QueryRow("SELECT id, class, title FROM content_piece WHERE id = ?", id)
	err := row.Scan(&piece.ID, &piece.Class, &piece.Title)
	if err != nil {
		return piece, err
	}

	query := `
		SELECT
			cpc.contlet_id,
			CASE
				WHEN cp.id IS NOT NULL THEN 'paragraph'
				WHEN ci.id IS NOT NULL THEN 'image'
				WHEN ch.id IS NOT NULL THEN 'heading'
				ELSE 'unknown'
			END AS class,
			cp.text_content,
			ci.src,
			ci.alt_text,
			ch.text_content,
			ch.level
		FROM content_piece_contlets cpc
		LEFT JOIN contlet_paragraph cp ON cpc.contlet_id = cp.id
		LEFT JOIN contlet_image ci ON cpc.contlet_id = ci.id
		LEFT JOIN contlet_heading ch ON cpc.contlet_id = ch.id
		WHERE cpc.content_piece_id = ?
		ORDER BY cpc.sort_order ASC`

	rows, err := db.Query(query, id)
	if err != nil {
		return piece, err
	}
	defer rows.Close()

	for rows.Next() {
		var cd ContletDetail
		var paraText, headingText, src, altText sql.NullString
		var level sql.NullInt64

		err := rows.Scan(
			&cd.ID, &cd.Class,
			&paraText, &src, &altText,
			&headingText, &level,
		)
		if err != nil {
			return piece, err
		}

		switch cd.Class {
		case "paragraph":
			cd.TextContent = paraText.String
		case "heading":
			cd.TextContent = headingText.String
			cd.Level = int(level.Int64)
		case "image":
			cd.Src = src.String
			cd.AltText = altText.String
		}
		piece.Contlets = append(piece.Contlets, cd)
	}
	return piece, nil
}

// createContentPiece creates a new content piece object and returns its ID.
func createContentPiece(title, class string) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	// Create a new entity first to get a unique ID.
	res, err := tx.Exec("INSERT INTO entity () VALUES ()")
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to create entity for piece: %w", err)
	}
	id, _ := res.LastInsertId()

	// Now create the content piece with the new ID.
	_, err = tx.Exec("INSERT INTO content_piece (id, title, class) VALUES (?, ?, ?)", id, title, class)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to insert into content_piece: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

// updateContentPiece updates an existing content piece object.
func updateContentPiece(id int, title, class string) error {
	_, err := db.Exec("UPDATE content_piece SET title = ?, class = ? WHERE id = ?", title, class, id)
	if err != nil {
		return fmt.Errorf("failed to update content_piece with id %d: %w", id, err)
	}
	return nil
}

// deleteContentPiece deletes a content piece object.
// It deletes from the 'entity' table, and the CASCADE constraint handles the rest.
func deleteContentPiece(id int) error {
	_, err := db.Exec("DELETE FROM entity WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete entity for piece with id %d: %w", id, err)
	}
	return nil
}
