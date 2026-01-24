package game

import (
	"bytes"
	img "image"
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	toolbarUI      *ebitenui.UI
	toolbarFace    text.Face
	uiRef          *UI
	confirmWindow  *widget.Window
	confirmContent *widget.Container
	helpWindow     *widget.Window
	helpContent    *widget.Container
	aboutWindow    *widget.Window
	aboutContent   *widget.Container
	lastScreenW    int
	lastScreenH    int
)

func initToolbar(u *UI) {
	uiRef = u
	buildToolbar(u)
}

func buildToolbar(u *UI) {
	// Load font if not already loaded
	if toolbarFace == nil {
		src, _ := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
		toolbarFace = &text.GoTextFace{Source: src, Size: 14}
	}

	// Button colors with rounded corners
	const buttonRadius float32 = 4
	buttonImage := &widget.ButtonImage{
		Idle:    createRoundedButtonImage(color.RGBA{40, 80, 50, 255}, buttonRadius),
		Hover:   createRoundedButtonImage(color.RGBA{60, 110, 70, 255}, buttonRadius),
		Pressed: createRoundedButtonImage(color.RGBA{30, 60, 40, 255}, buttonRadius),
	}

	textColor := &widget.ButtonTextColor{
		Idle:  color.RGBA{200, 200, 200, 255},
		Hover: color.RGBA{255, 255, 255, 255},
	}

	padding := &widget.Insets{Left: 12, Right: 12, Top: 6, Bottom: 6}

	// Create buttons
	newBtn := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("New", &toolbarFace, textColor),
		widget.ButtonOpts.TextPadding(padding),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			showConfirmDialog(u)
		}),
	)

	undoBtn := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Undo", &toolbarFace, textColor),
		widget.ButtonOpts.TextPadding(padding),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			u.game.Undo()
		}),
	)

	// Auto toggle button with checkbox graphic
	checkboxImg := createCheckboxImage(u.autoMove)
	autoBtn := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.TextAndImage("Auto Move", &toolbarFace, &widget.GraphicImage{Idle: checkboxImg, Disabled: checkboxImg}, textColor),
		widget.ButtonOpts.TextPadding(padding),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			u.autoMove = !u.autoMove
			buildToolbar(u)
		}),
	)

	helpBtn := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Help", &toolbarFace, textColor),
		widget.ButtonOpts.TextPadding(padding),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			showHelpDialog(u)
		}),
	)

	aboutBtn := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("About", &toolbarFace, textColor),
		widget.ButtonOpts.TextPadding(padding),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			showAboutDialog(u)
		}),
	)

	// Row container for buttons
	toolbar := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: 20, Right: 20, Top: 5, Bottom: 15}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
			}),
		),
	)

	toolbar.AddChild(newBtn)
	toolbar.AddChild(undoBtn)
	toolbar.AddChild(autoBtn)
	toolbar.AddChild(helpBtn)
	toolbar.AddChild(aboutBtn)

	// Root container
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	root.AddChild(toolbar)

	toolbarUI = &ebitenui.UI{Container: root}
}

