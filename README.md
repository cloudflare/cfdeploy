# Cloudflare Deployment Tool

This tool allows you to easily deploy Docker Image(s) to Marathon.

Features:

* Has a single dependency - Go
* Validates your Marathon YAML file and converts to JSON
* Checks that your Docker images are published *before* deploying to Marathon
* Automatically interpolates image tags via customizable template
    * e.g. `{{ .GitRevCount }}-{{ .GitRevShort }}` = `93-5814f5e`
* Supports multiple deployment targets/environments

## Installation

Assuming:

1. you have a correctly configured `$GOPATH`
2. you have `$GOPATH/bin` in your `$PATH`

```
go get -u github.com/cloudflare/cfdeploy
cfdeploy # shows help message
```

## Setup

Create a `deploy.yaml` file alongside your Marathon `staging.yaml` and `prod.yaml` files such as this:

```
marathon:
  host: marathon.example.com
image:
  repository: index.docker.io
  tagTemplate: "{{ .GitRevCount }}-{{ .GitRevShort }}"

environments:
  prod:
    marathon:
      file: prod.yaml
    images:
      svc:
        name: library/hello-world
  staging:
    marathon:
      file: staging.yaml
    images:
      svc:
        name: library/hello-world
```

Then, modify your Marathon files to have the Docker image replaced into them, e.g:

```
...
    container:
      type: DOCKER
      docker:
        image: {{ index .Images "svc" }}
...
```

Note: the key `"svc"` must match the key under `environments.ENV.images.KEY` in your `deploy.yaml` file.

## Usage

If you have direct (unauthenticated) access to your Marathon instance:

`cfdeploy -e staging`

or

`cfdeploy -e staging -y` to skip the confirmation prompt.

If you need to specify a custom Marathon hostname or headers:

```
cfdeploy -e staging
    -marathon.host my-marathon.example.com \
    -marathon.curlopts '-H "OauthEmail: ..." -H "OauthAccessToken: ..." -H "OauthExpires: ..."'
```
