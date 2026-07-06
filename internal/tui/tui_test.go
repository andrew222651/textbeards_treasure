package tui

import (
	"strings"
	"testing"
	"time"

	"pirates/internal/game"
)

func TestLegacyControlPressNudgesWithoutStayingHeld(t *testing.T) {
	g := game.New(game.Config{
		Width:     100,
		Height:    100,
		ShipSpeed: 10,
	})
	start := g.Ship()

	quit := handleEvent(g, Event{Type: EventControlPress, Control: KeyForward, Legacy: true})
	if quit {
		t.Fatal("legacy movement should not quit")
	}

	if g.IsControlPressed(game.ControlForward) {
		t.Fatal("legacy movement should not leave forward held")
	}
	if got := g.Ship(); got.Y >= start.Y || got.X != start.X {
		t.Fatalf("expected legacy forward nudge, started at %#v and got %#v", start, got)
	}
}

func TestModernForwardPressAndReleaseControlsHeldMovement(t *testing.T) {
	g := game.New(game.Config{
		Width:     100,
		Height:    100,
		ShipSpeed: 10,
	})
	start := g.Ship()

	handleEvent(g, Event{Type: EventControlPress, Control: KeyForward})
	if !g.IsControlPressed(game.ControlForward) {
		t.Fatal("modern W press should start forward movement")
	}

	g.Update(100 * time.Millisecond)
	moved := g.Ship()
	if moved.Y >= start.Y || moved.X != start.X {
		t.Fatalf("expected ship to move forward, started at %#v and got %#v", start, moved)
	}

	handleEvent(g, Event{Type: EventControlRelease, Control: KeyForward})
	if g.IsControlPressed(game.ControlForward) {
		t.Fatal("modern W release should stop forward movement")
	}

	g.Update(300 * time.Millisecond)
	if got := g.Ship(); got != moved {
		t.Fatalf("expected ship to stop after release, got %#v after %#v", got, moved)
	}
}

func TestTurnPressRotatesAndReleaseStopsHeldTurning(t *testing.T) {
	g := game.New(game.Config{
		Width:        100,
		Height:       100,
		TurnInterval: 100 * time.Millisecond,
	})

	handleEvent(g, Event{Type: EventControlPress, Control: KeyTurnRight})
	if got := g.Heading(); got != game.HeadingNE {
		t.Fatalf("expected D press to rotate right to NE, got %v", got)
	}

	handleEvent(g, Event{Type: EventControlRelease, Control: KeyTurnRight})
	g.Update(time.Second)
	if got := g.Heading(); got != game.HeadingNE {
		t.Fatalf("expected heading to remain after D release, got %v", got)
	}
}

func TestRepeatInputsDoNotAccelerateHeldControls(t *testing.T) {
	g := game.New(game.Config{
		Width:        100,
		Height:       100,
		ShipSpeed:    10,
		TurnInterval: 100 * time.Millisecond,
	})

	handleEvent(g, Event{Type: EventControlPress, Control: KeyForward})
	g.Update(100 * time.Millisecond)
	moved := g.Ship()

	handleEvent(g, Event{Type: EventControlRepeat, Control: KeyForward})
	handleEvent(g, Event{Type: EventControlPress, Control: KeyForward, Legacy: true})
	if got := g.Ship(); got != moved {
		t.Fatalf("expected repeat input not to add extra movement while held, got %#v after %#v", got, moved)
	}

	handleEvent(g, Event{Type: EventControlPress, Control: KeyTurnRight})
	heading := g.Heading()
	handleEvent(g, Event{Type: EventControlRepeat, Control: KeyTurnRight})
	if got := g.Heading(); got != heading {
		t.Fatalf("expected repeat input not to rotate while held, got %v after %v", got, heading)
	}
}

