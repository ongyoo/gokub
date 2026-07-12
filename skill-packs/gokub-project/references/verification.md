# Verification Reference

## During Development

```bash
gofmt -w <changed-go-files>
go test ./path/to/changed/package
```

## Before Completion

```bash
go test -race ./...
go vet ./...
go build ./...
gokub doctor
```

Use project-native Make targets when available:

```bash
make fmt
make test
make vet
make build
make doctor
```

For HTTP changes, verify `/health/live` and `/health/ready` still return success and
secure headers remain present. For configuration changes, update `.env.example`
without adding real credentials. For manifest or feature changes, run `gokub status`
and confirm `.gokub.yaml` matches generated files.

Do not claim verification that did not run. Record missing services, network access,
credentials, or platform tooling that prevented a check.
