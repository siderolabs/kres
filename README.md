# Kres

Kres is a tool to automate generation of build instructions based on project structure.

At the moment only Go projects are supported. Kres is opinionated, that's by design.

Following output files are generated automatically:

* `Makefile`
* `Dockerfile`
* `.drone.yaml`
* `.dockerignore`
* `.gitignore`
* `.golangci.yml`
* `LICENSE`

## Running Kres

When running Kres for the first time, run it manually via Docker container:

    docker run --rm -v ${PWD}:/src -w /src autonomy/kres:latest

To update build intstructions:

    make rekres
