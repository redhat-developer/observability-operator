# Observability Operator (early WIP)

### Running Operator:
- operator largely intended to be ran locally via `make` for now:
  - if changes are made, first do:
    - `make generate`
    - `make manifests`
  - `make install`
  - `make run ENABLE_WEBHOOKS=false`
  - see [operator-sdk documentation](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/) for further info

**NOTE**: when running the operator, it will:
- create its own operand (Observability CR) on start-up
  - reference `main.go:L82-96`
- create any resources found in the `config/observability` directory
  - some value substitutions are made, see `observability_controller.go:L143-145`
- delete the CR on `SIGINT/KILL` which will garbage-collect all owned resources it creates 
  - reference `main.go:L107-114`
