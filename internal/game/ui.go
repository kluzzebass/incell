package game

import (
	"image/color"
	"math"

	"incell/internal/assets"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Layout ratios relative to card width
const (
	spacingRatio     = 0.06 // Space between cards as fraction of card width
	stackOffsetRatio = 0.18 // Vertical offset for stacked cards as fraction of card height
	marginRatio      = 0.12 // Margin as fraction of card width
	topRowGapRatio   = 0.24 // Gap between free cells and foundations as fraction of card width
)

// CardAnimation represents a card flying from one position to another
type CardAnimation struct {
	Card         Card
	FromX        float64
	FromY        float64
	ToX          float64
	ToY          float64
	Progress     float64 // 0.0 to 1.0
	ToFoundation int     // Which foundation pile (-1 if not going to foundation)
}

// UI handles the game rendering and input
type UI struct {
	game        *FreeCell
	dragging    bool
	dragStartX  int
	dragStartY  int
	dragOffsetX int
	dragOffsetY int
	cardW       float64
	cardH       float64
	cardScale   float64
	width       int
	height      int
	feltImage   *ebiten.Image
	autoMove    bool // Auto-move aces to foundation
	animations  []CardAnimation
	pendingAuto bool // Whether we need to check for auto-moves after animation
}

// NewUI creates a new game UI
func NewUI() *UI {
	u := &UI{
		game:     New(),
		autoMove: true,
	}
	initToolbar(u)
	return u
}

// Layout implements ebiten.Game
func (u *UI) Layout(outsideWidth, outsideHeight int) (int, int) {
	u.width = outsideWidth
	u.height = outsideHeight

	// Calculate the ideal card size based on window dimensions
	unitCardW := 1.0
	unitCardH := unitCardW / assets.AspectRatio()
	unitSpacing := unitCardW * spacingRatio
	unitMargin := unitCardW * marginRatio
	unitTopRowGap := unitCardW * topRowGapRatio

	// Top row: 4 free cells + gap + 4 foundations
	topRowWidth := 2*unitMargin + 8*unitCardW + 7*unitSpacing + unitTopRowGap

	// Tableau: 8 columns centered
	tableauWidth := 8*unitCardW + 7*unitSpacing + 2*unitMargin

	requiredWidth := max(topRowWidth, tableauWidth)

	// Vertical: margin + card + spacing + tableau with stacked cards
	maxStackedCards := 13
	unitStackOffset := unitCardH * stackOffsetRatio
	requiredHeight := unitMargin + unitCardH + unitSpacing*2 + unitCardH + float64(maxStackedCards-1)*unitStackOffset + unitMargin

	// Find the scale that fits both dimensions
	scaleX := float64(outsideWidth) / requiredWidth
	scaleY := float64(outsideHeight) / requiredHeight
	scale := min(scaleX, scaleY)

	u.cardW = unitCardW * scale
	u.cardH = unitCardH * scale
	u.cardScale = u.cardW / float64(assets.BaseWidth())

	return outsideWidth, outsideHeight
}

func (u *UI) cardWidth() float64   { return u.cardW }
func (u *UI) cardHeight() float64  { return u.cardH }
func (u *UI) spacing() float64     { return u.cardWidth() * spacingRatio }
func (u *UI) margin() float64      { return u.cardWidth() * marginRatio }
func (u *UI) stackOffset() float64 { return u.cardHeight() * stackOffsetRatio }
func (u *UI) topRowGap() float64   { return u.cardWidth() * topRowGapRatio }
func (u *UI) topRowY() float64     { return u.margin() }

// Top row layout
func (u *UI) topRowWidth() float64 {
	return 8*u.cardWidth() + 7*u.spacing() + u.topRowGap()
}

func (u *UI) topRowStartX() float64 {
	return (float64(u.width) - u.topRowWidth()) / 2
}

func (u *UI) freeCellX(i int) float64 {
	return u.topRowStartX() + float64(i)*(u.cardWidth()+u.spacing())
}

func (u *UI) foundationX(i int) float64 {
	return u.topRowStartX() + 4*(u.cardWidth()+u.spacing()) + u.topRowGap() + float64(i)*(u.cardWidth()+u.spacing())
}

// Tableau layout
func (u *UI) tableauWidth() float64 {
	return float64(NumTableau)*u.cardWidth() + float64(NumTableau-1)*u.spacing()
}

func (u *UI) tableauStartX() float64 {
	return (float64(u.width) - u.tableauWidth()) / 2
}

func (u *UI) tableauStartY() float64 {
	return u.topRowY() + u.cardHeight() + u.spacing()*2
}

func (u *UI) tableauColumnX(col int) float64 {
	return u.tableauStartX() + float64(col)*(u.cardWidth()+u.spacing())
}

const animationSpeed = 0.05 // Progress per frame (higher = faster)

// Update implements ebiten.Game
func (u *UI) Update() error {
	// Update animations
	u.updateAnimations()

	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		u.game.Deal()
		u.tryAutoMove()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyU) || inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		u.game.Undo()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		for u.game.AutoMoveToFoundation() {
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		closeDialogs()
	}
	updateToolbar()
	u.handleMouse()
	return nil
}

