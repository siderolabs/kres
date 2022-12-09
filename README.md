# Kres

Kres is a tool to automate generation of build instructions based on project structure.

At the moment only Go projects are supported.
Kres is opinionated, that's by design.

Following output files are generated automatically:

* `Makefile`
* `Dockerfile`
* `.drone.yaml`
* `.dockerignore`
* `.gitignore`
* `.golangci.yml`
* `.markdownlint.json`
* `.golangci.yaml`
* `.codecov.yml`
* `LICENSE`

## Access Tokens

Kres can leverage API access tokens to set up build environment or settings for the project:

* `GITHUB_TOKEN` environment variable should contain GitHub API personal access token with `repo` scope.

## Running Kres

When running Kres for the first time, run it manually via Docker container:

    docker run --rm -v ${PWD}:/src -w /src -e GITHUB_TOKEN ghcr.io/siderolabs/kres:latest

To update build instructions:

    make rekres
