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
- [x] Add QOS
    - [x] Shapshot config storage
    - [x] QOS config
    - [ ] CircuitBreaker creation
    - [ ] Handle retries
    - [ ] Add request hedging support
- [ ] Handle url path params
- [x] Handle url query params
- [x] Handle `HEAD` method
- [x] Handle `GET` method
- [x] Handle `POST` method
- [x] Handle `PUT` method
- [x] Handle `DELETE` method
- [ ] Generate multi-file references (e.g. file A has `$ref:
  "../fileB.yaml#/definitions/SomeType"`)
- [ ] Generate `oneOf` and `anyOf` types

