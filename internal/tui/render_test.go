package tui

import (
	"strings"
	"testing"
	"time"

	"pirates/internal/game"
)

func TestRenderUsesRawTerminalSafeLineEndings(t *testing.T) {
	g := game.New(game.Config{Width: 4, Height: 3})

	frame := Render(g, 4, 3)

	if strings.Contains(frame, "\n") && !strings.Contains(frame, "\r\n") {
		t.Fatalf("expected raw-terminal-safe CRLF line endings, got %q", frame)
	}
	if got := strings.Count(frame, "\r\n"); got != 2 {
		t.Fatalf("expected 2 CRLF row separators for a 3-row frame, got %d in %q", got, frame)
	}
}

func TestRenderUsesColorsForObjectTypes(t *testing.T) {
	g := game.New(game.Config{Width: 50, Height: 20})
	g.FireCannon(game.CannonLeft)

	frame := Render(g, 50, 20)
	for _, want := range []string{
		"\x1b[34m~", // water
		"\x1b[90m%", // map edge
		"\x1b[33m#", // port
		"\x1b[32m",  // player ship
		"\x1b[31m",  // enemy ship
		"\x1b[93m@", // cannonball
		"\x1b[36mGold",
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("expected frame to contain color sequence %q, got %q", want, frame)
		}
	}
}

func TestRenderDrawsIslandInMiddleOfMap(t *testing.T) {
	g := game.New(game.Config{})

	frame := Render(g, g.Width(), g.Height())
	if !strings.Contains(frame, "[92m+") {
		t.Fatalf("expected island land cells to render in green, got %q", frame)
	}
}

func TestRenderDrawsLargerShipWithBowSternAndSides(t *testing.T) {
	g := game.New(game.Config{Width: 30, Height: 15})

	frame := Render(g, 30, 15)

	assertGlyphCount(t, frame, "^", 1)
	assertGlyphCount(t, frame, "O", 5)
	assertGlyphCount(t, frame, "B", 1)
	assertGlyphCount(t, frame, "=", 8)
}

func TestRenderDrawsBowForHeading(t *testing.T) {
	g := game.New(game.Config{Width: 30, Height: 15})
	g.NudgeControl(game.ControlTurnRight, 0)
	g.NudgeControl(game.ControlTurnRight, 0)

	frame := Render(g, 30, 15)

	assertGlyphCount(t, frame, ">", 1)
	assertGlyphCount(t, frame, "O", 5)
	assertGlyphCount(t, frame, "B", 1)
}

func TestRenderDrawsEnemyShipUntilDestroyed(t *testing.T) {
	g := game.New(game.Config{Width: 50, Height: 11, ShotSpeed: 10, CannonRange: 100, CannonCooldown: time.Nanosecond})

	frame := Render(g, 50, 11)
	assertGlyphCount(t, frame, "x", 5)
	assertGlyphCount(t, frame, "-", 8)
	assertGlyphCount(t, frame, "A", 1)
	assertGlyphCount(t, frame, "Z", 1)

	for i := 0; i < 3; i++ {
		if !g.FireCannon(game.CannonRight) {
			t.Fatalf("expected cannon shot %d to fire", i+1)
		}
		g.Update(time.Second)
	}
	frame = Render(g, 50, 11)
	assertGlyphCount(t, frame, "x", 0)
	assertGlyphCount(t, frame, "-", 0)
	assertGlyphCount(t, frame, "A", 0)
	assertGlyphCount(t, frame, "Z", 0)
}

func TestRenderShowsGoldInventoryAndPort(t *testing.T) {
	g := game.New(game.Config{
		Width:       30,
		Height:      10,
		PortCorners: []game.MapCorner{game.CornerSW, game.CornerNE},
		PortPrices: map[game.Good]int{
			game.GoodRum: 10,
		},
	})

	frame := Render(g, 30, 10)
	if !strings.Contains(frame, "Gold: 100") || !strings.Contains(frame, "HP: 5/5") || !strings.Contains(frame, "Cargo: 0/10") {
		t.Fatalf("expected gold, HP, and cargo status, got %q", frame)
	}
	if !strings.Contains(frame, "Rum: 0") || !strings.Contains(frame, "Sugar: 0") || !strings.Contains(frame, "Tobacco: 0") {
		t.Fatalf("expected inventory status, got %q", frame)
	}
	if !strings.Contains(frame, "PIER") {
		t.Fatalf("expected bottom-left port marker, got %q", frame)
	}
}

