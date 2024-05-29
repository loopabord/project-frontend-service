package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	projectv1 "projectfrontendservice/gen/project/v1"
	"projectfrontendservice/gen/project/v1/projectv1connect"

	"connectrpc.com/connect"

	"github.com/nats-io/nats.go"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	_ "github.com/joho/godotenv/autoload"
)

type ProjectServer struct {
	nc *nats.Conn
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
	log.Println(id)
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

	// natsURL := "nats://nats.loopabord.svc.cluster.local:4222"
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Println(err)
	}
	defer nc.Close()

	server := &ProjectServer{nc: nc}
	mux := http.NewServeMux()
	path, handler := projectv1connect.NewProjectFrontendServiceHandler(server)
	mux.Handle(path, handler)

	// CORS middleware
	corsHandler := cors.AllowAll().Handler(mux)

	// Start server
	http.ListenAndServe("0.0.0.0:8080", h2c.NewHandler(corsHandler, &http2.Server{}))
}
