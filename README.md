# GitHubActionsExporter

GitHubActionsExporter is Prometheus Exporter that collects GitHub Actions statistics of the specified repository.

## Installation

```shell
$ kubectl apply -k manifests
```

## Usage

```shell
$ curl http://github-actions-exporter:9090/metrics | grep github_actions_runs | head -n 5
# HELP github_actions_runs List how many workflow runs each repository actions
# TYPE github_actions_runs gauge
github_actions_runs{repository="kaidotdev/github-actions-exporter",status="completed"} 10
github_actions_runs{repository="kaidotdev/github-actions-exporter",status="in_progress"} 1
github_actions_runs{repository="kaidotdev/github-actions-exporter",status="queued"} 1
```

## How to develop

### `skaffold dev`

```sh
$ make dev
```

### Test

```sh
$ make test
```

### Lint

```sh
$ make lint
```