func TestRenderShowsPortMenuWhenShipTouchesPort(t *testing.T) {
	g := game.New(game.Config{
		Width:       60,
		Height:      16,
		PortCorners: []game.MapCorner{game.CornerSW, game.CornerNE},
		PortPrices: map[game.Good]int{
			game.GoodRum:     10,
			game.GoodSugar:   20,
			game.GoodTobacco: 30,
		},
		PortUpgrades: map[string]game.UpgradeKind{
			"Port Royal": game.UpgradeCargo,
		},
	})
	sailToPort(t, g)
	g.SelectTradeGood(game.GoodTobacco)
	g.IncreaseTradeQuantity()
	g.BuySelected()

	frame := Render(g, 60, 16)
	if !strings.Contains(frame, "Port Royal Market") {
		t.Fatalf("expected port menu while in port, got %q", frame)
	}
	if !strings.Contains(frame, "U Cargo +5 1000g") {
		t.Fatalf("expected port menu to show one-time upgrade offer, got %q", frame)
	}
	if !strings.Contains(frame, ">3 Tobacco") || !strings.Contains(frame, "30g") {
		t.Fatalf("expected selected tobacco price row, got %q", frame)
	}
	if !strings.Contains(frame, "Gold 40") || !strings.Contains(frame, "HP 5/5") || !strings.Contains(frame, "Repair 25g") || !strings.Contains(frame, "Cargo 2/10") || !strings.Contains(frame, "Qty 2") {
		t.Fatalf("expected trade and repair status in menu, got %q", frame)
	}
}

func TestRenderScrollsToBottomLeftMapEdgeAndPort(t *testing.T) {
	g := game.New(game.Config{Width: 60, Height: 30, PortCorners: []game.MapCorner{game.CornerSW, game.CornerNE}})
	sailToPort(t, g)

	frame := Render(g, 30, 10)
	if !strings.Contains(frame, "Port Royal Market") {
		t.Fatalf("expected Port Royal market after scrolling to bottom-left port, got %q", frame)
	}
	if !strings.Contains(frame, "%") {
		t.Fatalf("expected visible map edge marker, got %q", frame)
	}
}

func TestRenderScrollsToTopRightMapEdgeAndPort(t *testing.T) {
	g := game.New(game.Config{Width: 60, Height: 30, PortCorners: []game.MapCorner{game.CornerSW, game.CornerNE}})
	sailToTopRightPort(t, g)

	frame := Render(g, 30, 10)
	if !strings.Contains(frame, "Havana Market") {
		t.Fatalf("expected Havana market after scrolling to top-right port, got %q", frame)
	}
	if !strings.Contains(frame, "%") {
		t.Fatalf("expected visible map edge marker, got %q", frame)
	}
}

func TestRenderShowsNorthWestPortBelowStatusRows(t *testing.T) {
	g := game.New(game.Config{Width: 60, Height: 30, PortCorners: []game.MapCorner{game.CornerNW, game.CornerSE}})
	sailNearNorthWestPort(t, g)

	cam := cameraFor(g, 30, 20)
	if cam.y != -statusRows {
		t.Fatalf("expected camera to show %d rows outside north map edge, got %#v", statusRows, cam)
	}

	frame := stripANSI(Render(g, 30, 20))
	lines := strings.Split(frame, "\r\n")
	if len(lines) < statusRows+2 {
		t.Fatalf("expected enough rendered lines, got %d in %q", len(lines), frame)
	}
	if !strings.Contains(lines[statusRows], "##########") || !strings.Contains(lines[statusRows+1], "PIER") {
		t.Fatalf("expected NW port below status rows, got lines %q and %q in %q", lines[statusRows], lines[statusRows+1], frame)
	}
}

