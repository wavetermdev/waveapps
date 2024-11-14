package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/waveapp"
)

//go:embed style.css
var styleCSS []byte

var AppClient = waveapp.MakeClient(waveapp.AppOpts{
	CloseOnCtrlC: true,
	GlobalStyles: styleCSS,
})

type DataPoint struct {
	Value float64
}

const (
	canvasWidth  = 800
	canvasHeight = 400
	padding      = 40
	pointRadius  = 3 // Made slightly smaller
)

func calculateStats(points []DataPoint) (count int, avg float64) {
	count = len(points)
	if count == 0 {
		return count, 0
	}

	sum := 0.0
	for _, p := range points {
		sum += p.Value
	}
	return count, sum / float64(count)
}

var App = waveapp.DefineComponent(AppClient, "App",
	func(ctx context.Context, _ any) any {
		// State for our data points and update counter
		points, _, setPointsFn := vdom.UseStateWithFn(ctx, []DataPoint{})
		updateCount, _, setUpdateCountFn := vdom.UseStateWithFn(ctx, 0)

		// Reference for the canvas
		canvasRef := vdom.UseVDomRef(ctx)

		// Reference for managing the data reading goroutine
		readerState := vdom.UseRef(ctx, struct {
			done   chan bool
			active bool
		}{
			done:   make(chan bool),
			active: false,
		})

		// Function to draw the graph
		drawGraph := func() {
			if !canvasRef.HasCurrent || len(points) == 0 {
				return
			}

			// Clear canvas
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "clearRect",
				Params: []any{0, 0, canvasWidth, canvasHeight},
			})

			// Find max value for scaling
			maxVal := points[0].Value
			for _, p := range points {
				if p.Value > maxVal {
					maxVal = p.Value
				}
			}
			maxVal = maxVal * 1.05 // Add 5% buffer

			// Calculate scales
			xScale := float64(canvasWidth-2*padding) / math.Max(float64(len(points)-1), 1)
			yScale := float64(canvasHeight-2*padding) / maxVal

			// Draw grid (new!)
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "strokeStyle",
				Params: []any{"#333333"},
			})
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "lineWidth",
				Params: []any{1},
			})

			// Vertical grid lines
			for i := 0; i < 10; i++ {
				x := padding + (float64(i) * (canvasWidth - 2*padding) / 9)
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "beginPath",
					Params: nil,
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "moveTo",
					Params: []any{x, padding},
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "lineTo",
					Params: []any{x, canvasHeight - padding},
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "stroke",
					Params: nil,
				})
			}

			// Horizontal grid lines
			for i := 0; i < 10; i++ {
				y := padding + (float64(i) * (canvasHeight - 2*padding) / 9)
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "beginPath",
					Params: nil,
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "moveTo",
					Params: []any{padding, y},
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "lineTo",
					Params: []any{canvasWidth - padding, y},
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "stroke",
					Params: nil,
				})
			}

			// Draw axes
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "beginPath",
				Params: nil,
			})
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "strokeStyle",
				Params: []any{"#666666"},
			})
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "lineWidth",
				Params: []any{2},
			})

			// Y axis
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "moveTo",
				Params: []any{padding, padding},
			})
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "lineTo",
				Params: []any{padding, canvasHeight - padding},
			})

			// X axis
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "moveTo",
				Params: []any{padding, canvasHeight - padding},
			})
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "lineTo",
				Params: []any{canvasWidth - padding, canvasHeight - padding},
			})
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "stroke",
				Params: nil,
			})

			// Draw data line
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "beginPath",
				Params: nil,
			})
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "strokeStyle",
				Params: []any{"#4488ff"},
			})
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "lineWidth",
				Params: []any{2},
			})

			// Draw lines connecting points
			for i := 0; i < len(points); i++ {
				x := padding + (float64(i) * xScale)
				y := canvasHeight - padding - (points[i].Value * yScale)

				if i == 0 {
					vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
						Op:     "moveTo",
						Params: []any{x, y},
					})
				} else {
					vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
						Op:     "lineTo",
						Params: []any{x, y},
					})
				}
			}
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "stroke",
				Params: nil,
			})

			// Draw points
			for i := 0; i < len(points); i++ {
				x := padding + (float64(i) * xScale)
				y := canvasHeight - padding - (points[i].Value * yScale)

				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "beginPath",
					Params: nil,
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "fillStyle",
					Params: []any{"#4488ff"},
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "arc",
					Params: []any{x, y, pointRadius, 0, 2 * math.Pi},
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "fill",
					Params: nil,
				})
			}
		}

		// Effect to start reading data
		vdom.UseEffect(ctx, func() func() {
			if readerState.Current.active {
				return nil
			}

			readerState.Current.active = true
			go func() {
				scanner := bufio.NewScanner(os.Stdin)

				for scanner.Scan() {
					select {
					case <-readerState.Current.done:
						return
					default:
						text := strings.TrimSpace(scanner.Text())
						if value, err := strconv.ParseFloat(text, 64); err == nil {
							setPointsFn(func(points []DataPoint) []DataPoint {
								return append(points, DataPoint{Value: value})
							})
							setUpdateCountFn(func(updateCount int) int {
								return updateCount + 1
							})
							AppClient.SendAsyncInitiation()
						}
					}
				}
			}()

			return func() {
				close(readerState.Current.done)
				readerState.Current.active = false
				readerState.Current.done = make(chan bool)
			}
		}, []any{})

		// Effect to handle drawing
		vdom.UseEffect(ctx, func() func() {
			drawGraph()
			return nil
		}, []any{updateCount, canvasRef.HasCurrent})

		// Calculate stats
		count, avg := calculateStats(points)

		return vdom.E("div",
			vdom.Class("graph-container"),
			vdom.E("h1",
				vdom.Class("graph-title"),
				"Live Data Graph",
			),
			vdom.E("canvas",
				vdom.Class("graph-canvas"),
				vdom.P("ref", canvasRef),
				vdom.P("width", canvasWidth),
				vdom.P("height", canvasHeight),
			),
			vdom.E("div",
				vdom.Class("graph-stats"),
				fmt.Sprintf("Points: %d   Average: %.2f", count, avg),
			),
		)
	},
)

func main() {
	AppClient.RunMain()
}
