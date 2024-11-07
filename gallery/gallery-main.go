package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/vdom/vdomclient"
)

//go:embed style.css
var styleCSS []byte

// Command-line args (accessible in components)
var galleryPath string

var GalleryClient *vdomclient.Client = vdomclient.MakeClient(vdomclient.ApplicationOpts{
	Name:         "gallery",
	Use:          "gallery <path>",
	Description:  "Display an image gallery for a directory",
	CloseOnCtrlC: true,
	GlobalStyles: styleCSS,
})

type ImageInfo struct {
	Path string `json:"path"`
}

// Image gallery components
type GalleryProps struct {
	Images []ImageInfo `json:"images"`
}

type ImageViewProps struct {
	Image   ImageInfo `json:"image"`
	OnClose func()    `json:"onClose"`
	OnNext  func()    `json:"onNext"`
	OnPrev  func()    `json:"onPrev"`
	HasNext bool      `json:"hasNext"`
	HasPrev bool      `json:"hasPrev"`
}

var ImageView = vdomclient.DefineComponent[ImageViewProps](GalleryClient, "ImageView",
	func(ctx context.Context, props ImageViewProps) any {
		return vdom.E("div",
			vdom.Class("image-view"),
			// Close button
			vdom.E("button",
				vdom.Class("close-button"),
				vdom.P("onClick", props.OnClose),
				"×",
			),
			// Navigation buttons
			vdom.If(props.HasPrev,
				vdom.E("button",
					vdom.Class("nav-button prev"),
					vdom.P("onClick", props.OnPrev),
					"←",
				),
			),
			vdom.If(props.HasNext,
				vdom.E("button",
					vdom.Class("nav-button next"),
					vdom.P("onClick", props.OnNext),
					"→",
				),
			),
			// Image
			vdom.E("img",
				vdom.Class("full-image"),
				vdom.P("src", fmt.Sprintf("vdom:///img/%s", props.Image.Path)),
				vdom.P("alt", props.Image.Path),
			),
		)
	},
)

var App = vdomclient.DefineComponent(GalleryClient, "App",
	func(ctx context.Context, props vdomclient.AppProps) any {
		fmt.Printf("App props: %+v\n", props)
		galleryPath = props.Args[0]
		log.Printf("galleryPath: %q\n", galleryPath)
		// Get images from the provided path
		images, err := scanDirectory(galleryPath)
		if err != nil {
			return vdom.E("div",
				vdom.Class("error"),
				fmt.Sprintf("Error scanning directory: %v", err),
			)
		}

		selectedIndex, setSelectedIndex := vdom.UseState(ctx, -1)

		keyDown := &vdom.VDomFunc{
			Type: vdom.ObjectType_Func,
			Fn: func(event vdom.VDomEvent) {
				if event.KeyData == nil {
					return
				}
				if selectedIndex >= 0 {
					switch event.KeyData.Key {
					case "Escape":
						setSelectedIndex(-1)
					case "ArrowRight":
						if selectedIndex < len(images)-1 {
							setSelectedIndex(selectedIndex + 1)
						}
					case "ArrowLeft":
						if selectedIndex > 0 {
							setSelectedIndex(selectedIndex - 1)
						}
					}
				}
			},
			Keys: []string{"Escape", "ArrowRight", "ArrowLeft"},
		}

		// Prepare ImageView props only if we have a valid index
		var imageView any
		if selectedIndex >= 0 && selectedIndex < len(images) {
			imageView = ImageView(ImageViewProps{
				Image:   images[selectedIndex],
				OnClose: func() { setSelectedIndex(-1) },
				OnNext: func() {
					if selectedIndex < len(images)-1 {
						setSelectedIndex(selectedIndex + 1)
					}
				},
				OnPrev: func() {
					if selectedIndex > 0 {
						setSelectedIndex(selectedIndex - 1)
					}
				},
				HasNext: selectedIndex < len(images)-1,
				HasPrev: selectedIndex > 0,
			})
		}

		return vdom.E("div",
			vdom.Class("gallery"),
			vdom.P("onKeyDown", keyDown),
			vdom.E("div",
				vdom.Class("gallery-header"),
				vdom.E("h1", nil, "Image Gallery"),
			),
			vdom.Fragment(
				// Grid view when no image is selected
				vdom.If(selectedIndex == -1,
					vdom.E("div",
						vdom.Class("image-grid"),
						vdom.ForEachIdx(images, func(img ImageInfo, i int) any {
							return vdom.E("div",
								vdom.Class("image-item"),
								vdom.P("onClick", func() {
									setSelectedIndex(i)
								}),
								vdom.E("img",
									vdom.P("src", fmt.Sprintf("vdom:///img/%s", img.Path)),
									vdom.P("alt", img.Path),
								),
							)
						}),
					),
				),
				// Image view
				vdom.If(imageView != nil, imageView),
			),
		)
	},
)

func scanDirectory(root string) ([]ImageInfo, error) {
	var images []ImageInfo
	validExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true,
		".gif": true, ".webp": true,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if validExts[ext] {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			images = append(images, ImageInfo{Path: relPath})
		}
		return nil
	})

	return images, err
}

func main() {
	log.Printf("hello\n")
	// Set up image handlers
	GalleryClient.RegisterFilePrefixHandler("/img/", func(path string) (*vdomclient.FileHandlerOption, error) {
		imgPath := strings.TrimPrefix(path, "/img/")
		fullPath := filepath.Join(GalleryClient.CommandArgs[0], imgPath)

		// Get file info first for both existence check and ETag generation
		fileInfo, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			return nil, nil // Return nil for 404
		}
		if err != nil {
			return nil, err
		}

		// Generate ETag from file size and modification time
		etag := fmt.Sprintf(`"%x-%x"`, fileInfo.Size(), fileInfo.ModTime().Unix())

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}

		return &vdomclient.FileHandlerOption{
			Data: data,
			ETag: etag,
		}, nil
	})
	GalleryClient.Command.Args = cobra.ExactArgs(1)
	GalleryClient.RunMain()
}
