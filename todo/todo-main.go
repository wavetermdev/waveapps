package main

import (
	"context"
	_ "embed"

	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/waveapp"
)

//go:embed style.css
var styleCSS []byte

// Initialize client with embedded styles and ctrl-c handling
var AppClient *waveapp.Client = waveapp.MakeClient(waveapp.AppOpts{
	CloseOnCtrlC: true,
	GlobalStyles: styleCSS,
})

// Basic domain types with json tags for props
type Todo struct {
	Id        int    `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// Prop types demonstrate parent->child data flow
type TodoListProps struct {
	Todos    []Todo    `json:"todos"`
	OnToggle func(int) `json:"onToggle"`
	OnDelete func(int) `json:"onDelete"`
}

type TodoItemProps struct {
	Todo     Todo   `json:"todo"`
	OnToggle func() `json:"onToggle"`
	OnDelete func() `json:"onDelete"`
}

type InputFieldProps struct {
	Value    string       `json:"value"`
	OnChange func(string) `json:"onChange"`
	OnEnter  func()       `json:"onEnter"`
}

// Reusable input component showing keyboard event handling
var InputField = waveapp.DefineComponent[InputFieldProps](AppClient, "InputField",
	func(ctx context.Context, props InputFieldProps) any {
		// Example of special key handling with VDomFunc
		keyDown := &vdom.VDomFunc{
			Type:            vdom.ObjectType_Func,
			Fn:              func(event vdom.VDomEvent) { props.OnEnter() },
			StopPropagation: true,
			PreventDefault:  true,
			Keys:            []string{"Enter", "Cmd:Enter"},
		}

		return vdom.E("input",
			vdom.Class("todo-input"),
			// Basic input element props
			vdom.P("type", "text"),
			vdom.P("placeholder", "What needs to be done?"),
			vdom.P("value", props.Value),
			// Event handler accessing target value
			vdom.P("onChange", func(e vdom.VDomEvent) {
				props.OnChange(e.TargetValue)
			}),
			vdom.P("onKeyDown", keyDown),
		)
	},
)

// Item component showing conditional classes and event handling
var TodoItem = waveapp.DefineComponent(AppClient, "TodoItem",
	func(ctx context.Context, props TodoItemProps) any {
		return vdom.E("div",
			vdom.Class("todo-item"),
			// Conditional class example
			vdom.ClassIf(props.Todo.Completed, "completed"),
			vdom.E("input",
				vdom.Class("todo-checkbox"),
				vdom.P("type", "checkbox"),
				vdom.P("checked", props.Todo.Completed),
				vdom.P("onChange", props.OnToggle),
			),
			vdom.E("span",
				vdom.Class("todo-text"),
				props.Todo.Text,
			),
			vdom.E("button",
				vdom.Class("todo-delete"),
				vdom.P("onClick", props.OnDelete),
				"×",
			),
		)
	},
)

// List component demonstrating mapping over data
var TodoList = waveapp.DefineComponent(AppClient, "TodoList",
	func(ctx context.Context, props TodoListProps) any {
		return vdom.E("div",
			vdom.Class("todo-list"),
			// ForEach example with props passing
			vdom.ForEach(props.Todos, func(todo Todo) any {
				return TodoItem(TodoItemProps{
					Todo:     todo,
					OnToggle: func() { props.OnToggle(todo.Id) },
					OnDelete: func() { props.OnDelete(todo.Id) },
				})
			}),
		)
	},
)

// Root component showing state management and composition
var App = waveapp.DefineComponent(AppClient, "App",
	func(ctx context.Context, _ any) any {
		// Multiple state hooks example
		todos, setTodos := vdom.UseState(ctx, []Todo{
			{Id: 1, Text: "Learn VDOM", Completed: false},
			{Id: 2, Text: "Build a todo app", Completed: false},
		})
		nextId, setNextId := vdom.UseState(ctx, 3)
		inputText, setInputText := vdom.UseState(ctx, "")

		// Event handlers modifying multiple pieces of state
		addTodo := func() {
			if inputText == "" {
				return
			}
			setTodos(append(todos, Todo{
				Id:        nextId,
				Text:      inputText,
				Completed: false,
			}))
			setNextId(nextId + 1)
			setInputText("")
		}

		// Immutable state update pattern
		toggleTodo := func(id int) {
			newTodos := make([]Todo, len(todos))
			copy(newTodos, todos)
			for i := range newTodos {
				if newTodos[i].Id == id {
					newTodos[i].Completed = !newTodos[i].Completed
					break
				}
			}
			setTodos(newTodos)
		}

		// Filter pattern for deletion
		deleteTodo := func(id int) {
			newTodos := make([]Todo, 0, len(todos)-1)
			for _, todo := range todos {
				if todo.Id != id {
					newTodos = append(newTodos, todo)
				}
			}
			setTodos(newTodos)
		}

		return vdom.E("div",
			vdom.Class("todo-app"),
			vdom.E("div",
				vdom.Class("todo-header"),
				vdom.E("h1", nil, "Todo List"),
			),
			// Component composition with props
			vdom.E("div",
				vdom.Class("todo-form"),
				InputField(InputFieldProps{
					Value:    inputText,
					OnChange: setInputText,
					OnEnter:  addTodo,
				}),
				vdom.E("button",
					vdom.Class("todo-button"),
					vdom.P("onClick", addTodo),
					"Add Todo",
				),
			),
			TodoList(TodoListProps{
				Todos:    todos,
				OnToggle: toggleTodo,
				OnDelete: deleteTodo,
			}),
		)
	},
)

func main() {
	AppClient.RunMain()
}
