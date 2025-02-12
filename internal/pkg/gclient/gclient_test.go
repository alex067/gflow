package gclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/alex067/gflow/internal/pkg/gcontext"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

var (
	configContext gcontext.GConfigContext
	gcf           gcontext.GConfigFile
	newContext    gcontext.GContext
)

func init() {
	if _, ok := os.LookupEnv("DOCKER_API_VERSION"); !ok {
		os.Setenv("DOCKER_API_VERSION", "1.43")
	}
}

func startGrafanaContainer(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatalf("error initializing docker client: %s", err)
	}

	reader, err := dockerCli.ImagePull(ctx, "docker.io/grafana/grafana:11.4.0", image.PullOptions{})
	if err != nil {
		t.Fatalf("error pulling grafana image: %s", err)
	}
	defer reader.Close()

	// Don't know why we need this but tests won't pass in Github otherwise lol
	buf := new(strings.Builder)
	_, err = io.Copy(buf, reader)
	if err != nil {
		t.Fatalf("error reading pull output: %s", err)
	}

	hostPortBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: "3000",
	}

	containerPort, err := nat.NewPort("tcp", "3000")
	if err != nil {
		t.Fatalf("error mapping container port: %s", err)
	}
	portBindings := nat.PortMap{containerPort: []nat.PortBinding{hostPortBinding}}

	resp, err := dockerCli.ContainerCreate(
		ctx,
		&container.Config{
			Image: "grafana/grafana:11.4.0",
			Env: []string{
				"GF_SECURITY_ADMIN_PASSWORD=admin",
				"GF_SECURITY_ADMIN_USER=admin",
			},
			Healthcheck: &container.HealthConfig{},
		},
		&container.HostConfig{
			PortBindings: portBindings,
		}, nil, nil, "grafana",
	)
	if err != nil {
		t.Fatalf("error creating grafana container: %s", err)
	}

	if err = dockerCli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		t.Fatalf("error starting grafana container: %s", err)
	}

	// Wait for container to be ready
	// Should expect 302 repsonse
	interval := time.Second * 2
	retries := 10

	httpClient := http.DefaultClient
	req, _ := http.NewRequest("GET", "http://127.0.0.1:3000", nil)

	for retry := 0; retry < retries; retry++ {
		time.Sleep(interval)
		t.Logf("pinging grafana... attempt: %d", retry)
		resp, err := httpClient.Do(req)
		if err != nil {
			t.Logf("error processing grafana healthcheck: %s", err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Logf("grafana healthcheck response: %d", resp.StatusCode)
		} else {
			t.Log("grafana container healthcheck passed!")
			break
		}
	}
}

func cleanup(t *testing.T) {
	ctx := context.Background()
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatalf("error initializing docker client: %s", err)
	}
	if err := dockerCli.ContainerRemove(ctx, "grafana", container.RemoveOptions{Force: true}); err != nil {
		t.Fatalf("failed to kill grafana container: %v", err)
	}

	_, absConfigFilePath, _ := gcf.GetAbsolutePath()
	err = os.Remove(absConfigFilePath)
	if err != nil {
		t.Fatal("error cleaning up config file: ", err)
	}
}

func generateServiceAccountToken(t *testing.T) string {
	t.Helper()

	payload := map[string]interface{}{
		"name":       "gflow-admin",
		"role":       "Admin",
		"isDisabled": false,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("error constructing payload: %s", err)
	}

	req, err := http.NewRequest("POST", "http://127.0.0.1:3000/api/serviceaccounts", bytes.NewBuffer(payloadJson))
	if err != nil {
		t.Fatalf("error constructing payload request: %s", err)
	}

	req.SetBasicAuth("admin", "admin")
	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("service account request error: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create service account: %d", resp.StatusCode)
	}

	var serviceAccount struct {
		ID int `json:"id"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&serviceAccount); err != nil {
		t.Fatalf("failed to read service account result from body: %s", err)
	}

	tokenPayload := map[string]interface{}{
		"name": "gflow-token",
		"role": "Admin",
	}

	tokenPayloadJson, err := json.Marshal(tokenPayload)
	if err != nil {
		t.Fatalf("error constructing payload: %s", err)
	}

	req, err = http.NewRequest(
		"POST",
		fmt.Sprintf("http://127.0.0.1:3000/api/serviceaccounts/%d/tokens", serviceAccount.ID),
		bytes.NewBuffer(tokenPayloadJson),
	)
	if err != nil {
		t.Fatalf("error constructing payload request: %s", err)
	}

	req.SetBasicAuth("admin", "admin")
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("service account token request error: %s", err)
	}
	defer resp.Body.Close()

	var result struct {
		Key string `json:"key"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to read token result from body: %s", err)
	}

	return result.Key
}

func TestWatchingDashboard(t *testing.T) {
	startGrafanaContainer(t)
	token := generateServiceAccountToken(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	gcf.Base = dir
	gcf.Directory = "test"
	gcf.Name = "config.yaml"

	newContext.Url = "http://localhost:3000"
	newContext.Name = "test"
	newContext.Authentication.Grafana.Token = token
	newContext.Context.Dashboards.Path = dir
	newContext.Context.Dashboards.GrafanaTenant = "test"

	err := configContext.CreateNewContext(newContext, gcf)
	if err != nil {
		t.Fatal("should create new context: ", err)
	}

	t.Run("test watcher dashboard", func(t *testing.T) {
		gc := &GrafanaClient{
			Url:      "http://127.0.0.1:3000",
			TenantId: "local",
			ApiKey:   token,
			Interval: time.Duration(1) * time.Second,
			Logger:   logger,
			HttpClient: &http.Client{
				Timeout: 60 * time.Second,
			},
		}

		_, filename, _, _ := runtime.Caller(0)
		dir := filepath.Dir(filename)

		dbClient := &GrafanaDashboardClient{}
		dbClient.FilePath = filepath.Join(dir, gcf.Directory, "dashboard_test.json")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan error, 1)
		// chan ensures the goroutine started and sends the signal in time
		started := make(chan struct{}, 1)

		go func() {
			close(started)
			done <- gc.StartWatchingDashboard(ctx, configContext, dbClient)
		}()

		<-started
		time.Sleep(2 * time.Second)

		if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
			t.Fatalf("failed to send interrupt signal: %v", err)
		}

		exitErr := <-done

		if exitErr == nil {
			t.Errorf("expected to receive shutdown signal")
		}

		err := gc.GetDashboardChanges(dbClient)
		if err != nil {
			t.Fatalf("should get dashboard changes: %v", err)
		}

		// Version 1 means we have successfully watched for the temp dashboard in Grafana
		if dbClient.LastVersion != 1 {
			t.Fatalf("expected dashboard version 1, got: %d", dbClient.LastVersion)
		}

		if dbClient.IsDashboardChanged {
			t.Fatalf("expected dashboard changed to be false, got: %t", dbClient.IsDashboardChanged)
		}

		err = gc.SaveChangesToDisk(dbClient)
		if err != nil {
			t.Fatalf("should save changes to disk: %v", err)
		}

		t.Cleanup(func() {
			cleanup(t)
		})
	})
}
