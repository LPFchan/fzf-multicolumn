package fzf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/junegunn/fzf/src/util"
)

type gridSpanAtomicResult struct {
	status string
	final  bool
}

func TestGridSpanReloadSyncAtomicSnapshot(t *testing.T) {
	for _, test := range []struct {
		name  string
		type_ actionType
		sync  bool
	}{
		{"reload", actReload, false},
		{"reload-sync", actReloadSync, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			started := filepath.Join(tmp, "started")
			release := filepath.Join(tmp, "release")
			opts, err := ParseOptions(false, []string{"--grid=3", "--grid-span-prefix=@@", "--height=10", "--no-clear"})
			if err != nil {
				t.Fatal(err)
			}
			opts.Input = closedStringChannel("@@1@@initial")
			opts.Output = make(chan string, 4)
			opts.runtimeTestRenderer = newGridSpanTestRenderer()
			terminalReady := make(chan *Terminal, 1)
			readEvents := make(chan bool, 8)
			readDone := make(chan bool, 8)
			published := make(chan bool, 8)
			opts.runtimeTestHook = func(terminal *Terminal) { terminalReady <- terminal }
			opts.runtimeReadHook = func(evt util.EventType, useSnapshot bool) {
				if evt == EvtReadNew {
					select {
					case readEvents <- useSnapshot:
					default:
					}
				}
			}
			opts.runtimeReadDoneHook = func(evt util.EventType, useSnapshot bool) {
				if evt == EvtReadNew {
					select {
					case readDone <- useSnapshot:
					default:
					}
				}
			}
			var terminal *Terminal
			opts.runtimePublishHook = func(final bool) {
				select {
				case published <- final:
				default:
				}
			}
			runDone := make(chan gridSpanAtomicResult, 1)
			go func() {
				code, runErr := Run(opts)
				runDone <- gridSpanAtomicResult{status: fmt.Sprintf("code=%d err=%v", code, runErr), final: code == ExitOk && runErr == nil}
			}()

			cleanup := func() {
				_ = os.WriteFile(release, []byte("release"), 0o600)
				if terminal != nil {
					select {
					case terminal.serverInputChan <- []*action{{t: actAbort}}:
					default:
					}
				}
			}
			defer cleanup()

			select {
			case terminal = <-terminalReady:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for terminal")
			}
			waitStatusForeground(t, terminal, "initial result", func(status string) bool { return strings.Contains(status, "initial") })
			initialPublicationCount := len(published)
			for range initialPublicationCount {
				<-published
			}
			terminal.serverInputChan <- []*action{{t: test.type_, a: gridSpanPhasedHelperCommand(started, release)}}
			waitFileForeground(t, started)

			var observedSnapshot bool
			select {
			case observedSnapshot = <-readEvents:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for partial EvtReadNew")
			}
			if observedSnapshot != test.sync {
				t.Fatalf("runtime useSnapshot=%v want %v", observedSnapshot, test.sync)
			}
			select {
			case processedSnapshot := <-readDone:
				if processedSnapshot != test.sync {
					t.Fatalf("post-processing useSnapshot=%v want %v", processedSnapshot, test.sync)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for post-EvtReadNew acknowledgement")
			}

			if test.sync {
				status := terminal.dumpStatus(getParams{limit: 8})
				if !strings.Contains(status, "initial") || strings.Contains(status, "partial") {
					t.Fatalf("reload-sync exposed non-atomic status: %s", status)
				}
				select {
				case final := <-published:
					if !final {
						t.Fatalf("reload-sync published partial matcher result")
					}
				default:
				}
			} else {
				waitStatusForeground(t, terminal, "partial async result", func(status string) bool {
					return strings.Contains(status, "partial") && !strings.Contains(status, "initial")
				})
				select {
				case final := <-published:
					if final {
						t.Fatalf("async reload first publication was final")
					}
				case <-time.After(5 * time.Second):
					t.Fatal("timed out waiting for async partial publication")
				}
			}

			if err := os.WriteFile(release, []byte("release"), 0o600); err != nil {
				t.Fatal(err)
			}
			waitStatusForeground(t, terminal, "completed replacement", func(status string) bool {
				return strings.Contains(status, "replacement") && !strings.Contains(status, "initial")
			})
			terminal.serverInputChan <- []*action{{t: actAccept}}
			select {
			case result := <-runDone:
				if !result.final {
					t.Fatalf("Run failed: %s", result.status)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for Run completion")
			}
		})
	}
}

func waitStatusForeground(t *testing.T, terminal *Terminal, description string, predicate func(string) bool) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	last := ""
	for time.Now().Before(deadline) {
		last = terminal.dumpStatus(getParams{limit: 8})
		if predicate(last) {
			return last
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s; last status: %s", description, last)
	return ""
}

func waitFileForeground(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		} else {
			last = err
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for helper signal %s: %v", path, last)
}
