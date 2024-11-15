# WaveApp System Guide

Wave Terminal includes a powerful WaveApp system that lets developers create rich HTML/React-based UI applications directly from Go code. The system translates Go components and elements into React components that are rendered within Wave Terminal's UI. It's particularly well-suited for administrative interfaces, monitoring dashboards, data visualization, configuration managers, and form-based applications where you want a graphical interface but don't need complex browser-side interactions.

This guide explains how to use the WaveApp system (and corresponding VDOM, virtual DOM, component) to create interactive applications that run in Wave Terminal. While the patterns will feel familiar to React developers (components, props, hooks), the implementation is pure Go and takes advantage of Go's strengths like goroutines for async operations. Note that complex browser-side interactions like drag-and-drop, rich text editing, or heavy JavaScript functionality are not supported - the framework is designed for straightforward, practical terminal-based applications.

You'll learn how to:
- Create and compose components
- Manage state and handle events
- Work with styles and CSS
- Handle async operations with goroutines
- Create rich UIs that render in Wave Terminal

The included todo-main.go provides a complete example application showing these patterns in action.

## Client Setup and Registration

The WaveApp client should be created as a global variable using AppOpts:

```go
// Create client at package level
var AppClient *waveapp.Client = waveapp.MakeClient(waveapp.AppOpts{
    CloseOnCtrlC: true,
    GlobalStyles: styleCSS,
})

// Components are registered with the client
var MyComponent = waveapp.DefineComponent[MyProps](AppClient, "MyComponent",
    func(ctx context.Context, props MyProps) any {
        // component logic
    },
)
```

## Building Elements with vdom.H()

The H function creates virtual DOM elements following a React-like pattern. It takes a tag name, a props map, and any number of children:

```go
// Basic element with no props
vdom.H("div", nil, "Hello world")

// Element with props
vdom.H("div", map[string]any{
    "className": "container",
    "id": "main",
    "onClick": func() {
        fmt.Println("clicked!")
    },
}, 
    "child content",
)

// Element with style
vdom.H("div", map[string]any{
    "style": map[string]any{
        "marginTop": 10,      // Numbers automatically convert to px
        "color": "red",
        "display": "flex",
    },
})

// Working with classes
vdom.H("div", map[string]any{
    "className": vdom.Classes(
        "base",                     // Static classes
        vdom.If(isActive, "active"),    // Conditional class: condition first, then class
        vdom.If(isDisabled, "disabled"), // Another conditional
    ),
})

// Nesting elements
vdom.H("div", map[string]any{
    "className": "container",
}, 
    vdom.H("h1", map[string]any{
        "className": "title",
    }, "Hello"),
    vdom.H("p", map[string]any{
        "className": "content",
    }, "Some content"),
)

// Handling events
vdom.H("button", map[string]any{
    "onClick": func() { 
        handleClick() 
    },
    "onKeyDown": &vdom.VDomFunc{
        Fn: handleKey,
        Keys: []string{"Enter", "Space"},
        PreventDefault: true,
    },
})

// List rendering
vdom.H("ul", nil,
    vdom.ForEachIdx(items, func(item string, idx int) any {
        return vdom.H("li", map[string]any{
            "key": idx,
            "className": "list-item",
        }, item)
    }),
)

// Conditional rendering
vdom.H("div", nil,
    vdom.If(isVisible, vdom.H("span", nil, "Visible content")),
)
```

Arguments to H:
1. `tag` (string): The HTML tag name
2. `props` (map[string]any or nil): Props map including:
   - className: String of space-separated classes
   - style: map[string]any of CSS properties
   - Event handlers (onClick, onChange, etc)
   - Any other valid HTML attributes
3. `children` (...any): Any number of child elements:
   - Other H() elements
   - Strings (become text nodes)
   - Numbers (converted to string)
   - Arrays of the above
   - nil values are ignored
   - Anything with String() method becomes text

Best practices:
- Use Classes() with If() for conditional classes
- Use camelCase for style properties (matching React)
- Numbers in style are automatically converted to pixel values
- Always create new slices when updating arrays in state
- Use ForEach or ForEachIdx for list rendering
- Include key prop when rendering lists

