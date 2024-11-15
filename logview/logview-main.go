package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/wavetermdev/waveterm/pkg/util/logview"
	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/waveapp"
)

//go:embed style.css
var styleCSS []byte

// CLI arguments
var windowSize = flag.Int64("l", 20, "number of lines to display")
var logFilePath string

var AppClient = waveapp.MakeClient(waveapp.AppOpts{
	CloseOnCtrlC:         true,
	GlobalKeyboardEvents: true,
	GlobalStyles:         styleCSS,
})

// Props types
type LogContentProps struct {
	Lines     [][]byte `json:"lines"`
	Error     string   `json:"error"`
	LineStart int64    `json:"lineStart"`
}

type FilterInputProps struct {
	Value      string       `json:"value"`
	OnChange   func(string) `json:"onChange"`
	OnError    func(string) `json:"onError"`
	ClearError func()       `json:"clearError"`
}

var FilterInput = waveapp.DefineComponent[FilterInputProps](AppClient, "FilterInput",
	func(ctx context.Context, props FilterInputProps) any {
		handleChange := func(e vdom.VDomEvent) {
			filter := e.TargetValue
			props.OnChange(filter)
		}

		return vdom.H("div", map[string]any{
			"className": "filter-container",
		},
			vdom.H("input", map[string]any{
				"type":        "text",
				"className":   "filter-input",
				"placeholder": "Filter regexp (e.g. 'error|warn')",
				"value":       props.Value,
				"onChange":    handleChange,
			}),
		)
	},
)

// LogContent component to display the log lines
var LogContent = waveapp.DefineComponent[LogContentProps](AppClient, "LogContent",
	func(ctx context.Context, props LogContentProps) any {
		if props.Error != "" {
			return vdom.H("div", map[string]any{
				"className": "log-error",
			}, props.Error)
		}

		return vdom.H("pre", map[string]any{
			"className": "log-content",
		},
			vdom.ForEachIdx(props.Lines, func(line []byte, idx int) any {
				lineNum := props.LineStart + int64(idx)
				return vdom.H("div", map[string]any{
					"key":       idx,
					"className": "log-line",
				},
					vdom.H("span", map[string]any{
						"className": "line-number",
					}, fmt.Sprintf("%6d ", lineNum)),
					vdom.H("span", map[string]any{
						"className": "line-content",
					}, string(line)),
				)
			}),
		)
	},
)

