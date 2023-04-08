## winhchk

A simple Windows Service written in Golang for making HTTP requests every minute
to a heart beat service such as [healthchecks.io](https://healthchecks.io/).

### Usage

```
Add-MpPreference -ExclusionPath C:\winhchk.exe
curl "https://github.com/samuelkadolph/winhchk/releases/download/v0.1.2/winhchk.exe" -o C:\winhchk.exe
C:\winhchk.exe -url https://hc-ping.com/eb095278-f28d-448d-87fb-7b75c171a6aa install
C:\winhchk.exe start
```

```
C:\winhchk.exe stop
C:\winhchk.exe remove
Remove-Item C:\winhchk.exe
Remove-MpPreference -ExclusionPath C:\winhchk.exe
```
