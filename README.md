# MTPlugins

Wrapper around [MTServer](https://github.com/sarafanfm/mtserver) that allows you to hide business logic from prying eyes.

## Idea

Sometimes it becomes necessary to restrict access to a part of the project functionality.
For example, you may have several developers, some of whom should not see the billing system algorithms.
In this case, the standard functionality of Go, which is called [plugins](https://pkg.go.dev/plugin), can come in handy.

The idea is that at the build stage of the project, the restricted parts of the project have already been compiled.
More precisely, we will compile all the business logic before building the project.

In other words, this project will help form a strongly-typed `client->api->server->[service]` structure, where the `service` is a functional with limited access.

## System components

In the structure described above, one element is actually missing - `ServiceInterface`.
Correct request chain is `client->api->server->ServiceInterface->[service]`.
So we can store all of it except `service` in one repo, something like `api`.

### API

API repo can contains:
- common `plugin interface`: `service` startup arguments convention
- all `*.proto` files,
- files are the result of proto compilation,
- service interfaces for each service in your project,
- gRPC server implementation that call service methods
- gRPC client that **implement** service interface for intra-server communications

### Repo per service

Each service can have its own repo, into which the `api` repo [git-submodule](https://git-scm.com/docs/git-submodule) is connected.

One service may need to connect to another service. For example service `post` can try to get author from `user` service.
For this, `post` service initializes `user`'s gRPC client as `user` service interface.
And `user`'s gRPC client must implement `user` service interface with calls to `user`'s gRPC server.

Each service must expose the `plugin interface` and must be compiled with `-buildmode=plugin`.

### Executable repo

TODO