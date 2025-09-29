// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// package web provides an ability to parse command line flags and easily run server for both ADK WEB UI and ADK REST API
package web

import (
	"flag"
	"fmt"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a2aproject/a2a-go/a2agrpc"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"google.golang.org/adk/adka2a"
	"google.golang.org/adk/artifactservice"
	"google.golang.org/adk/cmd/restapi/config"
	"google.golang.org/adk/cmd/restapi/services"
	restapiweb "google.golang.org/adk/cmd/restapi/web"
	"google.golang.org/adk/sessionservice"
	"google.golang.org/grpc"
)

// WebConfig is a struct with parameters to run a WebServer.
type WebConfig struct {
	LocalPort      int
	UIDistPath     string
	FrontEndServer string
	StartRestApi   bool
	StartWebUI     bool
	StartA2A       bool
}

// ParseArgs parses the arguments for the ADK API server.
func ParseArgs() *WebConfig {
	localPortFlag := flag.Int("port", 8080, "Port to listen on")
	frontendServerFlag := flag.String("front_address", "localhost:8001", "Front address to allow CORS requests from")
	startRespApi := flag.Bool("start_restapi", true, "Set to start a rest api endpoint '/api'")
	startWebUI := flag.Bool("start_webui", true, "Set to start a web ui endpoint '/ui'")
	webuiDist := flag.String("webui_path", "", "Points to a static web ui dist path with the built version of ADK Web UI")
	startA2A := flag.Bool("a2a", true, "Set to expose a root agent via A2A protocol over gRPC")

	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		panic("Failed to parse flags")
	}
	return &(WebConfig{
		LocalPort:      *localPortFlag,
		FrontEndServer: *frontendServerFlag,
		StartRestApi:   *startRespApi,
		StartWebUI:     *startWebUI,
		UIDistPath:     *webuiDist,
		StartA2A:       *startA2A,
	})
}

func logRequestHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r)
	}
	return http.HandlerFunc(fn)
}

type ServeConfig struct {
	SessionService  sessionservice.Service
	AgentLoader     services.AgentLoader
	ArtifactService artifactservice.Service

	A2AOptions []a2asrv.RequestHandlerOption
}

// Serve initiates the http server and starts it according to WebConfig parameters
func Serve(c *WebConfig, serveConfig *ServeConfig) {
	serverConfig := config.ADKAPIRouterConfigs{
		SessionService:  serveConfig.SessionService,
		AgentLoader:     serveConfig.AgentLoader,
		ArtifactService: serveConfig.ArtifactService,
	}
	serverConfig.Cors = *cors.New(cors.Options{
		AllowedOrigins:   []string{c.FrontEndServer},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions, http.MethodDelete, http.MethodPut},
		AllowCredentials: true,
	})

	rBase := mux.NewRouter().StrictSlash(true)
	_ = logRequestHandler
	// rBase.Use(logRequestHandler)

	if c.StartWebUI {
		rUi := rBase.Methods("GET").PathPrefix("/ui/").Subrouter()
		rUi.Methods("GET").Handler(http.StripPrefix("/ui/", http.FileServer(http.Dir(c.UIDistPath))))
	}

	if c.StartRestApi {
		rApi := rBase.Methods("GET", "POST", "DELETE").PathPrefix("/api/").Subrouter()
		rApi.Use(serverConfig.Cors.Handler)
		restapiweb.SetupRouter(rApi, &serverConfig)
	}

	var handler http.Handler
	if c.StartA2A {
		grpcSrv := grpc.NewServer()
		newA2AHandler(serveConfig).RegisterWith(grpcSrv)
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
				grpcSrv.ServeHTTP(w, r)
			} else {
				rBase.ServeHTTP(w, r)
			}
		})
	} else {
		handler = rBase
	}

	handler = h2c.NewHandler(handler, &http2.Server{})

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(c.LocalPort), handler))
}

func newA2AHandler(serveConfig *ServeConfig) *a2agrpc.GRPCHandler {
	agent := serveConfig.AgentLoader.Root()
	executor := adka2a.NewExecutor(&adka2a.ExecutorConfig{
		AppName:         agent.Name(),
		Agent:           agent,
		SessionService:  serveConfig.SessionService,
		ArtifactService: serveConfig.ArtifactService,
	})
	reqHandler := a2asrv.NewHandler(executor, serveConfig.A2AOptions...)
	grpcHandler := a2agrpc.NewHandler(&adka2a.CardProducer{Agent: agent}, reqHandler)
	return grpcHandler
}