func showConfirmDialog(u *UI) {
	if toolbarUI.IsWindowOpen(confirmWindow) {
		return
	}

	const dialogButtonRadius float32 = 4
	buttonImage := &widget.ButtonImage{
		Idle:    createRoundedButtonImage(color.RGBA{50, 90, 60, 255}, dialogButtonRadius),
		Hover:   createRoundedButtonImage(color.RGBA{70, 120, 80, 255}, dialogButtonRadius),
		Pressed: createRoundedButtonImage(color.RGBA{40, 70, 50, 255}, dialogButtonRadius),
	}

	cancelButtonImage := &widget.ButtonImage{
		Idle:    createRoundedButtonImage(color.RGBA{90, 60, 60, 255}, dialogButtonRadius),
		Hover:   createRoundedButtonImage(color.RGBA{120, 80, 80, 255}, dialogButtonRadius),
		Pressed: createRoundedButtonImage(color.RGBA{70, 50, 50, 255}, dialogButtonRadius),
	}

	textColor := &widget.ButtonTextColor{
		Idle:  color.RGBA{220, 220, 220, 255},
		Hover: color.RGBA{255, 255, 255, 255},
	}

	padding := &widget.Insets{Left: 16, Right: 16, Top: 8, Bottom: 8}

	// Window content
	content := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(createRoundedButtonImage(color.RGBA{30, 50, 35, 255}, 8)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(15),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: 20, Right: 20, Top: 20, Bottom: 20}),
		)),
	)

	// Message
	content.AddChild(widget.NewText(
		widget.TextOpts.Text("Start a new game?", &toolbarFace, color.RGBA{220, 220, 220, 255}),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))

	// Button row
	buttonRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	)

	yesBtn := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Yes", &toolbarFace, textColor),
		widget.ButtonOpts.TextPadding(padding),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			u.game.Deal()
			u.tryAutoMove()
			confirmWindow.Close()
		}),
	)

	noBtn := widget.NewButton(
		widget.ButtonOpts.Image(cancelButtonImage),
		widget.ButtonOpts.Text("No", &toolbarFace, textColor),
		widget.ButtonOpts.TextPadding(padding),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			confirmWindow.Close()
		}),
	)

	buttonRow.AddChild(yesBtn)
	buttonRow.AddChild(noBtn)
	content.AddChild(buttonRow)

	// Create window
	confirmWindow = widget.NewWindow(
		widget.WindowOpts.Contents(content),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.CLICK_OUT),
	)

	confirmContent = content
	centerWindow(confirmWindow, confirmContent)
	toolbarUI.AddWindow(confirmWindow)
}

func showAboutDialog(u *UI) {
	if toolbarUI.IsWindowOpen(aboutWindow) {
		return
	}

	const dialogButtonRadius float32 = 4
	buttonImage := &widget.ButtonImage{
		Idle:    createRoundedButtonImage(color.RGBA{50, 90, 60, 255}, dialogButtonRadius),
		Hover:   createRoundedButtonImage(color.RGBA{70, 120, 80, 255}, dialogButtonRadius),
		Pressed: createRoundedButtonImage(color.RGBA{40, 70, 50, 255}, dialogButtonRadius),
	}

	textColor := &widget.ButtonTextColor{
		Idle:  color.RGBA{220, 220, 220, 255},
		Hover: color.RGBA{255, 255, 255, 255},
	}

	// Window content
	content := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(createRoundedButtonImage(color.RGBA{30, 50, 35, 255}, 8)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: 25, Right: 25, Top: 20, Bottom: 20}),
		)),
	)

	// Title
	content.AddChild(widget.NewText(
		widget.TextOpts.Text("Incell", &toolbarFace, color.RGBA{255, 215, 0, 255}),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))

	// Copyright
	content.AddChild(widget.NewText(
		widget.TextOpts.Text("© 2026 Jan Fredrik Leversund", &toolbarFace, color.RGBA{200, 200, 200, 255}),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))

	// Close button
	padding := &widget.Insets{Left: 16, Right: 16, Top: 8, Bottom: 8}
	closeBtn := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Close", &toolbarFace, textColor),
		widget.ButtonOpts.TextPadding(padding),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			aboutWindow.Close()
		}),
	)
	content.AddChild(closeBtn)

	// Create window
	aboutWindow = widget.NewWindow(
		widget.WindowOpts.Contents(content),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.NONE),
	)

	aboutContent = content
	centerWindow(aboutWindow, aboutContent)
	toolbarUI.AddWindow(aboutWindow)
}

