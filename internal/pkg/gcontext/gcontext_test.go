package gcontext

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var (
	configContext GConfigContext
	newContext    GContext
	gcf           GConfigFile
)

func cleanupFiles(t *testing.T, gcf GConfigFile) {
	_, absConfigFilePath, _ := gcf.GetAbsolutePath()

	err := os.Remove(absConfigFilePath)
	if err != nil {
		t.Fatal("error cleaning up config file: ", err)
	}
}

func TestContextFiles(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	gcf.Base = dir
	gcf.Directory = "test"
	gcf.Name = "config.yaml"

	t.Run("test create context", func(t *testing.T) {
		newContext.Url = "http://localhost:3000"
		newContext.Name = "test"
		newContext.Authentication.Grafana.Token = "test"
		newContext.Context.Dashboards.Path = filepath.Join(dir, "test")
		newContext.Context.Dashboards.GrafanaTenant = "test"

		err := configContext.CreateNewContext(newContext, gcf)
		if err != nil {
			t.Fatal("should create new context: ", err)
		}

		err = configContext.ReadConfigFile(gcf)
		if err != nil {
			t.Fatal("should read new context: ", err)
		}

		gContext, err := configContext.GetContext("test")
		if err != nil {
			t.Fatal("should retrieve new context: ", err)
		}

		if gContext.Name != newContext.Name {
			t.Errorf("got %s, want %s", gContext.Name, newContext.Name)
		}

		t.Cleanup(func() {
			cleanupFiles(t, gcf)
		})
	})
}
