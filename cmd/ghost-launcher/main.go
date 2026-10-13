package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"ghostbrowser/internal/undetectable"
	"os"
	"os/signal"
	"time"
)

func main() {
	path := flag.String("config", "", "profile JSON path")
	verify := flag.Bool("verify", false, "probe live values and close browser")
	prepare := flag.Bool("prepare", false, "only convert JSON to startInfo")
	port := flag.Int("port", 0, "CDP port; 0 chooses a free port")
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "usage: ghost-launcher -config profile.json [-verify|-prepare]")
		os.Exit(2)
	}
	spec, err := undetectable.LoadSpec(*path)
	if err != nil {
		fail(err)
	}
	if *prepare {
		_, dir, err := spec.Prepare()
		if err != nil {
			fail(err)
		}
		printJSON(map[string]any{"prepared": true, "profilesRoot": spec.ProfilesRoot, "profileDir": dir})
		return
	}
	s, err := spec.Start(*port)
	if err != nil {
		fail(err)
	}
	printJSON(map[string]any{"profileId": s.ProfileID, "pid": s.Cmd.Process.Pid, "port": s.Port, "websocket": s.WSURL, "supervising": true})
	if *verify {
		fp, err := undetectable.ProbeNewTab(s.Port)
		// Localhost is a secure context and does not require an external website.
		if err != nil {
			s.Close()
			fail(err)
		}
		mismatches := undetectable.CompareFull(s.Config, fp)
		printJSON(map[string]any{"live": fp, "mismatches": mismatches, "checkedAt": time.Now().Format(time.RFC3339)})
		s.Close()
		if len(mismatches) > 0 {
			os.Exit(1)
		}
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	select {
	case <-signals:
		s.Close()
	case <-s.Done:
		s.Close()
	}
}

func printJSON(v any) { _ = json.NewEncoder(os.Stdout).Encode(v) }
func fail(err error)  { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
