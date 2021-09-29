package server

// Server implements webhook server (i.e., event listener server) for git events and report server for jobs

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	port = 24335

	paramKeyNamespace = "namespace"
	paramKeyAppName   = "appName"
)

var logger = logf.Log.WithName("server")

// Server is an interface of server
type Server interface {
	Start()
}

// server is a HTTP server for git webhook API and report page
type server struct {
	k8sClient client.Client
	router    *mux.Router
}

// New is a constructor of a server
func New(c client.Client, cfg *rest.Config) *server {
	r := mux.NewRouter()

	// Add webhook handler
	r.Methods(http.MethodPost).Subrouter().Handle(webhookPath, &webhookHandler{k8sClient: c})

	return &server{
		k8sClient: c,
		router:    r,
	}
}

// Start starts the server
func (s *server) Start() {
	httpAddr := fmt.Sprintf("0.0.0.0:%d", port)

	logger.Info(fmt.Sprintf("Server is running on %s", httpAddr))
	if err := http.ListenAndServe(httpAddr, s.router); err != nil {
		logger.Error(err, "cannot launch http server")
		os.Exit(1)
	}
}

func logAndRespond(w http.ResponseWriter, log logr.Logger, code int, respMsg, logMsg string) {
	_ = utils.RespondError(w, code, respMsg)
	log.Info(logMsg)
}
