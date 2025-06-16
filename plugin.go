package main

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sync"

	"github.com/NextronSystems/jsonlog/thorlog/v3"
	"github.com/NextronSystems/thor-plugin"
)

const serverUrlEnv = "THOR_PLUGIN_FINDING_WEBHOOK_URL"

func Init(config thor.Configuration, logger thor.Logger, actions thor.RegisterActions) {
	serverUrl := os.Getenv(serverUrlEnv)
	if serverUrl == "" {
		logger.Error("No server URL set for finding webhook plugin, will not upload findings")
		return
	}
	var uploader = findingUploader{
		ServerUrl: serverUrl,
	}
	actions.AddPostProcessingHook(uploader.UploadFile)
}

type findingUploader struct {
	ServerUrl string
}

func (f findingUploader) UploadFile(logger thor.Logger, object thor.MatchedObject) {
	if object.Finding.Score <= 0 {
		return // No need to upload if the score is 0
	}

	pipeReader, pipeWriter := io.Pipe()
	defer func() {
		_ = pipeReader.Close()
	}()
	multipartWriter := multipart.NewWriter(pipeWriter)
	var multipartWriteDone sync.WaitGroup
	multipartWriteDone.Add(1)
	defer multipartWriteDone.Wait()
	go func() {
		defer multipartWriteDone.Done()
		defer func() {
			_ = multipartWriter.Close()
			_ = pipeWriter.Close()
		}()
		part, err := multipartWriter.CreateFormField(FindingField)
		if err != nil {
			logger.Error("Failed to create multipart form field", "error", err.Error())
			return
		}
		err = json.NewEncoder(part).Encode(object.Finding)
		if err != nil {
			logger.Error("Failed to write JSON object to multipart form field", "error", err.Error())
			return
		}

		if object.Content == nil {
			return
		}
		if _, isProcess := object.Finding.Subject.(*thorlog.Process); isProcess {
			// Processes have a content, but since it's not a compact set, but separated into memory regions,
			// trying to copy it as a whole would fail once the first unmapped region is reached.
			return
		}

		part, err = multipartWriter.CreateFormFile(ContentField, "content")
		if err != nil {
			logger.Error("Failed to create multipart form file", "error", err.Error())
			return
		}
		_, err = io.Copy(part, object.Content)
		if err != nil {
			logger.Error("Failed to copy content to multipart form file", "error", err.Error())
			return
		}
	}()
	response, err := http.Post(f.ServerUrl, multipartWriter.FormDataContentType(), pipeReader)
	if err != nil {
		logger.Error("Failed to upload finding", "error", err.Error())
		return
	}
	if response.StatusCode != http.StatusOK {
		logger.Error("Failed to upload finding", "status_code", response.StatusCode)
		return
	}
}
