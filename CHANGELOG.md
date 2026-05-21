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