func showHelpDialog(u *UI) {
	if toolbarUI.IsWindowOpen(helpWindow) {
		return
	}

	const dialogButtonRadius float32 = 4
	buttonImage := &widget.ButtonImage{
		Idle:    createRoundedButtonImage(color.RGBA{50, 90, 60, 255}, dialogButtonRadius),
		Hover:   createRoundedButtonImage(color.RGBA{70, 120, 80, 255}, dialogButtonRadius),
		Pressed: createRoundedButtonImage(color.RGBA{40, 70, 50, 255}, dialogButtonRadius),
	}

	textColor := &widget.ButtonTextColor{
		Idle:  color.RGBA{220, 220, 220, 255},
		Hover: color.RGBA{255, 255, 255, 255},
	}

	helpText := `[color=#90EE90]OBJECTIVE[/color]
Move all 52 cards to the four foundation piles, building each suit from Ace to King.

[color=#90EE90]LAYOUT[/color]
• 4 free cells (top left): Temporary storage for single cards.
• 4 Foundations (top right): Build up by suit from Ace to King.
• 8 Tableau columns: Build down by alternating colors.

[color=#90EE90]HOW TO PLAY[/color]
• Drag cards to move them.

[color=#90EE90]RULES[/color]
• Only one card can be placed in a free cell.
• Tableau columns build down in alternating colors (red on black, black on red).
• Empty tableau columns can hold any card.
• You can move multiple cards at once if enough free cells and empty columns exist.

[color=#90EE90]CONTROLS[/color]
• New: Start a new game
• Undo: Take back the last move
• Auto Move: Automatically move cards to foundations when safe

[color=#90EE90]TIPS[/color]
• Keep free cells open as long as possible.
• Try to empty a tableau column early for more flexibility.
• Build foundations evenly to avoid blocking needed cards.`

	// Window content
	content := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(createRoundedButtonImage(color.RGBA{30, 50, 35, 255}, 8)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: 20, Right: 20, Top: 20, Bottom: 20}),
		)),
	)

	// Title
	content.AddChild(widget.NewText(
		widget.TextOpts.Text("How to Play Incell", &toolbarFace, color.RGBA{255, 215, 0, 255}),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	))

	// Create text area (no scrollbar needed - content fits)
	textArea := widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Position: widget.RowLayoutPositionCenter,
				}),
				widget.WidgetOpts.MinSize(400, 420),
			),
		),
		widget.TextAreaOpts.FontFace(&toolbarFace),
		widget.TextAreaOpts.FontColor(color.RGBA{220, 220, 220, 255}),
		widget.TextAreaOpts.Text(helpText),
		widget.TextAreaOpts.TextPadding(widget.Insets{Left: 10, Right: 10, Top: 10, Bottom: 10}),
		widget.TextAreaOpts.ProcessBBCode(true),
		widget.TextAreaOpts.ScrollContainerImage(&widget.ScrollContainerImage{
			Idle: image.NewNineSliceColor(color.RGBA{25, 45, 30, 255}),
			Mask: image.NewNineSliceColor(color.RGBA{255, 255, 255, 255}),
		}),
	)

	content.AddChild(textArea)

	// Close button
	padding := &widget.Insets{Left: 16, Right: 16, Top: 8, Bottom: 8}
	closeBtn := widget.NewButton(
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text("Close", &toolbarFace, textColor),
		widget.ButtonOpts.TextPadding(padding),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			helpWindow.Close()
		}),
	)
	content.AddChild(closeBtn)

	// Create window
	helpWindow = widget.NewWindow(
		widget.WindowOpts.Contents(content),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.NONE),
	)

	helpContent = content
	centerWindow(helpWindow, helpContent)
	toolbarUI.AddWindow(helpWindow)
}

func ptr(i int) *int { return &i }

func centerWindow(win *widget.Window, content *widget.Container) {
	w, h := content.PreferredSize()
	screenW, screenH := ebiten.WindowSize()
	x := (screenW - w) / 2
	y := (screenH - h) / 2
	win.SetLocation(img.Rect(x, y, x+w, y+h))
}