// Main App component
var App = waveapp.DefineComponent(AppClient, "App",
	func(ctx context.Context, _ any) any {
		// State for storing log lines and error
		lines, setLines := vdom.UseState(ctx, [][]byte{})
		errorMsg, setErrorMsg := vdom.UseState(ctx, "")
		currentLineNum, setCurrentLineNum := vdom.UseState(ctx, int64(0))
		filterText, setFilterText := vdom.UseState(ctx, "")
		currentLinePtr := vdom.UseRef(ctx, (*logview.LinePtr)(nil))
		logViewRef := vdom.UseRef(ctx, (*logview.LogView)(nil))

		// Handle filter changes
		handleFilterChange := func(filter string) {
			setFilterText(filter)

			if logViewRef.Current == nil {
				return
			}

			// If filter is empty, clear the regexp
			if filter == "" {
				logViewRef.Current.MatchRe = nil
			} else {
				// Try to compile the regexp
				re, err := regexp.Compile(filter)
				if err != nil {
					setErrorMsg(fmt.Sprintf("Invalid regexp: %v", err))
					return
				}
				setErrorMsg("")
				logViewRef.Current.MatchRe = re
			}

			// Reset to first matching line
			newPtr, err := logViewRef.Current.FirstLinePtr()
			if err != nil {
				setErrorMsg(fmt.Sprintf("Error finding first matching line: %v", err))
				return
			}

			if newPtr == nil {
				setLines([][]byte{})
				setCurrentLineNum(0)
				currentLinePtr.Current = nil
				return
			}

			currentLinePtr.Current = newPtr
			setCurrentLineNum(newPtr.LineNum)

			// Read new window of lines
			newLines, err := logViewRef.Current.ReadWindow(newPtr, int(*windowSize))
			if err != nil {
				setErrorMsg(fmt.Sprintf("Error reading lines: %v", err))
				return
			}

			setLines(newLines)
		}

		// Load initial log data and setup keyboard handler
		vdom.UseEffect(ctx, func() func() {
			file, err := os.Open(logFilePath)
			if err != nil {
				setErrorMsg(fmt.Sprintf("Error opening file: %v", err))
				return nil
			}

			lv := logview.MakeLogView(file)
			logViewRef.Current = lv

			// Setup keyboard handler
			AppClient.SetGlobalEventHandler(func(client *waveapp.Client, event vdom.VDomEvent) {
				if event.EventType != "onKeyDown" || event.KeyData == nil {
					return
				}

				if logViewRef.Current == nil {
					return
				}

				key := event.KeyData.Key
				var newPtr *logview.LinePtr
				var err error

				switch key {
				case "Home":
					newPtr, err = logViewRef.Current.FirstLinePtr()
					if err != nil {
						setErrorMsg(fmt.Sprintf("Error moving to first line: %v", err))
						return
					}

				case "End":
					lastPtr, err := logViewRef.Current.LastLinePtr(currentLinePtr.Current)
					if err != nil {
						setErrorMsg(fmt.Sprintf("Error moving to last line: %v", err))
						return
					}
					if lastPtr == nil {
						return
					}

					_, backPtr, err := logViewRef.Current.Move(lastPtr, -int(*windowSize-1))
					if err != nil {
						setErrorMsg(fmt.Sprintf("Error adjusting view: %v", err))
						return
					}
					if backPtr != nil {
						newPtr = backPtr
					} else {
						newPtr, _ = logViewRef.Current.FirstLinePtr()
					}

				case "ArrowUp":
					if currentLinePtr.Current == nil {
						newPtr, _ = logViewRef.Current.FirstLinePtr()
					} else {
						_, newPtr, _ = logViewRef.Current.Move(currentLinePtr.Current, -1)
					}

				case "ArrowDown":
					if currentLinePtr.Current == nil {
						newPtr, _ = logViewRef.Current.FirstLinePtr()
					} else {
						_, newPtr, _ = logViewRef.Current.Move(currentLinePtr.Current, 1)
					}

				case "PageUp":
					if currentLinePtr.Current == nil {
						newPtr, _ = logViewRef.Current.FirstLinePtr()
					} else if currentLinePtr.Current.LineNum < *windowSize {
						newPtr, _ = logViewRef.Current.FirstLinePtr()
					} else {
						_, newPtr, _ = logViewRef.Current.Move(currentLinePtr.Current, -int(*windowSize))
					}

				case "PageDown":
					if currentLinePtr.Current == nil {
						newPtr, _ = logViewRef.Current.FirstLinePtr()
					} else {
						_, newPtr, _ = logViewRef.Current.Move(currentLinePtr.Current, int(*windowSize))
					}

				default:
					return
				}

				if newPtr == nil {
					return
				}

				currentLinePtr.Current = newPtr

				// Read new window of lines
				newLines, err := logViewRef.Current.ReadWindow(newPtr, int(*windowSize))
				if err != nil {
					setErrorMsg(fmt.Sprintf("Error reading lines: %v", err))
					return
				}

				setLines(newLines)
				setCurrentLineNum(newPtr.LineNum)
				client.SendAsyncInitiation()
			})

			// Get first line pointer
			linePtr, err := lv.FirstLinePtr()
			if err != nil {
				setErrorMsg(fmt.Sprintf("Error getting first line: %v", err))
				return nil
			}

			if linePtr == nil {
				setLines([][]byte{})
				return nil
			}

			currentLinePtr.Current = linePtr
			setCurrentLineNum(linePtr.LineNum)

			// Read initial window of lines
			initialLines, err := lv.ReadWindow(linePtr, int(*windowSize))
			if err != nil {
				setErrorMsg(fmt.Sprintf("Error reading lines: %v", err))
				return nil
			}

			setLines(initialLines)

			// Cleanup function
			return func() {
				lv.Close()
				file.Close()
			}
		}, []any{})

		return vdom.H("div", map[string]any{
			"className": "log-viewer",
		},
			vdom.H("div", map[string]any{
				"className": "log-header",
			},
				vdom.H("h1", nil, "Log Viewer"),
				vdom.H("div", map[string]any{
					"className": "log-info",
				},
					"Showing ", *windowSize, " lines starting at line ", currentLineNum,
					" of ", logFilePath),
				FilterInput(FilterInputProps{
					Value:    filterText,
					OnChange: handleFilterChange,
				}),
			),
			LogContent(LogContentProps{
				Lines:     lines,
				Error:     errorMsg,
				LineStart: currentLineNum,
			}),
		)
	},
)

func main() {
	AppClient.RegisterDefaultFlags()
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: logviewer [flags] <logfile>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	logFilePath = flag.Arg(0)

	AppClient.RunMain()
}
