## winhchk

A simple Windows Service written in Golang for making HTTP requests every minute
to a heart beat service such as [healthchecks.io](https://healthchecks.io/).

### Usage

```
winhchk.exe -url https://hc-ping.com/eb095278-f28d-448d-87fb-7b75c171a6aa install
winhchk.exe start
```

```
winhchk.exe stop
winhchk.exe remove
```
