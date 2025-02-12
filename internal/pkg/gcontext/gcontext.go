package gcontext

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type GContextGrafanaResource struct {
	Uid  string `yaml:"uid"`
	Path string `yaml:"path"`
}

type GContext struct {
	Name           string `yaml:"name"`
	Url            string `yaml:"url"`
	Authentication struct {
		Grafana struct {
			Token string `yaml:"token"`
		} `yaml:"grafana"`
	} `yaml:"auth"`
	Context struct {
		Dashboards struct {
			Path          string `yaml:"path"`
			GrafanaTenant string `yaml:"tenant"`
			// Stores the temp generated resources to watch over
			GrafanResources struct {
				FolderUid string                    `yaml:"folderUid"`
				Resources []GContextGrafanaResource `yaml:"resources"`
			} `yaml:"watching"`
		} `yaml:"dashboards"`
	} `yaml:"context"`
}

type GConfigContext struct {
	Contexts       []GContext `yaml:"contexts"`
	CurrentContext string     `yaml:"currentContext"`
}

type GConfigFile struct {
	Base      string
	Name      string
	Directory string
}

var ConfigFileName = "config.yaml"
var ConfigDirectory = ".gsync"

func (c *GConfigFile) GetAbsolutePath() (string, string, error) {
	var err error
	dirname := c.Base
	if dirname == "" {
		dirname, err = os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
	}

	absConfigPath := filepath.Join(dirname, c.Directory)
	absConfigFilePath := filepath.Join(absConfigPath, c.Name)

	return absConfigPath, absConfigFilePath, nil
}

func (c *GContext) TrimInputs() {
	c.Authentication.Grafana.Token = strings.TrimSpace(c.Authentication.Grafana.Token)
	c.Name = strings.TrimSpace(c.Name)
	c.Url = strings.TrimSpace(c.Url)
	c.Context.Dashboards.Path = strings.TrimSpace(c.Context.Dashboards.Path)
	c.Context.Dashboards.GrafanaTenant = strings.TrimSpace(c.Context.Dashboards.GrafanaTenant)
	c.Context.Dashboards.GrafanResources.FolderUid = strings.TrimSpace(c.Context.Dashboards.GrafanResources.FolderUid)
}

func (c *GConfigContext) writeChangesToDisk() error {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	absConfigPath := filepath.Join(dirname, ConfigDirectory, ConfigFileName)
	if _, err := os.Stat(absConfigPath); err != nil {
		return err
	}

	file, err := os.OpenFile(absConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return err
	}
	return nil
}

func (c *GConfigContext) ReadConfigFile(gcf GConfigFile) error {
	_, absConfigFilePath, err := gcf.GetAbsolutePath()
	if err != nil {
		return err
	}

	configFile, err := os.Open(absConfigFilePath)
	if err != nil {
		return err
	}
	defer configFile.Close()

	if configFile != nil {
		decoder := yaml.NewDecoder(configFile)
		if err = decoder.Decode(c); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("empty config file content")
	}

	return nil
}

func (c *GConfigContext) SetCurrentContext(name string, isTemp bool) error {
	isFound := false
	for _, context := range c.Contexts {
		if context.Name == name {
			c.CurrentContext = name
			isFound = true
			break
		}
	}

	if !isFound {
		return fmt.Errorf("provided context not found")
	}

	// Context can be set at runtime through a flag
	if !isTemp {
		err := c.writeChangesToDisk()
		return err
	}
	return nil
}

func (c *GConfigContext) SetNewResource(uid, jsonPath string) error {
	updateResourceFlag := false
	for i, context := range c.Contexts {
		if context.Name == c.CurrentContext {
			for j, resource := range context.Context.Dashboards.GrafanResources.Resources {
				// Skip process if UID is already recorded
				if resource.Uid == uid {
					return nil
				}
				// Replace path with new uid
				if resource.Path == jsonPath {
					c.Contexts[i].Context.Dashboards.GrafanResources.Resources[j].Uid = uid
					updateResourceFlag = true
					break
				}
			}
			if !updateResourceFlag {
				c.Contexts[i].Context.Dashboards.GrafanResources.Resources = append(
					context.Context.Dashboards.GrafanResources.Resources, GContextGrafanaResource{
						Uid:  uid,
						Path: jsonPath,
					},
				)
			}
			break
		}
	}
	// Write changes to disk
	err := c.writeChangesToDisk()
	return err
}