## Conditional Rendering and Lists

The system provides helper functions for conditional and list rendering:

```go
// Conditional rendering with vdom.If()
vdom.H("div", nil,
    vdom.If(isVisible, 
        vdom.H("span", nil, "Visible content"),
    ),
)

// Branching with vdom.IfElse()
vdom.H("div", nil,
    vdom.IfElse(isActive,
        vdom.H("span", nil, "Active"),
        vdom.H("span", nil, "Inactive"),
    ),
)

// List rendering (adding "key" prop to li element)
items := []string{"A", "B", "C"}
vdom.H("ul", nil,
    vdom.ForEachIdx(items, func(item string, idx int) any {
        return vdom.H("li", map[string]any{
            "key": idx,
            "className": "list-item",
        }, item)
    }),
)
```

Helper functions:
- `vdom.Fragment(...any)` - Combines elements into a group without adding a DOM node. Useful with conditional functions.
- `vdom.If(cond bool, part any) any` - Returns part if condition is true, nil otherwise
- `vdom.IfElse(cond bool, truePart any, falsePart any) any` - Returns truePart if condition is true, falsePart otherwise
- `vdom.ForEach[T any](items []T, fn func(T) any) []any` - Maps over items without index
- `vdom.ForEachIdx[T any](items []T, fn func(T, int) any) []any` - Maps over items with index
- `vdom.Filter[T any](items []T, fn func(T) bool) []T` - Filters items based on condition
- `vdom.FilterIdx[T any](items []T, fn func(T, int) bool) []T` - Filters items with index access

- The same If/IfElse functions are used for both conditional rendering and conditional classes, always following the pattern of condition first, then value(s).
- Remember to use vdom.IfElse if you need a true ternary condition.  vdom.If will return nil on false and does not allow a 3rd argument.

## Style Handling

Styles are defined using a map[string]any in the props:

```go
vdom.H("div", map[string]any{
    "style": map[string]any{
        "marginRight": 10,         // Numbers for px values
        "backgroundColor": "#fff", // Colors as strings
        "display": "flex",         // CSS values as strings
        "fontSize": 16,            // More numbers
        "borderRadius": 4,         // Numbers to px
    },
})

// Multiple style properties can be combined with dynamic values
vdom.H("div", map[string]any{
    "style": map[string]any{
        "marginTop": spacing,      // Variables work too
        "color": vdom.IfElse(isActive, "blue", "gray"),
        "display": "flex",
        "opacity": vdom.If(isVisible, 1.0),  // Conditional styles
    },
})
```

Properties use camelCase (must match React) and values can be:
- Numbers (automatically converted to pixel values)
- Colors as strings
- Other CSS values as strings
- Conditional values using If/IfElse

The style map in props mirrors React's style object pattern, making it familiar to React developers while maintaining type safety in Go.

## External CSS Files

Your application styles should be stored in an external file, canonically called "style.css".
Embed and provide them through AppOpts:

```go
//go:embed style.css
var styleCSS []byte

var AppClient *waveapp.Client = waveapp.MakeClient(waveapp.AppOpts{
    CloseOnCtrlC: true,
    GlobalStyles: styleCSS,
})
```

Note that the default environment where these apps will launch is a dark mode terminal.  So try to pick text colors and background colors that will work well against a dark client.  Normally the background is transparent (to a dark-ish background).  So either try to leave the background transparent (untouched), or if the transparency is not desirable, then choose a dark background color that would be appropriate.

## Component Definition Pattern

Create typed, reusable components using the client:

