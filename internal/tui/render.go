package tui

import (
	"fmt"
	"math"
	"strings"

	"textbeards_treasure/internal/game"
)

const (
	waterGlyph           byte = 126
	mapEdgeGlyph         byte = 37
	islandGlyph          byte = 43
	sternGlyph           byte = 66
	hullGlyph            byte = 79
	sideGlyph            byte = 61
	enemyBowGlyph        byte = 65
	enemySternGlyph      byte = 90
	enemyHullGlyph       byte = 120
	enemySideGlyph       byte = 45
	smallEnemyBowGlyph   byte = 97
	smallEnemyGunGlyph   byte = 33
	smallEnemyHullGlyph  byte = 111
	smallEnemySternGlyph byte = 122
	cannonballGlyph      byte = 64
	grapeShotGlyph       byte = 42
	aimLineGlyph         byte = 46
	statusRows           int  = 3
	ansiReset                 = "\x1b[0m"
)

type cellStyle int

const (
	styleNone cellStyle = iota
	styleWater
	styleMapEdge
	styleIsland
	stylePort
	stylePlayerShip
	styleEnemyShip
	styleCannonball
	styleGrapeShot
	styleAimLine
	styleStatus
	styleMenu
	styleGameOver
)

type camera struct {
	x int
	y int
}

func Render(g *game.Game, width, height int) string {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}

	cam := cameraFor(g, width, height)
	rows := make([][]byte, height)
	styles := make([][]cellStyle, height)
	for y := range rows {
		rows[y] = []byte(strings.Repeat(" ", width))
		styles[y] = make([]cellStyle, width)
		for x := range rows[y] {
			worldX := cam.x + x
			worldY := cam.y + y
			if worldX < 0 || worldY < 0 || worldX >= g.Width() || worldY >= g.Height() {
				continue
			}
			if (worldX+worldY)%11 == 0 {
				setCell(rows, styles, x, y, waterGlyph, styleWater)
			}
		}
	}

	drawAimLines(rows, styles, cam, g.AimLines())
	drawIslands(rows, styles, cam, g)
	drawMapEdges(rows, styles, cam, g.Width(), g.Height())
	for _, port := range g.Ports() {
		drawPort(rows, styles, cam, port)
	}
	for _, enemy := range g.Enemies() {
		drawEnemyShip(rows, styles, cam, enemy)
	}
	drawShots(rows, styles, cam, g.Shots())
	drawShip(rows, styles, cam, g.Ship(), g.Heading())
	drawStatus(rows, styles, g)
	if g.InPort() {
		drawPortMenu(rows, styles, g)
	}
	if g.GameOver() {
		drawGameOver(rows, styles)
	}

	return renderRows(rows, styles)
}

func cameraFor(g *game.Game, width, height int) camera {
	ship := g.Ship()
	maxX := maxInt(0, g.Width()-width)
	maxY := maxInt(0, g.Height()-height)
	desiredX := int(math.Round(ship.X)) - width/2
	desiredY := int(math.Round(ship.Y)) - height/2
	y := clampInt(desiredY, 0, maxY)
	if height < g.Height() && desiredY <= 0 {
		y = -topMapPadding(height)
	}
	return camera{x: clampInt(desiredX, 0, maxX), y: y}
}

func topMapPadding(height int) int {
	return minInt(statusRows, maxInt(0, height-1))
}

func drawMapEdges(rows [][]byte, styles [][]cellStyle, cam camera, mapWidth, mapHeight int) {
	for y := range rows {
		for x := range rows[y] {
			worldX := cam.x + x
			worldY := cam.y + y
			if worldX < 0 || worldY < 0 || worldX >= mapWidth || worldY >= mapHeight {
				continue
			}
			if worldX == 0 || worldY == 0 || worldX == mapWidth-1 || worldY == mapHeight-1 {
				setCell(rows, styles, x, y, mapEdgeGlyph, styleMapEdge)
			}
		}
	}
}

func drawIslands(rows [][]byte, styles [][]cellStyle, cam camera, g *game.Game) {
	for y := range rows {
		for x := range rows[y] {
			worldX := cam.x + x
			worldY := cam.y + y
			if g.CellIsIsland(worldX, worldY) {
				setCell(rows, styles, x, y, islandGlyph, styleIsland)
			}
		}
	}
}

func drawStatus(rows [][]byte, styles [][]cellStyle, g *game.Game) {
	loadLabel := "Load: 1 Cannonballs"
	if g.CannonLoad() == game.LoadGrapeShot {
		loadLabel = "Load: 2 Grape Shot"
	}
	drawText(rows, styles, 0, 0, fmt.Sprintf("Gold: %d  High: %d  HP: %d/%d", g.Gold(), g.HighScore(), g.PlayerHitPoints(), g.MaxShipHitPoints()), styleStatus)
	drawText(rows, styles, 0, 1, fmt.Sprintf("Cargo: %d/%d  %s", g.CargoUsed(), g.CargoCapacity(), loadLabel), styleStatus)
	drawText(rows, styles, 0, 2, fmt.Sprintf("Rum: %d  Sugar: %d  Tobacco: %d", g.InventoryFor(game.GoodRum), g.InventoryFor(game.GoodSugar), g.InventoryFor(game.GoodTobacco)), styleStatus)
}

