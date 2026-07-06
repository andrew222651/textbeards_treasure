package game

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestDefaultMapIsLargerThanTypicalViewport(t *testing.T) {
	g := New(Config{})

	if g.Width() != 240 || g.Height() != 144 {
		t.Fatalf("expected default map 240x144, got %dx%d", g.Width(), g.Height())
	}
}

func TestDefaultMapHasMiddleIslandWithWaterAroundIt(t *testing.T) {
	g := New(Config{})
	islands := g.Islands()
	if len(islands) != 3 {
		t.Fatalf("expected main island plus two small islands on the default map, got %d", len(islands))
	}

	island := islands[0]
	if !g.CellIsIsland(int(math.Round(island.Center.X)), int(math.Round(island.Center.Y))) {
		t.Fatalf("expected island to occupy its center at %#v", island.Center)
	}
	if island.Center.X < float64(g.Width())*0.45 || island.Center.X > float64(g.Width())*0.55 || island.Center.Y < float64(g.Height())*0.45 || island.Center.Y > float64(g.Height())*0.55 {
		t.Fatalf("expected island near map center, got %#v on %dx%d", island.Center, g.Width(), g.Height())
	}

	cx := int(math.Round(island.Center.X))
	cy := int(math.Round(island.Center.Y))
	eastExtent := islandExtent(g, cx, cy, 1, 0)
	westExtent := islandExtent(g, cx, cy, -1, 0)
	if eastExtent == westExtent {
		t.Fatalf("expected irregular island shoreline, got equal east/west extents %d", eastExtent)
	}

	leftWater := int(math.Round(island.Center.X)) - island.RadiusX - 20
	rightWater := int(math.Round(island.Center.X)) + island.RadiusX + 20
	topWater := int(math.Round(island.Center.Y)) - island.RadiusY - 12
	bottomWater := int(math.Round(island.Center.Y)) + island.RadiusY + 12
	for _, cell := range []gridCell{{x: leftWater, y: int(math.Round(island.Center.Y))}, {x: rightWater, y: int(math.Round(island.Center.Y))}, {x: int(math.Round(island.Center.X)), y: topWater}, {x: int(math.Round(island.Center.X)), y: bottomWater}} {
		if g.CellIsIsland(cell.x, cell.y) {
			t.Fatalf("expected navigable water around island at %#v", cell)
		}
	}
	if g.shipIntersectsIsland(g.Ship(), g.Heading()) {
		t.Fatalf("expected player to start in open water, got ship at %#v", g.Ship())
	}
}

func TestDefaultMapHasTwoSmallIslandsWithoutPorts(t *testing.T) {
	g := New(Config{})
	islands := g.Islands()
	if len(islands) != 3 {
		t.Fatalf("expected main island plus two small islands, got %d", len(islands))
	}

	main := islands[0]
	for i, island := range islands[1:] {
		if island.RadiusX >= main.RadiusX || island.RadiusY >= main.RadiusY {
			t.Fatalf("expected island %d to be smaller than main island, main=%#v small=%#v", i+1, main, island)
		}
		if islandsOverlap(main, island) {
			t.Fatalf("expected small island %d not to overlap main island, main=%#v small=%#v", i+1, main, island)
		}
		for _, port := range g.Ports() {
			if portOverlapsIsland(port, island) {
				t.Fatalf("expected small island %d to have no port, port=%#v island=%#v", i+1, port, island)
			}
		}
	}
}

func TestDefaultMapHasIslandPortAndTwoDistinctCornerPorts(t *testing.T) {
	g := New(Config{})
	ports := g.Ports()
	if len(ports) != 3 {
		t.Fatalf("expected two corner ports plus one island port, got %d", len(ports))
	}

	if ports[0].OnIsland || ports[1].OnIsland {
		t.Fatalf("expected first two ports to be corner ports, got %#v", ports[:2])
	}
	if !validMapCorner(ports[0].Corner) || !validMapCorner(ports[1].Corner) || ports[0].Corner == ports[1].Corner {
		t.Fatalf("expected distinct valid random corner ports, got %#v and %#v", ports[0].Corner, ports[1].Corner)
	}
	if ports[0].Position != cornerPortPosition(ports[0].Corner, g.Width(), g.Height()) || ports[1].Position != cornerPortPosition(ports[1].Corner, g.Width(), g.Height()) {
		t.Fatalf("expected corner port positions to match corners, got %#v", ports[:2])
	}

	islandPort := ports[2]
	if islandPort.Name != "Tortuga" || !islandPort.OnIsland {
		t.Fatalf("expected island port Tortuga, got %#v", islandPort)
	}
	if !g.CellIsIsland(int(math.Round(islandPort.Position.X))+portWidth/2, int(math.Round(islandPort.Position.Y))) {
		t.Fatalf("expected Tortuga to sit on the island shoreline, got %#v", islandPort.Position)
	}
}

func TestPortsStartAtMapCornersWithIndependentPrices(t *testing.T) {
	g := New(Config{
		Width:       80,
		Height:      40,
		PortCorners: []MapCorner{CornerSW, CornerNE},
		PortPrices: map[Good]int{
			GoodRum:     10,
			GoodSugar:   11,
			GoodTobacco: 12,
		},
		TopRightPortPrices: map[Good]int{
			GoodRum:     20,
			GoodSugar:   21,
			GoodTobacco: 22,
		},
	})

	ports := g.Ports()
	if len(ports) != 2 {
		t.Fatalf("expected two ports, got %d", len(ports))
	}
	if ports[0].Name != "Port Royal" {
		t.Fatalf("expected first port Port Royal, got %q", ports[0].Name)
	}
	assertPosition(t, ports[0].Position, 0, 37)
	if ports[0].Prices[GoodRum] != 10 || ports[0].Prices[GoodSugar] != 11 || ports[0].Prices[GoodTobacco] != 12 {
		t.Fatalf("unexpected Port Royal prices: %#v", ports[0].Prices)
	}

	if ports[1].Name != "Havana" {
		t.Fatalf("expected second port Havana, got %q", ports[1].Name)
	}
	assertPosition(t, ports[1].Position, 70, 0)
	if ports[1].Prices[GoodRum] != 20 || ports[1].Prices[GoodSugar] != 21 || ports[1].Prices[GoodTobacco] != 22 {
		t.Fatalf("unexpected Havana prices: %#v", ports[1].Prices)
	}
}

func TestNoControlsPressedShipDoesNotMove(t *testing.T) {
	g := New(Config{Width: 20, Height: 10, ShipSpeed: 10})
	start := g.Ship()

	g.Update(time.Second)

	assertPosition(t, g.Ship(), start.X, start.Y)
}

func TestForwardMovesShipAlongHeading(t *testing.T) {
	tests := []struct {
		name    string
		heading Heading
		wantDX  float64
		wantDY  float64
	}{
		{name: "north", heading: HeadingN, wantDX: 0, wantDY: -10},
		{name: "northeast", heading: HeadingNE, wantDX: 10, wantDY: -10},
		{name: "east", heading: HeadingE, wantDX: 10, wantDY: 0},
		{name: "southeast", heading: HeadingSE, wantDX: 10, wantDY: 10},
		{name: "south", heading: HeadingS, wantDX: 0, wantDY: 10},
		{name: "southwest", heading: HeadingSW, wantDX: -10, wantDY: 10},
		{name: "west", heading: HeadingW, wantDX: -10, wantDY: 0},
		{name: "northwest", heading: HeadingNW, wantDX: -10, wantDY: -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(Config{Width: 100, Height: 100, ShipSpeed: 10})
			g.heading = tt.heading
			start := g.Ship()

			g.PressControl(ControlForward)
			g.Update(time.Second)

			assertPosition(t, g.Ship(), start.X+tt.wantDX, start.Y+tt.wantDY)
		})
	}
}

func TestDiagonalMovementCommitsBothAxesTogether(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, ShipSpeed: 10})
	g.heading = HeadingNE
	start := g.Ship()

	g.PressControl(ControlForward)
	g.Update(50 * time.Millisecond)
	assertPosition(t, g.Ship(), start.X, start.Y)

	g.Update(50 * time.Millisecond)
	assertPosition(t, g.Ship(), start.X+1, start.Y-1)

	g.Update(100 * time.Millisecond)
	assertPosition(t, g.Ship(), start.X+2, start.Y-2)
}

func TestBackwardMovesOppositeHeadingAtHalfForwardSpeed(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, ShipSpeed: 10})
	g.heading = HeadingE
	start := g.Ship()

	g.PressControl(ControlBackward)
	g.Update(time.Second)

	assertPosition(t, g.Ship(), start.X-5, start.Y)
}

func TestBackwardMovementAccumulatesHalfSpeedRemainder(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, ShipSpeed: 10})
	g.heading = HeadingE
	start := g.Ship()

	g.PressControl(ControlBackward)
	g.Update(100 * time.Millisecond)
	assertPosition(t, g.Ship(), start.X, start.Y)

	g.Update(100 * time.Millisecond)
	assertPosition(t, g.Ship(), start.X-1, start.Y)
}

func TestForwardAndBackwardCancelMovement(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, ShipSpeed: 10})
	start := g.Ship()

	g.PressControl(ControlForward)
	g.PressControl(ControlBackward)
	g.Update(time.Second)

	assertPosition(t, g.Ship(), start.X, start.Y)
}

func TestTurnControlsRotateShipInEightDirections(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, TurnInterval: 100 * time.Millisecond})

	g.PressControl(ControlTurnRight)
	assertHeading(t, g, HeadingNE)

	g.Update(99 * time.Millisecond)
	assertHeading(t, g, HeadingNE)

	g.Update(time.Millisecond)
	assertHeading(t, g, HeadingE)

	g.ReleaseControl(ControlTurnRight)
	g.PressControl(ControlTurnLeft)
	assertHeading(t, g, HeadingNE)
}

func TestHeldTurnCatchesUpByInterval(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, TurnInterval: 100 * time.Millisecond})

	g.PressControl(ControlTurnRight)
	g.Update(350 * time.Millisecond)

	assertHeading(t, g, HeadingS)
}

func TestOppositeTurnControlsCancelAfterImmediatePresses(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, TurnInterval: 100 * time.Millisecond})

	g.PressControl(ControlTurnRight)
	g.PressControl(ControlTurnLeft)
	assertHeading(t, g, HeadingN)

	g.Update(time.Second)
	assertHeading(t, g, HeadingN)
}

