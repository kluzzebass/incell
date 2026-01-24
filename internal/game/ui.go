package game

import (
	"image/color"
	"math"
	"math/rand"

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

// CardAnimation represents cards flying from one position to another
type CardAnimation struct {
	Cards        []Card // Cards being animated (can be a stack)
	FromX        float64
	FromY        float64
	ToX          float64
	ToY          float64
	Progress     float64 // 0.0 to 1.0
	ToFoundation int     // Which foundation pile (-1 if not going to foundation)
}

// Particle represents a firework particle
type Particle struct {
	X, Y   float64
	VX, VY float64
	Life   float64 // 0.0 to 1.0, decreases over time
	Color  color.RGBA
	Size   float64
}

// CardWiggle represents a wiggling card animation
type CardWiggle struct {
	Card  Card
	Time  float64 // 0.0 to 1.0, increases over time
	Phase float64 // Random starting phase offset
}

// UI handles the game rendering and input
type UI struct {
	game         *FreeCell
	dragging     bool
	dragStartX   int
	dragStartY   int
	dragOffsetX  int
	dragOffsetY  int
	cardW        float64
	cardH        float64
	cardScale    float64
	width        int
	height       int
	feltImage    *ebiten.Image
	autoMove     bool // Auto-move aces to foundation
	iGetIt       bool // "I get it, very funny" option
	animations   []CardAnimation
	pendingAuto  bool // Whether we need to check for auto-moves after animation
	particles    []Particle
	gameOver     bool    // True when no more valid moves
	gameOverFade float64 // 0.0 to 1.0 for grayscale fade
	cardWiggles  []CardWiggle
	hintIndex    int // Current hint index for cycling
}

// NewUI creates a new game UI
func NewUI() *UI {
	settings := LoadSettings()
	g := New(settings.IGetIt)

	// Try to load saved game state
	if HasSavedState() {
		g.LoadState()
	}

	u := &UI{
		game:        g,
		autoMove:    settings.AutoMove,
		iGetIt:      settings.IGetIt,
		pendingAuto: true, // Check for auto-moves on first frame
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

const animationSpeed = 0.1 // Progress per frame (higher = faster)

// Update implements ebiten.Game
func (u *UI) Update() error {
	// Update animations
	u.updateAnimations()
	u.updateParticles()

	// Check for win - spawn fireworks
	if u.game.IsWon() {
		u.spawnFirework()
	}

	// Check for no valid moves (only when not animating and not already game over)
	if !u.gameOver && !u.game.IsWon() && len(u.animations) == 0 && !u.game.HasValidMoves() {
		u.gameOver = true
	}

	// Handle game over fade and rain
	if u.gameOver {
		if u.gameOverFade < 1.0 {
			u.gameOverFade += 0.02
			if u.gameOverFade > 1.0 {
				u.gameOverFade = 1.0
			}
		}
		u.spawnRain()
	}

	// Update card wiggles
	if len(u.cardWiggles) > 0 {
		remaining := u.cardWiggles[:0]
		for i := range u.cardWiggles {
			u.cardWiggles[i].Time += 0.02 // ~1 second duration (60 frames * 0.02 = 1.2)
			if u.cardWiggles[i].Time < 1.0 {
				remaining = append(remaining, u.cardWiggles[i])
			}
		}
		u.cardWiggles = remaining
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
		// No animations, check if we need to do auto-moves (but not while mouse is held down)
		if u.pendingAuto && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			u.pendingAuto = false
			u.doNextAutoMove()
			// If no more auto-moves pending, save state
			if !u.pendingAuto {
				u.game.SaveState()
			}
		}
		return
	}

	// Update all animations in parallel
	remaining := u.animations[:0]
	for i := range u.animations {
		u.animations[i].Progress += animationSpeed
		if u.animations[i].Progress < 1.0 {
			remaining = append(remaining, u.animations[i])
		}
	}
	u.animations = remaining
}

func (u *UI) isAnimating() bool {
	return len(u.animations) > 0
}

func (u *UI) updateParticles() {
	if len(u.particles) == 0 {
		return
	}

	remaining := u.particles[:0]
	for i := range u.particles {
		p := &u.particles[i]
		p.X += p.VX
		p.Y += p.VY
		p.VY += 0.15 // gravity
		p.Life -= 0.015
		if p.Life > 0 {
			remaining = append(remaining, *p)
		}
	}
	u.particles = remaining
}

func (u *UI) spawnFirework() {
	// Randomly spawn fireworks
	if rand.Float64() > 0.08 {
		return
	}

	// Random position in upper portion of screen
	x := rand.Float64() * float64(u.width)
	y := rand.Float64() * float64(u.height) * 0.6

	// Random bright color
	colors := []color.RGBA{
		{255, 100, 100, 255}, // Red
		{100, 255, 100, 255}, // Green
		{100, 100, 255, 255}, // Blue
		{255, 255, 100, 255}, // Yellow
		{255, 100, 255, 255}, // Magenta
		{100, 255, 255, 255}, // Cyan
		{255, 200, 100, 255}, // Orange
		{255, 255, 255, 255}, // White
	}
	c := colors[rand.Intn(len(colors))]

	// Spawn particles in a burst
	numParticles := 30 + rand.Intn(20)
	for i := 0; i < numParticles; i++ {
		angle := rand.Float64() * 2 * math.Pi
		speed := 2 + rand.Float64()*4
		u.particles = append(u.particles, Particle{
			X:     x,
			Y:     y,
			VX:    math.Cos(angle) * speed,
			VY:    math.Sin(angle) * speed,
			Life:  0.7 + rand.Float64()*0.3,
			Color: c,
			Size:  2 + rand.Float64()*3,
		})
	}
}

func (u *UI) spawnRain() {
	// Spawn a few raindrops each frame
	for i := 0; i < 3; i++ {
		u.particles = append(u.particles, Particle{
			X:     rand.Float64() * float64(u.width),
			Y:     -10,
			VX:    -1 + rand.Float64()*0.5, // Slight wind
			VY:    8 + rand.Float64()*4,    // Falling fast
			Life:  1.5 + rand.Float64()*0.5,
			Color: color.RGBA{150, 150, 180, 255},
			Size:  1.5 + rand.Float64(),
		})
	}
}

// isCardAnimating checks if a specific card is currently being animated
func (u *UI) isCardAnimating(card Card) bool {
	for _, anim := range u.animations {
		for _, c := range anim.Cards {
			if c == card {
				return true
			}
		}
	}
	return false
}

// getDestinationScreenPosition returns the screen position for a destination
func (u *UI) getDestinationScreenPosition(dest Position) (float64, float64) {
	switch dest.Location {
	case LocFreeCell:
		return u.freeCellX(dest.Index), u.topRowY()
	case LocFoundation:
		return u.foundationX(dest.Index), u.topRowY()
	case LocTableau:
		pile := u.game.Tableau[dest.Index]
		y := u.tableauStartY() + float64(len(pile))*u.stackOffset()
		return u.tableauColumnX(dest.Index), y
	}
	return 0, 0
}

// animatedMove performs a move with animation
func (u *UI) animatedMove(from Position, dest Position) bool {
	// Get the cards to move
	var cards []Card
	switch from.Location {
	case LocFreeCell:
		if u.game.FreeCells[from.Index] == nil {
			return false
		}
		cards = []Card{*u.game.FreeCells[from.Index]}
	case LocTableau:
		cards = u.game.GetMovableCards(from.Index, from.CardIdx)
		if len(cards) == 0 {
			return false
		}
	default:
		return false
	}

	// Get positions before the move
	fromXInt, fromYInt := u.getCardScreenPosition(from)
	fromX, fromY := float64(fromXInt), float64(fromYInt)

	// Perform the actual move
	u.game.Select(from)
	if !u.game.TryMove(dest) {
		u.game.ClearSelection()
		return false
	}
	u.game.ClearSelection()

	// Get destination position (after move, so pile length is correct for tableau)
	var toX, toY float64
	switch dest.Location {
	case LocFreeCell:
		toX, toY = u.freeCellX(dest.Index), u.topRowY()
	case LocFoundation:
		toX, toY = u.foundationX(dest.Index), u.topRowY()
	case LocTableau:
		// The cards are now at the end of the pile, so we need to calculate where they landed
		pile := u.game.Tableau[dest.Index]
		cardIdx := len(pile) - len(cards)
		toY = u.tableauStartY() + float64(cardIdx)*u.stackOffset()
		toX = u.tableauColumnX(dest.Index)
	}

	// Create animation
	toFoundation := -1
	if dest.Location == LocFoundation {
		toFoundation = dest.Index
	}

	u.animations = append(u.animations, CardAnimation{
		Cards:        cards,
		FromX:        fromX,
		FromY:        fromY,
		ToX:          toX,
		ToY:          toY,
		Progress:     0,
		ToFoundation: toFoundation,
	})

	return true
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
			// Clear wiggles when clicking on the game area
			u.cardWiggles = nil
			// Just select the card for potential dragging - moves happen on release
			u.game.Select(*pos)
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
			// Check if we actually dragged (moved more than a few pixels)
			dx := mx - u.dragStartX
			dy := my - u.dragStartY
			actuallyDragged := dx*dx+dy*dy > 4 // More than 2 pixels

			if actuallyDragged {
				// User dragged - try to move to drop position
				if pos := u.getPositionAt(mx, my); pos != nil {
					if u.game.TryMove(*pos) {
						u.tryAutoMove()
					}
				}
			} else {
				// User clicked without dragging - do auto-move
				from := u.game.Selected.Position
				if dest := u.game.FindBestDestination(from); dest != nil {
					u.game.ClearSelection()
					if u.animatedMove(from, *dest) {
						u.tryAutoMove()
					}
				}
			}
		}
		u.dragging = false
		u.game.ClearSelection()
	}
}

