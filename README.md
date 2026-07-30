# wsl-extended

WSL distrolarını ve `wslc` kapsayıcılarını tek bir terminal arayüzünden yönetmek için Go ile yazılmış TUI.

Sekmeli yapı: **Distros · Containers · Images · Volumes · Networks**

## Durum

Tasarım aşaması. Ayrıntılı yol haritası ve mimari kararlar için [docs/PLAN.md](docs/PLAN.md).

## Gereksinimler

- Go 1.26+
- WSL 2.9+ (`wsl.exe`)
- `wslc.exe` (WSL Kapsayıcı CLI) — kapsayıcı sekmeleri için

## Geliştirme

```bash
go run ./cmd/wsl-extended
```
