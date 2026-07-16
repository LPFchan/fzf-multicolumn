package fzf

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/junegunn/fzf/src/util"
)

func runGridSpanCLI(t *testing.T, input []string, args ...string) (int, []string, error) {
	t.Helper()
	opts, err := ParseOptions(false, args)
	if err != nil {
		return ExitError, nil, err
	}
	opts.Input = make(chan string, len(input))
	for _, item := range input {
		opts.Input <- item
	}
	close(opts.Input)
	opts.Output = make(chan string, len(input)+4)
	code, runErr := Run(opts)
	close(opts.Output)
	output := make([]string, 0, len(opts.Output))
	for item := range opts.Output {
		output = append(output, item)
	}
	return code, output, runErr
}

func TestGridSpanCLITransformsAndOutput(t *testing.T) {
	code, output, err := runGridSpanCLI(t,
		[]string{"@@2@@id1|alpha", "@@1@@id2|beta"},
		"--grid=3", "--grid-span-prefix=@@", "--delimiter=|",
		"--with-nth=2", "--accept-nth=1", "--filter=alpha")
	if err != nil || code != ExitOk || !reflect.DeepEqual(output, []string{"id1"}) {
		t.Fatalf("code=%d output=%#v err=%v", code, output, err)
	}
}

func TestGridSpanCLIFilterAndReadZero(t *testing.T) {
	code, output, err := runGridSpanCLI(t,
		[]string{"@@2@@alpha\nbeta", "@@1@@other"},
		"--grid=3", "--grid-span-prefix=@@", "--read0", "--filter=alpha")
	if err != nil || code != ExitOk || !reflect.DeepEqual(output, []string{"alpha\nbeta"}) {
		t.Fatalf("code=%d output=%#v err=%v", code, output, err)
	}
}

func TestGridSpanCLISelectedOutput(t *testing.T) {
	code, output, err := runGridSpanCLI(t, []string{"@@3@@chosen"},
		"--grid=3", "--grid-span-prefix=@@", "--filter=chosen")
	if err != nil || code != ExitOk || !reflect.DeepEqual(output, []string{"chosen"}) {
		t.Fatalf("code=%d output=%#v err=%v", code, output, err)
	}
}

func TestGridSpanHelperProcess(t *testing.T) {
	mode := ""
	var helperArgs []string
	for idx, arg := range os.Args {
		if arg == "--" && idx+1 < len(os.Args) {
			mode = os.Args[idx+1]
			helperArgs = os.Args[idx+2:]
			break
		}
	}
	if mode == "" {
		return
	}
	switch mode {
	case "invalid":
		fmt.Println("@@0@@bad")
	case "valid":
		fmt.Println("@@2@@replacement")
	case "live-invalid":
		fmt.Println("good")
		fmt.Println("bad")
		select {}
	case "phased":
		if len(helperArgs) != 2 {
			os.Exit(3)
		}
		fmt.Println("@@1@@partial")
		if err := os.WriteFile(helperArgs[0], []byte("started"), 0o600); err != nil {
			os.Exit(4)
		}
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(helperArgs[1]); err == nil {
				fmt.Println("@@2@@replacement")
				os.Exit(0)
			}
			time.Sleep(time.Millisecond)
		}
		os.Exit(5)
	}
	os.Exit(0)
}

func gridSpanHelperCommand(mode string) string {
	executor := util.NewExecutor("")
	return fmt.Sprintf("%s -test.run=TestGridSpanHelperProcess -- %s", executor.QuoteEntry(os.Args[0]), mode)
}

func gridSpanPhasedHelperCommand(started, release string) string {
	executor := util.NewExecutor("")
	return fmt.Sprintf("%s -test.run=TestGridSpanHelperProcess -- phased %s %s",
		executor.QuoteEntry(os.Args[0]), executor.QuoteEntry(started), executor.QuoteEntry(release))
}

