package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wavetermdev/waveterm/pkg/util/envutil"
	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/waveapp"
)

//go:embed style.css
var styleCSS []byte

var envPath string

var AppClient = waveapp.MakeClient(waveapp.AppOpts{
	CloseOnCtrlC: true,
	GlobalStyles: styleCSS,
})

type EnvItemProps struct {
	Key       string               `json:"key"`
	Value     string               `json:"value"`
	OnEdit    func()               `json:"onEdit"`
	OnDelete  func()               `json:"onDelete"`
	OnSave    func(string, string) `json:"onSave"`   // Add this
	OnCancel  func()               `json:"onCancel"` // Add this
	IsEditing bool                 `json:"isEditing"`
	Highlight bool                 `json:"highlight"`
}

var EnvItem = waveapp.DefineComponent[EnvItemProps](AppClient, "EnvItem",
	func(ctx context.Context, props EnvItemProps) any {
		if props.IsEditing {
			return EditForm(EditFormProps{
				Key:      props.Key,
				Value:    props.Value,
				IsNew:    false,
				OnSave:   props.OnSave,   // Use the passed save handler
				OnCancel: props.OnCancel, // Use the passed cancel handler
			})
		}

		return vdom.H("div", map[string]any{
			"className": vdom.Classes(
				"env-item",
				vdom.If(props.Highlight, "highlight"),
			),
		},
			vdom.H("div", map[string]any{
				"className": "env-item-key",
			}, props.Key),
			vdom.H("div", map[string]any{
				"className": "env-item-value",
			}, props.Value),
			vdom.H("div", map[string]any{
				"className": "env-item-actions",
			},
				vdom.H("button", map[string]any{
					"className": "env-item-edit",
					"onClick":   props.OnEdit,
					"title":     "Edit",
				},
					vdom.H("i", map[string]any{
						"className": "fa fa-pencil",
					}),
				),
				vdom.H("button", map[string]any{
					"className": "env-item-delete",
					"onClick":   props.OnDelete,
					"title":     "Delete",
				},
					vdom.H("i", map[string]any{
						"className": "fa fa-times",
					}),
				),
			),
		)
	},
)

type EditFormProps struct {
	Key      string               `json:"key"`
	Value    string               `json:"value"`
	IsNew    bool                 `json:"isNew"`
	OnSave   func(string, string) `json:"onSave"`
	OnCancel func()               `json:"onCancel"`
}

var EditForm = waveapp.DefineComponent[EditFormProps](AppClient, "EditForm",
	func(ctx context.Context, props EditFormProps) any {
		key, setKey := vdom.UseState(ctx, props.Key)
		value, setValue := vdom.UseState(ctx, props.Value)
		error, setError := vdom.UseState(ctx, "")

		handleSave := func() {
			if props.IsNew && key == "" {
				setError("Key cannot be empty")
				return
			}
			if strings.ContainsAny(key, "=\x00") {
				setError("Key cannot contain '=' or null character")
				return
			}
			if strings.Contains(value, "\x00") {
				setError("Value cannot contain null character")
				return
			}
			props.OnSave(key, value)
		}

		keyHandler := &vdom.VDomFunc{
			Type: vdom.ObjectType_Func,
			Fn: func(e vdom.VDomEvent) {
				switch e.KeyData.Key {
				case "Enter":
					if !e.KeyData.Shift {
						handleSave()
					}
				case "Escape":
					props.OnCancel()
				}
			},
			Keys:           []string{"Enter", "Escape"},
			PreventDefault: true,
		}

		return vdom.H("div", map[string]any{
			"className": "env-item env-item-editing",
		},
			vdom.H("div", map[string]any{
				"className": "env-item-key",
			},
				vdom.IfElse(props.IsNew,
					vdom.H("input", map[string]any{
						"className":   "env-edit-key",
						"value":       key,
						"onChange":    func(e vdom.VDomEvent) { setKey(e.TargetValue) },
						"placeholder": "Key",
						"onKeyDown":   keyHandler,
					}),
					vdom.H("div", map[string]any{
						"className": "env-key-static",
					}, props.Key),
				),
			),
			vdom.H("div", map[string]any{
				"className": "env-item-value",
			},
				vdom.H("textarea", map[string]any{
					"className":   "env-edit-value",
					"value":       value,
					"onChange":    func(e vdom.VDomEvent) { setValue(e.TargetValue) },
					"placeholder": "Value",
					"onKeyDown":   keyHandler,
					"rows":        3,
				}),
			),
			vdom.If(error != "",
				vdom.H("div", map[string]any{
					"className": "env-edit-error",
				}, error),
			),
			vdom.H("div", map[string]any{
				"className": "env-edit-actions",
			},
				vdom.H("button", map[string]any{
					"className": "env-edit-cancel",
					"onClick":   props.OnCancel,
				}, "Cancel"),
				vdom.H("button", map[string]any{
					"className": "env-edit-save",
					"onClick":   handleSave,
				},
					vdom.H("i", map[string]any{
						"className": "fa fa-check",
					}),
					" Save",
				),
			),
		)
	},
)