func drawPort(rows [][]byte, styles [][]cellStyle, cam camera, port game.Port) {
	x := int(math.Round(port.Position.X)) - cam.x
	y := int(math.Round(port.Position.Y)) - cam.y
	drawText(rows, styles, x, y, "##########", stylePort)
	drawText(rows, styles, x, y+1, "# PIER   #", stylePort)
	drawText(rows, styles, x, y+2, "##########", stylePort)
}

func drawGameOver(rows [][]byte, styles [][]cellStyle) {
	message := "GAME OVER"
	y := len(rows) / 2
	x := 0
	if len(rows) > 0 {
		x = maxInt(0, (len(rows[0])-len(message))/2)
	}
	drawText(rows, styles, x, y, message, styleGameOver)
}

func portMenuTitle(port game.Port) string {
	if port.UpgradePurchased {
		return port.Name + " Market  Upgrade sold"
	}
	if port.Upgrade == game.UpgradeNone {
		return port.Name + " Market  No upgrade"
	}
	return fmt.Sprintf("%s Market  U %s %dg", port.Name, game.UpgradeName(port.Upgrade), port.UpgradePrice)
}

func drawPortMenu(rows [][]byte, styles [][]cellStyle, g *game.Game) {
	port, ok := g.CurrentPort()
	if !ok {
		return
	}
	lines := []string{
		portMenuTitle(port),
		fmt.Sprintf("Gold %d  HP %d/%d  Cargo %d/%d  Qty %d  Repair %dg", g.Gold(), g.PlayerHitPoints(), g.MaxShipHitPoints(), g.CargoUsed(), g.CargoCapacity(), g.TradeQuantity(), g.RepairFee()),
	}
	for i, good := range game.Goods() {
		marker := " "
		if good == g.SelectedTradeGood() {
			marker = ">"
		}
		lines = append(lines, fmt.Sprintf("%s%d %-7s %2dg  Hold %d", marker, i+1, game.GoodName(good), g.Price(good), g.InventoryFor(good)))
	}
	lines = append(lines, "1/2/3 select  [/] qty  B buy  X sell  R repair  U upgrade")

	startY := 3
	if len(rows)-startY < len(lines) {
		startY = maxInt(0, len(rows)-len(lines))
	}
	for i, line := range lines {
		drawText(rows, styles, 0, startY+i, line, styleMenu)
	}
}

func drawAimLines(rows [][]byte, styles [][]cellStyle, cam camera, lines []game.AimLine) {
	for _, line := range lines {
		for _, cell := range line.Cells {
			x := int(math.Round(cell.X)) - cam.x
			y := int(math.Round(cell.Y)) - cam.y
			setCell(rows, styles, x, y, aimLineGlyph, styleAimLine)
		}
	}
}

func drawShots(rows [][]byte, styles [][]cellStyle, cam camera, shots []game.Shot) {
	for _, shot := range shots {
		x := int(math.Round(shot.Position.X)) - cam.x
		y := int(math.Round(shot.Position.Y)) - cam.y
		setCell(rows, styles, x, y, shotGlyph(shot.Load), shotStyle(shot.Load))
	}
}

func shotGlyph(load game.CannonLoad) byte {
	if load == game.LoadGrapeShot {
		return grapeShotGlyph
	}
	return cannonballGlyph
}

func shotStyle(load game.CannonLoad) cellStyle {
	if load == game.LoadGrapeShot {
		return styleGrapeShot
	}
	return styleCannonball
}

type shipGlyphs struct {
	stern byte
	hull  byte
	bow   byte
	side  byte
}

func drawShip(rows [][]byte, styles [][]cellStyle, cam camera, position game.Position, heading game.Heading) {
	drawShipShape(rows, styles, cam, position, heading, stylePlayerShip, shipGlyphs{
		stern: sternGlyph,
		hull:  hullGlyph,
		bow:   bowGlyph(heading),
		side:  sideGlyph,
	})
}

func drawEnemyShip(rows [][]byte, styles [][]cellStyle, cam camera, enemy game.EnemyShip) {
	if enemy.Kind == game.EnemyShipSmall {
		drawSmallEnemyShip(rows, styles, cam, enemy.Position, enemy.Heading)
		return
	}
	drawShipShape(rows, styles, cam, enemy.Position, enemy.Heading, styleEnemyShip, shipGlyphs{
		stern: enemySternGlyph,
		hull:  enemyHullGlyph,
		bow:   enemyBowGlyph,
		side:  enemySideGlyph,
	})
}

