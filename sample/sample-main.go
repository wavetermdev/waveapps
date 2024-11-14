// Copyright 2024, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	_ "embed"
	"log"

	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/waveapp"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wshrpc"
	"github.com/wavetermdev/waveterm/pkg/wshrpc/wshclient"
)

//go:embed style.css
var styleCSS []byte

var AppClient *waveapp.Client = waveapp.MakeClient(waveapp.AppOpts{
	CloseOnCtrlC: true,
	GlobalStyles: styleCSS,
})

// Prop Types
type BgItemProps struct {
	Bg      string `json:"bg"`
	Label   string `json:"label"`
	OnClick func() `json:"onClick"`
}

type BgListProps struct {
	Items []BgItem `json:"items"`
}

type BgItem struct {
	Bg    string `json:"bg"`
	Label string `json:"label"`
}

// Components
var Style = waveapp.DefineComponent[struct{}](AppClient, "Style",
	func(ctx context.Context, _ struct{}) any {
		return vdom.E("wave:style",
			vdom.P("src", "vdom:///style.css"),
		)
	},
)

var BgItemTag = waveapp.DefineComponent[BgItemProps](AppClient, "BgItem",
	func(ctx context.Context, props BgItemProps) any {
		return vdom.E("div",
			vdom.Class("bg-item"),
			vdom.E("div",
				vdom.Class("bg-preview"),
				vdom.PStyle("background", props.Bg),
			),
			vdom.E("div",
				vdom.Class("bg-label"),
				props.Label,
			),
			vdom.P("onClick", props.OnClick),
		)
	},
)

var BgList = waveapp.DefineComponent[BgListProps](AppClient, "BgList",
	func(ctx context.Context, props BgListProps) any {
		setBackground := func(bg string) func() {
			return func() {
				blockInfo, err := wshclient.BlockInfoCommand(AppClient.RpcClient, AppClient.RpcContext.BlockId, nil)
				if err != nil {
					log.Printf("error getting block info: %v\n", err)
					return
				}
				err = wshclient.SetMetaCommand(AppClient.RpcClient, wshrpc.CommandSetMetaData{
					ORef: waveobj.ORef{OType: "tab", OID: blockInfo.TabId},
					Meta: map[string]any{"bg": bg},
				}, nil)
				if err != nil {
					log.Printf("error setting meta: %v\n", err)
				}
			}
		}

		return vdom.E("div",
			vdom.Class("background"),
			vdom.E("div",
				vdom.Class("background-inner"),
				vdom.ForEach(props.Items, func(item BgItem) any {
					return BgItemTag(BgItemProps{
						Bg:      item.Bg,
						Label:   item.Label,
						OnClick: setBackground(item.Bg),
					})
				}),
			),
		)
	},
)

var App = waveapp.DefineComponent[struct{}](AppClient, "App",
	func(ctx context.Context, _ struct{}) any {
		inputText, setInputText := vdom.UseState(ctx, "start")

		bgItems := []BgItem{
			{Bg: "", Label: "default"},
			{Bg: "#ff0000", Label: "red"},
			{Bg: "#00ff00", Label: "green"},
			{Bg: "#0000ff", Label: "blue"},
		}

		return vdom.E("div",
			vdom.Class("root"),
			Style(struct{}{}),
			vdom.E("h1", nil, "Set Background"),
			vdom.E("div", nil,
				vdom.E("wave:markdown",
					vdom.P("text", "*quick vdom application to set background colors*"),
					vdom.P("scrollable", false),
					vdom.P("rehype", false),
				),
			),
			vdom.E("div", nil,
				BgList(BgListProps{Items: bgItems}),
			),
			vdom.E("div", nil,
				vdom.E("img",
					vdom.PStyle("width", "100%"),
					vdom.PStyle("height", "100%"),
					vdom.PStyle("maxWidth", "300px"),
					vdom.PStyle("maxHeight", "300px"),
					vdom.PStyle("objectFit", "contain"),
					vdom.P("src", "vdom:///test.png"),
				),
			),
			vdom.E("div", nil,
				vdom.E("input",
					vdom.P("type", "text"),
					vdom.P("value", inputText),
					vdom.P("onChange", func(e vdom.VDomEvent) {
						setInputText(e.TargetValue)
					}),
				),
				vdom.E("div", nil, "text ", inputText),
			),
		)
	},
)

func main() {
	AppClient.RegisterFileHandler("/test.png", waveapp.FileHandlerOption{
		FilePath: "~/Downloads/IMG_1939.png",
	})
	AppClient.RunMain()
}
