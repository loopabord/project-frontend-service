package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	projectv1 "projectfrontendservice/gen/project/v1"
	"projectfrontendservice/gen/project/v1/projectv1connect"

	"connectrpc.com/connect"

	"github.com/nats-io/nats.go"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/form3tech-oss/jwt-go"

	_ "github.com/joho/godotenv/autoload"
)

type ProjectServer struct {
	nc *nats.Conn
}

// func authenticate(_ context.Context, req authn.Request) (any, error) {
// }

// Ping implements projectv1connect.ProjectFrontendServiceHandler.
func (p *ProjectServer) Ping(context.Context, *connect.Request[projectv1.PingRequest]) (*connect.Response[projectv1.PingResponse], error) {
	result := 0.0
	for i := 0; i < 1000000000; i++ {
		result += math.Sqrt(float64(i))
	}
	return connect.NewResponse(&projectv1.PingResponse{}), nil
}

// CreateProject implements projectv1connect.ProjectFrontendServiceHandler.
func (p *ProjectServer) CreateProject(ctx context.Context, req *connect.Request[projectv1.CreateProjectRequest]) (*connect.Response[projectv1.CreateProjectResponse], error) {
	project := req.Msg.GetProject()
	data, err := json.Marshal(project)
	if err != nil {
		return nil, err
	}

	msg, err := p.nc.Request("CreateProject", data, nats.DefaultTimeout)
	if err != nil {
		return nil, err
	}

	var createdProject projectv1.Project
	err = json.Unmarshal(msg.Data, &createdProject)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&projectv1.CreateProjectResponse{Id: createdProject.Id}), nil
}

// DeleteProject implements projectv1connect.ProjectFrontendServiceHandler.
func (p *ProjectServer) DeleteProject(ctx context.Context, req *connect.Request[projectv1.DeleteProjectRequest]) (*connect.Response[projectv1.DeleteProjectResponse], error) {
	id := req.Msg.GetId()

	_, err := p.nc.Request("DeleteProject", []byte(id), nats.DefaultTimeout)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&projectv1.DeleteProjectResponse{}), nil
}

// ReadAllProjects implements projectv1connect.ProjectFrontendServiceHandler.
func (p *ProjectServer) ReadAllProjects(ctx context.Context, req *connect.Request[projectv1.ReadAllProjectsRequest]) (*connect.Response[projectv1.ReadAllProjectsResponse], error) {
	authorId := req.Msg.GetAuthorId()

	msg, err := p.nc.Request("ReadAllProjects", []byte(authorId), nats.DefaultTimeout)
	if err != nil {
		return nil, err
	}

	var projects []*projectv1.Project
	err = json.Unmarshal(msg.Data, &projects)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&projectv1.ReadAllProjectsResponse{Projects: projects}), nil
}

// ReadProject implements projectv1connect.ProjectFrontendServiceHandler.
func (p *ProjectServer) ReadProject(ctx context.Context, req *connect.Request[projectv1.ReadProjectRequest]) (*connect.Response[projectv1.ReadProjectResponse], error) {
	id := req.Msg.GetId()
	msg, err := p.nc.Request("ReadProject", []byte(id), nats.DefaultTimeout)
	if err != nil {
		return nil, err
	}

	var project projectv1.Project
	err = json.Unmarshal(msg.Data, &project)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&projectv1.ReadProjectResponse{Project: &project}), nil
}

// UpdateProject implements projectv1connect.ProjectFrontendServiceHandler.
func (p *ProjectServer) UpdateProject(ctx context.Context, req *connect.Request[projectv1.UpdateProjectRequest]) (*connect.Response[projectv1.UpdateProjectResponse], error) {
	project := req.Msg.GetProject()
	data, err := json.Marshal(project)
	if err != nil {
		return nil, err
	}

	// Accessing 'sub' from context
	token, ok := ctx.Value("user").(*jwt.Token)
	if !ok {
		return nil, errors.New("unable to retrieve JWT token from context")
	}
	claims := token.Claims.(jwt.MapClaims)
	sub := claims["sub"].(string)

	if sub != project.GetAuthorId() {
		return nil, errors.New("unauthorized")
	}

	msg, err := p.nc.Request("UpdateProject", data, nats.DefaultTimeout)
	if err != nil {
		return nil, err
	}

	var updatedProject projectv1.Project
	err = json.Unmarshal(msg.Data, &updatedProject)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&projectv1.UpdateProjectResponse{}), nil
}

func main() {
	natsURL := os.Getenv("NATS_URL")
	log.Println(natsURL)

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Println(err)
	}
	defer nc.Close()

	server := &ProjectServer{nc: nc}
	mux := http.NewServeMux()
	path, handler := projectv1connect.NewProjectFrontendServiceHandler(server)
	mux.Handle(path, handler)

	// Handle CORS headers
	corsWrapper := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow requests from any origin
			w.Header().Set("Access-Control-Allow-Origin", "https://loopabord.nl, http://localhost:5173")
			// Allow specific headers
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version")
			// Allow specific methods
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Call the actual handler
			h.ServeHTTP(w, r)
		})
	}

	// Wrap the CORS middleware around your mux
	corsHandler := corsWrapper(mux)

	// Create the authentication middleware
	jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			claims := token.Claims.(jwt.MapClaims)
			for key, value := range claims {
				fmt.Printf("Claim[%s]: %v\n", key, value)
			}

			// aud := os.Getenv("AUTH0_API_IDENTIFIER")
			// checkAud := token.Claims.(jwt.MapClaims).VerifyAudience(aud, false)
			// if !checkAud {
			// 	return token, errors.New("Invalid audience.")
			// }

			iss := "https://" + os.Getenv("AUTH0_DOMAIN") + "/"
			checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
			if !checkIss {
				return token, errors.New("Invalid issuer.")
			}

			cert, err := getPemCert(token)
			if err != nil {
				return nil, err
			}
			return jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
		},
		SigningMethod: jwt.SigningMethodRS256,
		Extractor: func(r *http.Request) (string, error) {
			cookie, err := r.Cookie("token")
			if err != nil {
				return "", err
			}
			return cookie.Value, nil
		},
	})

	// Apply the JWT middleware to the mux
	protectedHandler := jwtMiddleware.Handler(corsHandler)

	// Start server
	http.ListenAndServe("0.0.0.0:8080", h2c.NewHandler(protectedHandler, &http2.Server{}))
}

// Helper function to fetch the JWT's signing certificate
func getPemCert(token *jwt.Token) (string, error) {
	cert := ""
	resp, err := http.Get("https://" + os.Getenv("AUTH0_DOMAIN") + "/.well-known/jwks.json")

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []struct {
			Kty string   `json:"kty"`
			Kid string   `json:"kid"`
			Use string   `json:"use"`
			N   string   `json:"n"`
			E   string   `json:"e"`
			X5c []string `json:"x5c"`
		} `json:"keys"`
	}

	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		return cert, errors.New("Unable to find appropriate key.")
	}

	return cert, nil
}
