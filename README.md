# DataLimiter

DataLimiter is a Windows CLI that helps users save internet data while connected
to a mobile hotspot or other metered connection.

When enabled, DataLimiter switches Windows Firewall outbound policy to block and
then allows only Chrome browsing, DNS, and DHCP traffic. The goal is to reduce
unexpected background data usage from apps, updaters, launchers, sync clients,
and other software while still letting the user browse the web in Chrome.

The tool is intended for short, reversible sessions on a personal Windows
device. For example, a user can connect a laptop to a phone hotspot, run
`datalimiter enable`, browse in Chrome with fewer background data leaks, and then
run `datalimiter disable` when they return to normal Wi-Fi.

DataLimiter does not compress traffic, throttle bandwidth, or enforce per-site
limits. It is a simple firewall-based browsing-only mode for data-conscious
hotspot use.

## Commands

```powershell
datalimiter enable
datalimiter disable
datalimiter status
datalimiter repair
datalimiter app add <name-or-path>
datalimiter app remove <name-or-path>
```

`enable`, `disable`, and `repair` require Administrator privileges.
`app add` and `app remove` also require Administrator privileges because they
rewrite DataLimiter firewall rules.

`app add` allows another executable to use the internet alongside Chrome while
DataLimiter is active. You can pass a command name that Windows can resolve, or
a full executable path:

```powershell
datalimiter app add slack
datalimiter app add "C:\Program Files\SomeApp\SomeApp.exe"
```

`app remove` removes a previously allowed executable by name or path:

```powershell
datalimiter app remove slack
```

`status` shows Chrome and any extra apps currently allowed to access the
internet.

## Typical Hotspot Workflow

Open PowerShell as Administrator:

```powershell
datalimiter enable
```

Use Chrome for browsing while connected to the mobile hotspot.

When finished:

```powershell
datalimiter disable
```

Check current state at any time:

```powershell
datalimiter status
```

Allow another app for the current DataLimiter session:

```powershell
datalimiter app add teams
```

Remove it later:

```powershell
datalimiter app remove teams
```

## Build

```powershell
go test ./...
GOOS=windows GOARCH=amd64 go build -o datalimiter-windows-amd64.exe ./cmd/datalimiter
```

The first release artifact is intended for GitHub Releases and later `winget`
submission.