func TestReleaseStopsMovementAndTurning(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, ShipSpeed: 10, TurnInterval: 100 * time.Millisecond})

	g.PressControl(ControlForward)
	g.PressControl(ControlTurnRight)
	g.Update(time.Second)
	moved := g.Ship()
	heading := g.Heading()

	g.ReleaseControl(ControlForward)
	g.ReleaseControl(ControlTurnRight)
	g.Update(time.Second)

	assertPosition(t, g.Ship(), moved.X, moved.Y)
	assertHeading(t, g, heading)
}

func TestSetBoundsPreservesRelativeShipPosition(t *testing.T) {
	g := New(Config{})
	start := g.Ship()

	g.SetBounds(20, 10)

	wantX := scaleCoordinate(start.X, defaultWidth, 20)
	wantY := scaleCoordinate(start.Y, defaultHeight, 10)
	_, maxY := shipBounds(10)
	assertPosition(t, g.Ship(), wantX, math.Min(wantY, maxY))
}

func TestShipCenterIsClampedToKeepLargerShipVisible(t *testing.T) {
	g := New(Config{Width: 9, Height: 9, ShipSpeed: 100})

	g.PressControl(ControlForward)
	g.Update(time.Second)
	assertPosition(t, g.Ship(), 4, 3)

	g.ReleaseControl(ControlForward)
	g.heading = HeadingSE
	g.PressControl(ControlForward)
	g.Update(time.Second)
	assertPosition(t, g.Ship(), 5, 5)
}

func TestPlayerShipCannotSailThroughIsland(t *testing.T) {
	g := New(Config{ShipSpeed: 20})
	island := g.islands[0]
	start := Position{X: island.Center.X, Y: island.Center.Y + float64(island.RadiusY) + 10}
	g.ship = start
	g.heading = HeadingN

	g.PressControl(ControlForward)
	g.Update(time.Second)

	if g.shipIntersectsIsland(g.Ship(), g.Heading()) {
		t.Fatalf("expected ship to stop before island, got %#v", g.Ship())
	}
	if g.Ship().Y >= start.Y {
		t.Fatalf("expected ship to move toward island before stopping, start=%#v got=%#v", start, g.Ship())
	}
	if g.Ship().Y <= island.Center.Y+float64(island.RadiusY) {
		t.Fatalf("expected ship not to pass through island, island=%#v ship=%#v", island, g.Ship())
	}
}

func TestCannonShotDisappearsWhenItHitsIsland(t *testing.T) {
	g := New(Config{ShotSpeed: 10, CannonRange: 100})
	g.enemyDestroyed = true
	island := g.islands[0]
	g.shots = []Shot{{
		Position: Position{X: island.Center.X, Y: island.Center.Y - float64(island.RadiusY) - 2},
		Heading:  HeadingS,
		Load:     LoadCannonballs,
		Range:    10,
	}}

	g.Update(300 * time.Millisecond)

	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected shot to disappear after hitting island, got %d shots", got)
	}
}

func TestNudgeControlMovesOrTurnsWithoutHeldState(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, ShipSpeed: 10})
	start := g.Ship()

	g.NudgeControl(ControlForward, 100*time.Millisecond)
	if g.IsControlPressed(ControlForward) {
		t.Fatal("nudge should not leave forward held")
	}
	assertPosition(t, g.Ship(), start.X, start.Y-1)

	g.NudgeControl(ControlTurnRight, 0)
	assertHeading(t, g, HeadingNE)
}

func TestDefaultAndSelectCannonLoad(t *testing.T) {
	g := New(Config{})
	if got := g.CannonLoad(); got != LoadCannonballs {
		t.Fatalf("expected default load cannonballs, got %v", got)
	}

	g.SelectCannonLoad(LoadGrapeShot)
	if got := g.CannonLoad(); got != LoadGrapeShot {
		t.Fatalf("expected grape shot selected, got %v", got)
	}

	g.SelectCannonLoad(CannonLoad(99))
	if got := g.CannonLoad(); got != LoadGrapeShot {
		t.Fatalf("invalid load should not change selection, got %v", got)
	}
}

func TestCannonballFiresFromShipSideAndMovesBroadside(t *testing.T) {
	tests := []struct {
		name        string
		side        CannonSide
		wantHeading Heading
		wantY       float64
		wantMovedY  float64
	}{
		{name: "left side", side: CannonLeft, wantHeading: HeadingN, wantY: 6.5, wantMovedY: 5.5},
		{name: "right side", side: CannonRight, wantHeading: HeadingS, wantY: 12.5, wantMovedY: 13.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(Config{Width: 20, Height: 20, ShotSpeed: 10})
			g.heading = HeadingE
			ship := g.Ship()

			if !g.FireCannon(tt.side) {
				t.Fatal("expected cannon to fire")
			}
			shots := g.Shots()
			if len(shots) != 1 {
				t.Fatalf("expected one cannonball shot, got %d", len(shots))
			}
			if shots[0].Load != LoadCannonballs {
				t.Fatalf("expected cannonball load, got %v", shots[0].Load)
			}
			if shots[0].Heading != tt.wantHeading {
				t.Fatalf("expected shot heading %v, got %v", tt.wantHeading, shots[0].Heading)
			}
			assertPosition(t, shots[0].Position, ship.X, tt.wantY)

			g.Update(100 * time.Millisecond)
			shots = g.Shots()
			if len(shots) != 1 {
				t.Fatalf("expected shot to remain after moving, got %d", len(shots))
			}
			assertPosition(t, shots[0].Position, ship.X, tt.wantMovedY)
		})
	}
}

func TestDefaultCannonRangeIsShorterThanEnemyAggroRange(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, EnemyAggroRange: 20})

	if g.cannonRange <= 0 {
		t.Fatalf("expected positive cannon range, got %.6f", g.cannonRange)
	}
	if g.cannonRange >= g.enemyAggroRange {
		t.Fatalf("expected cannon range %.6f to be shorter than enemy aggro range %.6f", g.cannonRange, g.enemyAggroRange)
	}
}

func TestConfiguredCannonRangeIsCappedBelowEnemyAggroRange(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, EnemyAggroRange: 20, CannonRange: 99})

	if g.cannonRange >= g.enemyAggroRange {
		t.Fatalf("expected configured cannon range to be capped below enemy aggro range, range=%.6f aggro=%.6f", g.cannonRange, g.enemyAggroRange)
	}
}

func TestCannonShotsExpireAtRangeBeforeMapEdge(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, ShotSpeed: 10, EnemyAggroRange: 20, CannonRange: 3})
	g.enemyDestroyed = true
	g.heading = HeadingE

	if !g.FireCannon(CannonRight) {
		t.Fatal("expected cannon to fire")
	}
	g.Update(200 * time.Millisecond)
	if got := len(g.Shots()); got != 1 {
		t.Fatalf("expected shot to remain before reaching range, got %d shots", got)
	}

	g.Update(100 * time.Millisecond)
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected shot to disappear at max range, got %d shots", got)
	}
}

func TestCannonShotCanHitAtEndOfRange(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, ShotSpeed: 10, EnemyAggroRange: 20, CannonRange: 3})
	g.heading = HeadingE
	ship := g.Ship()
	g.enemy.Position = Position{X: ship.X, Y: ship.Y + 6}
	g.enemy.Heading = HeadingN

	if !g.FireCannon(CannonRight) {
		t.Fatal("expected cannon to fire")
	}
	g.Update(300 * time.Millisecond)

	if _, ok := g.Enemy(); !ok {
		t.Fatal("expected enemy to survive one cannonball hit")
	}
	if got, want := g.enemy.hitPoints, shipHitPoints-cannonballDamage; got != want {
		t.Fatalf("expected cannonball hit at end of range to leave %d hit points, got %d", want, got)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected shot to disappear on hit, got %d shots", got)
	}
}

func TestCannonShotFallsShortOfTargetBeyondRange(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, ShotSpeed: 10, EnemyAggroRange: 20, CannonRange: 3})
	g.heading = HeadingE
	ship := g.Ship()
	g.enemy.Position = Position{X: ship.X, Y: ship.Y + 8}
	g.enemy.Heading = HeadingN

	if !g.FireCannon(CannonRight) {
		t.Fatal("expected cannon to fire")
	}
	g.Update(time.Second)

	if _, ok := g.Enemy(); !ok {
		t.Fatal("expected target beyond cannon range to survive")
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected out-of-range shot to disappear, got %d shots", got)
	}
}

func TestCannonCooldownPreventsRefireForThreeSeconds(t *testing.T) {
	g := New(Config{Width: 200, Height: 200, ShotSpeed: 1, EnemyAggroRange: 200, CannonRange: 100})
	g.enemyDestroyed = true
	g.heading = HeadingE

	if !g.FireCannon(CannonLeft) {
		t.Fatal("expected first cannon shot to fire")
	}
	if g.FireCannon(CannonRight) {
		t.Fatal("expected immediate refire to be blocked by cooldown")
	}
	if got := len(g.Shots()); got != 1 {
		t.Fatalf("expected one shot during cooldown, got %d", got)
	}

	g.Update(2999 * time.Millisecond)
	if g.FireCannon(CannonRight) {
		t.Fatal("expected refire before 3 seconds to be blocked")
	}
	if got := len(g.Shots()); got != 1 {
		t.Fatalf("expected one shot before cooldown expires, got %d", got)
	}

	g.Update(time.Millisecond)
	if !g.FireCannon(CannonRight) {
		t.Fatal("expected refire at 3 seconds to be allowed")
	}
	if got := len(g.Shots()); got != 2 {
		t.Fatalf("expected two shots after cooldown expires, got %d", got)
	}
}

func TestGrapeShotFiresBroadsideSpread(t *testing.T) {
	g := New(Config{Width: 20, Height: 20})
	g.SelectCannonLoad(LoadGrapeShot)
	g.heading = HeadingN

	g.FireCannon(CannonLeft)
	shots := g.Shots()
	if len(shots) != 3 {
		t.Fatalf("expected three grape shot pellets, got %d", len(shots))
	}
	wantHeadings := []Heading{HeadingSW, HeadingW, HeadingNW}
	for i, shot := range shots {
		if shot.Load != LoadGrapeShot {
			t.Fatalf("shot %d: expected grape shot load, got %v", i, shot.Load)
		}
		if shot.Heading != wantHeadings[i] {
			t.Fatalf("shot %d: expected heading %v, got %v", i, wantHeadings[i], shot.Heading)
		}
	}
}