```go
// Define prop types with json tags
type TodoItemProps struct {
    Todo     Todo    `json:"todo"`
    OnToggle func()  `json:"onToggle"`
    IsActive bool    `json:"isActive"`
}

// Create component with typed props
var TodoItem = waveapp.DefineComponent[TodoItemProps](AppClient, "TodoItem",
    func(ctx context.Context, props TodoItemProps) any {
        return vdom.H("div", map[string]any{
            "className": vdom.Classes(
                "todo-item",
                vdom.If(props.IsActive, "active"),
            ),
            "onClick": props.OnToggle,
            "style": map[string]any{
                "cursor": "pointer",
                "opacity": vdom.If(props.IsActive, 1.0, 0.7),
            },
        }, props.Todo.Text)
    },
)

// Usage in parent component:
vdom.H("div", map[string]any{
    "className": "todo-list",
},
    TodoItem(TodoItemProps{
        Todo: todo,
        OnToggle: handleToggle,
        IsActive: isCurrentItem,
    }),
)

// Usage with key (when in lists)
TodoItem(TodoItemProps{
    Todo: todo,
    OnToggle: handleToggle,
}).WithKey(fmt.Sprint(idx))
```

Components in WaveApp:
- Use Go structs with json tags for props
- Take a context and props as arguments
- Return elements created with vdom.H()
- Can use all hooks (useState, useRef, etc)
- Are registered with the client and given a name
- Are called as functions with their props struct

Special Handling for Component "key" prop:
- Use the WithKey(key string) chaining func to set a key on a component
- Keys must be added for components rendered in lists (just like in React)
- Keys should be unique among siblings and stable across renders
- Keys are handled at the framework level and should not be declared in component props

This pattern matches React's functional components while maintaining Go's type safety and explicit props definition.

## Handler Functions

For most event handling, passing a function directly in the props map works:

```go
vdom.H("button", map[string]any{
    "onClick": func() { 
        fmt.Println("clicked!") 
    },
})

// With event data
vdom.H("input", map[string]any{
    "onChange": func(e vdom.VDomEvent) {
        fmt.Println("new value:", e.TargetValue)
    },
})
```

For keyboard events that need special handling, preventDefault, or stopPropagation, use VDomFunc:

```go
// Handle specific keys with onKeyDown
keyHandler := &vdom.VDomFunc{
    Type:            vdom.ObjectType_Func,
    Fn:              func(event vdom.VDomEvent) {
        // handle key press
    },
    StopPropagation: true,    // Stop event bubbling
    PreventDefault: true,     // Prevent default browser behavior
    Keys: []string{
        "Enter",              // Just Enter key
        "Shift:Tab",          // Shift+Tab
        "Control:c",          // Ctrl+C
        "Meta:v",             // Meta+V (Windows)/Command+V (Mac)
        "Alt:x",              // Alt+X
        "Cmd:s",             // Command+S (Mac)/Alt+S (Windows)
        "Option:f",          // Option+F (Mac)/Meta+F (Windows)
    },
}

vdom.H("input", map[string]any{
    "className": "special-input",
    "onKeyDown": keyHandler,
})

// Common pattern for form handling
vdom.H("form", map[string]any{
    "onSubmit": &vdom.VDomFunc{
        Fn:             handleSubmit,
        PreventDefault: true,  // Prevent form submission
    },
})
```

The Keys field on VDomFunc:
- Only works with onKeyDown events
- Format is "[modifier]:key" or just "key"
- Modifiers:
  - Shift, Control, Meta, Alt: work as expected
  - Cmd: maps to Meta on Mac, Alt on Windows/Linux
  - Option: maps to Alt on Mac, Meta on Windows/Linux

Event handlers follow React patterns while providing additional type safety and explicit control over event behavior through VDomFunc.

## State Management with Hooks

```go
func MyComponent(ctx context.Context, props MyProps) any {
    // UseState: returns current value and setter function
    // The setter function triggers a re-render when called
    count, setCount := vdom.UseState(ctx, 0)     // Initial value of 0
    items, setItems := vdom.UseState(ctx, []string{}) // Initial value of empty slice
    
    // Example of state updates
    incrementCount := func() {
        setCount(count + 1)  // Triggers re-render
    }

    addItem := func(item string) {
        // When updating slices/maps, create new value
        setItems(append([]string{}, items..., item))
    }
    
    // Refs for values that persist between renders but don't trigger updates
    counter := vdom.UseRef(ctx, 0)
    counter.Current++  // Doesn't cause re-render
    
    // DOM refs for accessing elements directly
    inputRef := vdom.UseVDomRef(ctx)
    
    // Side effects
    vdom.UseEffect(ctx, func() func() {
        // Can use refs in effects
        fmt.Printf("Render count: %d\n", counter.Current)
        
        return func() {
            // cleanup
        }
    }, []any{count})

    return vdom.H("div", nil,
        // Use DOM ref to get element properties
        vdom.H("input", map[string]any{
            "ref": inputRef,
            "type": "text",
        }),
        vdom.H("div", nil, 
            fmt.Sprintf("State: %d, Renders: %d", count, counter.Current),
        ),
    )
}
```