func TestLoadSelectionAndFireEvents(t *testing.T) {
	g := game.New(game.Config{Width: 20, Height: 20})

	handleEvent(g, Event{Type: EventLoadSelect, Load: KeyGrapeShot})
	if got := g.CannonLoad(); got != game.LoadGrapeShot {
		t.Fatalf("expected grape shot selected, got %v", got)
	}

	handleEvent(g, Event{Type: EventFirePress, Fire: KeyFireLeft})
	if got := len(g.Shots()); got != 3 {
		t.Fatalf("expected grape shot fire to create three shots, got %d", got)
	}

	handleEvent(g, Event{Type: EventFireRepeat, Fire: KeyFireLeft})
	handleEvent(g, Event{Type: EventFireRelease, Fire: KeyFireLeft})
	if got := len(g.Shots()); got != 3 {
		t.Fatalf("expected repeat and release not to fire extra shots, got %d", got)
	}

	handleEvent(g, Event{Type: EventLoadSelect, Load: KeyCannonballs})
	if got := g.CannonLoad(); got != game.LoadCannonballs {
		t.Fatalf("expected cannonballs selected, got %v", got)
	}
}

func TestFireEventsUseRequestedShipSide(t *testing.T) {
	g := game.New(game.Config{Width: 20, Height: 20})

	handleEvent(g, Event{Type: EventFirePress, Fire: KeyFireRight})
	shots := g.Shots()
	if len(shots) != 1 {
		t.Fatalf("expected one shot, got %d", len(shots))
	}
	if shots[0].Heading != game.HeadingE {
		t.Fatalf("expected right-side cannon to fire east from a north-facing ship, got %v", shots[0].Heading)
	}
}

func TestTradeEventsOnlyApplyInPort(t *testing.T) {
	g := game.New(game.Config{
		Width:  40,
		Height: 20,
		PortPrices: map[game.Good]int{
			game.GoodRum:     10,
			game.GoodSugar:   20,
			game.GoodTobacco: 30,
		},
	})

	handleEvent(g, Event{Type: EventTradeQuantityIncrease})
	handleEvent(g, Event{Type: EventTradeBuy})
	if g.TradeQuantity() != 1 || g.Gold() != 100 || g.CargoUsed() != 0 {
		t.Fatalf("expected trade controls away from port to do nothing, qty=%d gold=%d cargo=%d", g.TradeQuantity(), g.Gold(), g.CargoUsed())
	}

	g = game.New(game.Config{
		Width:  8,
		Height: 6,
		PortPrices: map[game.Good]int{
			game.GoodRum:     10,
			game.GoodSugar:   20,
			game.GoodTobacco: 30,
		},
	})
	if !g.InPort() {
		t.Fatal("expected compact test map to start in port")
	}

	g.SelectCannonLoad(game.LoadGrapeShot)
	handleEvent(g, Event{Type: EventLoadSelect, Load: KeyCannonballs})
	if g.SelectedTradeGood() != game.GoodRum {
		t.Fatalf("expected 1 to select rum while in port, got %v", g.SelectedTradeGood())
	}
	if g.CannonLoad() != game.LoadGrapeShot {
		t.Fatalf("expected cannon load not to change while in port, got %v", g.CannonLoad())
	}

	handleEvent(g, Event{Type: EventTradeGoodSelect, TradeGood: KeyTradeTobacco})
	handleEvent(g, Event{Type: EventTradeQuantityIncrease})
	handleEvent(g, Event{Type: EventTradeBuy})
	if g.InventoryFor(game.GoodTobacco) != 2 || g.Gold() != 40 {
		t.Fatalf("expected to buy 2 tobacco, inventory=%d gold=%d", g.InventoryFor(game.GoodTobacco), g.Gold())
	}

	handleEvent(g, Event{Type: EventTradeQuantityDecrease})
	handleEvent(g, Event{Type: EventTradeSell})
	if g.InventoryFor(game.GoodTobacco) != 1 || g.Gold() != 70 {
		t.Fatalf("expected to sell 1 tobacco, inventory=%d gold=%d", g.InventoryFor(game.GoodTobacco), g.Gold())
	}
}