func TestCannonFireCallbackRunsOncePerPlayerCannonFire(t *testing.T) {
	fires := 0
	g := New(Config{Width: 20, Height: 20, OnCannonFire: func() { fires++ }})
	g.SelectCannonLoad(LoadGrapeShot)

	if !g.FireCannon(CannonLeft) {
		t.Fatal("expected cannon to fire")
	}
	if got := len(g.Shots()); got != 3 {
		t.Fatalf("expected grape shot spread, got %d shots", got)
	}
	if fires != 1 {
		t.Fatalf("expected one callback for one grape shot cannon fire, got %d", fires)
	}

	if g.FireCannon(CannonRight) {
		t.Fatal("expected cooldown to block immediate refire")
	}
	if fires != 1 {
		t.Fatalf("expected blocked refire not to call callback, got %d", fires)
	}
}

func TestCannonFireCallbackRunsForEnemyCannonFire(t *testing.T) {
	fires := 0
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 1, EnemyAggroRange: 20, OnCannonFire: func() { fires++ }})
	g.enemy.Position = Position{X: 40, Y: 20}
	g.enemy.Heading = HeadingN
	g.ship = Position{X: 30, Y: 20}

	g.Update(time.Millisecond)

	if got := len(g.Shots()); got != 1 {
		t.Fatalf("expected enemy to fire one shot, got %d", got)
	}
	if fires != 1 {
		t.Fatalf("expected enemy cannon fire to call callback once, got %d", fires)
	}
}

func TestPortStartsInBottomLeftWithGoldAndPrices(t *testing.T) {
	g := New(Config{
		Width:       40,
		Height:      20,
		PortCorners: []MapCorner{CornerSW, CornerNE},
		PortPrices: map[Good]int{
			GoodRum:     10,
			GoodSugar:   7,
			GoodTobacco: 18,
		},
	})

	port := g.Port()
	if port.Name != "Port Royal" {
		t.Fatalf("expected port name Port Royal, got %q", port.Name)
	}
	assertPosition(t, port.Position, 0, 17)
	if g.Gold() != 100 {
		t.Fatalf("expected 100 starting gold, got %d", g.Gold())
	}
	if g.CargoCapacity() != 10 || g.CargoUsed() != 0 {
		t.Fatalf("expected empty 10-unit cargo hold, got %d/%d", g.CargoUsed(), g.CargoCapacity())
	}
	if g.Price(GoodRum) != 10 || g.Price(GoodSugar) != 7 || g.Price(GoodTobacco) != 18 {
		t.Fatalf("expected configured prices, got %#v", port.Prices)
	}
}

func TestRandomPortPricesArePositiveAndFixedForGame(t *testing.T) {
	g := New(Config{Width: 40, Height: 20})

	prices := g.Port().Prices
	for _, good := range Goods() {
		if prices[good] <= 0 {
			t.Fatalf("expected positive random price for %s, got %d", GoodName(good), prices[good])
		}
		if g.Price(good) != prices[good] {
			t.Fatalf("expected price for %s to stay fixed", GoodName(good))
		}
	}
}

func TestInPortWhenShipTouchesPort(t *testing.T) {
	g := New(Config{Width: 40, Height: 20, PortCorners: []MapCorner{CornerSW, CornerNE}})
	if g.InPort() {
		t.Fatal("expected starting ship to be away from port")
	}

	dockAtPortRoyal(g)
	if !g.InPort() {
		t.Fatal("expected ship touching bottom-left port to be in port")
	}
}

func TestBuyingAndSellingGoodsAtPort(t *testing.T) {
	g := New(Config{
		Width:       40,
		Height:      20,
		PortCorners: []MapCorner{CornerSW, CornerNE},
		PortPrices: map[Good]int{
			GoodRum:     10,
			GoodSugar:   20,
			GoodTobacco: 30,
		},
	})
	dockAtPortRoyal(g)

	if bought := g.Buy(GoodRum, 7); bought != 7 {
		t.Fatalf("expected to buy 7 rum, bought %d", bought)
	}
	if g.Gold() != 30 || g.InventoryFor(GoodRum) != 7 || g.CargoUsed() != 7 {
		t.Fatalf("unexpected state after rum purchase: gold=%d rum=%d cargo=%d", g.Gold(), g.InventoryFor(GoodRum), g.CargoUsed())
	}

	if bought := g.Buy(GoodSugar, 10); bought != 1 {
		t.Fatalf("expected gold to limit sugar purchase to 1, bought %d", bought)
	}
	if g.Gold() != 10 || g.InventoryFor(GoodSugar) != 1 || g.CargoUsed() != 8 {
		t.Fatalf("unexpected state after sugar purchase: gold=%d sugar=%d cargo=%d", g.Gold(), g.InventoryFor(GoodSugar), g.CargoUsed())
	}

	if sold := g.Sell(GoodRum, 99); sold != 7 {
		t.Fatalf("expected to sell only 7 rum in cargo, sold %d", sold)
	}
	if g.Gold() != 80 || g.InventoryFor(GoodRum) != 0 || g.CargoUsed() != 1 {
		t.Fatalf("unexpected state after sale: gold=%d rum=%d cargo=%d", g.Gold(), g.InventoryFor(GoodRum), g.CargoUsed())
	}
}

func TestTradeCallbackRunsOnlyWhenGoodsAreBoughtOrSold(t *testing.T) {
	trades := 0
	g := New(Config{
		Width:       40,
		Height:      20,
		OnTrade:     func() { trades++ },
		PortCorners: []MapCorner{CornerSW, CornerNE},
		PortPrices: map[Good]int{
			GoodRum: 10,
		},
	})

	if bought := g.Buy(GoodRum, 1); bought != 0 {
		t.Fatalf("expected buy away from port to fail, bought %d", bought)
	}
	if trades != 0 {
		t.Fatalf("expected failed buy not to call callback, got %d", trades)
	}

	dockAtPortRoyal(g)
	if bought := g.Buy(GoodRum, 0); bought != 0 {
		t.Fatalf("expected zero-quantity buy to fail, bought %d", bought)
	}
	if trades != 0 {
		t.Fatalf("expected zero-quantity buy not to call callback, got %d", trades)
	}

	if bought := g.Buy(GoodRum, 2); bought != 2 {
		t.Fatalf("expected successful buy of 2 rum, bought %d", bought)
	}
	if trades != 1 {
		t.Fatalf("expected successful buy to call callback once, got %d", trades)
	}

	if sold := g.Sell(GoodSugar, 1); sold != 0 {
		t.Fatalf("expected sell with no sugar to fail, sold %d", sold)
	}
	if trades != 1 {
		t.Fatalf("expected failed sell not to call callback, got %d", trades)
	}

	if sold := g.Sell(GoodRum, 1); sold != 1 {
		t.Fatalf("expected successful sale of 1 rum, sold %d", sold)
	}
	if trades != 2 {
		t.Fatalf("expected successful sell to call callback once, got %d", trades)
	}
}

func TestTradingUsesCurrentPortPrices(t *testing.T) {
	g := New(Config{
		Width:       80,
		Height:      40,
		PortCorners: []MapCorner{CornerSW, CornerNE},
		PortPrices: map[Good]int{
			GoodRum: 10,
		},
		TopRightPortPrices: map[Good]int{
			GoodRum: 25,
		},
	})

	dockAtPortRoyal(g)
	if port, ok := g.CurrentPort(); !ok || port.Name != "Port Royal" {
		t.Fatalf("expected current port Port Royal, got %#v ok=%v", port, ok)
	}
	if bought := g.Buy(GoodRum, 2); bought != 2 {
		t.Fatalf("expected to buy 2 rum at Port Royal, bought %d", bought)
	}
	if g.Gold() != 80 || g.InventoryFor(GoodRum) != 2 {
		t.Fatalf("expected Port Royal price 10, gold=%d rum=%d", g.Gold(), g.InventoryFor(GoodRum))
	}

	dockAtHavana(g)
	if port, ok := g.CurrentPort(); !ok || port.Name != "Havana" {
		t.Fatalf("expected current port Havana, got %#v ok=%v", port, ok)
	}
	if sold := g.Sell(GoodRum, 1); sold != 1 {
		t.Fatalf("expected to sell 1 rum at Havana, sold %d", sold)
	}
	if g.Gold() != 105 || g.InventoryFor(GoodRum) != 1 {
		t.Fatalf("expected Havana price 25, gold=%d rum=%d", g.Gold(), g.InventoryFor(GoodRum))
	}
}

func TestPortStateCallbackRunsOnEnterAndLeave(t *testing.T) {
	states := []bool{}
	g := New(Config{Width: 80, Height: 40, PortCorners: []MapCorner{CornerSW, CornerNE}, OnPortStateChange: func(inPort bool) {
		states = append(states, inPort)
	}})

	dockAtPortRoyal(g)
	g.updatePortVisit()
	g.updatePortVisit()
	if len(states) != 1 || !states[0] {
		t.Fatalf("expected one enter-port callback, got %#v", states)
	}

	leavePorts(g)
	g.updatePortVisit()
	g.updatePortVisit()
	if len(states) != 2 || states[1] {
		t.Fatalf("expected one leave-port callback after enter, got %#v", states)
	}
}

func TestVisitingDifferentPortRegeneratesOtherPortPricesNearPreviousPrices(t *testing.T) {
	g := New(Config{
		Width:       80,
		Height:      40,
		PortCorners: []MapCorner{CornerSW, CornerNE},
		PortPrices: map[Good]int{
			GoodRum:     10,
			GoodSugar:   20,
			GoodTobacco: 30,
		},
		TopRightPortPrices: map[Good]int{
			GoodRum:     50,
			GoodSugar:   60,
			GoodTobacco: 70,
		},
	})
	g.priceRNG = rand.New(rand.NewSource(1))
	portRoyalBefore := g.ports[0].Prices
	havanaBefore := g.ports[1].Prices

	dockAtPortRoyal(g)
	g.updatePortVisit()
	if g.ports[0].Prices != portRoyalBefore || g.ports[1].Prices != havanaBefore {
		t.Fatalf("expected first port visit not to regenerate prices, got %#v", g.Ports())
	}

	leavePorts(g)
	g.updatePortVisit()
	dockAtHavana(g)
	g.updatePortVisit()

	if g.ports[1].Prices != havanaBefore {
		t.Fatalf("expected current port prices to stay fixed, before=%#v after=%#v", havanaBefore, g.ports[1].Prices)
	}
	assertRegeneratedNear(t, g.ports[0].Prices, portRoyalBefore)

	portRoyalAfter := g.ports[0].Prices
	g.updatePortVisit()
	if g.ports[0].Prices != portRoyalAfter {
		t.Fatalf("expected prices not to regenerate repeatedly while still docked, before=%#v after=%#v", portRoyalAfter, g.ports[0].Prices)
	}
}

