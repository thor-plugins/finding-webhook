package main

// Server side definition of the API used to communicate with the server.
// This must be kept in sync with the client side definition in api.go.

// Fields in the multipart form data that is sent to the server
const (
	FindingField = "finding" // Contains a thorlog.Finding object, JSON encoded
	ContentField = "content" // Contains the object content, if any. If no content exists, should not be set.
)