func drawSmallEnemyShip(rows [][]byte, styles [][]cellStyle, cam camera, position game.Position, heading game.Heading) {
	cx := int(math.Round(position.X)) - cam.x
	cy := int(math.Round(position.Y)) - cam.y
	dx, dy := headingCellVector(heading)
	leftDX, leftDY := leftCellVector(heading)
	set := func(forward, lateral int, glyph byte) {
		setCell(rows, styles, cx+forward*dx+lateral*leftDX, cy+forward*dy+lateral*leftDY, glyph, styleEnemyShip)
	}

	set(-2, 0, smallEnemySternGlyph)
	set(-1, 0, smallEnemyHullGlyph)
	set(0, -1, enemySideGlyph)
	set(0, 0, smallEnemyHullGlyph)
	set(0, 1, enemySideGlyph)
	set(1, 0, smallEnemyGunGlyph)
	set(2, 0, smallEnemyBowGlyph)
}

func drawShipShape(rows [][]byte, styles [][]cellStyle, cam camera, position game.Position, heading game.Heading, style cellStyle, glyphs shipGlyphs) {
	cx := int(math.Round(position.X)) - cam.x
	cy := int(math.Round(position.Y)) - cam.y
	dx, dy := headingCellVector(heading)
	leftDX, leftDY := leftCellVector(heading)
	set := func(forward, lateral int, glyph byte) {
		setCell(rows, styles, cx+forward*dx+lateral*leftDX, cy+forward*dy+lateral*leftDY, glyph, style)
	}

	for forward := -1; forward <= 1; forward++ {
		set(forward, -1, glyphs.side)
		set(forward, 1, glyphs.side)
	}
	set(0, -2, glyphs.side)
	set(0, 2, glyphs.side)
	for forward := -2; forward <= 2; forward++ {
		set(forward, 0, glyphs.hull)
	}
	set(-3, 0, glyphs.stern)
	set(3, 0, glyphs.bow)
}

func drawText(rows [][]byte, styles [][]cellStyle, x, y int, text string, style cellStyle) {
	if y < 0 || y >= len(rows) || x >= len(rows[y]) {
		return
	}
	if x < 0 {
		text = trimLeft(text, -x)
		x = 0
	}
	if x >= len(rows[y]) || text == "" {
		return
	}
	if len(text) > len(rows[y])-x {
		text = text[:len(rows[y])-x]
	}
	copy(rows[y][x:], text)
	for i := 0; i < len(text); i++ {
		styles[y][x+i] = style
	}
}

func trimLeft(text string, count int) string {
	if count >= len(text) {
		return ""
	}
	return text[count:]
}

func setCell(rows [][]byte, styles [][]cellStyle, x, y int, glyph byte, style cellStyle) {
	if y < 0 || y >= len(rows) || x < 0 || x >= len(rows[y]) {
		return
	}
	rows[y][x] = glyph
	styles[y][x] = style
}

func renderRows(rows [][]byte, styles [][]cellStyle) string {
	var out strings.Builder
	out.WriteString("\x1b[H")
	for y, row := range rows {
		writeStyledRow(&out, row, styles[y])
		if y < len(rows)-1 {
			out.WriteString("\r\n")
		}
	}
	out.WriteString(ansiReset)
	return out.String()
}

func writeStyledRow(out *strings.Builder, row []byte, styles []cellStyle) {
	current := styleNone
	for x, cell := range row {
		style := styles[x]
		if style != current {
			out.WriteString(ansiForStyle(style))
			current = style
		}
		out.WriteByte(cell)
	}
	if current != styleNone {
		out.WriteString(ansiReset)
	}
}

func ansiForStyle(style cellStyle) string {
	switch style {
	case styleWater:
		return "\x1b[34m"
	case styleMapEdge:
		return "\x1b[90m"
	case styleIsland:
		return "\x1b[92m"
	case stylePort:
		return "\x1b[33m"
	case stylePlayerShip:
		return "\x1b[32m"
	case styleEnemyShip:
		return "\x1b[31m"
	case styleCannonball:
		return "\x1b[93m"
	case styleGrapeShot:
		return "\x1b[35m"
	case styleAimLine:
		return "\x1b[37m"
	case styleStatus:
		return "\x1b[36m"
	case styleMenu:
		return "\x1b[96m"
	case styleGameOver:
		return "\x1b[91m"
	default:
		return ansiReset
	}
}

func leftCellVector(heading game.Heading) (int, int) {
	dx, dy := headingCellVector(heading)
	return dy, -dx
}

func headingCellVector(heading game.Heading) (int, int) {
	switch heading {
	case game.HeadingN:
		return 0, -1
	case game.HeadingNE:
		return 1, -1
	case game.HeadingE:
		return 1, 0
	case game.HeadingSE:
		return 1, 1
	case game.HeadingS:
		return 0, 1
	case game.HeadingSW:
		return -1, 1
	case game.HeadingW:
		return -1, 0
	case game.HeadingNW:
		return -1, -1
	default:
		return 0, -1
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func bowGlyph(heading game.Heading) byte {
	switch heading {
	case game.HeadingN:
		return 94
	case game.HeadingNE:
		return 47
	case game.HeadingE:
		return 62
	case game.HeadingSE:
		return 92
	case game.HeadingS:
		return 118
	case game.HeadingSW:
		return 47
	case game.HeadingW:
		return 60
	case game.HeadingNW:
		return 92
	default:
		return 94
	}
}