func TestRenderShowsSelectedCannonLoadInTopLeft(t *testing.T) {
	g := game.New(game.Config{Width: 60, Height: 12})

	frame := Render(g, 60, 12)
	if !strings.Contains(frame, "Load: 1 Cannonballs") {
		t.Fatalf("expected cannonball load status, got %q", frame)
	}

	g.SelectCannonLoad(game.LoadGrapeShot)
	frame = Render(g, 60, 12)
	if !strings.Contains(frame, "Load: 2 Grape Shot") {
		t.Fatalf("expected grape shot load status, got %q", frame)
	}
}

func TestRenderShowsGameOverWhenPlayerIsDestroyed(t *testing.T) {
	g := game.New(game.Config{Width: 80, Height: 40, ShotSpeed: 10, CannonRange: 25, EnemyAggroRange: 30, EnemyShipSpeed: 0.1, CannonCooldown: time.Nanosecond})
	for i := 0; i < 3 && !g.GameOver(); i++ {
		g.Update(2 * time.Second)
	}
	if !g.GameOver() {
		t.Fatal("expected repeated enemy cannon fire to destroy the player")
	}

	frame := Render(g, 40, 12)
	if !strings.Contains(frame, "GAME OVER") {
		t.Fatalf("expected game-over message, got %q", frame)
	}
}

func TestRenderDrawsShots(t *testing.T) {
	g := game.New(game.Config{Width: 30, Height: 10})
	g.FireCannon(game.CannonLeft)
	frame := Render(g, 30, 10)
	assertGlyphCount(t, frame, "@", 1)

	g = game.New(game.Config{Width: 30, Height: 10})
	g.SelectCannonLoad(game.LoadGrapeShot)
	g.FireCannon(game.CannonLeft)
	frame = Render(g, 30, 10)
	assertGlyphCount(t, frame, "*", 3)
}

func sailToTopRightPort(t *testing.T, g *game.Game) {
	t.Helper()

	g.NudgeControl(game.ControlTurnRight, 0)
	g.NudgeControl(game.ControlTurnRight, 0)
	g.NudgeControl(game.ControlForward, 8*time.Second)
	g.NudgeControl(game.ControlTurnLeft, 0)
	g.NudgeControl(game.ControlTurnLeft, 0)
	g.NudgeControl(game.ControlForward, 8*time.Second)
	if !g.InPort() {
		t.Fatal("expected ship to reach top-right port")
	}
}

func sailNearNorthWestPort(t *testing.T, g *game.Game) {
	t.Helper()

	g.NudgeControl(game.ControlTurnLeft, 0)
	g.NudgeControl(game.ControlTurnLeft, 0)
	g.NudgeControl(game.ControlForward, 8*time.Second)
	g.NudgeControl(game.ControlTurnRight, 0)
	g.NudgeControl(game.ControlTurnRight, 0)
	g.NudgeControl(game.ControlForward, 475*time.Millisecond)
	if g.InPort() {
		t.Fatal("expected ship to be near the NW port without opening the port menu")
	}
}

func sailToPort(t *testing.T, g *game.Game) {
	t.Helper()

	g.NudgeControl(game.ControlTurnLeft, 0)
	g.NudgeControl(game.ControlTurnLeft, 0)
	g.NudgeControl(game.ControlForward, 8*time.Second)
	g.NudgeControl(game.ControlTurnLeft, 0)
	g.NudgeControl(game.ControlTurnLeft, 0)
	g.NudgeControl(game.ControlForward, 8*time.Second)
	if !g.InPort() {
		t.Fatal("expected ship to reach port")
	}
}

func assertGlyphCount(t *testing.T, frame, glyph string, want int) {
	t.Helper()

	if got := strings.Count(frame, glyph); got != want {
		t.Fatalf("expected %d %q glyphs, got %d in %q", want, glyph, got, frame)
	}
}

func stripANSI(text string) string {
	var out strings.Builder
	inEscape := false
	for i := 0; i < len(text); i++ {
		b := text[i]
		if inEscape {
			if b >= 64 && b <= 126 {
				inEscape = false
			}
			continue
		}
		if b == 27 {
			inEscape = true
			continue
		}
		out.WriteByte(b)
	}
	return out.String()
}
