package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alex067/gflow/internal/pkg/gcontext"
	"github.com/manifoldco/promptui"
)

type ContextSelectItem struct {
	Name         string
	PaddedName   string
	Tenant       string
	PaddedTenant string
	Dashboards   string
	Active       bool
	PaddedActive string
}

type DashboardSelectItem struct {
	Name       string
	PaddedName string
	Watching   string
	Path       string
	StripPath  string
}

type MultiSelector struct{}

func (c *MultiSelector) RunContextSelectMenu(currentContext string, configContexts []gcontext.GContext) (string, error) {
	maxWidths := struct {
		name   int
		tenant int
	}{
		name:   10,
		tenant: 10,
	}

	var selectItems []ContextSelectItem

	for _, context := range configContexts {
		if len(context.Name) > maxWidths.name {
			maxWidths.name = len(context.Name) + 10
		}
		if len(context.Context.Dashboards.GrafanaTenant) > maxWidths.tenant {
			maxWidths.tenant = len(context.Context.Dashboards.GrafanaTenant) + 10
		}

		selectItem := ContextSelectItem{
			Name:       context.Name,
			Tenant:     context.Context.Dashboards.GrafanaTenant,
			Dashboards: context.Context.Dashboards.Path,
			Active:     currentContext == context.Name,
		}
		selectItems = append(selectItems, selectItem)
	}

	// Add padded strings
	for i := range selectItems {
		selectItems[i].PaddedName = selectItems[i].Name + strings.Repeat(" ", maxWidths.name-len(selectItems[i].Name))
		selectItems[i].PaddedTenant = selectItems[i].Tenant + strings.Repeat(" ", maxWidths.tenant-len(selectItems[i].Tenant))
		if selectItems[i].Active {
			selectItems[i].PaddedActive = "*" + strings.Repeat(" ", 9)
		} else {
			selectItems[i].PaddedActive = " " + strings.Repeat(" ", 9)
		}
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "{{.PaddedActive}}{{.PaddedName}}{{.PaddedTenant}}{{.Dashboards}}",
		Inactive: "{{.PaddedActive}}{{.PaddedName | faint}}{{.PaddedTenant | faint}}{{.Dashboards | faint}}",
		Selected: "✔ Selected context: {{.Name}}",
	}

	headerFormat := func(name string, maxWidth int) string {
		return name + strings.Repeat(" ", maxWidth-len(name))
	}

	// Create header
	header := "  " + fmt.Sprintf(
		"%s%s%s%s",
		headerFormat("ACTIVE", 10),
		headerFormat("NAME", maxWidths.name),
		headerFormat("TENANT", maxWidths.tenant),
		"DASHBOARDS",
	)

	prompt := promptui.Select{
		Label:        header,
		Items:        selectItems,
		Size:         len(selectItems),
		Templates:    templates,
		HideSelected: false,
		HideHelp:     false,
	}

	index, _, err := prompt.Run()

	if err == promptui.ErrInterrupt || err == promptui.ErrAbort {
		return currentContext, nil
	} else if err != nil {
		return currentContext, err
	}
	return selectItems[index].Name, nil
}

func (c *MultiSelector) RunDashboardSelectMenu(dashboardPath string, watchedDashboards []gcontext.GContextGrafanaResource) (string, error) {
	var selectItems []DashboardSelectItem

	var maxWidth int

	// Read all dashboard files in current dashboard directory
	if err := filepath.Walk(dashboardPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only parse through json files
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			var dashboardSelectItem DashboardSelectItem
			dashboardSelectItem.Name = strings.Trim(filepath.Base(path), "")
			dashboardSelectItem.Path = strings.Trim(path, "")
			dashboardSelectItem.StripPath = path[len(dashboardPath):]
			dashboardSelectItem.Watching = " " + strings.Repeat(" ", 2)

			for _, resource := range watchedDashboards {
				if resource.Path == path {
					dashboardSelectItem.Watching = "*" + strings.Repeat(" ", 2)
					break
				}
			}

			if len(dashboardSelectItem.Name) > maxWidth {
				maxWidth = len(dashboardSelectItem.Name) + 15
			}

			selectItems = append(selectItems, dashboardSelectItem)
		}
		return nil
	}); err != nil {
		return "", err
	}

	for i := range selectItems {
		selectItems[i].PaddedName = selectItems[i].Name + strings.Repeat(" ", maxWidth-len(selectItems[i].Name))
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "{{.Watching}}{{.PaddedName}}{{.StripPath}}",
		Inactive: "{{.Watching}}{{.PaddedName | faint}}{{.StripPath | faint}}",
		Selected: "✔ Selected dashboard: {{.Name }}",
	}

	// Create header
	header := "  " + strings.Repeat(" ", 3) + fmt.Sprintf("%s%s", "NAME"+strings.Repeat(" ", maxWidth-len("NAME")), "RELATIVE PATH")

	prompt := promptui.Select{
		Label:        header,
		Items:        selectItems,
		Size:         len(selectItems),
		Templates:    templates,
		HideSelected: false,
		HideHelp:     false,
	}

	index, _, err := prompt.Run()

	if err == promptui.ErrInterrupt || err == promptui.ErrAbort {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return selectItems[index].Path, nil
}

func (c *MultiSelector) RunGetContextDisplay(currentContext string, configContexts []gcontext.GContext) error {
	maxWidths := struct {
		name   int
		tenant int
	}{
		name:   10,
		tenant: 10,
	}

	var selectItems []ContextSelectItem

	for _, context := range configContexts {
		if len(context.Name) > maxWidths.name {
			maxWidths.name = len(context.Name) + 10
		}
		if len(context.Context.Dashboards.GrafanaTenant) > maxWidths.tenant {
			maxWidths.tenant = len(context.Context.Dashboards.GrafanaTenant) + 10
		}

		selectItem := ContextSelectItem{
			Name:       context.Name,
			Tenant:     context.Context.Dashboards.GrafanaTenant,
			Dashboards: context.Context.Dashboards.Path,
			Active:     currentContext == context.Name,
		}
		selectItems = append(selectItems, selectItem)
	}

	// Add padded strings
	for i := range selectItems {
		selectItems[i].PaddedName = selectItems[i].Name + strings.Repeat(" ", maxWidths.name-len(selectItems[i].Name))
		selectItems[i].PaddedTenant = selectItems[i].Tenant + strings.Repeat(" ", maxWidths.tenant-len(selectItems[i].Tenant))
		if selectItems[i].Active {
			selectItems[i].PaddedActive = "*" + strings.Repeat(" ", 9)
		} else {
			selectItems[i].PaddedActive = " " + strings.Repeat(" ", 9)
		}
	}

	headerFormat := func(name string, maxWidth int) string {
		return name + strings.Repeat(" ", maxWidth-len(name))
	}

	// Create header
	header := fmt.Sprintf(
		"%s%s%s%s",
		headerFormat("ACTIVE", 10),
		headerFormat("NAME", maxWidths.name),
		headerFormat("TENANT", maxWidths.tenant),
		"DASHBOARDS",
	)

	var contextSelectString string
	contextSelectString = header
	for _, selectItem := range selectItems {
		contextSelectString = contextSelectString + fmt.Sprintf(
			"\n%s%s%s%s",
			selectItem.PaddedActive,
			selectItem.PaddedName,
			selectItem.PaddedTenant,
			selectItem.Dashboards,
		)
	}

	fmt.Println(contextSelectString)
	return nil
}