func TestRegeneratedPricesBiasTowardCurrentGoodAverage(t *testing.T) {
	g := New(Config{
		Width:       80,
		Height:      40,
		PortCorners: []MapCorner{CornerSW, CornerNE},
		PortPrices: map[Good]int{
			GoodRum:     10,
			GoodSugar:   50,
			GoodTobacco: 30,
		},
		TopRightPortPrices: map[Good]int{
			GoodRum:     50,
			GoodSugar:   10,
			GoodTobacco: 30,
		},
	})
	g.priceRNG = rand.New(rand.NewSource(2))
	previous := g.ports[0].Prices

	g.regenerateOtherPortPrices(1)

	got := g.ports[0].Prices
	if got[GoodRum] <= previous[GoodRum] {
		t.Fatalf("expected low rum price %d to move up toward average, got %d", previous[GoodRum], got[GoodRum])
	}
	if got[GoodSugar] >= previous[GoodSugar] {
		t.Fatalf("expected high sugar price %d to move down toward average, got %d", previous[GoodSugar], got[GoodSugar])
	}
	assertRegeneratedNear(t, got, previous)
}

func TestNearbyPriceBiasesTowardAverageWithinPreviousPriceRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	lowPrevious := 10
	lowNext := nearbyPrice(lowPrevious, 50, rng)
	if lowNext <= lowPrevious {
		t.Fatalf("expected price below average to move up from %d, got %d", lowPrevious, lowNext)
	}
	if delta := absInt(lowNext - lowPrevious); delta > maxInt(1, lowPrevious/5) {
		t.Fatalf("expected low price move to stay near previous, previous=%d got=%d", lowPrevious, lowNext)
	}

	highPrevious := 50
	highNext := nearbyPrice(highPrevious, 10, rng)
	if highNext >= highPrevious {
		t.Fatalf("expected price above average to move down from %d, got %d", highPrevious, highNext)
	}
	if delta := absInt(highNext - highPrevious); delta > maxInt(1, highPrevious/5) {
		t.Fatalf("expected high price move to stay near previous, previous=%d got=%d", highPrevious, highNext)
	}
}

func TestReturningToSamePreviousPortDoesNotRegenerateOtherPorts(t *testing.T) {
	g := New(Config{
		Width:       80,
		Height:      40,
		PortCorners: []MapCorner{CornerSW, CornerNE},
		PortPrices: map[Good]int{
			GoodRum:     10,
			GoodSugar:   20,
			GoodTobacco: 30,
		},
		TopRightPortPrices: map[Good]int{
			GoodRum:     50,
			GoodSugar:   60,
			GoodTobacco: 70,
		},
	})
	havanaBefore := g.ports[1].Prices

	dockAtPortRoyal(g)
	g.updatePortVisit()
	leavePorts(g)
	g.updatePortVisit()
	dockAtPortRoyal(g)
	g.updatePortVisit()

	if g.ports[1].Prices != havanaBefore {
		t.Fatalf("expected other ports not to regenerate when returning to same previous port, before=%#v after=%#v", havanaBefore, g.ports[1].Prices)
	}
}

func TestTradeRequiresPortAndValidQuantity(t *testing.T) {
	g := New(Config{Width: 40, Height: 20, PortPrices: map[Good]int{GoodRum: 10}, PortCorners: []MapCorner{CornerSW, CornerNE}})

	if bought := g.Buy(GoodRum, 1); bought != 0 {
		t.Fatalf("expected buying away from port to fail, bought %d", bought)
	}
	if sold := g.Sell(GoodRum, 1); sold != 0 {
		t.Fatalf("expected selling away from port to fail, sold %d", sold)
	}

	dockAtPortRoyal(g)
	if bought := g.Buy(GoodRum, 0); bought != 0 {
		t.Fatalf("expected zero-quantity buy to fail, bought %d", bought)
	}
}

func TestRepairShipAtPortRestoresHitPointsForFixedFee(t *testing.T) {
	g := New(Config{Width: 40, Height: 20})
	hitPlayer(g, LoadCannonballs)
	dockAtPortRoyal(g)

	paid := g.RepairShip()
	if paid != shipRepairFee {
		t.Fatalf("expected repair to cost %d gold, paid %d", shipRepairFee, paid)
	}
	if got := g.PlayerHitPoints(); got != shipHitPoints {
		t.Fatalf("expected repair to restore player to %d hit points, got %d", shipHitPoints, got)
	}
	if got := g.Gold(); got != defaultGold-shipRepairFee {
		t.Fatalf("expected repair fee to be deducted from gold, got %d", got)
	}
}

func TestRepairCallbackRunsOnlyWhenRepairSucceeds(t *testing.T) {
	repairs := 0
	g := New(Config{Width: 40, Height: 20, OnRepair: func() { repairs++ }})

	if paid := g.RepairShip(); paid != 0 {
		t.Fatalf("expected repair away from port to fail, paid %d", paid)
	}
	if repairs != 0 {
		t.Fatalf("expected failed repair not to call callback, got %d", repairs)
	}

	dockAtPortRoyal(g)
	if paid := g.RepairShip(); paid != 0 {
		t.Fatalf("expected undamaged repair to fail, paid %d", paid)
	}
	if repairs != 0 {
		t.Fatalf("expected undamaged repair not to call callback, got %d", repairs)
	}

	hitPlayer(g, LoadCannonballs)
	if paid := g.RepairShip(); paid != shipRepairFee {
		t.Fatalf("expected successful repair to cost %d, paid %d", shipRepairFee, paid)
	}
	if repairs != 1 {
		t.Fatalf("expected successful repair to call callback once, got %d", repairs)
	}
}

func TestRepairShipRequiresPortDamageAndGold(t *testing.T) {
	g := New(Config{Width: 40, Height: 20})
	hitPlayer(g, LoadCannonballs)

	if paid := g.RepairShip(); paid != 0 {
		t.Fatalf("expected repair away from port to fail, paid %d", paid)
	}
	if got := g.PlayerHitPoints(); got != shipHitPoints-cannonballDamage {
		t.Fatalf("expected failed repair to leave damage unchanged, got %d hit points", got)
	}

	dockAtPortRoyal(g)
	g.gold = shipRepairFee - 1
	if paid := g.RepairShip(); paid != 0 {
		t.Fatalf("expected repair without enough gold to fail, paid %d", paid)
	}
	if got := g.PlayerHitPoints(); got != shipHitPoints-cannonballDamage {
		t.Fatalf("expected unaffordable repair to leave damage unchanged, got %d hit points", got)
	}

	g.gold = defaultGold
	g.playerHitPoints = shipHitPoints
	if paid := g.RepairShip(); paid != 0 {
		t.Fatalf("expected repairing an undamaged ship to do nothing, paid %d", paid)
	}
	if got := g.Gold(); got != defaultGold {
		t.Fatalf("expected undamaged repair to leave gold unchanged, got %d", got)
	}
}

func TestPortsOfferConfiguredOneTimeUpgrades(t *testing.T) {
	g := New(Config{
		Width:  80,
		Height: 40,
		PortUpgrades: map[string]UpgradeKind{
			"Port Royal": UpgradeCargo,
			"Havana":     UpgradeCannons,
		},
	})

	ports := g.Ports()
	if len(ports) != 2 {
		t.Fatalf("expected two ports, got %d", len(ports))
	}
	if ports[0].Upgrade != UpgradeCargo || ports[0].UpgradePurchased {
		t.Fatalf("expected Port Royal to offer unsold cargo upgrade, got %#v", ports[0])
	}
	if ports[1].Upgrade != UpgradeCannons || ports[1].UpgradePurchased {
		t.Fatalf("expected Havana to offer unsold cannon upgrade, got %#v", ports[1])
	}
}

func TestPortsOfferRandomUniqueValidUpgradesByDefault(t *testing.T) {
	g := New(Config{Width: 80, Height: 40})
	seen := map[UpgradeKind]string{}

	for _, port := range g.Ports() {
		if !validUpgrade(port.Upgrade) {
			t.Fatalf("expected %s to offer a valid upgrade, got %v", port.Name, port.Upgrade)
		}
		if otherPort := seen[port.Upgrade]; otherPort != "" {
			t.Fatalf("expected unique upgrades, both %s and %s offer %v", otherPort, port.Name, port.Upgrade)
		}
		seen[port.Upgrade] = port.Name
		if port.UpgradePurchased {
			t.Fatalf("expected %s upgrade to start unsold", port.Name)
		}
	}
}

func TestDuplicateConfiguredPortUpgradesAreReassigned(t *testing.T) {
	g := New(Config{
		Width:  80,
		Height: 40,
		PortUpgrades: map[string]UpgradeKind{
			"Port Royal": UpgradeHull,
			"Havana":     UpgradeHull,
		},
	})

	ports := g.Ports()
	if ports[0].Upgrade != UpgradeHull {
		t.Fatalf("expected first configured hull upgrade to be honored, got %#v", ports[0])
	}
	assertNoDuplicateValidUpgrades(t, ports)
}

func TestPortUpgradeAssignmentUsesNoUpgradeWhenUniqueUpgradesRunOut(t *testing.T) {
	upgrades := configuredPortUpgrades([]string{"A", "B", "C", "D", "E"}, nil, rand.New(rand.NewSource(1)))
	seen := map[UpgradeKind]bool{}
	noneCount := 0
	for _, upgrade := range upgrades {
		if upgrade == UpgradeNone {
			noneCount++
			continue
		}
		if seen[upgrade] {
			t.Fatalf("expected no duplicate valid upgrades, got %#v", upgrades)
		}
		seen[upgrade] = true
	}
	if noneCount != 1 {
		t.Fatalf("expected one port with no upgrade after exhausting four upgrade kinds, got %d in %#v", noneCount, upgrades)
	}
}