func TestRepairEventOnlyAppliesInPort(t *testing.T) {
	g := game.New(game.Config{Width: 80, Height: 40, ShotSpeed: 10, CannonRange: 25, EnemyAggroRange: 30, EnemyShipSpeed: 0.1, PortCorners: []game.MapCorner{game.CornerSW, game.CornerNE}})
	g.Update(2 * time.Second)
	if got := g.PlayerHitPoints(); got != 3 {
		t.Fatalf("expected enemy cannonball to damage player to 3 HP, got %d", got)
	}

	handleEvent(g, Event{Type: EventRepair})
	if got := g.PlayerHitPoints(); got != 3 {
		t.Fatalf("expected repair away from port to do nothing, got %d HP", got)
	}
	if got := g.Gold(); got != 100 {
		t.Fatalf("expected repair away from port not to spend gold, got %d", got)
	}

	sailToPort(t, g)
	handleEvent(g, Event{Type: EventRepair})
	if got := g.PlayerHitPoints(); got != g.MaxShipHitPoints() {
		t.Fatalf("expected repair in port to restore full HP, got %d", got)
	}
	if got := g.Gold(); got != 100-g.RepairFee() {
		t.Fatalf("expected repair in port to charge %d gold, got %d", g.RepairFee(), got)
	}
}

func TestMuteToggleEventTogglesGameMute(t *testing.T) {
	g := game.New(game.Config{})

	if handleEvent(g, Event{Type: EventMuteToggle}) {
		t.Fatal("mute event should not quit")
	}
	if !g.Muted() {
		t.Fatal("expected mute event to mute the game")
	}

	handleEvent(g, Event{Type: EventMuteToggle})
	if g.Muted() {
		t.Fatal("expected second mute event to unmute the game")
	}
}

func TestGoldCheatAddsGold(t *testing.T) {
	g := game.New(game.Config{})

	if handleEvent(g, Event{Type: EventGoldCheat}) {
		t.Fatal("gold cheat should not quit")
	}
	if got := g.Gold(); got != 1100 {
		t.Fatalf("expected Ctrl-G cheat to add 1000 gold, got %d", got)
	}
}

func TestUpdateGameFinalizesScoreWhenGameOverStarts(t *testing.T) {
	finalized := 0
	g := game.New(game.Config{
		Width:           80,
		Height:          40,
		ShotSpeed:       10,
		CannonRange:     25,
		EnemyAggroRange: 30,
		EnemyShipSpeed:  0.1,
		CannonCooldown:  time.Nanosecond,
		OnScoreFinalized: func(score int) (int, error) {
			finalized++
			return score, nil
		},
	})

	for i := 0; i < 3 && !g.GameOver(); i++ {
		if err := updateGame(g, 2*time.Second); err != nil {
			t.Fatalf("update game: %v", err)
		}
	}
	if !g.GameOver() {
		t.Fatal("expected repeated enemy cannon fire to destroy the player")
	}
	if finalized != 1 || !g.ScoreFinalized() {
		t.Fatalf("expected score to finalize once on game over, finalized=%d scoreFinalized=%v", finalized, g.ScoreFinalized())
	}

	if err := updateGame(g, time.Second); err != nil {
		t.Fatalf("update game after game over: %v", err)
	}
	if finalized != 1 {
		t.Fatalf("expected game-over score finalization to remain idempotent, got %d calls", finalized)
	}
}

func TestQuitEventRequestsExit(t *testing.T) {
	g := game.New(game.Config{})

	if !handleEvent(g, Event{Type: EventQuit}) {
		t.Fatal("quit event should request exit")
	}
}

func TestTerminalModesAreConfiguredForFullFrameRendering(t *testing.T) {
	if keyboardModeEnable != "\x1b[>10u" {
		t.Fatalf("expected keyboard mode enable sequence %q, got %q", "\x1b[>10u", keyboardModeEnable)
	}
	if !strings.Contains(enterScreen, "\x1b[?7l") {
		t.Fatalf("enter screen sequence should disable autowrap, got %q", enterScreen)
	}
	if !strings.Contains(leaveScreen, "\x1b[?7h") {
		t.Fatalf("leave screen sequence should restore autowrap, got %q", leaveScreen)
	}
}