// Draw implements ebiten.Game
func (u *UI) Draw(screen *ebiten.Image) {
	// Draw game to offscreen buffer if we need grayscale effect
	target := screen
	var offscreen *ebiten.Image
	if u.gameOverFade > 0 {
		offscreen = ebiten.NewImage(u.width, u.height)
		target = offscreen
	}

	u.drawFelt(target)

	u.drawEmptySlots(target)
	u.drawFoundations(target)
	u.drawFreeCells(target)
	u.drawTableau(target)

	if u.dragging && u.game.Selected != nil {
		mx, my := ebiten.CursorPosition()
		u.drawDraggedCards(target, mx, my)
	}

	// Draw animated cards
	u.drawAnimations(target)

	// Draw hint highlight
	u.drawHint(target)

	// Apply grayscale fade if game over
	if u.gameOverFade > 0 {
		u.drawWithGrayscale(screen, offscreen, u.gameOverFade)
	}

	// Draw particles (fireworks or rain) on top
	u.drawParticles(screen)

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

		// Draw all cards in the animation (handles stacks)
		for i, card := range anim.Cards {
			cardY := y + float64(i)*u.stackOffset()
			u.drawCard(screen, card, x, cardY, false)
		}
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

	// Toolbar area height (approximate)
	toolbarHeight := 75.0

	for y := 0; y < u.height; y++ {
		for x := 0; x < u.width; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx+dy*dy) / maxDist

			// Base green, darken towards edges
			factor := 1.0 - dist*0.25

			// Darken the bottom toolbar area
			bottomDist := float64(u.height-y) / toolbarHeight
			if bottomDist < 1.0 {
				factor *= 0.7 + 0.3*bottomDist
			}

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

	// Draw a slightly brighter background for free cells and foundations
	if slot == slotFreeCell || slot == slotFoundation {
		bgColor := color.RGBA{255, 255, 255, 15}
		u.fillRoundedRect(screen, float32(x), float32(y), float32(u.cardWidth()), float32(u.cardHeight()), radius, bgColor)
	}

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

func (u *UI) fillRoundedRect(screen *ebiten.Image, x, y, w, h, radius float32, c color.Color) {
	var path vector.Path
	path.MoveTo(x+radius, y)
	path.LineTo(x+w-radius, y)
	path.ArcTo(x+w, y, x+w, y+radius, radius)
	path.LineTo(x+w, y+h-radius)
	path.ArcTo(x+w, y+h, x+w-radius, y+h, radius)
	path.LineTo(x+radius, y+h)
	path.ArcTo(x, y+h, x, y+h-radius, radius)
	path.LineTo(x, y+radius)
	path.ArcTo(x, y, x+radius, y, radius)
	path.Close()

	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = colorToFloats(c)
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)
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
			if anim.ToFoundation == i && len(anim.Cards) > 0 && anim.Cards[0] == topCard {
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
			if u.dragging && selected {
				continue
			}
			// Skip cards that are currently being animated
			if u.isCardAnimating(card) {
				continue
			}
			x := u.freeCellX(i)
			// Apply wiggle rotation if this card is wiggling
			if wiggling, angle := u.isCardWiggling(card); wiggling {
				u.drawCardWithRotation(screen, card, x, y, angle, selected)
			} else {
				u.drawCard(screen, card, x, y, selected)
			}
		}
	}
}