func TestBuyingHullUpgradeIncreasesMaxAndCurrentHitPointsOnce(t *testing.T) {
	g := New(Config{
		Width:  40,
		Height: 20,
		PortUpgrades: map[string]UpgradeKind{
			"Port Royal": UpgradeHull,
		},
	})
	dockAtPortRoyal(g)
	g.gold = upgradeCost

	paid := g.BuyPortUpgrade()
	if paid != upgradeCost {
		t.Fatalf("expected upgrade to cost %d, paid %d", upgradeCost, paid)
	}
	if got, want := g.MaxShipHitPoints(), shipHitPoints+hullUpgradeHitPoints; got != want {
		t.Fatalf("expected max HP %d after hull upgrade, got %d", want, got)
	}
	if got, want := g.PlayerHitPoints(), shipHitPoints+hullUpgradeHitPoints; got != want {
		t.Fatalf("expected current HP %d after hull upgrade, got %d", want, got)
	}
	if got := g.Gold(); got != 0 {
		t.Fatalf("expected gold to be spent, got %d", got)
	}
	port, ok := g.CurrentPort()
	if !ok || !port.UpgradePurchased {
		t.Fatalf("expected current port upgrade to be marked purchased, got %#v ok=%v", port, ok)
	}

	g.gold = upgradeCost
	if paid := g.BuyPortUpgrade(); paid != 0 {
		t.Fatalf("expected purchased port not to sell another upgrade, paid %d", paid)
	}
	if got, want := g.MaxShipHitPoints(), shipHitPoints+hullUpgradeHitPoints; got != want {
		t.Fatalf("expected max HP to remain %d after repeat buy, got %d", want, got)
	}
	if got := g.Gold(); got != upgradeCost {
		t.Fatalf("expected repeat buy not to spend gold, got %d", got)
	}
}

func TestBuyingCannonUpgradeReducesCooldownAndClampsRemainingCooldown(t *testing.T) {
	g := New(Config{
		Width:          40,
		Height:         20,
		CannonCooldown: 3 * time.Second,
		PortUpgrades: map[string]UpgradeKind{
			"Port Royal": UpgradeCannons,
		},
	})
	dockAtPortRoyal(g)
	g.gold = upgradeCost
	g.cannonCooldownRemaining = 3 * time.Second

	paid := g.BuyPortUpgrade()
	if paid != upgradeCost {
		t.Fatalf("expected upgrade to cost %d, paid %d", upgradeCost, paid)
	}
	if got, want := g.cannonCooldown, 2*time.Second; got != want {
		t.Fatalf("expected cannon cooldown %s after upgrade, got %s", want, got)
	}
	if got, want := g.cannonCooldownRemaining, 2*time.Second; got != want {
		t.Fatalf("expected remaining cooldown to clamp to %s, got %s", want, got)
	}
}

func TestBuyingCannonUpgradeDoesNotReduceCooldownBelowMinimum(t *testing.T) {
	g := New(Config{
		Width:          40,
		Height:         20,
		CannonCooldown: 1500 * time.Millisecond,
		PortUpgrades: map[string]UpgradeKind{
			"Port Royal": UpgradeCannons,
		},
	})
	dockAtPortRoyal(g)
	g.gold = upgradeCost

	g.BuyPortUpgrade()
	if got := g.cannonCooldown; got != minimumCannonCooldown {
		t.Fatalf("expected cannon cooldown to clamp at %s, got %s", minimumCannonCooldown, got)
	}
}

func TestBuyingCannonUpgradeDoesNotReduceEnemyCooldown(t *testing.T) {
	g := New(Config{
		Width:           80,
		Height:          40,
		ShotSpeed:       1,
		EnemyAggroRange: 20,
		EnemyShipSpeed:  0.1,
		CannonCooldown:  3 * time.Second,
		PortUpgrades: map[string]UpgradeKind{
			"Port Royal": UpgradeCannons,
		},
	})
	dockAtPortRoyal(g)
	g.gold = upgradeCost
	g.BuyPortUpgrade()
	if got, want := g.cannonCooldown, 2*time.Second; got != want {
		t.Fatalf("expected player cannon cooldown %s after upgrade, got %s", want, got)
	}

	g.ship = Position{X: 30, Y: 20}
	g.heading = HeadingN
	g.enemy.Position = Position{X: 40, Y: 20}
	g.enemy.Heading = HeadingN
	g.shots = nil

	g.Update(time.Millisecond)
	if got := len(g.Shots()); got != 1 {
		t.Fatalf("expected initial enemy shot, got %d", got)
	}

	g.Update(2 * time.Second)
	if got := len(g.Shots()); got != 1 {
		t.Fatalf("expected enemy cooldown to remain at original 3 seconds, got %d shots after 2 seconds", got)
	}

	g.Update(time.Second)
	if got := len(g.Shots()); got != 2 {
		t.Fatalf("expected enemy refire after original 3-second cooldown, got %d shots", got)
	}
}

func TestBuyingAimLinesUpgradeEnablesCurrentLoadAimLines(t *testing.T) {
	g := New(Config{
		Width:        40,
		Height:       20,
		StartingGold: upgradeCost,
		PortUpgrades: map[string]UpgradeKind{
			"Port Royal": UpgradeAimLines,
		},
	})
	if g.AimLinesEnabled() || len(g.AimLines()) != 0 {
		t.Fatal("expected aim lines to start disabled")
	}
	dockAtPortRoyal(g)

	paid := g.BuyPortUpgrade()
	if paid != upgradeCost {
		t.Fatalf("expected upgrade to cost %d, paid %d", upgradeCost, paid)
	}
	if !g.AimLinesEnabled() {
		t.Fatal("expected aim lines upgrade to enable aim lines")
	}

	lines := g.AimLines()
	if len(lines) != 2 {
		t.Fatalf("expected cannonball aim lines for both broadsides, got %d", len(lines))
	}
	for _, line := range lines {
		if line.Load != LoadCannonballs || len(line.Cells) == 0 {
			t.Fatalf("expected non-empty cannonball aim line, got %#v", line)
		}
	}

	g.SelectCannonLoad(LoadGrapeShot)
	lines = g.AimLines()
	if len(lines) != 6 {
		t.Fatalf("expected grape shot aim lines for three pellets on both broadsides, got %d", len(lines))
	}
	for _, line := range lines {
		if line.Load != LoadGrapeShot || len(line.Cells) == 0 {
			t.Fatalf("expected non-empty grape shot aim line, got %#v", line)
		}
	}
}

func TestBuyingCargoUpgradeIncreasesCapacityAndTradeQuantityCap(t *testing.T) {
	g := New(Config{
		Width:  40,
		Height: 20,
		PortUpgrades: map[string]UpgradeKind{
			"Port Royal": UpgradeCargo,
		},
	})
	dockAtPortRoyal(g)
	g.gold = upgradeCost

	paid := g.BuyPortUpgrade()
	if paid != upgradeCost {
		t.Fatalf("expected upgrade to cost %d, paid %d", upgradeCost, paid)
	}
	if got, want := g.CargoCapacity(), defaultCargoCapacity+cargoUpgradeCapacity; got != want {
		t.Fatalf("expected cargo capacity %d after upgrade, got %d", want, got)
	}
	for i := 0; i < 50; i++ {
		g.IncreaseTradeQuantity()
	}
	if got, want := g.TradeQuantity(), defaultCargoCapacity+cargoUpgradeCapacity; got != want {
		t.Fatalf("expected trade quantity cap %d after cargo upgrade, got %d", want, got)
	}
}

func TestBuyingPortUpgradeRequiresPortAndGold(t *testing.T) {
	g := New(Config{
		Width:  40,
		Height: 20,
		PortUpgrades: map[string]UpgradeKind{
			"Port Royal": UpgradeHull,
		},
	})
	g.gold = upgradeCost

	if paid := g.BuyPortUpgrade(); paid != 0 {
		t.Fatalf("expected upgrade purchase away from port to fail, paid %d", paid)
	}
	if got := g.MaxShipHitPoints(); got != shipHitPoints {
		t.Fatalf("expected failed away-from-port upgrade to leave max HP unchanged, got %d", got)
	}

	dockAtPortRoyal(g)
	g.gold = upgradeCost - 1
	if paid := g.BuyPortUpgrade(); paid != 0 {
		t.Fatalf("expected unaffordable upgrade purchase to fail, paid %d", paid)
	}
	if got := g.Gold(); got != upgradeCost-1 {
		t.Fatalf("expected unaffordable upgrade not to spend gold, got %d", got)
	}
	port, ok := g.CurrentPort()
	if !ok || port.UpgradePurchased {
		t.Fatalf("expected unaffordable upgrade to remain available, got %#v ok=%v", port, ok)
	}
}

func TestSelectedTradeGoodAndQuantityControls(t *testing.T) {
	g := New(Config{})

	if g.SelectedTradeGood() != GoodRum || g.TradeQuantity() != 1 {
		t.Fatalf("expected default trade selection rum qty 1, got %v qty %d", g.SelectedTradeGood(), g.TradeQuantity())
	}
	g.SelectTradeGood(GoodTobacco)
	for i := 0; i < 20; i++ {
		g.IncreaseTradeQuantity()
	}
	if g.SelectedTradeGood() != GoodTobacco || g.TradeQuantity() != 10 {
		t.Fatalf("expected tobacco qty capped at 10, got %v qty %d", g.SelectedTradeGood(), g.TradeQuantity())
	}
	for i := 0; i < 20; i++ {
		g.DecreaseTradeQuantity()
	}
	if g.TradeQuantity() != 1 {
		t.Fatalf("expected quantity floor at 1, got %d", g.TradeQuantity())
	}
}

func TestEnemyShipMovesForwardWhenNotEngaging(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, EnemyShipSpeed: 4, EnemyAggroRange: 5})
	g.ship = Position{X: 10, Y: 20}
	g.enemy.Position = Position{X: 40, Y: 20}
	g.enemy.Heading = HeadingN

	g.Update(time.Second)

	enemy, ok := g.Enemy()
	if !ok {
		t.Fatal("expected enemy to remain alive after moving")
	}
	assertPosition(t, enemy.Position, 40, 16)
}

func TestEnemyShipMovesWhileEngagingAndCanRotate(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, EnemyShipSpeed: 10, EnemyAggroRange: 20, TurnInterval: 100 * time.Millisecond})
	g.enemy.Position = Position{X: 40, Y: 20}
	g.enemy.Heading = HeadingN
	g.ship = Position{X: 40, Y: 30}

	g.Update(200 * time.Millisecond)

	enemy, ok := g.Enemy()
	if !ok {
		t.Fatal("expected enemy to remain alive while engaging")
	}
	assertPosition(t, enemy.Position, 42, 20)
	if enemy.Heading != HeadingE {
		t.Fatalf("expected engaging enemy to rotate toward broadside, got %v", enemy.Heading)
	}
}

