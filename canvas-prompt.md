# VDOM Canvas Operations Guide

The VDOM system provides a powerful way to interact with HTML Canvas elements through the `QueueRefOp` function. This guide explains how to use canvas operations effectively in your VDOM applications.

## Basic Canvas Setup

First, create a canvas element and get a reference to it:

```go
canvasRef := vdom.UseVDomRef(ctx)
return vdom.E("canvas",
    vdom.P("ref", canvasRef),
    vdom.P("width", "300"),
    vdom.P("height", "300"),
    vdom.PStyle("width", 300),
    vdom.PStyle("height", 300),
)
```

## Queuing Canvas Operations

Use `QueueRefOp` to send operations to the canvas. Each operation maps directly to a Canvas 2D context method:

```go
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "fillStyle",
    Params: []any{"#ff0000"},
})

vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "fillRect",
    Params: []any{0, 0, 100, 100},
})
```

## Reference Management

The VDOM canvas system has two key uses for references:

1. **Capturing Canvas API Return Values**  
Some canvas operations return objects (like gradients) that cannot be directly serialized. Use the `Ref` field to capture these return values:

```go
// Create and capture a gradient
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "createLinearGradient",
    Params: []any{0, 0, 200, 0},
    Ref:    "myGradient",  // Captures the returned gradient object
})

// Use the gradient in another operation
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "fillStyle",
    Params: []any{"#ref:myGradient"},  // Reference the captured gradient
})
```

2. **Storing Complex Data**  
You can also store and reuse any JSON-compatible data:

```go
// Store configuration data
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "addRef",
    Ref:    "pointsConfig",
    Params: []any{[]float64{10, 20, 30, 40, 50, 60}},
})

// Use stored data with spreadRef
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "moveTo",
    Params: []any{"#spreadRef:pointsConfig"},
})

// Clean up when done
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "dropRef",
    Params: []any{"pointsConfig"},
})
```

Reference operations support:
- `#ref:id` - Use the referenced value directly (crucial for canvas-created objects)
- `#spreadRef:id` - Spread an array reference as multiple parameters

## Common Canvas Operations

Here are commonly used canvas operations:

1. **Basic Drawing**
```go
// Clear canvas
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "clearRect",
    Params: []any{0, 0, width, height},
})

// Set fill color
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "fillStyle",
    Params: []any{"rgba(255, 0, 0, 0.5)"},
})

// Draw rectangle
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "fillRect",
    Params: []any{x, y, width, height},
})
```

2. **Path Operations**
```go
// Begin a new path
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "beginPath",
    Params: nil,
})

// Draw circle
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "arc",
    Params: []any{x, y, radius, 0, 2 * math.Pi},
})

// Fill the path
vdom.QueueRefOp(ctx, canvasRef, vdom.VDomRefOperation{
    Op:     "fill",
    Params: nil,
})
```

## Animation and Updates

For smooth animations, use UseEffect with a render timestamp:

```go
renderTs := vdom.UseRenderTs(ctx)
lastRenderTs := vdom.UseRef(ctx, int64(0))

vdom.UseEffect(ctx, func() func() {
    if !canvasRef.HasCurrent {
        return nil
    }
    // Limit frame rate
    if renderTs-lastRenderTs.Current < 30 {
        return nil
    }
    lastRenderTs.Current = renderTs
    
    // Queue drawing operations here
    
    // Schedule next frame
    go func() {
        time.Sleep(60 * time.Millisecond)
        AppClient.SendAsyncInitiation()
    }()
    
    return nil
}, []any{renderTs})
```

## Best Practices

1. **Performance**
   - Batch related operations together
   - Use `clearRect` instead of resizing the canvas
   - Limit animation frame rates with render timestamps
   - Clean up any animation loops in UseEffect cleanup functions

2. **Reference Management**
   - Clean up references with `dropRef` when no longer needed
   - Use `#spreadRef` when passing array data to operations
   - Store frequently used complex data as references

3. **Error Handling**
   - Check `canvasRef.HasCurrent` before operations
   - Validate coordinates and dimensions
   - Use proper cleanup in UseEffect returns

## Complete Example: Particle System

```go
type Particle struct {
    X, Y      float64
    VelocityX float64
    VelocityY float64
    Color     string
    Size      float64
}

type CanvasUpdaterProps struct {
    CanvasRef *vdom.VDomRef
}

var CanvasUpdater = waveapp.DefineComponent[CanvasUpdaterProps](
    Client, "CanvasUpdater",
    func(ctx context.Context, props CanvasUpdaterProps) any {
        particles, setParticles := vdom.UseState(ctx, initParticles(10))
        lastRenderTs := vdom.UseRef(ctx, int64(0))
        renderTs := vdom.UseRenderTs(ctx)
        
        vdom.UseEffect(ctx, func() func() {
            if !props.CanvasRef.HasCurrent {
                return nil
            }
            if renderTs-lastRenderTs.Current < 30 {
                return nil
            }
            lastRenderTs.Current = renderTs

            // Clear canvas
            vdom.QueueRefOp(ctx, props.CanvasRef, vdom.VDomRefOperation{
                Op:     "clearRect",
                Params: []any{0, 0, 300, 300},
            })

            // Draw particles
            for _, p := range particles {
                // Set color
                vdom.QueueRefOp(ctx, props.CanvasRef, vdom.VDomRefOperation{
                    Op:     "fillStyle",
                    Params: []any{p.Color},
                })
                
                // Draw circle
                vdom.QueueRefOp(ctx, props.CanvasRef, vdom.VDomRefOperation{
                    Op:     "beginPath",
                    Params: nil,
                })
                vdom.QueueRefOp(ctx, props.CanvasRef, vdom.VDomRefOperation{
                    Op:     "arc",
                    Params: []any{p.X, p.Y, p.Size, 0, 2 * math.Pi},
                })
                vdom.QueueRefOp(ctx, props.CanvasRef, vdom.VDomRefOperation{
                    Op:     "fill",
                    Params: nil,
                })
            }

            // Update positions for next frame
            newParticles := updateParticles(particles)
            setParticles(newParticles)

            // Schedule next frame
            go func() {
                time.Sleep(60 * time.Millisecond)
                Client.SendAsyncInitiation()
            }()

            return nil
        }, []any{renderTs})

        return nil
    },
)
```

This guide covers the essential patterns for working with Canvas in VDOM applications. Remember that while Canvas operations are powerful, they should be used judiciously in terminal-based applications to maintain good performance.