func (u *UI) updateAnimations() {
	if len(u.animations) == 0 {
		// No animations, check if we need to do auto-moves
		if u.pendingAuto {
			u.pendingAuto = false
			u.doNextAutoMove()
		}
		return
	}

	// Update the first animation (sequential)
	u.animations[0].Progress += animationSpeed
	if u.animations[0].Progress >= 1.0 {
		// Animation complete, remove it
		u.animations = u.animations[1:]
	}
}

func (u *UI) isAnimating() bool {
	return len(u.animations) > 0
}

func (u *UI) handleMouse() {
	// Don't process mouse input when a dialog is open
	if isDialogOpen() {
		return
	}

	mx, my := ebiten.CursorPosition()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		pos := u.getPositionAt(mx, my)
		if pos != nil {
			if u.game.Selected != nil {
				if u.game.TryMove(*pos) {
					u.tryAutoMove()
				} else {
					u.game.Select(*pos)
				}
			} else {
				u.game.Select(*pos)
			}
			u.dragging = true
			u.dragStartX = mx
			u.dragStartY = my
			if u.game.Selected != nil {
				cardX, cardY := u.getCardScreenPosition(u.game.Selected.Position)
				u.dragOffsetX = mx - cardX
				u.dragOffsetY = my - cardY
			}
		} else {
			u.game.ClearSelection()
		}
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if u.dragging && u.game.Selected != nil {
			if pos := u.getPositionAt(mx, my); pos != nil {
				if u.game.TryMove(*pos) {
					u.tryAutoMove()
				}
			}
		}
		u.dragging = false
		u.game.ClearSelection()
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if pos := u.getPositionAt(mx, my); pos != nil {
			u.game.Select(*pos)
			if u.game.Selected != nil && len(u.game.Selected.Cards) == 1 {
				card := u.game.Selected.Cards[0]
				if foundIdx := u.game.FindFoundationForCard(card); foundIdx >= 0 {
					if u.game.TryMove(Position{Location: LocFoundation, Index: foundIdx}) {
						u.tryAutoMove()
					}
				}
			}
			u.game.ClearSelection()
		}
	}
}

// Draw implements ebiten.Game
func (u *UI) Draw(screen *ebiten.Image) {
	u.drawFelt(screen)

	u.drawEmptySlots(screen)
	u.drawFoundations(screen)
	u.drawFreeCells(screen)
	u.drawTableau(screen)

	if u.dragging && u.game.Selected != nil {
		mx, my := ebiten.CursorPosition()
		u.drawDraggedCards(screen, mx, my)
	}

	// Draw animated cards
	u.drawAnimations(screen)

	if u.game.IsWon() {
		u.drawWinMessage(screen)
	}

	drawToolbarUI(screen)
}