## Available Hooks

The system provides four main types of hooks:

1. `UseState` - For values that trigger re-renders when changed:
   - Returns current value and setter function
   - Setter function triggers component re-render
   - Create new values for slices/maps when updating
   ```go
   count, setCount := vdom.UseState(ctx, 0)
   setCount(count + 1) // Triggers re-render
   ```

2. `UseStateWithFn` - For safe state updates based on current value:
   - Returns current value, direct setter, and functional setter
   - Useful for state updates in goroutines or when new value depends on current
   - Ensures you're always working with latest state value
   ```go
   count, setCount, setCountFn := vdom.UseStateWithFn(ctx, 0)
   // Direct update when you have the value:
   setCount(42)
   // Functional update when you need current value:
   setCountFn(func(current int) int {
       return current + 1
   })
   ```

3. `UseRef` - For values that persist between renders without triggering updates:
   - Holds mutable values that survive re-renders
   - Changes don't cause re-renders
   - Perfect for:
     - Managing goroutine state
     - Storing timers/channels
     - Tracking subscriptions
     - Holding complex state structures
   ```go
   timerRef := vdom.UseRef(ctx, &TimerState{
       done: make(chan bool),
   })
   ```

4. `UseVDomRef` - For accessing DOM elements directly:
   - Creates refs for DOM interaction
   - Useful for:
     - Accessing DOM element properties
     - Managing focus
     - Measuring elements
     - Direct DOM manipulation when needed
   ```go
   inputRef := vdom.UseVDomRef(ctx)
   vdom.H("input", map[string]any{
       "ref": inputRef,
       "type": "text",
   })
   ```

Best Practices:
- Use `UseState` for simple UI state
- Use `UseStateWithFn` when updating state from goroutines or based on current value
- Use `UseRef` for complex state that goroutines need to access
- Always clean up timers, channels, and goroutines in UseEffect cleanup functions
     
## State Management and Async Updates

While React patterns typically avoid globals, in Go WaveApp applications it's perfectly fine and often clearer to use global variables. However, when dealing with async operations and goroutines, special care must be taken:

```go
// Global state is fine!
var globalTodos []Todo
var globalFilter string

// For async operations, consider using a state struct
type TimerState struct {
    ticker   *time.Ticker
    done     chan bool
    isActive bool
}

var TodoApp = waveapp.DefineComponent[struct{}](AppClient, "TodoApp",
    func(ctx context.Context, _ struct{}) any {
        // Local state for UI updates
        count, setCount := vdom.UseState(ctx, 0)
        
        // For state updates that depend on current value, use UseStateWithFn
        seconds, setSeconds, setSecondsFn := vdom.UseStateWithFn(ctx, 0)
        
        // Use refs to store complex state that goroutines need to access
        stateRef := vdom.UseRef(ctx, &TimerState{
            done: make(chan bool),
        })

        // Example of safe goroutine management
        startAsync := func() {
            if stateRef.Current.isActive {
                return // Prevent multiple goroutines
            }
            
            stateRef.Current.isActive = true
            go func() {
                defer func() {
                    stateRef.Current.isActive = false
                }()
                
                // Use channels for cleanup
                for {
                    select {
                    case <-stateRef.Current.done:
                        return
                    case <-time.After(time.Second):
                        // Use functional updates for state that depends on current value
                        setSecondsFn(func(s int) int {
                            return s + 1
                        })
                        // Notify UI of update
                        AppClient.SendAsyncInitiation()
                    }
                }
            }()
        }

        // Always clean up goroutines
        stopAsync := func() {
            if stateRef.Current.isActive {
                close(stateRef.Current.done)
                stateRef.Current.done = make(chan bool)
            }
        }

        // Use UseEffect for cleanup on unmount
        vdom.UseEffect(ctx, func() func() {
            return func() {
                stopAsync()
            }
        }, []any{})

        return vdom.H("div", nil,
            vdom.ForEach(globalTodos, func(todo Todo) any {
                return TodoItem(TodoItemProps{Todo: todo})
            }),
        )
    },
)
```

