// In file: main.go
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	resetDBFlag := flag.Bool("reset-db", false, "Drop and recreate the database for development.")
	noSampleDataFlag := flag.Bool("no-sample-data", false, "Do not insert sample data into the database.")
	flag.Parse()

	if *resetDBFlag {
		resetDB()
	}
	setupDatabase()
	connectToDB()
	createSchemaFromArchitecture()
	seedSampleData(!*noSampleDataFlag)

	log.Println("Registering application routes...")

	// Serve static files (like fixi.js)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// --- Application Routes ---
	http.HandleFunc("/", dashboardHandler)
	http.HandleFunc("/pieces", piecesHandler)
	http.HandleFunc("/contlets", contletsHandler)
	http.HandleFunc("/contlets/", contletsRouter)
	http.HandleFunc("/tags", tagsHandler)
	http.HandleFunc("/schema", schemaHandler)
	http.HandleFunc("/pieces/", piecesRouter)
	http.HandleFunc("/schema/", updateSchemaHandler)

	log.Println("✅ Application ready: http://localhost:8080")
	if *resetDBFlag {
		log.Println("💡 Tip: Database was reset because the --reset-db flag was used.")
	}
	log.Fatal(http.ListenAndServe(":8080", nil))
}