// HeaderProps breaks out all the header functionality
type HeaderProps struct {
	Path      string `json:"path"`
	OnAddNew  func() `json:"onAddNew"`
	IsEditing bool   `json:"isEditing"`
}

var Header = waveapp.DefineComponent[HeaderProps](AppClient, "Header",
	func(ctx context.Context, props HeaderProps) any {
		return vdom.H("div", map[string]any{
			"className": "env-header",
		},
			vdom.H("h1", nil, "Environment Editor"),
			vdom.H("div", map[string]any{
				"className": "env-path",
			}, "Path: ", props.Path),
			vdom.H("button", map[string]any{
				"className": vdom.Classes(
					"env-add",
					vdom.If(props.IsEditing, "env-add-active"),
				),
				"onClick": props.OnAddNew,
				"title":   vdom.IfElse(props.IsEditing, "Close editor", "Add variable"),
			},
				vdom.H("i", map[string]any{
					"className": vdom.Classes(
						"fa",
						vdom.IfElse(props.IsEditing, "fa-times", "fa-plus"),
					),
				}),
				vdom.IfElse(props.IsEditing, " Close", " Add Variable"),
			),
		)
	},
)

var App = waveapp.DefineComponent(AppClient, "App",
	func(ctx context.Context, _ any) any {
		envMap, setEnvMap := vdom.UseState(ctx, map[string]string{})
		editingKey, setEditingKey := vdom.UseState(ctx, "")
		error, setError := vdom.UseState(ctx, "")
		highlightKey, setHighlightKey := vdom.UseState(ctx, "")

		// Clear highlight after delay
		vdom.UseEffect(ctx, func() func() {
			if highlightKey != "" {
				timer := time.AfterFunc(1500*time.Millisecond, func() {
					setHighlightKey("")
					AppClient.SendAsyncInitiation()
				})
				return func() {
					timer.Stop()
				}
			}
			return nil
		}, []any{highlightKey})

		// Load environment file on mount
		vdom.UseEffect(ctx, func() func() {
			content, err := os.ReadFile(envPath)
			if err != nil && !os.IsNotExist(err) {
				setError(fmt.Sprintf("Error reading file: %v", err))
				return nil
			}
			if len(content) > 0 {
				setEnvMap(envutil.EnvToMap(string(content)))
			}
			return nil
		}, []any{})

		// Save environment to file
		saveToFile := func(newMap map[string]string) {
			content := envutil.MapToEnv(newMap)
			err := os.WriteFile(envPath, []byte(content), 0644)
			if err != nil {
				setError(fmt.Sprintf("Error saving file: %v", err))
				return
			}
			setEnvMap(newMap)
		}

		handleAdd := func() {
			if editingKey != "" {
				setEditingKey("")
			} else {
				setEditingKey("__new__")
			}
		}

		handleEdit := func(key string) {
			if editingKey == key {
				setEditingKey("")
			} else {
				setEditingKey(key)
			}
		}

		handleDelete := func(key string) {
			newMap := make(map[string]string)
			for k, v := range envMap {
				if k != key {
					newMap[k] = v
				}
			}
			saveToFile(newMap)
		}

		handleSave := func(key, value string) {
			newMap := make(map[string]string)
			for k, v := range envMap {
				newMap[k] = v
			}
			newMap[key] = value
			saveToFile(newMap)
			setEditingKey("")
			setHighlightKey(key)
		}

		handleCancel := func() {
			setEditingKey("")
		}

		// Get sorted keys
		var keys []string
		for k := range envMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		return vdom.H("div", map[string]any{
			"className": "env-editor",
		},
			Header(HeaderProps{
				Path:      envPath,
				OnAddNew:  handleAdd,
				IsEditing: editingKey != "",
			}),

			vdom.If(error != "",
				vdom.H("div", map[string]any{
					"className": "env-error",
				}, error),
			),

			vdom.H("div", map[string]any{
				"className": "env-list",
			},
				vdom.ForEach(keys, func(key string) any {
					return EnvItem(EnvItemProps{
						Key:       key,
						Value:     envMap[key],
						OnEdit:    func() { handleEdit(key) },
						OnDelete:  func() { handleDelete(key) },
						OnSave:    handleSave,   // Add this
						OnCancel:  handleCancel, // Add this
						IsEditing: key == editingKey,
						Highlight: key == highlightKey,
					})
				}),
				// Add new item form at bottom
				vdom.If(editingKey == "__new__",
					EditForm(EditFormProps{
						Key:      "",
						Value:    "",
						IsNew:    true,
						OnSave:   handleSave,
						OnCancel: handleCancel,
					}),
				),
			),
		)
	},
)

func main() {
	AppClient.RegisterDefaultFlags()
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: env-editor [flags] <env-file-path>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	envPath = flag.Arg(0)

	// Verify directory exists
	dir := filepath.Dir(envPath)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Directory does not exist: %s\n", dir)
		os.Exit(1)
	}

	AppClient.RunMain()
}
