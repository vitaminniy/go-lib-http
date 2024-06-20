# go-lib-http

`go-lib-http` provides code generator for OpenAPI 3.0 specification standart.

## Help

In order to get help run following command:

```shell
make
```

or

```shell
make help
```

## Generator roadmap

- [ ] Handle inline-defined properties
- [x] Add client constructor
- [ ] Add QOS
    - [ ] Shapshot config storage
    - [ ] QOS config
    - [ ] CircuitBreaker creation
    - [ ] Handle retries
    - [ ] Add request hedging support
- [ ] Handle url path params
- [ ] Handle url query params
- [ ] Handle `HEAD` method
- [x] Handle `GET` method
- [x] Handle `POST` method
- [ ] Handle `PUT` method
- [ ] Handle `DELETE` method
- [ ] Generate multi-file references (e.g. file A has `$ref:
  "../fileB.yaml#/definitions/SomeType"`)
- [ ] Generate `oneOf` and `anyOf` types

