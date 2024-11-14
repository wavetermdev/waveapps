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

## Building Elements with vdom.E()

The E function creates virtual DOM elements. Its first argument is always a tag name (string), followed by any number of:

```go
// Props can be set in several ways:

// 1. Individual props with P(name, value):
vdom.E("div",
    vdom.P("className", "my-class"),
    vdom.P("onClick", handleClick),
)

// 2. Direct prop maps (map[string]any):
vdom.E("div",
    map[string]any{
        "className": "container",
        "id": "main",
    },
)

// 3. Convert structs to props with Props():
type ButtonProps struct {
    Text    string `json:"text"`
    OnClick func() `json:"onClick"`
}
vdom.E("Button", 
    vdom.Props(ButtonProps{  // Props() required for structs
        Text: "Click me",
        OnClick: handleClick,
    }),
)

// 4. Class helpers (work directly in E() like P()):
vdom.E("div",
    vdom.Class("base"),                    // Always included
    vdom.ClassIf(isActive, "active"),      // Included if condition true
    vdom.ClassIfElse(isDone,              // Choose based on condition
        "completed",
        "pending",
    ),
)

// Children can be:
// 1. Other elements
vdom.E("div",
    vdom.E("span", nil, "Hello"),
    vdom.E("span", nil, "World"),
)

// 2. Strings (become text nodes)
vdom.E("div", "Hello world")

// 3. Numbers (converted to string text nodes)
vdom.E("div", 42)

// 4. Arrays of any of the above
elements := []any{
    vdom.E("span", nil, "First"),
    "Some text",
    42,
}
vdom.E("div", elements)

// Mix everything together:
vdom.E("div",
    vdom.Class("container"),          // base class
    vdom.ClassIf(isActive, "active"), // conditional class
    map[string]any{"id": "main"},     // prop map
    vdom.E("h1", nil, "Title"),       // child element
    "Some text",                      // text node
    42,                               // number -> text node
    elements,                         // array of children
)
```

All arguments after the tag name can be:
- Props: via P(), map[string]any, or Props(struct)
- Classes: via Class(), ClassIf(), ClassIfElse().  Note that Class() only supports a single class.  To add multiple, use multiple calls.
- Children: elements, strings, numbers, arrays
- Anything with String() method (uses result as text node)

## Conditional Rendering and Lists

The system provides helper functions for conditional and list rendering:

```go
// Conditional rendering with vdom.If()
vdom.E("div",
    vdom.If(isVisible, vdom.E("span", nil, "Visible content")),
)

// Branching with vdom.IfElse()
vdom.E("div",
    vdom.IfElse(isActive,
        vdom.E("span", "Active"),
        vdom.E("span", "Inactive"),
    ),
)

// List rendering with vdom.ForEach()
items := []string{"A", "B", "C"}
vdom.E("ul",
    vdom.ForEach(items, func(item string) any {
        return vdom.E("li", item)
    }),
)

// List rendering with indexes with vdom.ForEachIdx()
items := []string{"A", "B", "C"}
vdom.E("ul",
    vdom.ForEachIdx(items, func(item string, idx int) any {
        return vdom.E("li", vdom.P("key", idx), item)
    }),
)
```

- Use vdom.Fragment(...any) to combine elements together into a group.  Useful with the conditional functions.
- func If(cond bool, part any) any
- func IfElse(cond bool, part any, elsePart any) any
- func ForEach[T any](items []T, fn func(T) any) []any
- func ForEachIdx[T any](items []T, fn func(T, int) any) []any

## Style Handling

Use PStyle to set individual style properties:

```go
vdom.E("div",
    vdom.PStyle("marginRight", 10),     // Numbers for px values
    vdom.PStyle("backgroundColor", "#fff"),
    vdom.PStyle("display", "flex"),
    vdom.PStyle("fontSize", 16),
    vdom.PStyle("borderRadius", 4),
)
```

Properties use camelCase (must match React) and values can be:
- Numbers (automatically handled for px values)
- Colors as strings
- Other CSS values as strings

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

## Component Definition Pattern

Create typed, reusable components using the client:

```go
type TodoItemProps struct {
    Todo     Todo    `json:"todo"`
    OnToggle func()  `json:"onToggle"`
}

var TodoItem = waveapp.DefineComponent[TodoItemProps](AppClient, "TodoItem",
    func(ctx context.Context, props TodoItemProps) any {
        return vdom.E("div",
            vdom.P("className", "todo-item"),
            vdom.P("onClick", props.OnToggle),
            props.Todo.Text,
        )
    },
)

// Usage:
TodoItem(TodoItemProps{
    Todo: todo,
    OnToggle: handleToggle,
})
```

## Handler Functions

For most event handling, passing a function directly works:

```go
vdom.E("button",
    vdom.P("onClick", func() { 
        fmt.Println("clicked!") 
    }),
)
```

If you need to call stopPropagation, preventDefault, For keyboard events that need special handling, use VDomFunc:

```go
// Handle specific keys with onKeyDown
keyHandler := &vdom.VDomFunc{
    Type: vdom.ObjectType_Func,
    Fn: func(event vdom.VDomEvent) {
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

vdom.E("input",
    vdom.P("onKeyDown", keyHandler),
)
```

The Keys field on VDomFunc:
- Only works with onKeyDown events
- Format is "[modifier]:key" or just "key"
- Modifiers:
  - Shift, Control, Meta, Alt: work as expected
  - Cmd: maps to Meta on Mac, Alt on Windows/Linux
  - Option: maps to Alt on Mac, Meta on Windows/Linux


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

    return vdom.E("div",
        // Use DOM ref to get element properties
        vdom.E("input",
            vdom.P("ref", inputRef),
            vdom.P("type", "text"),
        ),
        vdom.E("div", nil, 
            fmt.Sprintf("State: %d, Renders: %d", count, counter.Current),
        ),
    )
}
```

## State Management with Hooks

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
   vdom.E("input",
       vdom.P("ref", inputRef),
       vdom.P("type", "text"),
   )
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

        return vdom.E("div",
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
vdom.E("img", 
    vdom.P("src", "vdom:///logo.png"),  // Note the vdom:// prefix
    vdom.P("alt", "Logo"),
    vdom.PStyle("background", "url(vdom:///logo.png)"), // vdom urls can be used in CSS as well
)
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
        return vdom.E("div", nil,
            // Access CLI flags/args directly
            vdom.E("div", nil, "Path: ", myPath),
            vdom.E("div", nil, "Verbose: ", *verbose),
            vdom.E("button",
                vdom.P("onClick", func() { setCount(count + 1) }),
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
- Use vdom.Props() when passing structured props to vdom.E()
- Always use waveapp.DefineComponent with the client instance
- Use PStyle for cleaner style property setting
- Call SendAsyncInitiation() after async state updates
- Provide keys when using ForEach() with lists
- Consider cleanup functions in UseEffect() for async operations
- <script> tags are not supported
- Applications consist of exactly two files:
  - main.go: Contains all Go code and component definitions
  - style.css: Contains all styling (embedded into the app)
- This is a pure Go system - do not attempt to write React components or JavaScript code
- All UI rendering, including complex visualizations, should be done through Go using vdom.E()

The attached todo app demonstrates all these patterns in a complete application.
