package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "winhchks"
const displayName = "winhchks"
const description = "This service makes a HTTP request to a URL every minute for a healthcheck."
const version = "v0.1.2"

type service struct {
	elog debug.Log
	url  string
}

func main() {
	var hchkURL string

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usge of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "  command\n    \tCommand to run, one of: debug install remove start stop")
	}
	flag.StringVar(&hchkURL, "url", "", "Healthcheck URL, required with debug and install")
	flag.Parse()

	if inService, err := svc.IsWindowsService(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to determine if we are running in service err='%v'", err)
		os.Exit(1)
	} else if inService {
		runService(hchkURL, false)
		return
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := strings.ToLower(flag.Arg(0))

	if (command == "debug" || command == "install") && hchkURL == "" {
		flag.Usage()
		os.Exit(1)
	}

	var err error

	switch command {
	case "debug":
		runService(hchkURL, true)
		return
	case "install":
		err = installService(hchkURL)
	case "remove":
		err = removeService()
	case "start":
		err = startService()
	case "stop":
		err = stopSerice()
	default:
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run command %s err='%v'", command, err)
		os.Exit(1)
	}
}

func calcNextRun() <-chan time.Time {
	now := time.Now()
	next := now.Add(time.Minute).Truncate(time.Minute).Add(2 * time.Second)

	return time.After(next.Sub(now))
}

func installService(url string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service is already installed")
	}

	s, err = m.CreateService(serviceName, executable, mgr.Config{
		Description: description,
		DisplayName: displayName,
		StartType:   mgr.StartAutomatic,
	}, "-url", url)
	if err != nil {
		return err
	}
	defer s.Close()

	if err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Info|eventlog.Warning); err != nil {
		s.Delete()
		return fmt.Errorf("eventlog install failed: %s", err)
	}

	return nil
}

func removeService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service is not installed")
	}
	defer s.Close()

	if err = s.Delete(); err != nil {
		return err
	}

	if err = eventlog.Remove(serviceName); err != nil {
		return fmt.Errorf("eventlog remove failed: %s", err)
	}

	return nil
}

func runService(url string, isDebug bool) {
	var elog debug.Log

	if isDebug {
		elog = debug.New(serviceName)
	} else {
		elog, _ = eventlog.Open(serviceName)
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("Starting service"))

	run := svc.Run
	if isDebug {
		run = debug.Run
	}

	if err := run(serviceName, &service{elog: elog, url: url}); err != nil {
		elog.Error(1, fmt.Sprintf("Service failed: %s", err))
	} else {
		elog.Info(1, fmt.Sprintf("Service stopped"))
	}
}

func startService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service is not installed")
	}
	defer s.Close()

	if err = s.Start(); err != nil {
		return fmt.Errorf("could not start service: %s", err)
	}

	return nil
}

func stopSerice() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service is not installed")
	}
	defer s.Close()

	if _, err = s.Control(svc.Stop); err != nil {
		return fmt.Errorf("could not stop service: %v", err)
	}

	return nil
}

func (s *service) Healthcheck() {
	if resp, err := http.Get(s.url); err != nil {
		s.elog.Error(1, fmt.Sprintf("Healthcheck err='%v'", err))
	} else {
		s.elog.Info(1, fmt.Sprintf("Healthcheck status='%s'", resp.Status))
		resp.Body.Close()
	}
}

func (s *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue

	changes <- svc.Status{State: svc.StartPending}

	paused := false
	run := calcNextRun()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	s.elog.Info(1, fmt.Sprintf("Service started url='%s' version='%s'", s.url, version))

	for {
		select {
		case <-run:
			if !paused {
				s.Healthcheck()
			}
			run = calcNextRun()
		case c := <-r:
			switch c.Cmd {
			case svc.Continue:
				paused = false
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				s.elog.Info(1, fmt.Sprintf("Service resumed"))
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Pause:
				paused = true
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				s.elog.Info(1, fmt.Sprintf("Service paused"))
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				return
			default:
				s.elog.Error(1, fmt.Sprintf("Unexpected control request %+v", c))
			}
		}
	}

	return
}
