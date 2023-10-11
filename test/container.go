package test

import (
	"context"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
	"testing"
)

// A Container is our wrapper around a testcontainers.Container
type Container struct {
	ctx context.Context
	c   testcontainers.Container
}

// MappedPort returns the mapped port for the exposed port from the container.
func (c *Container) MappedPort(port nat.Port) string {
	mappedPort, err := c.c.MappedPort(c.ctx, port)
	if err != nil {
		panic(err)
	}

	return mappedPort.Port()
}

// ComposeRequest contains the information needed to start a compose container.
type ComposeRequest struct {
	PathToComposeFile string
	Services          []ComposeService
}

// ComposeService are the individual services in a compose container.
type ComposeService struct {
	Name           string
	WaitStrategies []wait.Strategy
}

// Convenience function to create the most common container request.
func DefaultContainerRequest(dockerContext, mysqlPass, mysqlDb string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context: dockerContext,
			//Dockerfile:    "",
			PrintBuildLog: true,
		},
		ExposedPorts: []string{"22/tcp", "80/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": mysqlPass,
			"MYSQL_DATABASE":      mysqlDb,
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("22/tcp"),
			wait.ForHTTP("/").WithPort("80/tcp"),
			wait.ForLog("port: 3306  MySQL Community Server - GPL"),
		),
	}
}

// Convenience function to create the most common compose request.
func DefaultComposeRequest(pathToComposeFile string) ComposeRequest {
	return ComposeRequest{
		pathToComposeFile,
		[]ComposeService{
			{
				"wordpress",
				[]wait.Strategy{
					wait.ForListeningPort("22/tcp"),
					wait.ForHTTP("/").WithPort("80/tcp"),
				},
			},
			{
				"db",
				[]wait.Strategy{
					wait.ForLog("port: 3306  MySQL Community Server - GPL"),
				},
			},
		},
	}
}

// StartContainer uses testcontainers to start a container. It handles wrapping the container in our own Container struct
// as well as cleaning up the container when the test is finished.
func StartContainer(t *testing.T, req testcontainers.ContainerRequest) *Container {
	t.Helper()

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			panic(err)
		}
	})

	return &Container{ctx: ctx, c: container}
}

// StartComposeContainers uses testcontainers to start a containers from a compose file. It handles wrapping the
// containers in our own Container struct as well as cleaning up the containers when the test is finished. The individual
// containers are returned in a map keyed by their service name.
func StartComposeContainers(t *testing.T, req ComposeRequest) map[string]*Container {
	compose, err := tc.NewDockerCompose(req.PathToComposeFile)
	if err != nil {
		t.Errorf("Error creating container: %s", err)
	}

	t.Cleanup(func() {
		err = compose.Down(context.Background(), tc.RemoveOrphans(true), tc.RemoveImagesAll, tc.RemoveVolumes(true))
		if err != nil {
			t.Errorf("Error stopping container: %s", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Loop through all the services in the request and add their wait strategies
	for _, service := range req.Services {
		for _, waitStrategy := range service.WaitStrategies {
			compose.WaitForService(service.Name, waitStrategy)
		}
	}

	err = compose.Up(ctx, tc.Wait(true))

	if err != nil {
		t.Errorf("Error starting container: %s", err)
	}

	containers := make(map[string]*Container)

	for _, service := range req.Services {
		serviceContainer, err := compose.ServiceContainer(ctx, service.Name)
		if err != nil {
			t.Errorf("Error getting service container: %s", err)
		}

		containers[service.Name] = &Container{ctx: ctx, c: serviceContainer}
	}

	return containers
}