func TestEnemySteersTowardPlayerWhenEngagedOutsideCannonRange(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, EnemyShipSpeed: 10, EnemyAggroRange: 30, CannonRange: 5, TurnInterval: 100 * time.Millisecond})
	g.enemy.Position = Position{X: 60, Y: 50}
	g.enemy.Heading = HeadingN
	g.ship = Position{X: 40, Y: 50}
	before := g.enemyDistanceToPlayer(g.enemy.Position)

	g.Update(100 * time.Millisecond)

	enemy, ok := g.Enemy()
	if !ok {
		t.Fatal("expected enemy to remain alive while closing distance")
	}
	if enemy.Heading != HeadingNW {
		t.Fatalf("expected enemy to steer toward the player before broadside firing, got %v", enemy.Heading)
	}
	if after := g.enemyDistanceToPlayer(enemy.Position); after >= before {
		t.Fatalf("expected enemy to close distance from %.2f, got %.2f at %#v", before, after, enemy.Position)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected enemy not to fire while outside cannon range, got %d shots", got)
	}
}

func TestPrimaryEnemyDisappearsWhenItReachesMapEdge(t *testing.T) {
	g := New(Config{Width: 40, Height: 20, EnemyShipSpeed: 10, EnemyAggroRange: 1})
	g.ship = Position{X: 20, Y: 15}
	g.enemy.Position = Position{X: 20, Y: 4}
	g.enemy.Heading = HeadingN

	g.Update(100 * time.Millisecond)

	if _, ok := g.Enemy(); ok {
		t.Fatal("expected primary enemy to disappear at map edge")
	}
	if got := g.Gold(); got != defaultGold {
		t.Fatalf("expected edge despawn not to award gold, got %d", got)
	}
}

func TestSpawnedEnemyDisappearsWhenItReachesMapEdge(t *testing.T) {
	g := New(Config{Width: 40, Height: 20, EnemyShipSpeed: 10, EnemyAggroRange: 1})
	g.enemyDestroyed = true
	g.ship = Position{X: 20, Y: 15}
	g.spawnedEnemies = []EnemyShip{{Position: Position{X: 20, Y: 4}, Heading: HeadingN, hitPoints: shipHitPoints}}

	g.Update(100 * time.Millisecond)

	if got := len(g.spawnedEnemies); got != 0 {
		t.Fatalf("expected spawned enemy to disappear at map edge, got %d", got)
	}
	if got := g.Gold(); got != defaultGold {
		t.Fatalf("expected edge despawn not to award gold, got %d", got)
	}
}

func TestEnemyShipsSpawnOccasionallyOffScreenAndOutsideCannonRange(t *testing.T) {
	g := New(Config{
		Width:              200,
		Height:             200,
		CannonRange:        5,
		EnemyAggroRange:    20,
		EnemySpawnInterval: time.Second,
		EnemyDensityCells:  1000,
	})
	g.enemyDestroyed = true
	g.SetViewport(20, 10)
	g.enemySpawnRNG = rand.New(rand.NewSource(1))

	g.Update(time.Second - time.Millisecond)
	if got := len(g.spawnedEnemies); got != 0 {
		t.Fatalf("expected no spawn before interval, got %d", got)
	}

	g.Update(time.Millisecond)
	if got := len(g.spawnedEnemies); got != 1 {
		t.Fatalf("expected one spawned enemy after interval, got %d", got)
	}
	enemy := g.spawnedEnemies[0]
	if g.enemyVisible(enemy) {
		t.Fatalf("expected spawned enemy to be outside visible viewport, got %#v in %#v", enemy, g.visibleBounds())
	}
	if g.enemyWithinCannonRange(enemy) {
		t.Fatalf("expected spawned enemy outside cannon range %.2f, got %#v", g.cannonRange, enemy)
	}
}

func TestEnemySpawnRejectsVisibleAndInRangePositions(t *testing.T) {
	g := New(Config{Width: 100, Height: 100, CannonRange: 5, EnemyAggroRange: 20})
	g.enemyDestroyed = true

	g.SetViewport(20, 10)
	visibleEnemy := EnemyShip{Position: Position{X: g.ship.X + 6, Y: g.ship.Y}, Heading: HeadingN}
	if g.validEnemySpawn(visibleEnemy) {
		t.Fatalf("expected visible enemy spawn to be rejected")
	}

	g.SetViewport(1, 1)
	inRangeEnemy := EnemyShip{Position: Position{X: g.ship.X + 5, Y: g.ship.Y}, Heading: HeadingN}
	if g.validEnemySpawn(inRangeEnemy) {
		t.Fatalf("expected enemy spawn within cannon range to be rejected")
	}
}

func TestEnemySpawnRejectsIslandPositions(t *testing.T) {
	g := New(Config{})
	g.enemyDestroyed = true
	g.SetViewport(1, 1)
	island := g.islands[0]

	candidate := EnemyShip{Position: island.Center, Heading: HeadingN}
	if g.validEnemySpawn(candidate) {
		t.Fatalf("expected enemy spawn on island to be rejected")
	}
}

func TestEnemySpawnRejectsShipsTooCloseToEnemiesOrPorts(t *testing.T) {
	g := New(Config{Width: 120, Height: 80, CannonRange: 5, EnemyAggroRange: 20})
	g.SetViewport(1, 1)

	nearPrimaryEnemy := EnemyShip{Position: Position{X: g.enemy.Position.X + float64(enemySpawnClearance), Y: g.enemy.Position.Y}, Heading: HeadingN}
	if g.validEnemySpawn(nearPrimaryEnemy) {
		t.Fatalf("expected spawn near primary enemy to be rejected")
	}

	g.enemyDestroyed = true
	g.spawnedEnemies = []EnemyShip{{Position: Position{X: 90, Y: 50}, Heading: HeadingN}}
	nearSpawnedEnemy := EnemyShip{Position: Position{X: 90 + float64(enemySpawnClearance), Y: 50}, Heading: HeadingN}
	if g.validEnemySpawn(nearSpawnedEnemy) {
		t.Fatalf("expected spawn near existing spawned enemy to be rejected")
	}

	g.spawnedEnemies = nil
	port := g.ports[0]
	nearPort := EnemyShip{Position: Position{X: port.Position.X + portWidth/2, Y: port.Position.Y + 1}, Heading: HeadingN}
	if g.validEnemySpawn(nearPort) {
		t.Fatalf("expected spawn near port to be rejected")
	}
}

func TestEnemySpawnAllowsDistantShipsAwayFromPorts(t *testing.T) {
	g := New(Config{Width: 160, Height: 120, CannonRange: 5, EnemyAggroRange: 20})
	g.enemyDestroyed = true
	g.ship = Position{X: 10, Y: 10}
	g.SetViewport(1, 1)

	candidate := EnemyShip{Position: Position{X: 80, Y: 100}, Heading: HeadingN}
	if !g.validEnemySpawn(candidate) {
		t.Fatalf("expected distant spawn away from enemies, ports, and islands to be allowed")
	}
}

func TestEnemyShipsDoNotSpawnWhenEntireMapIsVisible(t *testing.T) {
	g := New(Config{
		Width:              40,
		Height:             20,
		EnemySpawnInterval: time.Millisecond,
		EnemyDensityCells:  10,
	})
	g.enemyDestroyed = true
	g.SetViewport(40, 20)
	g.enemySpawnRNG = rand.New(rand.NewSource(1))

	g.Update(time.Second)
	if got := len(g.spawnedEnemies); got != 0 {
		t.Fatalf("expected no spawned enemies when whole map is visible, got %d", got)
	}
}

func TestDefaultEnemyDensityIsReduced(t *testing.T) {
	g := New(Config{})

	if got := g.maxEnemyShips(); got != 14 {
		t.Fatalf("expected default enemy cap to be 14, got %d", got)
	}
}

func TestEnemySpawnRespectsMapDensityLimit(t *testing.T) {
	g := New(Config{
		Width:              100,
		Height:             100,
		CannonRange:        5,
		EnemyAggroRange:    20,
		EnemySpawnInterval: time.Millisecond,
		EnemyDensityCells:  2000,
	})
	g.SetViewport(10, 5)
	g.enemySpawnRNG = rand.New(rand.NewSource(2))

	g.Update(100 * time.Millisecond)
	if got, want := g.enemyCount(), g.maxEnemyShips(); got != want {
		t.Fatalf("expected enemy count to reach density cap %d, got %d", want, got)
	}

	count := g.enemyCount()
	g.Update(100 * time.Millisecond)
	if got := g.enemyCount(); got != count {
		t.Fatalf("expected enemy count to stay capped at %d, got %d", count, got)
	}
}

func TestEnemyStartsAliveAtDefaultPosition(t *testing.T) {
	g := New(Config{Width: 40, Height: 20})

	enemy, ok := g.Enemy()
	if !ok {
		t.Fatal("expected enemy to start alive")
	}
	assertPosition(t, enemy.Position, 29.25, 9.5)
	if enemy.Heading != HeadingN {
		t.Fatalf("expected enemy heading north, got %v", enemy.Heading)
	}
}

func TestPerpendicularShotHitsDiagonalEnemyThroughFilledHitbox(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 10, EnemyAggroRange: 1})
	g.ship = Position{X: 5, Y: 35}
	g.enemy.Position = Position{X: 20, Y: 20}
	g.enemy.Heading = HeadingNE
	g.shots = []Shot{{
		Position: Position{X: 23, Y: 24},
		Heading:  HeadingNW,
		Load:     LoadCannonballs,
		Range:    10,
	}}

	g.Update(300 * time.Millisecond)

	if got, want := g.enemy.hitPoints, shipHitPoints-cannonballDamage; got != want {
		t.Fatalf("expected perpendicular shot to hit diagonal enemy and leave %d hit points, got %d", want, got)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected perpendicular shot to disappear on hit, got %d shots", got)
	}
}

func TestPerpendicularEnemyShotHitsDiagonalPlayerThroughFilledHitbox(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 10, EnemyAggroRange: 1})
	g.enemyDestroyed = true
	g.ship = Position{X: 20, Y: 20}
	g.heading = HeadingNE
	g.shots = []Shot{{
		Position: Position{X: 23, Y: 24},
		Heading:  HeadingNW,
		Load:     LoadCannonballs,
		Owner:    ShotOwnerEnemy,
		Range:    10,
	}}

	g.Update(300 * time.Millisecond)

	if got, want := g.playerHitPoints, shipHitPoints-cannonballDamage; got != want {
		t.Fatalf("expected perpendicular enemy shot to hit diagonal player and leave %d hit points, got %d", want, got)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected perpendicular enemy shot to disappear on hit, got %d shots", got)
	}
}

