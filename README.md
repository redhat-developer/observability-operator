# Observability Operator (early WIP)

### Running Operator:
  - if image changes required, first do:
    - `make generate`
    - `make manifests`
  - `make install`
  - `make run ENABLE_WEBHOOKS=false`
  - see [operator-sdk documentation](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/) for further info