func (u *UI) drawAnimations(screen *ebiten.Image) {
	for _, anim := range u.animations {
		// Ease-in-out interpolation: accelerate then decelerate
		t := anim.Progress
		if t < 0.5 {
			t = 2 * t * t // Ease-in (accelerate)
		} else {
			t = 1 - 2*(1-t)*(1-t) // Ease-out (decelerate)
		}

		x := anim.FromX + (anim.ToX-anim.FromX)*t
		y := anim.FromY + (anim.ToY-anim.FromY)*t

		u.drawCard(screen, anim.Card, x, y, false)
	}
}

func (u *UI) drawFelt(screen *ebiten.Image) {
	// Cache the felt image, regenerate only on resize
	if u.feltImage == nil || u.feltImage.Bounds().Dx() != u.width || u.feltImage.Bounds().Dy() != u.height {
		u.feltImage = ebiten.NewImage(u.width, u.height)
		u.generateFelt()
	}
	screen.DrawImage(u.feltImage, nil)
}

func (u *UI) generateFelt() {
	// Radial gradient: lighter in center, darker at edges
	cx, cy := float64(u.width)/2, float64(u.height)/2
	maxDist := math.Sqrt(cx*cx + cy*cy)

	for y := 0; y < u.height; y++ {
		for x := 0; x < u.width; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx+dy*dy) / maxDist

			// Base green, darken towards edges
			factor := 1.0 - dist*0.25
			r := uint8(float64(0x1a) * factor)
			g := uint8(float64(0x6b) * factor)
			b := uint8(float64(0x35) * factor)
			u.feltImage.Set(x, y, color.RGBA{r, g, b, 0xff})
		}
	}
}

func (u *UI) drawEmptySlots(screen *ebiten.Image) {
	y := u.topRowY()

	for i := 0; i < NumFreeCells; i++ {
		u.drawEmptySlot(screen, u.freeCellX(i), y, slotFreeCell)
	}

	for i := 0; i < NumFoundation; i++ {
		u.drawEmptySlot(screen, u.foundationX(i), y, slotFoundation)
	}
}

type slotType int

const (
	slotTableau slotType = iota
	slotFreeCell
	slotFoundation
)

func (u *UI) drawEmptySlot(screen *ebiten.Image, x, y float64, slot slotType) {
	slotColor := color.RGBA{0x0f, 0x4d, 0x25, 0xff}
	radius := float32(u.cardWidth() * 0.05)
	u.strokeRoundedRect(screen, float32(x), float32(y), float32(u.cardWidth()), float32(u.cardHeight()), radius, 2, slotColor)

	// Draw symbol in center
	centerX := float32(x + u.cardWidth()/2)
	centerY := float32(y + u.cardHeight()/2)

	switch slot {
	case slotFreeCell:
		// Draw a star for free cells
		u.drawStar(screen, centerX, centerY, float32(u.cardWidth()*0.32), slotColor)
	case slotFoundation:
		// Draw "A" for foundations
		u.drawLetterA(screen, centerX, centerY, float32(u.cardWidth()*0.25), slotColor)
	}
	// slotTableau: no symbol, just the border
}

func (u *UI) strokeRoundedRect(screen *ebiten.Image, x, y, w, h, radius, strokeWidth float32, c color.Color) {
	var path vector.Path
	// Start at top-left, after the corner
	path.MoveTo(x+radius, y)
	// Top edge and top-right corner
	path.LineTo(x+w-radius, y)
	path.ArcTo(x+w, y, x+w, y+radius, radius)
	// Right edge and bottom-right corner
	path.LineTo(x+w, y+h-radius)
	path.ArcTo(x+w, y+h, x+w-radius, y+h, radius)
	// Bottom edge and bottom-left corner
	path.LineTo(x+radius, y+h)
	path.ArcTo(x, y+h, x, y+h-radius, radius)
	// Left edge and top-left corner
	path.LineTo(x, y+radius)
	path.ArcTo(x, y, x+radius, y, radius)
	path.Close()

	strokeOp := &vector.StrokeOptions{Width: strokeWidth}
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, strokeOp)
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = colorToFloats(c)
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)
}