func createRoundedButtonImage(c color.RGBA, radius float32) *image.NineSlice {
	// Create a small image with rounded corners that can be stretched
	size := int(radius*2 + 4)
	ebiImg := ebiten.NewImage(size, size)

	// Draw rounded rectangle
	vector.DrawFilledRect(ebiImg, 0, radius, float32(size), float32(size)-radius*2, c, false)
	vector.DrawFilledRect(ebiImg, radius, 0, float32(size)-radius*2, float32(size), c, false)
	vector.DrawFilledCircle(ebiImg, radius, radius, radius, c, false)
	vector.DrawFilledCircle(ebiImg, float32(size)-radius, radius, radius, c, false)
	vector.DrawFilledCircle(ebiImg, radius, float32(size)-radius, radius, c, false)
	vector.DrawFilledCircle(ebiImg, float32(size)-radius, float32(size)-radius, radius, c, false)

	// Create nine-slice with corners preserved
	corner := int(radius)
	return image.NewNineSlice(ebiImg, [3]int{corner, size - corner*2, corner}, [3]int{corner, size - corner*2, corner})
}

func createCheckboxImage(checked bool) *ebiten.Image {
	size := 14
	img := ebiten.NewImage(size, size)

	// Draw border
	borderColor := color.RGBA{180, 180, 180, 255}
	for x := 0; x < size; x++ {
		img.Set(x, 0, borderColor)
		img.Set(x, size-1, borderColor)
	}
	for y := 0; y < size; y++ {
		img.Set(0, y, borderColor)
		img.Set(size-1, y, borderColor)
	}

	// Fill inside
	fillColor := color.RGBA{30, 60, 40, 255}
	for y := 1; y < size-1; y++ {
		for x := 1; x < size-1; x++ {
			img.Set(x, y, fillColor)
		}
	}

	// Draw checkmark if checked
	if checked {
		checkColor := color.RGBA{220, 220, 220, 255}
		// Draw a simple checkmark
		img.Set(3, 7, checkColor)
		img.Set(4, 8, checkColor)
		img.Set(5, 9, checkColor)
		img.Set(6, 8, checkColor)
		img.Set(7, 7, checkColor)
		img.Set(8, 6, checkColor)
		img.Set(9, 5, checkColor)
		img.Set(10, 4, checkColor)
		// Make it thicker
		img.Set(3, 8, checkColor)
		img.Set(4, 9, checkColor)
		img.Set(5, 10, checkColor)
		img.Set(6, 9, checkColor)
		img.Set(7, 8, checkColor)
		img.Set(8, 7, checkColor)
		img.Set(9, 6, checkColor)
		img.Set(10, 5, checkColor)
	}

	return img
}

func isDialogOpen() bool {
	if toolbarUI == nil {
		return false
	}
	if helpWindow != nil && toolbarUI.IsWindowOpen(helpWindow) {
		return true
	}
	if confirmWindow != nil && toolbarUI.IsWindowOpen(confirmWindow) {
		return true
	}
	if aboutWindow != nil && toolbarUI.IsWindowOpen(aboutWindow) {
		return true
	}
	return false
}

func closeDialogs() {
	if helpWindow != nil && toolbarUI.IsWindowOpen(helpWindow) {
		helpWindow.Close()
	}
	if confirmWindow != nil && toolbarUI.IsWindowOpen(confirmWindow) {
		confirmWindow.Close()
	}
	if aboutWindow != nil && toolbarUI.IsWindowOpen(aboutWindow) {
		aboutWindow.Close()
	}
}

func updateToolbar() {
	if toolbarUI != nil {
		// Recenter windows if screen size changed
		screenW, screenH := ebiten.WindowSize()
		if screenW != lastScreenW || screenH != lastScreenH {
			lastScreenW = screenW
			lastScreenH = screenH
			if helpWindow != nil && toolbarUI.IsWindowOpen(helpWindow) && helpContent != nil {
				centerWindow(helpWindow, helpContent)
			}
			if confirmWindow != nil && toolbarUI.IsWindowOpen(confirmWindow) && confirmContent != nil {
				centerWindow(confirmWindow, confirmContent)
			}
			if aboutWindow != nil && toolbarUI.IsWindowOpen(aboutWindow) && aboutContent != nil {
				centerWindow(aboutWindow, aboutContent)
			}
		}
		toolbarUI.Update()
	}
}

func drawToolbarUI(screen *ebiten.Image) {
	if toolbarUI != nil {
		toolbarUI.Draw(screen)
	}
}
