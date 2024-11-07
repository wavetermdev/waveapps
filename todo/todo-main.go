package main

import (
	"context"
	_ "embed"

	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/vdom/vdomclient"
)

//go:embed style.css
var styleCSS []byte

var TodoVDomClient *vdomclient.Client = vdomclient.MakeClient(vdomclient.ApplicationOpts{
	CloseOnCtrlC: true,
	GlobalStyles: styleCSS,
})

type Todo struct {
	Id        int    `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

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

var InputField = vdomclient.DefineComponent[InputFieldProps](TodoVDomClient, "InputField",
	func(ctx context.Context, props InputFieldProps) any {
		keyDown := &vdom.VDomFunc{
			Type: vdom.ObjectType_Func,
			Fn: func(event vdom.VDomEvent) {
				if props.OnEnter != nil {
					props.OnEnter()
				}
			},
			StopPropagation: true,
			PreventDefault:  true,
			Keys:            []string{"Enter", "Cmd:Enter"}, // Handle both Enter and Cmd+Enter
		}

		return vdom.E("input",
			vdom.Class("todo-input"),
			vdom.P("type", "text"),
			vdom.P("placeholder", "What needs to be done?"),
			vdom.P("value", props.Value),
			vdom.P("onChange", func(e vdom.VDomEvent) {
				props.OnChange(e.TargetValue)
			}),
			vdom.P("onKeyDown", keyDown),
		)
	},
)

var TodoItem = vdomclient.DefineComponent(TodoVDomClient, "TodoItem",
	func(ctx context.Context, props TodoItemProps) any {
		return vdom.E("div",
			vdom.Class("todo-item"),
			vdom.ClassIf(props.Todo.Completed, "completed"),
			vdom.E("input",
				vdom.P("type", "checkbox"),
				vdom.Class("todo-checkbox"),
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

var TodoList = vdomclient.DefineComponent(TodoVDomClient, "TodoList",
	func(ctx context.Context, props TodoListProps) any {
		return vdom.E("div",
			vdom.Class("todo-list"),
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

var App = vdomclient.DefineComponent(TodoVDomClient, "App",
	func(ctx context.Context, _ any) any {
		// State using hooks
		todos, setTodos := vdom.UseState(ctx, []Todo{
			{Id: 1, Text: "Learn VDOM", Completed: false},
			{Id: 2, Text: "Build a todo app", Completed: false},
			{Id: 3, Text: "Profit!", Completed: false},
		})
		nextId, setNextId := vdom.UseState(ctx, 4)
		inputText, setInputText := vdom.UseState(ctx, "")

		// Event handlers
		addTodo := func() {
			if inputText == "" {
				return
			}
			newTodo := Todo{
				Id:        nextId,
				Text:      inputText,
				Completed: false,
			}
			setNextId(nextId + 1)
			setTodos(append(todos, newTodo))
			setInputText("")
		}

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
	TodoVDomClient.RunMain()
}
