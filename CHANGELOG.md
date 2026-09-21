# [1.2.0](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.1.6...v1.2.0) (2026-09-21)


### Features

* **ci:** baue den WLED-Mock pro PR und verlinke seine UI ([867f905](https://github.com/stuttgart-things/homerun2-light-catcher/commit/867f905b4f3a5c110aaae966aedce385841c7326)), closes [#74](https://github.com/stuttgart-things/homerun2-light-catcher/issues/74)
* **ci:** nimm die Preview-Domain aus der Org-Variable ([5ce5934](https://github.com/stuttgart-things/homerun2-light-catcher/commit/5ce5934f70a482181954bc7d6a7f3f756ac40909)), closes [#74](https://github.com/stuttgart-things/homerun2-light-catcher/issues/74)

## [1.1.6](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.1.5...v1.1.6) (2026-09-20)


### Bug Fixes

* **ci:** baue das PR-Image aus dem Head-Commit ([ff9a4bf](https://github.com/stuttgart-things/homerun2-light-catcher/commit/ff9a4bfcdea6559779cf7d91b73f7eaecd8b2c4c)), closes [#72](https://github.com/stuttgart-things/homerun2-light-catcher/issues/72)

## [1.1.5](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.1.4...v1.1.5) (2026-09-20)


### Bug Fixes

* **ci:** repariere die PR-Preview-Strecke ([2049ceb](https://github.com/stuttgart-things/homerun2-light-catcher/commit/2049ceb7bf73d0658364addb98376cd998107298)), closes [stuttgart-things/stuttgart-things#3065](https://github.com/stuttgart-things/stuttgart-things/issues/3065) [#70](https://github.com/stuttgart-things/homerun2-light-catcher/issues/70)

## [1.1.4](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.1.3...v1.1.4) (2026-09-11)


### Bug Fixes

* stop waiting for Redis when a shutdown signal arrives ([#64](https://github.com/stuttgart-things/homerun2-light-catcher/issues/64)) ([8c163f3](https://github.com/stuttgart-things/homerun2-light-catcher/commit/8c163f3e1baffe05a1f61831e293bf65f990f921)), closes [#61](https://github.com/stuttgart-things/homerun2-light-catcher/issues/61) [#63](https://github.com/stuttgart-things/homerun2-light-catcher/issues/63)

## [1.1.3](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.1.2...v1.1.3) (2026-09-11)


### Bug Fixes

* wait for Redis with bounded backoff instead of exiting at startup ([#60](https://github.com/stuttgart-things/homerun2-light-catcher/issues/60)) ([6ce8012](https://github.com/stuttgart-things/homerun2-light-catcher/commit/6ce80123c367a90d99c5dd2c87f0c7e0d5ec0333)), closes [#59](https://github.com/stuttgart-things/homerun2-light-catcher/issues/59) [stuttgart-things/homerun-library#123](https://github.com/stuttgart-things/homerun-library/issues/123) [#59](https://github.com/stuttgart-things/homerun2-light-catcher/issues/59)

## [1.1.2](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.1.1...v1.1.2) (2026-09-11)


### Bug Fixes

* skip the light for messages pitched longer ago than MAX_MESSAGE_AGE ([#58](https://github.com/stuttgart-things/homerun2-light-catcher/issues/58)) ([a33f08f](https://github.com/stuttgart-things/homerun2-light-catcher/commit/a33f08fb0a0dd14120d0cb2f92aae6bcd4dc78e4)), closes [#55](https://github.com/stuttgart-things/homerun2-light-catcher/issues/55) [#57](https://github.com/stuttgart-things/homerun2-light-catcher/issues/57)

## [1.1.1](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.1.0...v1.1.1) (2026-09-11)


### Bug Fixes

* start new consumer groups at $ and handle messages in stream order ([#56](https://github.com/stuttgart-things/homerun2-light-catcher/issues/56)) ([6262f87](https://github.com/stuttgart-things/homerun2-light-catcher/commit/6262f87211e6c647050a85833ad57270a3032da8)), closes [#55](https://github.com/stuttgart-things/homerun2-light-catcher/issues/55)

# [1.1.0](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.0.2...v1.1.0) (2026-09-11)


### Features

* effect rules that also match on message tags ([#54](https://github.com/stuttgart-things/homerun2-light-catcher/issues/54)) ([1ff1973](https://github.com/stuttgart-things/homerun2-light-catcher/commit/1ff197302223e0ab333caf925e4919bb10732ecc)), closes [#51](https://github.com/stuttgart-things/homerun2-light-catcher/issues/51)

## [1.0.2](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.0.1...v1.0.2) (2026-09-11)


### Bug Fixes

* **deps:** update module charm.land/bubbletea/v2 to v2.0.9 ([#39](https://github.com/stuttgart-things/homerun2-light-catcher/issues/39)) ([c7fceac](https://github.com/stuttgart-things/homerun2-light-catcher/commit/c7fceacdfd4ee750964385da9ae69d1b644f010d))
* match effects in profile order and keep older timers from turning off newer effects ([#53](https://github.com/stuttgart-things/homerun2-light-catcher/issues/53)) ([eea2ea3](https://github.com/stuttgart-things/homerun2-light-catcher/commit/eea2ea3a815aa7bc7921c0a87b3c7e9562bc1fb8)), closes [#50](https://github.com/stuttgart-things/homerun2-light-catcher/issues/50)

## [1.0.1](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v1.0.0...v1.0.1) (2026-09-11)


### Bug Fixes

* **deps:** update module charm.land/lipgloss/v2 to v2.0.6 ([#40](https://github.com/stuttgart-things/homerun2-light-catcher/issues/40)) ([6cb73cf](https://github.com/stuttgart-things/homerun2-light-catcher/commit/6cb73cf532e5625bb953c9fa222ba9fbb8d2a547))
* **kcl:** stop emitting Namespace from the wled-mock kustomize OCI ([#52](https://github.com/stuttgart-things/homerun2-light-catcher/issues/52)) ([b22174d](https://github.com/stuttgart-things/homerun2-light-catcher/commit/b22174d99d915d4722d2164333ea673a3a6b9b9b)), closes [#36](https://github.com/stuttgart-things/homerun2-light-catcher/issues/36) [#35](https://github.com/stuttgart-things/homerun2-light-catcher/issues/35)

# [1.0.0](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.7.4...v1.0.0) (2026-08-20)


* chore(deps)!: migrate to homerun-library v4 ([#48](https://github.com/stuttgart-things/homerun2-light-catcher/issues/48)) ([980f700](https://github.com/stuttgart-things/homerun2-light-catcher/commit/980f700cfde068118e41eba4cafdc69f51df9fdc)), closes [homerun-library#103](https://github.com/homerun-library/issues/103)


### BREAKING CHANGES

* homerun-library moved to the /v4 module path and renamed
Message.Url to Message.URL.

- Import path github.com/stuttgart-things/homerun-library/v3 -> /v4.
- Message.Url -> Message.URL. The JSON tag stays "url", so nothing on the wire,
  in Redis JSON or in the RediSearch index changes; this is a compile-time
  rename only.

What v4 brings beyond the rename:

- RediSearch indexes the message's own timestamp instead of the moment of
  indexing, as a NUMERIC field so range queries are expressible at all

## [0.7.4](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.7.3...v0.7.4) (2026-05-28)


### Bug Fixes

* **kcl:** stop emitting Namespace from kustomize OCI ([acfe976](https://github.com/stuttgart-things/homerun2-light-catcher/commit/acfe9763e26263c322a0aa6c24809641d7591f1e)), closes [#36](https://github.com/stuttgart-things/homerun2-light-catcher/issues/36)

## [0.7.3](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.7.2...v0.7.3) (2026-05-21)


### Bug Fixes

* match severities case-insensitively + fix bundled profile ([0fbef1f](https://github.com/stuttgart-things/homerun2-light-catcher/commit/0fbef1ff55b9d7b36eb0ae986a96b12bfd7add4e)), closes [#27](https://github.com/stuttgart-things/homerun2-light-catcher/issues/27)

## [0.7.2](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.7.1...v0.7.2) (2026-05-21)


### Bug Fixes

* **deps:** update module github.com/redis/go-redis/v9 to v9.19.0 ([#25](https://github.com/stuttgart-things/homerun2-light-catcher/issues/25)) ([13afce1](https://github.com/stuttgart-things/homerun2-light-catcher/commit/13afce15df9bb3027951eec2f58f6f8e1a316fe8))

## [0.7.1](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.7.0...v0.7.1) (2026-05-20)


### Bug Fixes

* **deps:** update module charm.land/bubbletea/v2 to v2.0.6 ([#18](https://github.com/stuttgart-things/homerun2-light-catcher/issues/18)) ([573d65b](https://github.com/stuttgart-things/homerun2-light-catcher/commit/573d65b4d62cebddbdf7024d391f8d4cf5e87f3a))

# [0.7.0](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.6.0...v0.7.0) (2026-05-20)


### Bug Fixes

* **deps:** update module charm.land/lipgloss/v2 to v2.0.3 ([#22](https://github.com/stuttgart-things/homerun2-light-catcher/issues/22)) ([06bdebb](https://github.com/stuttgart-things/homerun2-light-catcher/commit/06bdebb7362962c2671e220570f43eea3c28238e))


### Features

* PR-preview setup (Option B HTTPRoute + profile CM + 4 workflows) ([#20](https://github.com/stuttgart-things/homerun2-light-catcher/issues/20)) ([9ada80d](https://github.com/stuttgart-things/homerun2-light-catcher/commit/9ada80dcb709df7e01675f3b4998094e5aaf865b)), closes [stuttgart-things/homerun2-omni-pitcher#116](https://github.com/stuttgart-things/homerun2-omni-pitcher/issues/116) [stuttgart-things/argocd#116](https://github.com/stuttgart-things/argocd/issues/116) [#16](https://github.com/stuttgart-things/homerun2-light-catcher/issues/16)

# [0.6.0](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.5.0...v0.6.0) (2026-04-15)


### Features

* multi-stream subscription (list of Redis streams) ([#13](https://github.com/stuttgart-things/homerun2-light-catcher/issues/13)) ([0bad474](https://github.com/stuttgart-things/homerun2-light-catcher/commit/0bad4749c2386ad7b150404604cf6b183d1c77e8)), closes [stuttgart-things/homerun2-core-catcher#53](https://github.com/stuttgart-things/homerun2-core-catcher/issues/53) [#12](https://github.com/stuttgart-things/homerun2-light-catcher/issues/12) [stuttgart-things/homerun-library#83](https://github.com/stuttgart-things/homerun-library/issues/83)

# [0.5.0](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.4.1...v0.5.0) (2026-03-15)


### Features

* show trigger context (severity, system, effect) on mock dashboard ([88b89a1](https://github.com/stuttgart-things/homerun2-light-catcher/commit/88b89a1bedbac809d8043c5565638a0f4324dec3))

## [0.4.1](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.4.0...v0.4.1) (2026-03-15)


### Bug Fixes

* add explicit packages:write permission to wled-mock release job ([ab6939d](https://github.com/stuttgart-things/homerun2-light-catcher/commit/ab6939df986a65c4c52eec3890565116e74d9fe7))

# [0.4.0](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.3.0...v0.4.0) (2026-03-15)


### Features

* add light catcher dashboard and CI for both images ([843b905](https://github.com/stuttgart-things/homerun2-light-catcher/commit/843b9051473610bd187a19d1f959d910d5da5ad3))

# [0.3.0](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.2.0...v0.3.0) (2026-03-15)


### Features

* add build info to mock dashboard footer and lighten background ([2da7bd6](https://github.com/stuttgart-things/homerun2-light-catcher/commit/2da7bd6a744614e1df5adb71b201f42cad5f8581)), closes [#0f172a](https://github.com/stuttgart-things/homerun2-light-catcher/issues/0f172a) [#1e293b](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1e293b)
* add WLED mock container image build and KCL deployment manifests ([c00426c](https://github.com/stuttgart-things/homerun2-light-catcher/commit/c00426cfca022fc824037b18fc06992d490a2fec)), closes [#2](https://github.com/stuttgart-things/homerun2-light-catcher/issues/2)
* align WLED mock dashboard with core-catcher design ([096084c](https://github.com/stuttgart-things/homerun2-light-catcher/commit/096084cbb908634587e3884c12adb2767cc89a0f)), closes [#0f172a](https://github.com/stuttgart-things/homerun2-light-catcher/issues/0f172a) [#1e293b](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1e293b) [#334155](https://github.com/stuttgart-things/homerun2-light-catcher/issues/334155) [#818cf8](https://github.com/stuttgart-things/homerun2-light-catcher/issues/818cf8) [#64748b](https://github.com/stuttgart-things/homerun2-light-catcher/issues/64748b)

# [0.2.0](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.1.0...v0.2.0) (2026-03-15)


### Features

* add mkdocs documentation and backstage catalog-info ([2996f73](https://github.com/stuttgart-things/homerun2-light-catcher/commit/2996f732e0db5f87c8b844faf9dc54967f45d750))

## [0.1.1](https://github.com/stuttgart-things/homerun2-light-catcher/compare/v0.1.0...v0.1.1) (2026-03-14)


### Bug Fixes

* resolve 8 golangci-lint issues ([87ba1f2](https://github.com/stuttgart-things/homerun2-light-catcher/commit/87ba1f289372ef64dcb4136ab8824b706c6e1d3d))

# 0.1.0 (2026-03-14)


### Features

* initial homerun2-light-catcher service ([b0b12ea](https://github.com/stuttgart-things/homerun2-light-catcher/commit/b0b12ea9ac450048913fd6d9014e0124be720457)), closes [#1863](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1863) [#1864](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1864) [#1865](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1865) [#1866](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1866) [#1867](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1867) [#1868](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1868) [#1869](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1869) [#1870](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1870) [#1871](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1871) [#1872](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1872) [#1873](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1873) [#1874](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1874) [#1875](https://github.com/stuttgart-things/homerun2-light-catcher/issues/1875)
