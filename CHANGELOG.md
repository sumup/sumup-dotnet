# Changelog

## [0.0.9](https://github.com/sumup/sumup-dotnet/compare/v0.0.8...v0.0.9) (2026-03-13)


### Chores

* align release automation config ([#55](https://github.com/sumup/sumup-dotnet/issues/55)) ([83265e4](https://github.com/sumup/sumup-dotnet/commit/83265e4c8e7eaaf7804cfb0e27c7f1dc90a5cc7d))
* synced file(s) with sumup/apis ([#54](https://github.com/sumup/sumup-dotnet/issues/54)) ([7d025bd](https://github.com/sumup/sumup-dotnet/commit/7d025bdb39987bf5d36bb6c59c8d9e2275b698d6))

## [0.0.8](https://github.com/sumup/sumup-dotnet/compare/v0.0.7...v0.0.8) (2026-03-13)


### Features

* simpler types for flexible schemas ([ebc9db6](https://github.com/sumup/sumup-dotnet/commit/ebc9db6e79ebaa7112e546b6f66491b18ddb2262))
* update to latest openapi specs ([7133de9](https://github.com/sumup/sumup-dotnet/commit/7133de9f970e2d399e6464ca6afff0caf73072a9))


### Bug Fixes

* **client:** send problem+json accept header ([#50](https://github.com/sumup/sumup-dotnet/issues/50)) ([778b58f](https://github.com/sumup/sumup-dotnet/commit/778b58f594fb7db0105e49c4acbce9511cf8da3e))
* **codegen:** prefer problem+json for response schemas ([#51](https://github.com/sumup/sumup-dotnet/issues/51)) ([92283f4](https://github.com/sumup/sumup-dotnet/commit/92283f48e9f02701379a3055ea8b840c3825fe9c))

## [0.0.7](https://github.com/sumup/sumup-dotnet/compare/v0.0.6...v0.0.7) (2026-02-19)


### Features

* **cd:** format generated code before comitting ([1e218cc](https://github.com/sumup/sumup-dotnet/commit/1e218cc8cfa9d74a949d6fc6b9e9455dc2eea2f4))
* handling of alias schemas / allOf unwrap ([#32](https://github.com/sumup/sumup-dotnet/issues/32)) ([7181e11](https://github.com/sumup/sumup-dotnet/commit/7181e111fb477e884670d551254048b27f609cf8))
* improve error handling ([#20](https://github.com/sumup/sumup-dotnet/issues/20)) ([b0d7ced](https://github.com/sumup/sumup-dotnet/commit/b0d7cedda968dee3847aaa6c50c751f5bf54e97e))
* init ([0a8c79e](https://github.com/sumup/sumup-dotnet/commit/0a8c79ecd90928ec3578dd349500030798d15b21))
* init ([c9ee5f5](https://github.com/sumup/sumup-dotnet/commit/c9ee5f5597bf5e5e38b1236b58a700c8e3898192))
* **query:** support explicit null query parameters ([#34](https://github.com/sumup/sumup-dotnet/issues/34)) ([d42acbe](https://github.com/sumup/sumup-dotnet/commit/d42acbebdd6b0c94b92ff70ee4f804af08206207))
* report runtime info ([#13](https://github.com/sumup/sumup-dotnet/issues/13)) ([f99755f](https://github.com/sumup/sumup-dotnet/commit/f99755fa584d37f0524e956435652e894f90dd65))
* respect readonly fields ([#24](https://github.com/sumup/sumup-dotnet/issues/24)) ([7e959df](https://github.com/sumup/sumup-dotnet/commit/7e959dfe81823d796eae3e0a6287f59163a7aab1))
* **sdk:** package icon ([366b067](https://github.com/sumup/sumup-dotnet/commit/366b06757d8b252cccd46504be6a4b27b802f517))
* **tooling:** dependabot ([01e6594](https://github.com/sumup/sumup-dotnet/commit/01e65944ea8f57fa47644f10edd04daf1dfff4ea))


### Bug Fixes

* **codegen:** deserialize typed JSON responses instead of casting JsonDocument ([#31](https://github.com/sumup/sumup-dotnet/issues/31)) ([29cdd9e](https://github.com/sumup/sumup-dotnet/commit/29cdd9e39023c676a86c7e6e349f712f6d0c80a8))
* **codegen:** lint warnings ([74aaa22](https://github.com/sumup/sumup-dotnet/commit/74aaa22c56be978145fe309207803ff2c8c07398))
* license, package readme, community files ([5d0979e](https://github.com/sumup/sumup-dotnet/commit/5d0979e836347001d48bdef06248d728f0ef592e))
* user-agent version string ([542339a](https://github.com/sumup/sumup-dotnet/commit/542339a714f3072954b8ce8f6274b2ee1c42e9dc))

## [0.0.6](https://github.com/sumup/sumup-dotnet/compare/v0.0.5...v0.0.6) (2026-02-02)


### Features

* **cd:** format generated code before comitting ([1e218cc](https://github.com/sumup/sumup-dotnet/commit/1e218cc8cfa9d74a949d6fc6b9e9455dc2eea2f4))
* improve error handling ([#20](https://github.com/sumup/sumup-dotnet/issues/20)) ([b0d7ced](https://github.com/sumup/sumup-dotnet/commit/b0d7cedda968dee3847aaa6c50c751f5bf54e97e))
* init ([0a8c79e](https://github.com/sumup/sumup-dotnet/commit/0a8c79ecd90928ec3578dd349500030798d15b21))
* init ([c9ee5f5](https://github.com/sumup/sumup-dotnet/commit/c9ee5f5597bf5e5e38b1236b58a700c8e3898192))
* report runtime info ([#13](https://github.com/sumup/sumup-dotnet/issues/13)) ([f99755f](https://github.com/sumup/sumup-dotnet/commit/f99755fa584d37f0524e956435652e894f90dd65))
* **sdk:** package icon ([366b067](https://github.com/sumup/sumup-dotnet/commit/366b06757d8b252cccd46504be6a4b27b802f517))
* **tooling:** dependabot ([01e6594](https://github.com/sumup/sumup-dotnet/commit/01e65944ea8f57fa47644f10edd04daf1dfff4ea))


### Bug Fixes

* **codegen:** lint warnings ([74aaa22](https://github.com/sumup/sumup-dotnet/commit/74aaa22c56be978145fe309207803ff2c8c07398))
* license, package readme, community files ([5d0979e](https://github.com/sumup/sumup-dotnet/commit/5d0979e836347001d48bdef06248d728f0ef592e))
* user-agent version string ([542339a](https://github.com/sumup/sumup-dotnet/commit/542339a714f3072954b8ce8f6274b2ee1c42e9dc))

## [0.0.5](https://github.com/sumup/sumup-dotnet/compare/v0.0.4...v0.0.5) (2026-02-02)


### Features

* improve error handling ([#20](https://github.com/sumup/sumup-dotnet/issues/20)) ([b0d7ced](https://github.com/sumup/sumup-dotnet/commit/b0d7cedda968dee3847aaa6c50c751f5bf54e97e))


### Bug Fixes

* user-agent version string ([542339a](https://github.com/sumup/sumup-dotnet/commit/542339a714f3072954b8ce8f6274b2ee1c42e9dc))

## [0.0.4](https://github.com/sumup/sumup-dotnet/compare/v0.0.3...v0.0.4) (2026-01-20)


### Features

* **cd:** format generated code before comitting ([1e218cc](https://github.com/sumup/sumup-dotnet/commit/1e218cc8cfa9d74a949d6fc6b9e9455dc2eea2f4))
* report runtime info ([#13](https://github.com/sumup/sumup-dotnet/issues/13)) ([f99755f](https://github.com/sumup/sumup-dotnet/commit/f99755fa584d37f0524e956435652e894f90dd65))


### Bug Fixes

* **codegen:** lint warnings ([74aaa22](https://github.com/sumup/sumup-dotnet/commit/74aaa22c56be978145fe309207803ff2c8c07398))

## [0.0.3](https://github.com/sumup/sumup-dotnet/compare/v0.0.2...v0.0.3) (2026-01-04)


### Features

* **sdk:** package icon ([366b067](https://github.com/sumup/sumup-dotnet/commit/366b06757d8b252cccd46504be6a4b27b802f517))
* **tooling:** dependabot ([01e6594](https://github.com/sumup/sumup-dotnet/commit/01e65944ea8f57fa47644f10edd04daf1dfff4ea))


### Bug Fixes

* license, package readme, community files ([5d0979e](https://github.com/sumup/sumup-dotnet/commit/5d0979e836347001d48bdef06248d728f0ef592e))

## [0.0.2](https://github.com/sumup/sumup-dotnet/compare/v0.0.1...v0.0.2) (2026-01-04)


### Features

* init ([0a8c79e](https://github.com/sumup/sumup-dotnet/commit/0a8c79ecd90928ec3578dd349500030798d15b21))
* init ([c9ee5f5](https://github.com/sumup/sumup-dotnet/commit/c9ee5f5597bf5e5e38b1236b58a700c8e3898192))