func (u *UI) drawLetterA(screen *ebiten.Image, cx, cy, size float32, c color.Color) {
	// Draw an "A" shape using vector paths
	// Size is roughly half-height of the letter
	h := size * 2   // total height
	w := size * 1.3 // total width
	stroke := size * 0.19

	top := cy - h/2
	bottom := cy + h/2
	left := cx - w/2
	right := cx + w/2
	midY := cy + h*0.1 // crossbar slightly above center

	var path vector.Path
	// Left leg
	path.MoveTo(left, bottom)
	path.LineTo(cx, top)
	// Right leg
	path.LineTo(right, bottom)
	// Crossbar
	path.MoveTo(left+w*0.2, midY)
	path.LineTo(right-w*0.2, midY)

	strokeOp := &vector.StrokeOptions{Width: stroke, LineCap: vector.LineCapRound, LineJoin: vector.LineJoinRound}
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, strokeOp)
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = colorToFloats(c)
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)
}

func (u *UI) drawStar(screen *ebiten.Image, cx, cy, radius float32, c color.Color) {
	// 5-pointed star
	points := 5
	innerRadius := radius * 0.4

	var path vector.Path
	for i := 0; i < points*2; i++ {
		angle := float64(i)*math.Pi/float64(points) - math.Pi/2
		r := radius
		if i%2 == 1 {
			r = innerRadius
		}
		px := cx + float32(math.Cos(angle))*r
		py := cy + float32(math.Sin(angle))*r
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
	path.Close()

	// Stroke the star outline
	strokeOp := &vector.StrokeOptions{Width: radius * 0.15, LineCap: vector.LineCapRound, LineJoin: vector.LineJoinRound}
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, strokeOp)
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = colorToFloats(c)
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)
}

func colorToFloats(c color.Color) (float32, float32, float32, float32) {
	r, g, b, a := c.RGBA()
	return float32(r) / 0xffff, float32(g) / 0xffff, float32(b) / 0xffff, float32(a) / 0xffff
}

var emptyImage = func() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return img
}()

func (u *UI) drawFoundations(screen *ebiten.Image) {
	y := u.topRowY()
	for i := 0; i < NumFoundation; i++ {
		pile := u.game.Foundations[i]
		if len(pile) == 0 {
			continue
		}
		// Check if the top card is currently animating to this foundation
		topCard := pile[len(pile)-1]
		animating := false
		for _, anim := range u.animations {
			if anim.ToFoundation == i && anim.Card == topCard {
				animating = true
				break
			}
		}
		if animating {
			// Draw the card below instead (if any)
			if len(pile) > 1 {
				u.drawCard(screen, pile[len(pile)-2], u.foundationX(i), y, false)
			}
		} else {
			u.drawCard(screen, topCard, u.foundationX(i), y, false)
		}
	}
}

func (u *UI) drawFreeCells(screen *ebiten.Image) {
	y := u.topRowY()
	for i := 0; i < NumFreeCells; i++ {
		if u.game.FreeCells[i] != nil {
			card := *u.game.FreeCells[i]
			selected := u.game.Selected != nil &&
				u.game.Selected.Position.Location == LocFreeCell &&
				u.game.Selected.Position.Index == i
			if !u.dragging || !selected {
				u.drawCard(screen, card, u.freeCellX(i), y, selected)
			}
		}
	}
}

func (u *UI) drawTableau(screen *ebiten.Image) {
	startY := u.tableauStartY()

	for col := 0; col < NumTableau; col++ {
		x := u.tableauColumnX(col)
		pile := u.game.Tableau[col]

		if len(pile) == 0 {
			u.drawEmptySlot(screen, x, startY, slotTableau)
			continue
		}

		for cardIdx, card := range pile {
			y := startY + float64(cardIdx)*u.stackOffset()

			selected := u.game.Selected != nil &&
				u.game.Selected.Position.Location == LocTableau &&
				u.game.Selected.Position.Index == col &&
				cardIdx >= u.game.Selected.Position.CardIdx

			if u.dragging && selected {
				continue
			}

			u.drawCard(screen, card, x, y, selected && !u.dragging)
		}
	}
}

