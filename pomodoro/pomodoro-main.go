package main

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/vdom/vdomclient"
)

//go:embed style.css
var styleCSS []byte

var PomodoroVDomClient *vdomclient.Client = vdomclient.MakeClient(vdomclient.ApplicationOpts{
	CloseOnCtrlC: true,
	GlobalStyles: styleCSS,
})

type Mode struct {
	Name     string `json:"name"`
	Duration int    `json:"duration"` // in minutes
}

var (
	WorkMode  = Mode{Name: "Work", Duration: 25}
	BreakMode = Mode{Name: "Break", Duration: 5}
)

type TimerDisplayProps struct {
	Minutes int    `json:"minutes"`
	Seconds int    `json:"seconds"`
	Mode    string `json:"mode"`
}

type ControlButtonsProps struct {
	IsRunning bool      `json:"isRunning"`
	OnStart   func()    `json:"onStart"`
	OnPause   func()    `json:"onPause"`
	OnReset   func()    `json:"onReset"`
	OnMode    func(int) `json:"onMode"`
}

type TimerState struct {
	ticker    *time.Ticker
	done      chan bool
	startTime time.Time
	duration  time.Duration
	isActive  bool // Track if the timer goroutine is running
}

var TimerDisplay = vdomclient.DefineComponent[TimerDisplayProps](PomodoroVDomClient, "TimerDisplay",
	func(ctx context.Context, props TimerDisplayProps) any {
		return vdom.E("div",
			vdom.Class("timer-display"),
			vdom.E("div",
				vdom.Class("mode-indicator"),
				props.Mode,
			),
			vdom.E("div",
				vdom.Class("time"),
				fmt.Sprintf("%02d:%02d", props.Minutes, props.Seconds),
			),
		)
	},
)

var ControlButtons = vdomclient.DefineComponent[ControlButtonsProps](PomodoroVDomClient, "ControlButtons",
	func(ctx context.Context, props ControlButtonsProps) any {
		return vdom.E("div",
			vdom.Class("control-buttons"),
			vdom.IfElse(props.IsRunning,
				vdom.E("button",
					vdom.Class("control-btn"),
					vdom.P("onClick", props.OnPause),
					"Pause",
				),
				vdom.E("button",
					vdom.Class("control-btn"),
					vdom.P("onClick", props.OnStart),
					"Start",
				),
			),
			vdom.E("button",
				vdom.Class("control-btn"),
				vdom.P("onClick", props.OnReset),
				"Reset",
			),
			vdom.E("div",
				vdom.Class("mode-buttons"),
				vdom.E("button",
					vdom.Class("mode-btn"),
					vdom.P("onClick", func() { props.OnMode(WorkMode.Duration) }),
					"Work Mode",
				),
				vdom.E("button",
					vdom.Class("mode-btn"),
					vdom.P("onClick", func() { props.OnMode(BreakMode.Duration) }),
					"Break Mode",
				),
			),
		)
	},
)

var App = vdomclient.DefineComponent[struct{}](PomodoroVDomClient, "App",
	func(ctx context.Context, _ struct{}) any {
		isRunning, setIsRunning := vdom.UseState(ctx, false)
		minutes, setMinutes := vdom.UseState(ctx, WorkMode.Duration)
		seconds, setSeconds := vdom.UseState(ctx, 0)
		mode, setMode := vdom.UseState(ctx, WorkMode.Name)
		_, setIsComplete := vdom.UseState(ctx, false)
		timerRef := vdom.UseRef(ctx, &TimerState{
			done: make(chan bool),
		})

		stopTimer := func() {
			if timerRef.Current.ticker != nil {
				timerRef.Current.ticker.Stop()
				timerRef.Current.ticker = nil
			}
			if timerRef.Current.isActive {
				close(timerRef.Current.done)
				timerRef.Current.isActive = false
			}
			timerRef.Current.done = make(chan bool)
		}

		startTimer := func() {
			if timerRef.Current.isActive {
				return // Timer already running
			}

			// Stop any existing timer first
			stopTimer()

			setIsComplete(false)
			timerRef.Current.startTime = time.Now()
			timerRef.Current.duration = time.Duration(minutes) * time.Minute
			timerRef.Current.isActive = true
			setIsRunning(true)
			timerRef.Current.ticker = time.NewTicker(1 * time.Second)

			go func() {
				for {
					select {
					case <-timerRef.Current.done:
						return
					case <-timerRef.Current.ticker.C:
						elapsed := time.Since(timerRef.Current.startTime)
						remaining := timerRef.Current.duration - elapsed

						if remaining <= 0 {
							// Timer completed
							setIsRunning(false)
							setMinutes(0)
							setSeconds(0)
							setIsComplete(true)
							stopTimer()
							PomodoroVDomClient.SendAsyncInitiation()
							return
						}

						m := int(remaining.Minutes())
						s := int(remaining.Seconds()) % 60

						// Only send update if values actually changed
						if m != minutes || s != seconds {
							setMinutes(m)
							setSeconds(s)
							PomodoroVDomClient.SendAsyncInitiation()
						}
					}
				}
			}()
		}

		pauseTimer := func() {
			stopTimer()
			setIsRunning(false)
			PomodoroVDomClient.SendAsyncInitiation()
		}

		resetTimer := func() {
			stopTimer()
			setIsRunning(false)
			setIsComplete(false)
			if mode == WorkMode.Name {
				setMinutes(WorkMode.Duration)
			} else {
				setMinutes(BreakMode.Duration)
			}
			setSeconds(0)
			PomodoroVDomClient.SendAsyncInitiation()
		}

		changeMode := func(duration int) {
			stopTimer()
			setIsRunning(false)
			setIsComplete(false)
			setMinutes(duration)
			setSeconds(0)
			if duration == WorkMode.Duration {
				setMode(WorkMode.Name)
			} else {
				setMode(BreakMode.Name)
			}
			PomodoroVDomClient.SendAsyncInitiation()
		}

		// Cleanup on unmount
		vdom.UseEffect(ctx, func() func() {
			return func() {
				stopTimer()
			}
		}, []any{})

		return vdom.E("div",
			vdom.Class("pomodoro-app"),
			vdom.E("h1",
				vdom.Class("title"),
				"Pomodoro Timer",
			),
			TimerDisplay(TimerDisplayProps{
				Minutes: minutes,
				Seconds: seconds,
				Mode:    mode,
			}),
			ControlButtons(ControlButtonsProps{
				IsRunning: isRunning,
				OnStart:   startTimer,
				OnPause:   pauseTimer,
				OnReset:   resetTimer,
				OnMode:    changeMode,
			}),
		)
	},
)

func main() {
	PomodoroVDomClient.RunMain()
}