func TestCannonballHitDamagesEnemyForTwoHitPoints(t *testing.T) {
	g := New(Config{Width: 40, Height: 20, ShotSpeed: 10})
	g.enemy.Position = Position{X: g.Ship().X + 4, Y: g.Ship().Y}

	if !g.FireCannon(CannonRight) {
		t.Fatal("expected cannon to fire")
	}
	g.Update(100 * time.Millisecond)

	if _, ok := g.Enemy(); !ok {
		t.Fatal("expected enemy to survive one cannonball hit")
	}
	if got, want := g.enemy.hitPoints, shipHitPoints-cannonballDamage; got != want {
		t.Fatalf("expected cannonball to leave %d hit points, got %d", want, got)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected cannonball to disappear on hit, got %d shots", got)
	}
}

func TestShipHitCallbackRunsForPrimaryEnemyHits(t *testing.T) {
	hits := 0
	g := New(Config{Width: 40, Height: 20, OnShipHit: func() { hits++ }})

	hitPrimaryEnemy(g, LoadCannonballs)
	if hits != 1 {
		t.Fatalf("expected enemy hit callback once, got %d", hits)
	}

	hitPrimaryEnemy(g, LoadCannonballs)
	hitPrimaryEnemy(g, LoadGrapeShot)
	if hits != 3 {
		t.Fatalf("expected enemy hit callback for each hit including sinking hit, got %d", hits)
	}
}

func TestShipHitCallbackRunsForSpawnedEnemyHits(t *testing.T) {
	hits := 0
	g := New(Config{Width: 80, Height: 40, OnShipHit: func() { hits++ }})
	g.enemyDestroyed = true
	g.spawnedEnemies = []EnemyShip{{Position: Position{X: 20, Y: 20}, Heading: HeadingN, hitPoints: shipHitPoints}}

	g.shots = []Shot{{Position: g.spawnedEnemies[0].Position, Load: LoadCannonballs}}
	g.Update(time.Nanosecond)

	if hits != 1 {
		t.Fatalf("expected spawned enemy hit callback once, got %d", hits)
	}
}

func TestShipHitCallbackRunsForPlayerHits(t *testing.T) {
	hits := 0
	g := New(Config{Width: 80, Height: 40, OnShipHit: func() { hits++ }})

	hitPlayer(g, LoadCannonballs)
	if hits != 1 {
		t.Fatalf("expected player hit callback once, got %d", hits)
	}
}

func TestEnemyIsDestroyedAfterFiveDamage(t *testing.T) {
	sunk := 0
	g := New(Config{Width: 40, Height: 20, OnEnemySunk: func() { sunk++ }})

	hitPrimaryEnemy(g, LoadCannonballs)
	hitPrimaryEnemy(g, LoadCannonballs)
	if _, ok := g.Enemy(); !ok {
		t.Fatal("expected enemy to survive four damage")
	}
	if got := g.enemy.hitPoints; got != 1 {
		t.Fatalf("expected enemy to have 1 hit point after four damage, got %d", got)
	}
	if got := g.Gold(); got != defaultGold {
		t.Fatalf("expected no gold reward before enemy sinks, got %d", got)
	}
	if sunk != 0 {
		t.Fatalf("expected no sink callback before enemy sinks, got %d", sunk)
	}

	hitPrimaryEnemy(g, LoadGrapeShot)
	if _, ok := g.Enemy(); ok {
		t.Fatal("expected fifth damage to destroy enemy")
	}
	if enemySunkReward != 50 {
		t.Fatalf("expected enemy sunk reward to be 50 gold, got %d", enemySunkReward)
	}
	if got := g.Gold(); got != defaultGold+50 {
		t.Fatalf("expected sinking enemy to award 50 gold, got %d", got)
	}
	if sunk != 1 {
		t.Fatalf("expected sink callback once, got %d", sunk)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected final shot to disappear on hit, got %d shots", got)
	}
}

func TestSinkingSpawnedEnemyAwardsGold(t *testing.T) {
	sunk := 0
	g := New(Config{Width: 80, Height: 40, OnEnemySunk: func() { sunk++ }})
	g.enemyDestroyed = true
	g.spawnedEnemies = []EnemyShip{{Position: Position{X: 20, Y: 20}, Heading: HeadingN, hitPoints: grapeShotDamage}}

	g.shots = []Shot{{Position: g.spawnedEnemies[0].Position, Load: LoadGrapeShot}}
	g.Update(time.Nanosecond)

	if got := len(g.spawnedEnemies); got != 0 {
		t.Fatalf("expected spawned enemy to be removed after sinking, got %d spawned enemies", got)
	}
	if got := g.Gold(); got != defaultGold+50 {
		t.Fatalf("expected sinking spawned enemy to award 50 gold, got %d", got)
	}
	if sunk != 1 {
		t.Fatalf("expected sink callback once for spawned enemy, got %d", sunk)
	}
}

func TestGrapeShotHitDamagesEnemyForOneHitPoint(t *testing.T) {
	g := New(Config{Width: 40, Height: 20})

	hitPrimaryEnemy(g, LoadGrapeShot)
	if _, ok := g.Enemy(); !ok {
		t.Fatal("expected enemy to survive one grape shot hit")
	}
	if got, want := g.enemy.hitPoints, shipHitPoints-grapeShotDamage; got != want {
		t.Fatalf("expected grape shot to leave %d hit points, got %d", want, got)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected grape shot to disappear on hit, got %d shots", got)
	}
}

func TestFiveGrapeShotHitsDestroyEnemy(t *testing.T) {
	g := New(Config{Width: 40, Height: 20})

	for i := 0; i < shipHitPoints-1; i++ {
		hitPrimaryEnemy(g, LoadGrapeShot)
		if _, ok := g.Enemy(); !ok {
			t.Fatalf("expected enemy to survive grape shot hit %d", i+1)
		}
	}

	hitPrimaryEnemy(g, LoadGrapeShot)
	if _, ok := g.Enemy(); ok {
		t.Fatal("expected fifth grape shot hit to destroy enemy")
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected fifth grape shot to disappear on hit, got %d shots", got)
	}
}

func TestEnemyFiresWhenPlayerGetsClose(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 1, EnemyAggroRange: 20})
	g.enemy.Position = Position{X: 40, Y: 20}
	g.enemy.Heading = HeadingN
	g.ship = Position{X: 30, Y: 20}

	g.Update(time.Millisecond)

	shots := g.Shots()
	if len(shots) != 1 {
		t.Fatalf("expected enemy to fire one shot, got %d", len(shots))
	}
	if shots[0].Owner != ShotOwnerEnemy || shots[0].Load != LoadCannonballs || shots[0].Heading != HeadingW {
		t.Fatalf("expected enemy cannonball heading west, got %#v", shots[0])
	}
	assertPosition(t, shots[0].Position, 37, 20)
}

func TestEnemyUsesGrapeShotAtCloseRange(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 1, EnemyAggroRange: 24})
	g.enemy.Position = Position{X: 42, Y: 20}
	g.enemy.Heading = HeadingN
	g.ship = Position{X: 35, Y: 20}

	g.Update(time.Millisecond)

	shots := g.Shots()
	if len(shots) != 3 {
		t.Fatalf("expected enemy grape shot spread, got %d shots", len(shots))
	}
	wantHeadings := []Heading{HeadingSW, HeadingW, HeadingNW}
	for i, shot := range shots {
		if shot.Owner != ShotOwnerEnemy || shot.Load != LoadGrapeShot || shot.Heading != wantHeadings[i] {
			t.Fatalf("shot %d: expected enemy grape shot heading %v, got %#v", i, wantHeadings[i], shot)
		}
	}
}

func TestEnemyDoesNotFireOutsideAggroRange(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 1, EnemyAggroRange: 5})
	g.enemy.Position = Position{X: 40, Y: 20}
	g.enemy.Heading = HeadingN
	g.ship = Position{X: 30, Y: 20}

	g.Update(time.Second)

	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected no enemy shots outside aggro range, got %d", got)
	}
}

func TestEnemyDoesNotEngagePlayerInPort(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 1, EnemyAggroRange: 30, TurnInterval: 100 * time.Millisecond})
	dockAtPortRoyal(g)
	if !g.InPort() {
		t.Fatal("expected player to be in port")
	}
	g.enemy.Position = Position{X: 9, Y: 20}
	g.enemy.Heading = HeadingN

	g.Update(300 * time.Millisecond)

	enemy, ok := g.Enemy()
	if !ok {
		t.Fatal("expected enemy to remain on map")
	}
	if enemy.Heading != HeadingN {
		t.Fatalf("expected enemy not to rotate while player is in port, got %v", enemy.Heading)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected enemy not to fire while player is in port, got %d shots", got)
	}
}

func TestEnemyRotatesTowardBroadsideBeforeFiring(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 1, EnemyAggroRange: 20, TurnInterval: 100 * time.Millisecond})
	g.enemy.Position = Position{X: 40, Y: 20}
	g.enemy.Heading = HeadingN
	g.ship = Position{X: 40, Y: 30}

	g.Update(99 * time.Millisecond)
	if enemy, _ := g.Enemy(); enemy.Heading != HeadingN {
		t.Fatalf("expected enemy to wait for turn interval, got heading %v", enemy.Heading)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected no shot before enemy is aligned, got %d", got)
	}

	g.Update(time.Millisecond)
	if enemy, _ := g.Enemy(); enemy.Heading != HeadingNE {
		t.Fatalf("expected enemy to rotate one step toward broadside, got heading %v", enemy.Heading)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected no shot after only one rotation step, got %d", got)
	}

	g.Update(100 * time.Millisecond)
	if enemy, _ := g.Enemy(); enemy.Heading != HeadingE {
		t.Fatalf("expected enemy to finish rotating to east, got heading %v", enemy.Heading)
	}
	shots := g.Shots()
	if len(shots) != 1 {
		t.Fatalf("expected enemy to fire once aligned, got %d shots", len(shots))
	}
	if shots[0].Owner != ShotOwnerEnemy || shots[0].Heading != HeadingS {
		t.Fatalf("expected enemy right broadside shot heading south, got %#v", shots[0])
	}
}