func (u *UI) drawCard(screen *ebiten.Image, card Card, x, y float64, highlighted bool) {
	img := assets.GetCardImage(card.Rank, card.Suit)
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(u.cardScale, u.cardScale)
	op.GeoM.Translate(x, y)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(img, op)

	if highlighted {
		vector.StrokeRect(screen, float32(x), float32(y), float32(u.cardWidth()), float32(u.cardHeight()), 3, color.RGBA{0xff, 0xff, 0x00, 0xff}, false)
	}
}

func (u *UI) drawDraggedCards(screen *ebiten.Image, mx, my int) {
	if u.game.Selected == nil {
		return
	}

	x := float64(mx - u.dragOffsetX)
	y := float64(my - u.dragOffsetY)

	// Create offscreen image for the stack
	numCards := len(u.game.Selected.Cards)
	stackHeight := int(u.cardHeight() + float64(numCards-1)*u.stackOffset())
	stackImg := ebiten.NewImage(int(u.cardWidth()), stackHeight)

	// Draw cards to offscreen image
	for i, card := range u.game.Selected.Cards {
		img := assets.GetCardImage(card.Rank, card.Suit)
		if img == nil {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(u.cardScale, u.cardScale)
		op.GeoM.Translate(0, float64(i)*u.stackOffset())
		op.Filter = ebiten.FilterLinear
		stackImg.DrawImage(img, op)
	}

	// Draw soft shadow with a few offset copies
	offsets := []struct{ dx, dy, alpha float64 }{
		{4, 4, 0.15},
		{5, 5, 0.15},
		{6, 6, 0.1},
	}
	for _, o := range offsets {
		shadowOp := &ebiten.DrawImageOptions{}
		shadowOp.GeoM.Translate(x+o.dx, y+o.dy)
		shadowOp.ColorScale.Scale(0, 0, 0, 1)
		shadowOp.ColorScale.ScaleAlpha(float32(o.alpha))
		screen.DrawImage(stackImg, shadowOp)
	}

	// Draw the actual stack
	stackOp := &ebiten.DrawImageOptions{}
	stackOp.GeoM.Translate(x, y)
	screen.DrawImage(stackImg, stackOp)
}

func (u *UI) drawWinMessage(screen *ebiten.Image) {
	overlay := ebiten.NewImage(u.width, u.height)
	overlay.Fill(color.RGBA{0, 0, 0, 128})
	screen.DrawImage(overlay, nil)

	// Draw banner
	bannerW, bannerH := float32(250), float32(80)
	bannerX := float32(u.width)/2 - bannerW/2
	bannerY := float32(u.height)/2 - bannerH/2
	vector.DrawFilledRect(screen, bannerX, bannerY, bannerW, bannerH, color.RGBA{30, 80, 50, 255}, false)
	vector.StrokeRect(screen, bannerX, bannerY, bannerW, bannerH, 3, color.RGBA{255, 215, 0, 255}, false)

	// Draw text using the toolbar font
	if toolbarFace != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(u.width)/2-55, float64(u.height)/2-8)
		op.ColorScale.ScaleWithColor(color.RGBA{255, 215, 0, 255})
		text.Draw(screen, "You Win!", toolbarFace, op)
	}
}