// Appends a new context to the user config file
func (c *GConfigContext) CreateNewContext(
	newContext GContext,
	gcf GConfigFile,
) error {
	newContext.TrimInputs()

	if _, err := os.Stat(newContext.Context.Dashboards.Path); err != nil {
		return fmt.Errorf("dashboard absolute path not found in local filesystem")
	}

	if newContext.Authentication.Grafana.Token == "" {
		return fmt.Errorf("grafana auth token is required")
	}

	if newContext.Context.Dashboards.GrafanaTenant == "" {
		return fmt.Errorf("grafana tenant is required")
	}

	if len(newContext.Url) < 4 {
		return fmt.Errorf("must provide valid url to grafana instance")
	}

	// Prepend with https if not included
	if newContext.Url[0:4] != "http" {
		newContext.Url = fmt.Sprintf("https://%s", newContext.Url)
	}

	// Remove trailing slash
	if string(newContext.Url[len(newContext.Url)-1]) == "/" {
		newContext.Url = newContext.Url[0 : len(newContext.Url)-1]
	}

	if _, err := url.ParseRequestURI(newContext.Url); err != nil {
		return fmt.Errorf("must provide valid url to grafana instance")
	}

	_, absConfigFilePath, err := gcf.GetAbsolutePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(absConfigFilePath); err == nil {
		err = c.ReadConfigFile(gcf)
		if err != nil {
			return err
		}
		if _, err := c.SearchContext(newContext.Name); err == nil {
			if err = c.UpdateContext(newContext); err != nil {
				return err
			}
		} else {
			c.Contexts = append(c.Contexts, newContext)
		}
	} else if os.IsNotExist(err) {
		c.Contexts = append(c.Contexts, newContext)
		c.SetCurrentContext(newContext.Name, false)
	} else {
		return err
	}

	file, err := os.OpenFile(absConfigFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return err
	}

	return nil
}

func (c *GConfigContext) SearchContext(name string) (GContext, error) {
	for _, context := range c.Contexts {
		if context.Name == name {
			return context, nil
		}
	}
	return GContext{}, fmt.Errorf("context does not exist")
}

func (c *GConfigContext) UpdateContext(configContext GContext) error {
	for i, context := range c.Contexts {
		if context.Name == configContext.Name {
			c.Contexts[i] = configContext
			return nil
		}
	}
	return fmt.Errorf("provided context does not exist")
}

func (c *GConfigContext) GetContextNames() []string {
	var contextNames []string
	for _, context := range c.Contexts {
		contextNames = append(contextNames, context.Name)
	}
	return contextNames
}

func (c *GConfigContext) GetContext(contextName string) (GContext, error) {
	for _, context := range c.Contexts {
		if context.Name == contextName {
			return context, nil
		}
	}
	return GContext{}, fmt.Errorf("current context not found in config")
}

func (c *GConfigContext) GetResourceByPath(filePath string) string {
	for _, context := range c.Contexts {
		if context.Name == c.CurrentContext {
			for _, resource := range context.Context.Dashboards.GrafanResources.Resources {
				if resource.Path == filePath {
					return resource.Uid
				}
			}
		}
	}
	return ""
}

func (c *GConfigContext) GetWatchedDashboards() []GContextGrafanaResource {
	var watchedPaths []GContextGrafanaResource
	for _, context := range c.Contexts {
		if context.Name == c.CurrentContext {
			for _, resource := range context.Context.Dashboards.GrafanResources.Resources {
				watchedPaths = append(watchedPaths, GContextGrafanaResource{
					Path: resource.Path,
					Uid:  resource.Uid,
				})
			}
		}
	}
	return watchedPaths
}

func (c *GConfigContext) ClearResourceDashboardByPath(filePath string) error {
	var resourceLength int

	for _, context := range c.Contexts {
		if context.Name == c.CurrentContext {
			resourceLength = len(context.Context.Dashboards.GrafanResources.Resources)
			break
		}
	}

	newResources := make([]GContextGrafanaResource, resourceLength-1)

	for _, context := range c.Contexts {
		if context.Name == c.CurrentContext {
			for i, resource := range context.Context.Dashboards.GrafanResources.Resources {
				if resource.Path == filePath {
					copy(newResources, context.Context.Dashboards.GrafanResources.Resources[0:i])
					if i < len(context.Context.Dashboards.GrafanResources.Resources)-1 {
						copy(newResources[i:], context.Context.Dashboards.GrafanResources.Resources[i+1:])
					}
					break
				}
			}
			break
		}
	}

	for i, context := range c.Contexts {
		if context.Name == c.CurrentContext {
			c.Contexts[i].Context.Dashboards.GrafanResources.Resources = newResources
			break
		}
	}

	err := c.writeChangesToDisk()
	return err
}