func waitForGridSpanCondition(t *testing.T, description string, condition func() (bool, string)) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	last := ""
	for time.Now().Before(deadline) {
		ok, diagnostic := condition()
		last = diagnostic
		if ok {
			return diagnostic
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s; last state: %s", description, last)
	return ""
}

func waitForGridSpanFile(t *testing.T, path string) {
	t.Helper()
	waitForGridSpanCondition(t, "helper start signal", func() (bool, string) {
		_, err := os.Stat(path)
		return err == nil, fmt.Sprintf("stat error=%v", err)
	})
}

func TestGridSpanRuntimeRunActionEndToEnd(t *testing.T) {
	for _, test := range []struct {
		name  string
		type_ actionType
		mode  string
	}{
		{"reload", actReload, "valid"},
		{"reload-sync", actReloadSync, "valid"},
		{"reload-invalid", actReload, "invalid"},
		{"reload-sync-invalid", actReloadSync, "invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts, err := ParseOptions(false, []string{"--grid=3", "--grid-span-prefix=@@", "--height=10", "--no-clear"})
			if err != nil {
				t.Fatal(err)
			}
			opts.Input = closedStringChannel("@@1@@initial")
			opts.Output = make(chan string, 4)
			opts.runtimeTestRenderer = newGridSpanTestRenderer()
			opts.runtimeTestHook = func(terminal *Terminal) {
				go func() {
					waitForGridSpanCondition(t, "initial result", func() (bool, string) {
						status := terminal.dumpStatus(getParams{limit: 8})
						return strings.Contains(status, "initial"), status
					})
					terminal.serverInputChan <- []*action{{t: test.type_, a: gridSpanHelperCommand(test.mode)}}
					if test.mode == "valid" {
						waitForGridSpanCondition(t, "replacement result", func() (bool, string) {
							status := terminal.dumpStatus(getParams{limit: 8})
							return strings.Contains(status, "replacement") && !strings.Contains(status, "@@2@@"), status
						})
						terminal.serverInputChan <- []*action{{t: actAccept}}
					}
				}()
			}
			code, runErr := Run(opts)
			if test.mode == "invalid" {
				if code != ExitError || runErr == nil || !strings.Contains(runErr.Error(), "positive") {
					t.Fatalf("code=%d err=%v", code, runErr)
				}
			} else if code != ExitOk || runErr != nil {
				t.Fatalf("code=%d err=%v", code, runErr)
			}
		})
	}
}

func TestGridSpanCoreRuntimeReloadDispatch(t *testing.T) {
	for _, reading := range []bool{false, true} {
		t.Run(fmt.Sprintf("reading-%v", reading), func(t *testing.T) {
			reader := &Reader{termFunc: func() {}}
			command := &commandSpec{command: gridSpanHelperCommand("valid")}
			restarted := false
			next, _ := dispatchRuntimeReload(searchRequest{command: command}, reading, reader, func(got commandSpec, _ []string) {
				restarted = true
				if got.command != command.command {
					t.Fatalf("restart command=%q", got.command)
				}
			})
			if reading {
				if next != command || restarted {
					t.Fatalf("queued=%v restarted=%v", next, restarted)
				}
			} else if next != nil || !restarted {
				t.Fatalf("queued=%v restarted=%v", next, restarted)
			}
		})
	}
}