func (u *UI) drawTableau(screen *ebiten.Image) {
	startY := u.tableauStartY()

	for col := 0; col < NumTableau; col++ {
		baseX := u.tableauColumnX(col)
		pile := u.game.Tableau[col]

		if len(pile) == 0 {
			u.drawEmptySlot(screen, baseX, startY, slotTableau)
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

			// Skip cards that are currently being animated
			if u.isCardAnimating(card) {
				continue
			}

			x := baseX
			// Check if this card (or a card above it in the stack) is wiggling
			// Each card in the stack gets its own slightly offset angle
			var angle float64
			var wiggleBaseTime float64
			var wiggleBasePhase float64
			var isWiggling bool
			for checkIdx := 0; checkIdx <= cardIdx; checkIdx++ {
				for _, w := range u.cardWiggles {
					if w.Card == pile[checkIdx] {
						isWiggling = true
						wiggleBaseTime = w.Time
						wiggleBasePhase = w.Phase
						break
					}
				}
				if isWiggling {
					break
				}
			}
			if isWiggling {
				// Add a small phase offset for each card in the stack
				cardPhase := wiggleBasePhase + float64(cardIdx)*0.3
				angle = u.getWiggleAngle(wiggleBaseTime, cardPhase)
			}

			if angle != 0 {
				u.drawCardWithRotation(screen, card, x, y, angle, selected && !u.dragging)
			} else {
				u.drawCard(screen, card, x, y, selected && !u.dragging)
			}
		}
	}
}