func TestEnemyCannonCooldownPreventsRapidRefire(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 1, EnemyAggroRange: 20, EnemyShipSpeed: 0.1, CannonCooldown: 3 * time.Second})
	g.enemy.Position = Position{X: 40, Y: 20}
	g.enemy.Heading = HeadingN
	g.ship = Position{X: 30, Y: 20}

	g.Update(time.Millisecond)
	if got := len(g.Shots()); got != 1 {
		t.Fatalf("expected initial enemy shot, got %d", got)
	}

	g.Update(2999 * time.Millisecond)
	if got := len(g.Shots()); got != 1 {
		t.Fatalf("expected cooldown to block enemy refire, got %d shots", got)
	}

	g.Update(time.Millisecond)
	if got := len(g.Shots()); got != 2 {
		t.Fatalf("expected enemy refire after cooldown, got %d shots", got)
	}
}

func TestEnemyCannonballHitDamagesPlayerForTwoHitPoints(t *testing.T) {
	g := New(Config{Width: 80, Height: 40, ShotSpeed: 10, EnemyAggroRange: 20})
	g.enemy.Position = Position{X: 40, Y: 20}
	g.enemy.Heading = HeadingN
	g.ship = Position{X: 30, Y: 20}
	g.heading = HeadingN

	g.Update(time.Second)

	if g.GameOver() {
		t.Fatal("expected player to survive one cannonball hit")
	}
	if got, want := g.playerHitPoints, shipHitPoints-cannonballDamage; got != want {
		t.Fatalf("expected cannonball to leave player with %d hit points, got %d", want, got)
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected enemy shot to disappear on hit, got %d shots", got)
	}
}

func TestPlayerIsDestroyedAfterFiveDamage(t *testing.T) {
	g := New(Config{Width: 80, Height: 40})

	hitPlayer(g, LoadCannonballs)
	hitPlayer(g, LoadCannonballs)
	if g.GameOver() {
		t.Fatal("expected player to survive four damage")
	}
	if got := g.playerHitPoints; got != 1 {
		t.Fatalf("expected player to have 1 hit point after four damage, got %d", got)
	}

	hitPlayer(g, LoadGrapeShot)
	if !g.GameOver() {
		t.Fatal("expected fifth damage to destroy player")
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected final enemy shot to disappear on hit, got %d shots", got)
	}
}

func TestFiveEnemyGrapeShotHitsDestroyPlayer(t *testing.T) {
	g := New(Config{Width: 80, Height: 40})

	for i := 0; i < shipHitPoints-1; i++ {
		hitPlayer(g, LoadGrapeShot)
		if g.GameOver() {
			t.Fatalf("expected player to survive grape shot hit %d", i+1)
		}
	}

	hitPlayer(g, LoadGrapeShot)
	if !g.GameOver() {
		t.Fatal("expected fifth grape shot hit to destroy player")
	}
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected fifth grape shot to disappear on hit, got %d shots", got)
	}
}

func TestShotsAreRemovedAtScreenEdge(t *testing.T) {
	g := New(Config{Width: 8, Height: 8, ShotSpeed: 10, CannonRange: 100})
	g.heading = HeadingN

	g.FireCannon(CannonLeft)
	if got := len(g.Shots()); got != 1 {
		t.Fatalf("expected one shot, got %d", got)
	}

	g.Update(time.Second)
	if got := len(g.Shots()); got != 0 {
		t.Fatalf("expected shot to be removed after leaving screen, got %d", got)
	}
}

func dockAtPortRoyal(g *Game) {
	dockAtPortNamed(g, "Port Royal")
}

func dockAtHavana(g *Game) {
	dockAtPortNamed(g, "Havana")
}

func dockAtPortNamed(g *Game, name string) {
	for _, port := range g.ports {
		if port.Name == name {
			dockAtPort(g, port)
			return
		}
	}
}

func dockAtPort(g *Game, port Port) {
	if port.OnIsland {
		g.ship = Position{X: port.Position.X + portWidth/2, Y: port.Position.Y + portHeight + 3}
		g.heading = HeadingN
		return
	}

	switch port.Corner {
	case CornerNW:
		g.ship = Position{X: port.Position.X + portWidth - 1, Y: port.Position.Y + portHeight}
		g.heading = HeadingN
	case CornerNE:
		g.ship = Position{X: port.Position.X, Y: port.Position.Y + portHeight}
		g.heading = HeadingN
	case CornerSE:
		g.ship = Position{X: port.Position.X, Y: port.Position.Y - 1}
		g.heading = HeadingS
	default:
		g.ship = Position{X: port.Position.X + portWidth - 1, Y: port.Position.Y - 1}
		g.heading = HeadingS
	}
}

func leavePorts(g *Game) {
	g.ship = Position{X: float64(g.width) / 2, Y: float64(g.height) / 2}
}

func islandsOverlap(a, b Island) bool {
	left := minInt(int(math.Round(a.Center.X))-a.RadiusX-2, int(math.Round(b.Center.X))-b.RadiusX-2)
	right := maxInt(int(math.Round(a.Center.X))+a.RadiusX+2, int(math.Round(b.Center.X))+b.RadiusX+2)
	top := minInt(int(math.Round(a.Center.Y))-a.RadiusY-2, int(math.Round(b.Center.Y))-b.RadiusY-2)
	bottom := maxInt(int(math.Round(a.Center.Y))+a.RadiusY+2, int(math.Round(b.Center.Y))+b.RadiusY+2)
	for y := top; y <= bottom; y++ {
		for x := left; x <= right; x++ {
			if a.ContainsCell(x, y) && b.ContainsCell(x, y) {
				return true
			}
		}
	}
	return false
}

func portOverlapsIsland(port Port, island Island) bool {
	bounds := portBounds(port, 10000, 10000)
	for y := bounds.top; y <= bounds.bottom; y++ {
		for x := bounds.left; x <= bounds.right; x++ {
			if island.ContainsCell(x, y) {
				return true
			}
		}
	}
	return false
}

func islandExtent(g *Game, x, y, dx, dy int) int {
	extent := 0
	for g.CellIsIsland(x+dx*extent, y+dy*extent) {
		extent++
	}
	return extent
}

func assertRegeneratedNear(t *testing.T, got, previous [goodCount]int) {
	t.Helper()

	for _, good := range Goods() {
		if got[good] == previous[good] {
			t.Fatalf("expected %s price to regenerate from %d", GoodName(good), previous[good])
		}
		maxDelta := maxInt(1, previous[good]/5)
		if delta := absInt(got[good] - previous[good]); delta > maxDelta {
			t.Fatalf("expected %s price %d to stay within %d of previous price %d", GoodName(good), got[good], maxDelta, previous[good])
		}
	}
}

func assertHeading(t *testing.T, g *Game, want Heading) {
	t.Helper()

	if got := g.Heading(); got != want {
		t.Fatalf("expected heading %v, got %v", want, got)
	}
}

func hitPrimaryEnemy(g *Game, load CannonLoad) {
	g.shots = []Shot{{Position: g.enemy.Position, Load: load}}
	g.Update(time.Nanosecond)
}

func hitPlayer(g *Game, load CannonLoad) {
	g.shots = []Shot{{Position: g.ship, Load: load, Owner: ShotOwnerEnemy}}
	g.Update(time.Nanosecond)
}

func assertNoDuplicateValidUpgrades(t *testing.T, ports []Port) {
	t.Helper()
	seen := map[UpgradeKind]string{}
	for _, port := range ports {
		if !validUpgrade(port.Upgrade) {
			continue
		}
		if otherPort := seen[port.Upgrade]; otherPort != "" {
			t.Fatalf("expected unique valid upgrades, both %s and %s offer %v", otherPort, port.Name, port.Upgrade)
		}
		seen[port.Upgrade] = port.Name
	}
}

func assertPosition(t *testing.T, got Position, wantX, wantY float64) {
	t.Helper()

	if math.Abs(got.X-wantX) > 0.000001 || math.Abs(got.Y-wantY) > 0.000001 {
		t.Fatalf("expected position (%.6f, %.6f), got (%.6f, %.6f)", wantX, wantY, got.X, got.Y)
	}
}

func TestMuteToggleUpdatesStateAndCallback(t *testing.T) {
	states := []bool{}
	g := New(Config{OnMuteChange: func(muted bool) {
		states = append(states, muted)
	}})

	if g.Muted() {
		t.Fatal("expected new game to start unmuted")
	}
	if muted := g.ToggleMute(); !muted || !g.Muted() {
		t.Fatalf("expected first toggle to mute, returned %v and state is %v", muted, g.Muted())
	}
	if muted := g.ToggleMute(); muted || g.Muted() {
		t.Fatalf("expected second toggle to unmute, returned %v and state is %v", muted, g.Muted())
	}
	if len(states) != 2 || !states[0] || states[1] {
		t.Fatalf("expected callbacks for mute then unmute, got %#v", states)
	}
}

func TestAddGoldIncreasesGold(t *testing.T) {
	g := New(Config{})

	g.AddGold(1000)
	if got := g.Gold(); got != defaultGold+1000 {
		t.Fatalf("expected gold cheat to add 1000 gold, got %d", got)
	}

	g.AddGold(-100)
	if got := g.Gold(); got != defaultGold+1000 {
		t.Fatalf("expected negative gold addition to do nothing, got %d", got)
	}
}

func TestHighScoreStartsFromConfig(t *testing.T) {
	g := New(Config{HighScore: 250})

	if got := g.HighScore(); got != 250 {
		t.Fatalf("expected configured high score 250, got %d", got)
	}
}

func TestFinalizeScoreSavesCurrentGoldOnceAndUpdatesHighScore(t *testing.T) {
	calls := 0
	g := New(Config{
		HighScore: 90,
		OnScoreFinalized: func(score int) (int, error) {
			calls++
			if score != defaultGold {
				t.Fatalf("expected current gold score %d, got %d", defaultGold, score)
			}
			return score, nil
		},
	})

	if err := g.FinalizeScore(); err != nil {
		t.Fatalf("finalize score: %v", err)
	}
	if !g.ScoreFinalized() {
		t.Fatal("expected score to be marked finalized")
	}
	if got := g.HighScore(); got != defaultGold {
		t.Fatalf("expected high score to update to %d, got %d", defaultGold, got)
	}

	if err := g.FinalizeScore(); err != nil {
		t.Fatalf("second finalize score: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected score finalizer to run once, got %d calls", calls)
	}
}
