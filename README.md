# gflow. ![GitHub release (latest by date)](https://img.shields.io/github/v/release/alex067/gflow) 
---

gflow is a Grafana resource management tool aimed to streamline the workflow for making changes to Grafana resources. It creates a temporary resource on your Grafana instance of the thing you want to change, and syncs the changes back to your local filesystem.

Managing monitoring resources has never been easy. No one wants to deal with meticulously changing the x,y coordinates of a panel on a json file, which is expected if you manage Grafana resources through IaC. 

By using gflow, all changes are made on a temporary version of that resource right on your Grafana instance. The changes are saved back to your local filesystem as a json file. From there, simply push the changes to your Github and let your IaC do the rest.

This also makes it easier to review resource changes and get immediate feedback. Just share the link to your temporary resource. 

The tool is heavily inspired by *kubectl* and *grizzly*.

## How It Works
---

The tool expects you to store resource configs as json files in your local filesystem. The tool simply syncs changes back by overwriting the json file.

1. `gflow config create-context` initially create a context pointing to your Grafana instance. This works similar to `kubectl` where contexts point to specific grafana instances and tenants. If this is your first context, then it's automatically the active context. Run `gflow config use-context` to switch between contexts.
2. `gflow start dashboard` opens a dropdown menu to select a dashboard to work on. The selected dashboard will then be created on your Grafana instance as a temporary resource. The tool will now start watching for any changes made to this temporary resource.
3. Changes are synced back every 10 seconds or if you interrupt the process.
4. Open a new Github PR to push the changes. Since the changes are sycned by overwriting the json files.

## Future Development 
---
Currently the only supported resource are dashboards. Future changes are to refactor the codebase to be modular, where any resource can be added by providing a schema. Additional resources like alerts and library panels will be added. 

The tool also only supports resource configs stored as json files. Possibly supportying yaml is required.