func TestGridSpanRuntimeReloadActionLifecycle(t *testing.T) {
	terminal := &Terminal{
		executor:     util.NewExecutor(""),
		input:        []rune("runtime-query"),
		merger:       EmptyMerger(revision{}),
		resultMerger: EmptyMerger(revision{}),
		selected:     make(map[int32]selectedItem),
	}
	for _, test := range []struct {
		name  string
		type_ actionType
		mode  string
		sync  bool
	}{
		{"reload-success", actReload, "valid", false},
		{"reload-sync-success", actReloadSync, "valid", true},
		{"reload-invalid", actReload, "invalid", false},
		{"reload-sync-invalid", actReloadSync, "invalid", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			command, sync := terminal.prepareReloadAction(&action{t: test.type_, a: gridSpanHelperCommand(test.mode)})
			if command == nil || sync != test.sync {
				t.Fatalf("prepared command=%v sync=%v", command, sync)
			}
			records, spans, err := runGridSpanReloadCommand(command.command)
			if test.mode == "valid" {
				if err != nil || !reflect.DeepEqual(records, []string{"replacement"}) || !reflect.DeepEqual(spans, []int{2}) {
					t.Fatalf("records=%#v spans=%#v err=%v", records, spans, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "positive") {
				t.Fatalf("invalid reload err=%v", err)
			}
		})
	}
}

func runGridSpanReloadCommand(command string) ([]string, []int, error) {
	eventBox := util.NewEventBox()
	executor := util.NewExecutor("")
	var records []string
	var spans []int
	var ingestionErr error
	reader := NewReader(func(data []byte) bool {
		text, span, err := parseGridSpanRecord(data, "@@", 3)
		if err != nil {
			ingestionErr = err
			return false
		}
		records = append(records, string(text))
		spans = append(spans, span)
		return true
	}, eventBox, executor, false, false)
	reader.SetErrorCheck(func() error { return ingestionErr })
	ready := make(chan bool, 1)
	reader.restart(commandSpec{command: command}, nil, ready)
	<-ready
	return records, spans, ingestionErr
}

func TestGridSpanRuntimeReloadLifecycle(t *testing.T) {
	for _, mode := range []string{"valid", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			eventBox := util.NewEventBox()
			executor := util.NewExecutor("")
			var records []string
			var spans []int
			var ingestionErr error
			reader := NewReader(func(data []byte) bool {
				text, span, err := parseGridSpanRecord(data, "@@", 3)
				if err != nil {
					ingestionErr = err
					return false
				}
				records = append(records, string(text))
				spans = append(spans, span)
				return true
			}, eventBox, executor, false, false)
			reader.SetErrorCheck(func() error { return ingestionErr })

			// Establish the initial source before invoking the same restart path used
			// by runtime reload and reload-sync actions.
			if !reader.readChannel(closedStringChannel("@@1@@initial")) {
				t.Fatal("initial source failed")
			}
			records, spans = nil, nil
			ready := make(chan bool, 1)
			reader.restart(commandSpec{command: gridSpanHelperCommand(mode)}, nil, ready)
			<-ready

			if mode == "valid" {
				if ingestionErr != nil || !reflect.DeepEqual(records, []string{"replacement"}) || !reflect.DeepEqual(spans, []int{2}) {
					t.Fatalf("valid reload records=%#v spans=%#v err=%v", records, spans, ingestionErr)
				}
			} else if ingestionErr == nil || !strings.Contains(ingestionErr.Error(), "positive") {
				t.Fatalf("invalid reload err=%v", ingestionErr)
			}
		})
	}
}

func closedStringChannel(values ...string) chan string {
	channel := make(chan string, len(values))
	for _, value := range values {
		channel <- value
	}
	close(channel)
	return channel
}

func TestGridSpanCLIReloadSyncLifecycle(t *testing.T) {
	opts, err := ParseOptions(false, []string{
		"--grid=3", "--grid-span-prefix=@@", "--filter=bad",
		"--bind=start:reload-sync(" + gridSpanHelperCommand("invalid") + ")",
	})
	if err != nil {
		t.Fatal(err)
	}
	code, err := Run(opts)
	if code != ExitError || err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("reload-sync code=%d err=%v", code, err)
	}
}

func TestGridSpanCLIInvalidError(t *testing.T) {
	code, _, err := runGridSpanCLI(t, []string{"@@0@@bad"},
		"--grid=3", "--grid-span-prefix=@@", "--filter=bad")
	if code != ExitError || err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("code=%d err=%v", code, err)
	}
}
