# gflow. ![GitHub release (latest by date)](https://img.shields.io/github/v/release/alex067/gflow) 
gflow is a Grafana resource management tool aimed to streamline the workflow for making changes to Grafana resources. It creates a temporary resource on your Grafana instance of the thing you want to change, and syncs the changes back to your local filesystem.

Managing monitoring resources has never been easy. No one wants to deal with meticulously changing the x,y coordinates of a panel on a json file, which is expected if you manage Grafana resources through IaC. 

By using gflow, all changes are made on a temporary version of that resource right on your Grafana instance. The changes are saved back to your local filesystem as a json file. From there, simply push the changes to your Github and let your IaC do the rest.

This provides the following benefits:  
- 👀 Immediate feedback to changes
- 🚀 Resource files automatically synced
- 😃 Removes manual exporting of json files

The tool is heavily inspired by **kubectl** and **grizzly**.

## Workflow
The tool assumes you are hosting all your resource configs as json or yaml files on your local file system. For example, all your dashboard configs are stored under `/Users/name/monitoring_repo/dashboards/*.json` 

When configuring your context, one of the required fields is providing the absolute path to this directory. From there, simply provide which dashboard to work on and the tool will spin up a temporary resource on your Grafana instance.

## How It Works
The tool expects you to store resource configs as json files in your local filesystem. The tool simply syncs changes back by overwriting the json file.

1. `gflow config create-context` initially create a context pointing to your Grafana instance configuration. Each context points to specific grafana instances and tenants. 
2. `gflow start dashboard` to select a dashboard to work on. The dashboard will then be created on your Grafana instance as a temporary resource. The tool will now start watching for any changes made to this temporary resource.
3. Changes are synced back every 10 seconds or if you interrupt the process.
4. Open a Github PR to push the changes. Since the changes are sycned back by overwriting the json files.

## Configuration & Auth
Contexts allow you to switch between Grafana tenants and instances. The config is stored entirely on your local filesystem under `~/.gflow/config.yaml` and never sent anywhere else.

Run `gflow config use-context` to switch between contexts.

Creating a context requires you to provide the following:
1. Your Grafana instance url
2. Context name 
3. Absolute path to directory containing your dashboard json files
4. Grafana tenant name (if none simply use your context name)
5. Grafana auth token
6. Optional Grafana folder UID to host all your temporary resources

Grafana authentication is provided through your service account token. For roles and permissions, `editor` is required as the tool creates and deletes temporary resources.

## Future Development 
- [ ] Refactor codebase to be modular
- [ ] Enable support for Alerts
- [ ] Enable support for Library panels
- [ ] Enable support for yaml
