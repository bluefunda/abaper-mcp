# Changelog

## [0.10.4](https://github.com/bluefunda/abaper-mcp/compare/v0.10.3...v0.10.4) (2026-07-31)


### Bug Fixes

* **tools:** validate required inputs on 4 handlers (partial [#64](https://github.com/bluefunda/abaper-mcp/issues/64)), trigger release ([#72](https://github.com/bluefunda/abaper-mcp/issues/72)) ([a7bc0c7](https://github.com/bluefunda/abaper-mcp/commit/a7bc0c756e73a933cbaf9eb09f285d479d06e7d4))

## [0.10.3](https://github.com/bluefunda/abaper-mcp/compare/v0.10.2...v0.10.3) (2026-07-31)


### Bug Fixes

* script allowlist + optional SSE auth, coverage, and abaper-ts-&gt;abaper rename ([#69](https://github.com/bluefunda/abaper-mcp/issues/69)) ([7508ac9](https://github.com/bluefunda/abaper-mcp/commit/7508ac9a0fe9ff28f89fb440cad7ca2a8ccea2d3))

## [0.10.2](https://github.com/bluefunda/abaper-mcp/compare/v0.10.1...v0.10.2) (2026-07-31)


### Bug Fixes

* address medium-severity review findings (graceful shutdown, s4 perf, error preservation) ([#67](https://github.com/bluefunda/abaper-mcp/issues/67)) ([c91809a](https://github.com/bluefunda/abaper-mcp/commit/c91809a15a2aba4ea8ad16b93149a019606fcf6e))

## [0.10.1](https://github.com/bluefunda/abaper-mcp/compare/v0.10.0...v0.10.1) (2026-07-31)


### Bug Fixes

* resolve high-severity review findings + propagate context ([#65](https://github.com/bluefunda/abaper-mcp/issues/65)) ([339e8dd](https://github.com/bluefunda/abaper-mcp/commit/339e8dd8db9dfe336ae64b76e369c266d51c3d4c))

## [0.10.0](https://github.com/bluefunda/abaper-mcp/compare/v0.9.2...v0.10.0) (2026-06-03)


### Features

* **s4:** add remediation patterns P004-P010 ([dc98018](https://github.com/bluefunda/abaper-mcp/commit/dc9801852f5735e6f12125c3b6301a1cdf4a6ff6))

## [0.9.2](https://github.com/bluefunda/abaper-mcp/compare/v0.9.1...v0.9.2) (2026-06-02)


### Bug Fixes

* redeploy with source_code support in analyze-s4-remediation ([#47](https://github.com/bluefunda/abaper-mcp/issues/47)) ([18764db](https://github.com/bluefunda/abaper-mcp/commit/18764db896723802e7e0ecf297a1749c85663bcf))

## [0.9.1](https://github.com/bluefunda/abaper-mcp/compare/v0.9.0...v0.9.1) (2026-06-02)


### Bug Fixes

* add gitops-repo input for docker-deploy gitops update ([8a6db3b](https://github.com/bluefunda/abaper-mcp/commit/8a6db3bec77fbc3b9bbbd0dae6d8a26ecddca06f))
* add goprivate for private bluefunda module access in CI ([1844476](https://github.com/bluefunda/abaper-mcp/commit/1844476ae4dfd4507045290c08e39c93dd40af30))

## [0.9.0](https://github.com/bluefunda/abaper-mcp/compare/v0.8.0...v0.9.0) (2026-05-29)


### Features

* add source_code input to analyze-s4-remediation tool ([c36c15e](https://github.com/bluefunda/abaper-mcp/commit/c36c15e668dfdd8626b9714b2ed7a0f287e72e85))

## [0.8.0](https://github.com/bluefunda/abaper-mcp/compare/v0.7.0...v0.8.0) (2026-03-25)


### Features

* **tools:** add S4 batch analysis tools via s4-temporal API ([#24](https://github.com/bluefunda/abaper-mcp/issues/24)) ([eade5e9](https://github.com/bluefunda/abaper-mcp/commit/eade5e9cc4337144bef0ed637913ec6efd810309))

## [0.7.0](https://github.com/bluefunda/abaper-mcp/compare/v0.6.0...v0.7.0) (2026-02-28)


### Features

* generic create/update tools with DDIC support ([#19](https://github.com/bluefunda/abaper-mcp/issues/19)) ([1fe80be](https://github.com/bluefunda/abaper-mcp/commit/1fe80be3f461b0a23086fe1d9fa650e672dfb8e0))

## [0.6.0](https://github.com/bluefunda/abaper-mcp/compare/v0.5.0...v0.6.0) (2026-02-24)


### Features

* add create-and-activate composite tool (Phase 1) ([#16](https://github.com/bluefunda/abaper-mcp/issues/16)) ([fbdfd74](https://github.com/bluefunda/abaper-mcp/commit/fbdfd747eed343b40b3caebce118a4335326c70f))

## [0.5.0](https://github.com/bluefunda/abaper-mcp/compare/v0.4.1...v0.5.0) (2026-02-23)


### Features

* add idempotency guard and structured error responses ([#15](https://github.com/bluefunda/abaper-mcp/issues/15)) ([d057232](https://github.com/bluefunda/abaper-mcp/commit/d057232a718252d6885ce4e58713148f1faffb66))
* replace direct ADT calls with abaper-ts REST API ([bf3128f](https://github.com/bluefunda/abaper-mcp/commit/bf3128f6a35838f5ea2684ffcbe6dd838483b18f))

## [0.4.1](https://github.com/bluefunda/abaper-mcp/compare/v0.4.0...v0.4.1) (2026-02-20)


### Bug Fixes

* trigger release for internal CA cert addition ([#10](https://github.com/bluefunda/abaper-mcp/issues/10)) ([a2db494](https://github.com/bluefunda/abaper-mcp/commit/a2db494acb0918e95116464ee3516eb69dd52030))

## [0.4.0](https://github.com/bluefunda/abaper-mcp/compare/v0.3.8...v0.4.0) (2026-02-19)


### Features

* centralize NATS config to shared vault paths ([#7](https://github.com/bluefunda/abaper-mcp/issues/7)) ([78bc69b](https://github.com/bluefunda/abaper-mcp/commit/78bc69bc3db414e8b4e55f0d9cc9649ee8533ae5))
