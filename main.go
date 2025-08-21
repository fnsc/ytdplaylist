package main

import (
	"log"
	"os"

	"ytpd/excel"
	"ytpd/metadata"
	"ytpd/playlist"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Use: go run main.go <artist> <excel_file>")
	}

	artist, excelPath := os.Args[1], os.Args[2]

	links, err := excel.ExtractLinks(excelPath)
	if err != nil {
		log.Fatalf("Erro ao ler Excel: %v", err)
	}

	results := playlist.ProcessAll(links, artist)

	for _, result := range results {
		if result.Err != nil {
			log.Printf("Error %s: %v", result.URL, result.Err)
		} else {
			metadata.WriteAllMetadata(result, artist)
			log.Printf("Success: %s", result.Directory)
		}
	}
}
