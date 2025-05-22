package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	store "github.com/NextronSystems/finding-store"
	"github.com/NextronSystems/jsonlog/thorlog/parser"
	"github.com/NextronSystems/jsonlog/thorlog/v3"
)

const maxDataInMemory = 100 * 1024 * 1024 // 100 MB

func main() {
	var flags = flag.NewFlagSet("server", flag.ExitOnError)
	address := flags.String("address", ":8080", "address to bind to")
	storePath := flags.String("storePath", "./findings", "path where findings are stored")
	flat := flags.Bool("flat", false, "use flat storage (no subdirectories)")

	_ = flags.Parse(os.Args[1:])

	findingStore := store.New(*storePath)
	findingStore.Flat = *flat

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := os.MkdirAll(*storePath, 0700); err != nil {
		slog.Error("Could not create finding store directory", "error", err.Error())
		os.Exit(1)
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("POST /upload", func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseMultipartForm(maxDataInMemory); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(writer, "could not parse multipart form: %s", err.Error())
			slog.Info("could not parse multipart form", "error", err.Error())
			return
		}

		event, err := parser.ParseEvent([]byte(request.FormValue(FindingField)))
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(writer, "could not decode finding: %s", err.Error())
			slog.Info("could not decode finding", "error", err.Error())
			return
		}
		finding, ok := event.(*thorlog.Finding)
		if !ok {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(writer, "invalid finding type: %T", event)
			slog.Info("invalid finding type", "type", fmt.Sprintf("%T", event))
			return
		}
		content, _, err := request.FormFile(ContentField)
		if err != nil && !errors.Is(err, http.ErrMissingFile) {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(writer, "could not get content: %s", err.Error())
			slog.Info("could not get content", "error", err.Error())
			// Continue to store at least the finding
		}

		if err := findingStore.Store(finding, content); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(writer, "could not store finding: %s", err.Error())
			slog.Error("could not store finding", "finding", finding, "error", err.Error())
			return
		}
	})
	err := http.ListenAndServe(*address, serveMux)
	if err != nil {
		slog.Error("Could not start server", "error", err.Error())
	}
}