func (u *UI) getPositionAt(mx, my int) *Position {
	fx, fy := float64(mx), float64(my)

	// Free cells
	for i := 0; i < NumFreeCells; i++ {
		if u.pointInRect(fx, fy, u.freeCellX(i), u.topRowY(), u.cardWidth(), u.cardHeight()) {
			return &Position{Location: LocFreeCell, Index: i}
		}
	}

	// Foundations
	for i := 0; i < NumFoundation; i++ {
		if u.pointInRect(fx, fy, u.foundationX(i), u.topRowY(), u.cardWidth(), u.cardHeight()) {
			return &Position{Location: LocFoundation, Index: i}
		}
	}

	// Tableau
	startY := u.tableauStartY()
	for col := 0; col < NumTableau; col++ {
		x := u.tableauColumnX(col)
		pile := u.game.Tableau[col]

		if len(pile) == 0 {
			if u.pointInRect(fx, fy, x, startY, u.cardWidth(), u.cardHeight()) {
				return &Position{Location: LocTableau, Index: col, CardIdx: 0}
			}
			continue
		}

		for cardIdx := len(pile) - 1; cardIdx >= 0; cardIdx-- {
			y := startY + float64(cardIdx)*u.stackOffset()
			h := u.cardHeight()
			if cardIdx < len(pile)-1 {
				h = u.stackOffset()
			}
			if u.pointInRect(fx, fy, x, y, u.cardWidth(), h) {
				return &Position{Location: LocTableau, Index: col, CardIdx: cardIdx}
			}
		}
	}

	return nil
}

func (u *UI) getCardScreenPosition(pos Position) (int, int) {
	switch pos.Location {
	case LocFreeCell:
		return int(u.freeCellX(pos.Index)), int(u.topRowY())
	case LocFoundation:
		return int(u.foundationX(pos.Index)), int(u.topRowY())
	case LocTableau:
		y := u.tableauStartY() + float64(pos.CardIdx)*u.stackOffset()
		return int(u.tableauColumnX(pos.Index)), int(y)
	}
	return 0, 0
}

func (u *UI) pointInRect(px, py, rx, ry, rw, rh float64) bool {
	return px >= rx && px < rx+rw && py >= ry && py < ry+rh
}

func (u *UI) tryAutoMove() {
	if u.autoMove && !u.isAnimating() {
		u.pendingAuto = true
	}
}

func (u *UI) doNextAutoMove() {
	if !u.autoMove {
		return
	}

	// Find the next card that can be auto-moved
	// Try free cells first
	for i := 0; i < NumFreeCells; i++ {
		if u.game.FreeCells[i] == nil {
			continue
		}
		card := *u.game.FreeCells[i]
		if foundIdx := u.game.FindFoundationForCard(card); foundIdx >= 0 {
			// Get source position
			fromX, fromY := u.freeCellX(i), u.topRowY()
			// Get destination position
			toX, toY := u.foundationX(foundIdx), u.topRowY()

			// Do the move in game state
			pos := Position{Location: LocFreeCell, Index: i}
			u.game.MoveToFoundation(pos, foundIdx)

			// Create animation
			u.animations = append(u.animations, CardAnimation{
				Card:         card,
				FromX:        fromX,
				FromY:        fromY,
				ToX:          toX,
				ToY:          toY,
				Progress:     0,
				ToFoundation: foundIdx,
			})

			// Check for more auto-moves after this animation
			u.pendingAuto = true
			return
		}
	}

	// Try tableau
	for col := 0; col < NumTableau; col++ {
		pile := u.game.Tableau[col]
		if len(pile) == 0 {
			continue
		}
		card := pile[len(pile)-1]
		if foundIdx := u.game.FindFoundationForCard(card); foundIdx >= 0 {
			// Get source position
			fromX := u.tableauColumnX(col)
			fromY := u.tableauStartY() + float64(len(pile)-1)*u.stackOffset()
			// Get destination position
			toX, toY := u.foundationX(foundIdx), u.topRowY()

			// Do the move in game state
			pos := Position{Location: LocTableau, Index: col, CardIdx: len(pile) - 1}
			u.game.MoveToFoundation(pos, foundIdx)

			// Create animation
			u.animations = append(u.animations, CardAnimation{
				Card:         card,
				FromX:        fromX,
				FromY:        fromY,
				ToX:          toX,
				ToY:          toY,
				Progress:     0,
				ToFoundation: foundIdx,
			})

			// Check for more auto-moves after this animation
			u.pendingAuto = true
			return
		}
	}
}
