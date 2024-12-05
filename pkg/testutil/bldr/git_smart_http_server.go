package bldr

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

// StartGitSmartHTTPServer starts a Git Smart HTTP Server.
// Pushing to the Git Server is not supported.
// Shallow clones are not supported.
// Not thread-safe.
func StartGitSmartHTTPServer(t *testing.T, repo *git.Repository) string {
	listener, port := TCPPort(t).Listen("0")

	gitServer := NewGitSmartHTTPServer(listener, repo)
	gitServer.Serve()
	t.Cleanup(gitServer.Close)

	return fmt.Sprintf("http://127.0.0.1:%s/", port)
}

type GitSmartHTTPServer struct {
	repo       *git.Repository
	listener   net.Listener
	httpServer *http.Server
}

func NewGitSmartHTTPServer(listener net.Listener, repo *git.Repository) *GitSmartHTTPServer {
	return &GitSmartHTTPServer{
		repo:       repo,
		listener:   listener,
		httpServer: nil,
	}
}

func (s *GitSmartHTTPServer) Serve() {
	// See https://git-scm.com/docs/http-protocol/2.34.0#_smart_clients
	mux := http.NewServeMux()
	mux.HandleFunc("/info/refs", s.handleAdvertizedRefs)
	mux.HandleFunc("/git-upload-pack", s.handleUploadPack)
	s.httpServer = &http.Server{Handler: mux}

	go func() { _ = s.httpServer.Serve(s.listener) }()
}

func (s *GitSmartHTTPServer) Close() {
	if s.httpServer != nil {
		_ = s.httpServer.Close()
	}
}

func (s *GitSmartHTTPServer) handleAdvertizedRefs(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("service") != "git-upload-pack" {
		http.Error(w, `"Dumb" HTTP Git clients are not supported`, http.StatusNotImplemented)
		return
	}

	session, err := s.establishUploadPackSession()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to establish git-upload-pack session: %v", err), http.StatusInternalServerError)
		return
	}

	advRefs, err := session.AdvertisedReferencesContext(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve the advertised references: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/x-git-upload-pack-advertisement")
	w.Header().Add("Cache-Control", "no-cache")
	advRefs.Prefix = append(advRefs.Prefix, []byte("# service=git-upload-pack"), pktline.Flush)
	err = advRefs.Encode(w)
	if err != nil {
		panic(err) // too late to write the error to the response
	}
}

func (s *GitSmartHTTPServer) handleUploadPack(w http.ResponseWriter, r *http.Request) {
	uploadReq := packp.NewUploadPackRequest()
	err := uploadReq.Decode(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode HTTP request body: %v", err), http.StatusBadRequest)
		return
	}

	session, err := s.establishUploadPackSession()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to establish git-upload-pack session: %v", err), http.StatusInternalServerError)
		return
	}

	uploadResponse, err := session.UploadPack(r.Context(), uploadReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to upload pack: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/x-git-upload-pack-result")
	w.Header().Add("Cache-Control", "no-cache")
	err = uploadResponse.Encode(w)
	if err != nil {
		panic(err) // too late to write the error to the response
	}
}

func (s *GitSmartHTTPServer) establishUploadPackSession() (transport.UploadPackSession, error) {
	endpoint, _ := transport.NewEndpoint("/")
	gitServer := server.NewServer(server.MapLoader{endpoint.String(): s.repo.Storer})
	return gitServer.NewUploadPackSession(endpoint, nil)
}
