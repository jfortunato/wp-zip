package test

import (
	"context"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
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
