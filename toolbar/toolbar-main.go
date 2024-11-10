package main

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/vdom/vdomclient"
)

//go:embed style.css
var styleCSS []byte

var AppClient = vdomclient.MakeClient(vdomclient.AppOpts{
	CloseOnCtrlC:  true,
	GlobalStyles:  styleCSS,
	TargetToolbar: &vdom.VDomTargetToolbar{Toolbar: true, Height: "1.5em"},
})

type Metrics struct {
	CPUPercent float64
	LoadAvg    [3]float64
}

var App = vdomclient.DefineComponent(AppClient, "App",
	func(ctx context.Context, _ any) any {
		metrics, setMetrics := vdom.UseState(ctx, Metrics{})

		// Update metrics every second
		vdom.UseEffect(ctx, func() func() {
			done := make(chan bool)

			go func() {
				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-done:
						return
					case <-ticker.C:
						if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
							if loadAvg, err := load.Avg(); err == nil {
								setMetrics(Metrics{
									CPUPercent: cpuPercent[0],
									LoadAvg:    [3]float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15},
								})
								AppClient.SendAsyncInitiation()
							}
						}
					}
				}
			}()

			return func() {
				close(done)
			}
		}, []any{})

		// Helper to get color based on CPU usage
		getCPUColor := func(cpu float64) string {
			switch {
			case cpu >= 80:
				return "var(--high-usage)"
			case cpu >= 50:
				return "var(--medium-usage)"
			default:
				return "var(--low-usage)"
			}
		}

		return vdom.E("div",
			vdom.Class("toolbar"),
			// CPU Section
			vdom.E("div",
				vdom.Class("metric-group"),
				vdom.E("span",
					vdom.Class("label"),
					"CPU",
				),
				vdom.E("div",
					vdom.Class("progress-container"),
					vdom.E("div",
						vdom.Class("progress-bar"),
						vdom.PStyle("width", fmt.Sprintf("%d%%", int(metrics.CPUPercent))),
						vdom.PStyle("backgroundColor", getCPUColor(metrics.CPUPercent)),
					),
				),
				vdom.E("span",
					vdom.Class("value"),
					fmt.Sprintf("%.1f%%", metrics.CPUPercent),
				),
			),
			// Divider
			vdom.E("div", vdom.Class("divider")),
			// Load Average Section
			vdom.E("div",
				vdom.Class("metric-group"),
				vdom.E("span",
					vdom.Class("label"),
					"Load",
				),
				vdom.E("span",
					vdom.Class("value"),
					fmt.Sprintf("%.2f %.2f %.2f",
						metrics.LoadAvg[0],
						metrics.LoadAvg[1],
						metrics.LoadAvg[2],
					),
				),
			),
		)
	},
)

func main() {
	AppClient.RunMain()
}