// getWiggleAngle returns the rotation angle (in radians) for a wiggling card based on time (0-1) and phase offset
func (u *UI) getWiggleAngle(t, phase float64) float64 {
	// Wiggle with decreasing amplitude
	amplitude := 0.15 * (1.0 - t) // Starts at ~8.5 degrees, decreases to 0
	frequency := 20.0             // Fast wiggle
	return amplitude * math.Sin((t*frequency+phase)*math.Pi)
}

// isCardWiggling checks if a card is currently wiggling and returns its rotation angle
func (u *UI) isCardWiggling(card Card) (bool, float64) {
	for _, w := range u.cardWiggles {
		if w.Card == card {
			return true, u.getWiggleAngle(w.Time, w.Phase)
		}
	}
	return false, 0
}

func (u *UI) drawHint(screen *ebiten.Image) {
	// Wiggle is applied directly in drawFreeCells and drawTableau
	// No additional drawing needed here
}

func (u *UI) drawCard(screen *ebiten.Image, card Card, x, y float64, highlighted bool) {
	u.drawCardWithRotation(screen, card, x, y, 0, highlighted)
}

func (u *UI) drawCardWithRotation(screen *ebiten.Image, card Card, x, y, angle float64, highlighted bool) {
	img := assets.GetCardImage(card.Rank, card.Suit)
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(u.cardScale, u.cardScale)

	if angle != 0 {
		// Rotate around the center of the card
		cx := u.cardWidth() / 2
		cy := u.cardHeight() / 2
		op.GeoM.Translate(-cx, -cy)
		op.GeoM.Rotate(angle)
		op.GeoM.Translate(cx, cy)
	}

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

var particleImage *ebiten.Image

func init() {
	// Create a single white circle image for all particles
	const size = 16
	particleImage = ebiten.NewImage(size, size)
	vector.DrawFilledCircle(particleImage, size/2, size/2, size/2-1, color.White, true)
}

func (u *UI) drawParticles(screen *ebiten.Image) {
	for _, p := range u.particles {
		op := &ebiten.DrawImageOptions{}
		// Scale to desired size
		scale := p.Size / 7
		op.GeoM.Translate(-8, -8) // Center the image
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(p.X, p.Y)
		// Apply color and alpha (ease-out: stays bright longer, fades quickly at end)
		op.ColorScale.ScaleWithColor(p.Color)
		alpha := float32(1 - (1-p.Life)*(1-p.Life)) // quadratic ease-out
		op.ColorScale.ScaleAlpha(alpha)
		screen.DrawImage(particleImage, op)
	}
}

func (u *UI) drawWithGrayscale(screen, src *ebiten.Image, fade float64) {
	op := &ebiten.DrawImageOptions{}

	// Grayscale conversion using luminance weights
	// We interpolate between identity matrix and grayscale matrix
	r := 0.299 * fade
	g := 0.587 * fade
	b := 0.114 * fade
	inv := 1 - fade

	// Row 1 (R output): lerp between (1,0,0) and (0.299,0.587,0.114)
	op.ColorM.SetElement(0, 0, inv+r)
	op.ColorM.SetElement(0, 1, g)
	op.ColorM.SetElement(0, 2, b)

	// Row 2 (G output): lerp between (0,1,0) and (0.299,0.587,0.114)
	op.ColorM.SetElement(1, 0, r)
	op.ColorM.SetElement(1, 1, inv+g)
	op.ColorM.SetElement(1, 2, b)

	// Row 3 (B output): lerp between (0,0,1) and (0.299,0.587,0.114)
	op.ColorM.SetElement(2, 0, r)
	op.ColorM.SetElement(2, 1, g)
	op.ColorM.SetElement(2, 2, inv+b)

	screen.DrawImage(src, op)
}

func (u *UI) drawWinMessage(screen *ebiten.Image) {
	// No overlay - let the fireworks show through!

	// Draw banner
	bannerW, bannerH := float32(250), float32(80)
	bannerX := float32(u.width)/2 - bannerW/2
	bannerY := float32(u.height)/2 - bannerH/2
	vector.DrawFilledRect(screen, bannerX, bannerY, bannerW, bannerH, color.RGBA{30, 80, 50, 230}, false)
	vector.StrokeRect(screen, bannerX, bannerY, bannerW, bannerH, 3, color.RGBA{255, 215, 0, 255}, false)

	// Draw text using the toolbar font
	if toolbarFace != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(u.width)/2-55, float64(u.height)/2-8)
		op.ColorScale.ScaleWithColor(color.RGBA{255, 215, 0, 255})
		text.Draw(screen, "You Win!", toolbarFace, op)
	}
}

func (u *UI) drawGameOverMessage(screen *ebiten.Image) {
	overlay := ebiten.NewImage(u.width, u.height)
	overlay.Fill(color.RGBA{0, 0, 0, 128})
	screen.DrawImage(overlay, nil)

	// Draw banner
	bannerW, bannerH := float32(280), float32(80)
	bannerX := float32(u.width)/2 - bannerW/2
	bannerY := float32(u.height)/2 - bannerH/2
	vector.DrawFilledRect(screen, bannerX, bannerY, bannerW, bannerH, color.RGBA{80, 30, 30, 255}, false)
	vector.StrokeRect(screen, bannerX, bannerY, bannerW, bannerH, 3, color.RGBA{200, 100, 100, 255}, false)

	// Draw text using the toolbar font
	if toolbarFace != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(u.width)/2-85, float64(u.height)/2-8)
		op.ColorScale.ScaleWithColor(color.RGBA{255, 200, 200, 255})
		text.Draw(screen, "No More Moves", toolbarFace, op)
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
	if u.autoMove {
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
				Cards:        []Card{card},
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
				Cards:        []Card{card},
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