Key points for state management:
- Global state is fine for simple data structures
- Use `UseStateWithFn` when updating state based on its current value, especially in goroutines
- Store complex state in refs when it needs to be accessed by goroutines
- Use `UseEffect` cleanup function to handle component unmount
- Call `SendAsyncInitiation()` after state changes in goroutines (consider round trip performance, so don't call at very high speeds)
- Use atomic operations if globals are modified from multiple goroutines (or locks)

## Global Keyboard Handling

The WaveApp system provides two approaches for handling keyboard events:

1. Standard DOM event handling on elements:
```go
vdom.H("div", map[string]any{
    "onKeyDown": func(e vdom.VDomEvent) {
        // Handle key event
    },
})
```

2. Global keyboard event handling:
```go
var AppClient = waveapp.MakeClient(waveapp.AppOpts{
    CloseOnCtrlC:         true,
    GlobalKeyboardEvents: true,  // Enable global keyboard events
    GlobalStyles:         styleCSS,
})

// In main() or an effect:
AppClient.SetGlobalEventHandler(func(client *waveapp.Client, event vdom.VDomEvent) {
    if event.EventType != "onKeyDown" || event.KeyData == nil {
        return
    }

    switch event.KeyData.Key {
    case "ArrowUp":
        // Handle up arrow
    case "ArrowDown":
        // Handle down arrow
    }
})
```

The global handler approach is particularly useful when:
- You need to handle keyboard events regardless of focus state
- Building terminal-like applications that need consistent keyboard control
- Implementing application-wide keyboard shortcuts
- Managing navigation in full-screen applications

Key differences:
- Standard DOM events require the element to have focus
- Global events work regardless of focus state
- Global events can be used alongside regular DOM event handlers
- Global handler receives all keyboard events for the application

The event handler receives a VDomEvent with KeyData for keyboard events:
```go
type VDomEvent struct {
    EventType       string             // e.g., "onKeyDown"
    KeyData         *WaveKeyboardEvent `json:"keydata,omitempty"`
    // ... other fields
}

type WaveKeyboardEvent struct {
    Type     string // "keydown", "keyup", "keypress"
    Key      string // The key value (e.g., "ArrowUp")
    Code     string // Physical key code
    Shift    bool   // Modifier states
    Control  bool
    Alt      bool
    Meta     bool
    Cmd      bool   // Meta on Mac, Alt on Windows/Linux
    Option   bool   // Alt on Mac, Meta on Windows/Linux
}
```

When using global keyboard events, remember to:
1. Enable GlobalKeyboardEvents in AppOpts
2. Set up the handler in a place where you have access to necessary state updates

## File Handling

The WaveApp system can serve files to components. Any URL starting with `vdom://` will be handled by the registered handlers:

```go
// Register handlers for files (in the main func)
AppClient.RegisterFileHandler("/logo.png", waveapp.FileHandlerOption{
    FilePath: "./assets/logo.png",
})

// returning nil will produce a 404, path will be the full path, including the prefix
AppClient.RegisterFilePrefixHandler("/img/", func(path string) (*waveapp.FileHandlerOption, error) {
    return &waveapp.FileHandlerOption{Data: data, MimeType: "image/png"}
})

// Use in components with vdom:// prefix
vdom.H("img", map[string]any{
    "src": "vdom:///logo.png",  // Note the vdom:// prefix
    "alt": "Logo",
    "style": map[string]any{
        "background": "url(vdom:///logo.png)", // vdom urls can be used in CSS as well
    },
})
```

Files can come from:
- Disk (FilePath)
- Memory (Data + MimeType)
- Stream (Reader)

```
type FileHandlerOption struct {
	FilePath string    // optional file path on disk
	Data     []byte    // optional byte slice content (easy to use with go:embed)
	Reader   io.Reader // optional reader for content
	File     fs.File   // optional embedded or opened file
	MimeType string    // optional mime type
	ETag     string    // optional ETag (if set, resource may be cached)
}
```

Any URL passed to src attributes that starts with vdom:// will be handled by the registered handlers.  All registered paths should be absolute (start with "/").

Note that the system will attempt to detect the file type using the first 512 bytes of content.  This works great for images, videos, and binary files.  For text files which might be ambiguous (CSS, JSON, YAML, TOML, JavaScript, Go Code, Java, other programming languages) it can make sense to specify the mimetype (but usually only required if the frontend needs it for some reason).

By default, files will not be cached.  If you'd like to enable caching, pass an ETag.  If a subsequent request comes and the ETag matches the system will used the cached content.

## WaveApp Template

```go
package main

import (
    "context"
    _ "embed"
    "flag"
    "fmt"
    "os"
    "github.com/wavetermdev/waveterm/pkg/vdom"
    "github.com/wavetermdev/waveterm/pkg/waveapp"
)

//go:embed style.css
var styleCSS []byte

// Define CLI args/flags as global variables
var myPath string
var verbose = flag.Bool("v", false, "verbose output")

var AppClient = waveapp.MakeClient(waveapp.AppOpts{
    CloseOnCtrlC: true,
    GlobalStyles: styleCSS,
})

// Root component must be named "App" (it takes no props)
var App = waveapp.DefineComponent(AppClient, "App",
    func(ctx context.Context, _ any) any {
        count, setCount := vdom.UseState(ctx, 0)
        return vdom.H("div", nil,
            // Access CLI flags/args directly
            vdom.H("div", nil, "Path: ", myPath),
            vdom.H("div", nil, "Verbose: ", *verbose),
            vdom.H("button", map[string]any{
                "onClick": func() { setCount(count + 1) },
            },
                "Count: ", count,
            ),
        )
    },
)

func main() {
    // For custom CLI argument parsing:
    AppClient.RegisterDefaultFlags()
    flag.Parse()
    
    if flag.NArg() != 1 {
        fmt.Fprintf(os.Stderr, "Usage: myapp [flags] <path>\n")
        flag.PrintDefaults()
        os.Exit(1)
    }
    myPath = flag.Arg(0)
    
    // Optional: Add URL handlers
    AppClient.RegisterFileHandler("/api/data", waveapp.FileHandlerOption{...})
    AppClient.RegisterFilePrefixHandler("/img/", imageHandler)
    
    // Run the app
    AppClient.RunMain()
}
```

Key points:
1. Root component must be named "App"
2. Define any CLI flags/args as global variables
3. For apps with custom CLI parsing:
   - Call RegisterDefaultFlags() first (registers framework's -n flag)
   - Call flag.Parse()
   - Process args/show usage
4. If no custom CLI parsing is needed, do not call AppClient.RegisterDefaultFlags() or flag.Parse(). Just call AppClient.RunMain().

```
type AppOpts struct {
    CloseOnCtrlC         bool
    GlobalKeyboardEvents bool
    GlobalStyles         []byte
    RootComponentName    string // defaults to "App"
    NewBlockFlag         string // defaults to "n" (set to "-" to disable)
}
```

## Important Technical Details
- Props must be defined as Go structs with json tags
- Components take their props type directly: `func MyComponent(ctx context.Context, props MyProps) any`
- Always use waveapp.DefineComponent with the client instance
- Call SendAsyncInitiation() after async state updates
- Provide keys when using ForEachIdx() with lists
- Use Classes() with If() for combining static and conditional class names
- Consider cleanup functions in UseEffect() for async operations
- <script> tags are not supported
- Applications consist of exactly two files:
  - main.go: Contains all Go code and component definitions
  - style.css: Contains all styling (embedded into the app)
- This is a pure Go system - do not attempt to write React components or JavaScript code
- All UI rendering, including complex visualizations, should be done through Go using vdom.H()

The attached todo app demonstrates all these patterns in a complete application.
