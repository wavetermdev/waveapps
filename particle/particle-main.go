package main

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/waveapp"
)

var AppClient *waveapp.Client = waveapp.MakeClient(waveapp.AppOpts{
	CloseOnCtrlC: true,
})

type Particle struct {
	X, Y      float64 // Position
	VelocityX float64 // Horizontal speed
	VelocityY float64 // Vertical speed
	Color     string  // Color in rgba format
	Size      float64 // Radius of the particle
}

type CanvasUpdaterProps struct {
	CanvasRef *vdom.VDomRef
}

var CanvasUpdaterParticles = waveapp.DefineComponent[CanvasUpdaterProps](AppClient, "CanvasUpdaterParticles",
	func(ctx context.Context, props CanvasUpdaterProps) any {
		particles, setParticles := vdom.UseState(ctx, initParticles(10))
		lastRenderTs := vdom.UseRef(ctx, int64(0))
		renderTs := vdom.UseRenderTs(ctx)
		canvasRef := props.CanvasRef

		vdom.UseEffect(ctx, func() func() {
			if !canvasRef.HasCurrent {
				return nil
			}
			if renderTs-lastRenderTs.Current < 30 {
				return nil
			}
			lastRenderTs.Current = renderTs

			// Update particle positions and redraw
			newParticles := updateParticles(particles)
			setParticles(newParticles)

			// Drawing operations on canvas
			vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
				Op:     "clearRect",
				Params: []any{0, 0, 300, 300},
			})

			for _, particle := range newParticles {
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "fillStyle",
					Params: []any{particle.Color},
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "beginPath",
					Params: []any{},
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "arc",
					Params: []any{particle.X, particle.Y, particle.Size, 0, 2 * math.Pi},
				})
				vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
					Op:     "fill",
					Params: []any{},
				})
			}

			// Trigger re-render based on tickNum, not particles
			go func() {
				time.Sleep(60 * time.Millisecond)
				AppClient.SendAsyncInitiation()
			}()

			return nil
		}, []any{renderTs}) // Using tickNum as the dependency

		return nil
	},
)

// Helper function to get a random direction (-1 or 1)
func randomDirection() float64 {
	if rand.Intn(2) == 0 {
		return -1
	}
	return 1
}

// Initialize particles with random positions, velocities, sizes, and colors
func initParticles(count int) []Particle {
	particles := make([]Particle, count)
	for i := 0; i < count; i++ {
		particles[i] = Particle{
			X:         float64(rand.Intn(300)),
			Y:         float64(rand.Intn(300)),
			VelocityX: float64(rand.Intn(3)+1) * randomDirection(),
			VelocityY: float64(rand.Intn(3)+1) * randomDirection(),
			Color:     fmt.Sprintf("rgba(%d, %d, %d, 0.7)", rand.Intn(256), rand.Intn(256), rand.Intn(256)),
			Size:      float64(rand.Intn(10) + 3), // Radius between 3 and 12
		}
	}
	return particles
}

// Update particle positions, bouncing off edges
func updateParticles(particles []Particle) []Particle {
	for i, particle := range particles {
		// Update position based on velocity
		particle.X += particle.VelocityX
		particle.Y += particle.VelocityY

		// Bounce off edges
		if particle.X <= 0 || particle.X >= 300 {
			particle.VelocityX = -particle.VelocityX
		}
		if particle.Y <= 0 || particle.Y >= 300 {
			particle.VelocityY = -particle.VelocityY
		}
		particles[i] = particle
	}
	return particles
}

var App = waveapp.DefineComponent[struct{}](AppClient, "App",
	func(ctx context.Context, _ struct{}) any {
		canvasRef := vdom.UseVDomRef(ctx)
		return vdom.E("div",
			vdom.E("canvas",
				vdom.P("ref", canvasRef),
				vdom.P("width", "300"), vdom.P("height", "300"),
				vdom.PStyle("width", 300), vdom.PStyle("height", 300),
			),
			CanvasUpdaterParticles(CanvasUpdaterProps{canvasRef}),
		)
	},
)

func main() {
	AppClient.RunMain()
}
