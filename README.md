# DataLimiter

DataLimiter is a Windows CLI that limits outbound traffic to Chrome browsing by
managing Windows Firewall policy and rules.

## Commands

```powershell
datalimiter enable
datalimiter disable
datalimiter status
datalimiter repair
```

`enable`, `disable`, and `repair` require Administrator privileges.

## Build

```powershell
go test ./...
GOOS=windows GOARCH=amd64 go build -o datalimiter-windows-amd64.exe ./cmd/datalimiter
```

The first release artifact is intended for GitHub Releases and later `winget`
submission.
