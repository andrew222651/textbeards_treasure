package game

import (
	"math"
	"math/rand"
	"time"
)

const (
	defaultWidth              = 240
	defaultHeight             = 144
	defaultShipSpeed          = 18.0
	defaultShotSpeed          = 32.0
	defaultCannonCooldown     = 3 * time.Second
	defaultTurnInterval       = 140 * time.Millisecond
	defaultCannonRange        = 0.9
	defaultEnemySpawnInterval = 12 * time.Second
	defaultEnemyDensityCells  = 2400
	defaultEnemyShipSpeed     = 3.0
	enemySpawnClearance       = 8
	defaultGold               = 100
	shipHitPoints             = 5
	cannonballDamage          = 2
	grapeShotDamage           = 1
	shipRepairFee             = 25
	enemySunkReward           = 50
	defaultCargoCapacity      = 10
	upgradeCost               = 1000
	hullUpgradeHitPoints      = 2
	cargoUpgradeCapacity      = 5
	cooldownUpgradeReduction  = time.Second
	minimumCannonCooldown     = time.Second
	portWidth                 = 10
	portHeight                = 3
	shipMargin                = 3.0
	maxDefaultEnemyRange      = 24.0
)

type Heading int

const (
	HeadingN Heading = iota
	HeadingNE
	HeadingE
	HeadingSE
	HeadingS
	HeadingSW
	HeadingW
	HeadingNW
	headingCount
)

type Control int

const (
	ControlForward Control = iota
	ControlBackward
	ControlTurnLeft
	ControlTurnRight
)

type CannonLoad int

const (
	LoadCannonballs CannonLoad = iota
	LoadGrapeShot
)

type CannonSide int

const (
	CannonLeft CannonSide = iota
	CannonRight
)

type ShotOwner int

const (
	ShotOwnerPlayer ShotOwner = iota
	ShotOwnerEnemy
)

type EnemyShip struct {
	Position       Position
	Heading        Heading
	hitPoints      int
	cannonLoad     CannonLoad
	cannonCooldown time.Duration
	turnElapsed    time.Duration
	moveRemainder  float64
}

type Good int

const (
	GoodRum Good = iota
	GoodSugar
	GoodTobacco
	goodCount
)

type UpgradeKind int

const (
	UpgradeNone UpgradeKind = iota
	UpgradeHull
	UpgradeCannons
	UpgradeCargo
	UpgradeAimLines
)

func UpgradeName(upgrade UpgradeKind) string {
	switch upgrade {
	case UpgradeHull:
		return "Hull +2 HP"
	case UpgradeCannons:
		return "Cannons -1s"
	case UpgradeCargo:
		return "Cargo +5"
	case UpgradeAimLines:
		return "Aim Lines"
	default:
		return "No Upgrade"
	}
}

type Port struct {
	Name             string
	Position         Position
	Prices           [goodCount]int
	Upgrade          UpgradeKind
	UpgradePurchased bool
	Corner           MapCorner
	OnIsland         bool
}

type Island struct {
	Center  Position
	RadiusX int
	RadiusY int
}

func (island Island) ContainsCell(x, y int) bool {
	if island.RadiusX <= 0 || island.RadiusY <= 0 {
		return false
	}
	dx := (float64(x) - island.Center.X) / float64(island.RadiusX)
	dy := (float64(y) - island.Center.Y) / float64(island.RadiusY)
	angle := math.Atan2(dy, dx)
	limit := 1 + 0.12*math.Sin(3*angle+0.7) + 0.08*math.Sin(5*angle-1.1) + 0.05*math.Cos(7*angle+0.3)
	return dx*dx+dy*dy <= limit*limit
}

type MapCorner int

const (
	CornerNW MapCorner = iota
	CornerNE
	CornerSE
	CornerSW
)

type Config struct {
	Width              int
	Height             int
	ShipSpeed          float64
	ShotSpeed          float64
	CannonRange        float64
	CannonCooldown     time.Duration
	TurnInterval       time.Duration
	EnemyAggroRange    float64
	EnemySpawnInterval time.Duration
	EnemyDensityCells  int
	EnemyShipSpeed     float64
	PortPrices         map[Good]int
	TopRightPortPrices map[Good]int
	IslandPortPrices   map[Good]int
	PortUpgrades       map[string]UpgradeKind
	PortCorners        []MapCorner
	StartingGold       int
	HighScore          int
	OnMove             func(Position)
	OnCannonFire       func()
	OnShipHit          func()
	OnEnemySunk        func()
	OnTrade            func()
	OnRepair           func()
	OnPortStateChange  func(bool)
	OnMuteChange       func(bool)
	OnScoreFinalized   func(score int) (highScore int, err error)
}

type Position struct {
	X float64
	Y float64
}

type Shot struct {
	Position  Position
	Heading   Heading
	Load      CannonLoad
	Owner     ShotOwner
	Range     float64
	remainder float64
}

type AimLine struct {
	Cells   []Position
	Heading Heading
	Load    CannonLoad
}

type Game struct {
	width                       int
	height                      int
	shipSpeed                   float64
	shotSpeed                   float64
	cannonRange                 float64
	cannonCooldown              time.Duration
	cannonCooldownRemaining     time.Duration
	turnInterval                time.Duration
	ship                        Position
	heading                     Heading
	controls                    [4]bool
	turnElapsed                 time.Duration
	moveRemainder               float64
	cannonLoad                  CannonLoad
	shots                       []Shot
	enemy                       EnemyShip
	enemyCannonLoad             CannonLoad
	enemyCannonCooldown         time.Duration
	enemyCannonCooldownDuration time.Duration
	enemyTurnElapsed            time.Duration
	enemyAggroRange             float64
	enemyDestroyed              bool
	spawnedEnemies              []EnemyShip
	enemySpawnInterval          time.Duration
	enemySpawnElapsed           time.Duration
	enemyDensityCells           int
	enemyShipSpeed              float64
	enemySpawnRNG               *rand.Rand
	viewportWidth               int
	viewportHeight              int
	playerHitPoints             int
	maxShipHitPoints            int
	gameOver                    bool
	ports                       []Port
	islands                     []Island
	lastVisitedPort             int
	dockedPort                  int
	priceRNG                    *rand.Rand
	gold                        int
	highScore                   int
	scoreFinalized              bool
	inventory                   [goodCount]int
	cargoCapacity               int
	aimLinesEnabled             bool
	selectedTradeGood           Good
	tradeQuantity               int
	onMove                      func(Position)
	onCannonFire                func()
	onShipHit                   func()
	onEnemySunk                 func()
	muted                       bool
	onTrade                     func()
	onRepair                    func()
	onPortStateChange           func(bool)
	onMuteChange                func(bool)
	onScoreFinalized            func(score int) (highScore int, err error)
}

func New(config Config) *Game {
	if config.Width <= 0 {
		config.Width = defaultWidth
	}
	if config.Height <= 0 {
		config.Height = defaultHeight
	}
	if config.ShipSpeed <= 0 {
		config.ShipSpeed = defaultShipSpeed
	}
	if config.ShotSpeed <= 0 {
		config.ShotSpeed = defaultShotSpeed
	}
	if config.CannonCooldown <= 0 {
		config.CannonCooldown = defaultCannonCooldown
	}
	if config.TurnInterval <= 0 {
		config.TurnInterval = defaultTurnInterval
	}
	if config.EnemySpawnInterval <= 0 {
		config.EnemySpawnInterval = defaultEnemySpawnInterval
	}
	if config.EnemyDensityCells <= 0 {
		config.EnemyDensityCells = defaultEnemyDensityCells
	}
	if config.EnemyShipSpeed <= 0 {
		config.EnemyShipSpeed = defaultEnemyShipSpeed
	}
	if config.EnemyAggroRange <= 0 {
		config.EnemyAggroRange = defaultEnemyAggroRange(config.Width, config.Height)
	}
	if config.CannonRange <= 0 {
		config.CannonRange = defaultShotRange(config.EnemyAggroRange)
	}
	if config.StartingGold <= 0 {
		config.StartingGold = defaultGold
	}
	config.CannonRange = capShotRange(config.CannonRange, config.EnemyAggroRange)
	priceRNG := rand.New(rand.NewSource(time.Now().UnixNano()))
	enemySpawnRNG := rand.New(rand.NewSource(time.Now().UnixNano() + 1))
	portRoyalPrices := randomPortPrices(priceRNG)
	applyPortPrices(portRoyalPrices[:], config.PortPrices)
	havanaPrices := randomPortPrices(priceRNG)
	applyPortPrices(havanaPrices[:], config.TopRightPortPrices)
	tortugaPrices := randomPortPrices(priceRNG)
	applyPortPrices(tortugaPrices[:], config.IslandPortPrices)
	islands := defaultIslands(config.Width, config.Height, priceRNG)
	portNames := []string{"Port Royal", "Havana"}
	if len(islands) > 0 {
		portNames = append(portNames, "Tortuga")
	}
	portUpgrades := configuredPortUpgrades(portNames, config.PortUpgrades, priceRNG)
	corners := configuredPortCorners(config.PortCorners, priceRNG)
	ship := defaultPlayerPosition(config.Width, config.Height, islands)

	g := &Game{
		width:                       config.Width,
		height:                      config.Height,
		shipSpeed:                   config.ShipSpeed,
		shotSpeed:                   config.ShotSpeed,
		cannonRange:                 config.CannonRange,
		cannonCooldown:              config.CannonCooldown,
		turnInterval:                config.TurnInterval,
		ship:                        ship,
		heading:                     HeadingN,
		cannonLoad:                  LoadCannonballs,
		enemyCannonLoad:             LoadCannonballs,
		enemyCannonCooldownDuration: config.CannonCooldown,
		enemyAggroRange:             config.EnemyAggroRange,
		enemySpawnInterval:          config.EnemySpawnInterval,
		enemyDensityCells:           config.EnemyDensityCells,
		enemyShipSpeed:              config.EnemyShipSpeed,
		enemySpawnRNG:               enemySpawnRNG,
		enemy: EnemyShip{
			Position:  defaultEnemyPosition(config.Width, config.Height),
			Heading:   HeadingN,
			hitPoints: shipHitPoints,
		},
		lastVisitedPort:  -1,
		dockedPort:       -1,
		priceRNG:         priceRNG,
		playerHitPoints:  shipHitPoints,
		maxShipHitPoints: shipHitPoints,
		ports: []Port{
			{
				Name:     "Port Royal",
				Position: cornerPortPosition(corners[0], config.Width, config.Height),
				Prices:   portRoyalPrices,
				Upgrade:  portUpgrades[0],
				Corner:   corners[0],
			},
			{
				Name:     "Havana",
				Position: cornerPortPosition(corners[1], config.Width, config.Height),
				Prices:   havanaPrices,
				Upgrade:  portUpgrades[1],
				Corner:   corners[1],
			},
		},
		islands:           islands,
		gold:              config.StartingGold,
		highScore:         maxInt(0, config.HighScore),
		cargoCapacity:     defaultCargoCapacity,
		selectedTradeGood: GoodRum,
		tradeQuantity:     1,
		onMove:            config.OnMove,
		onCannonFire:      config.OnCannonFire,
		onShipHit:         config.OnShipHit,
		onEnemySunk:       config.OnEnemySunk,
		onTrade:           config.OnTrade,
		onRepair:          config.OnRepair,
		onPortStateChange: config.OnPortStateChange,
		onMuteChange:      config.OnMuteChange,
		onScoreFinalized:  config.OnScoreFinalized,
	}
	if len(islands) > 0 {
		g.ports = append(g.ports, Port{
			Name:     "Tortuga",
			Position: islandPortPosition(islands[0], config.Width, config.Height),
			Prices:   tortugaPrices,
			Upgrade:  portUpgrades[2],
			OnIsland: true,
		})
	}
	g.clampShip()
	return g
}

func (g *Game) PressControl(control Control) {
	if g.gameOver {
		return
	}
	if !validControl(control) {
		return
	}
	if g.controls[control] {
		return
	}

	g.controls[control] = true
	switch control {
	case ControlTurnLeft:
		g.rotate(-1)
		g.turnElapsed = 0
	case ControlTurnRight:
		g.rotate(1)
		g.turnElapsed = 0
	}
}

func (g *Game) ReleaseControl(control Control) {
	if !validControl(control) {
		return
	}
	g.controls[control] = false
	if !g.controls[ControlTurnLeft] && !g.controls[ControlTurnRight] {
		g.turnElapsed = 0
	}
	if !g.controls[ControlForward] && !g.controls[ControlBackward] {
		g.moveRemainder = 0
	}
}

func (g *Game) IsControlPressed(control Control) bool {
	if !validControl(control) {
		return false
	}
	return g.controls[control]
}

func (g *Game) NudgeControl(control Control, dt time.Duration) {
	if g.gameOver {
		return
	}
	switch control {
	case ControlForward:
		g.nudgeAlongHeading(1, dt)
	case ControlBackward:
		g.nudgeAlongHeading(-1, dt)
	case ControlTurnLeft:
		g.rotate(-1)
	case ControlTurnRight:
		g.rotate(1)
	}
}

func (g *Game) SelectCannonLoad(load CannonLoad) {
	if !validCannonLoad(load) {
		return
	}
	g.cannonLoad = load
}

func (g *Game) CannonLoad() CannonLoad {
	return g.cannonLoad
}

func (g *Game) FireCannon(side CannonSide) bool {
	if g.gameOver || !validCannonSide(side) || g.cannonCooldownRemaining > 0 {
		return false
	}

	g.fireCannonFrom(g.ship, g.heading, g.cannonLoad, side, ShotOwnerPlayer)
	g.cannonCooldownRemaining = g.cannonCooldown
	return true
}

func (g *Game) Shots() []Shot {
	shots := make([]Shot, len(g.shots))
	copy(shots, g.shots)
	return shots
}

func (g *Game) AimLines() []AimLine {
	if !g.aimLinesEnabled || g.cannonRange <= 0 {
		return nil
	}

	var lines []AimLine
	for _, side := range []CannonSide{CannonLeft, CannonRight} {
		lines = append(lines, g.aimLinesForSide(side)...)
	}
	return lines
}

func (g *Game) AimLinesEnabled() bool {
	return g.aimLinesEnabled
}

func (g *Game) aimLinesForSide(side CannonSide) []AimLine {
	shotHeading := cannonHeading(g.heading, side)
	base := cannonMouthPosition(g.ship, shotHeading)
	switch g.cannonLoad {
	case LoadCannonballs:
		return []AimLine{g.aimLineFrom(base, shotHeading, LoadCannonballs)}
	case LoadGrapeShot:
		return []AimLine{
			g.aimLineFrom(spreadShotPosition(base, rotatedHeading(shotHeading, -1)), rotatedHeading(shotHeading, -1), LoadGrapeShot),
			g.aimLineFrom(spreadShotPosition(base, shotHeading), shotHeading, LoadGrapeShot),
			g.aimLineFrom(spreadShotPosition(base, rotatedHeading(shotHeading, 1)), rotatedHeading(shotHeading, 1), LoadGrapeShot),
		}
	default:
		return nil
	}
}

func (g *Game) aimLineFrom(position Position, heading Heading, load CannonLoad) AimLine {
	dx, dy := headingStep(heading)
	line := AimLine{Heading: heading, Load: load}
	for remaining := g.cannonRange; remaining > 0; remaining-- {
		if !g.inBounds(position) || g.positionInIsland(position) {
			break
		}
		line.Cells = append(line.Cells, position)
		position.X += float64(dx)
		position.Y += float64(dy)
	}
	return line
}

func (g *Game) Enemy() (EnemyShip, bool) {
	if !g.enemyDestroyed {
		return g.enemy, true
	}
	if len(g.spawnedEnemies) > 0 {
		return g.spawnedEnemies[0], true
	}
	return EnemyShip{}, false
}

func (g *Game) Enemies() []EnemyShip {
	enemies := make([]EnemyShip, 0, g.enemyCount())
	if !g.enemyDestroyed {
		enemies = append(enemies, g.enemy)
	}
	enemies = append(enemies, g.spawnedEnemies...)
	return enemies
}

func (g *Game) SetViewport(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	g.viewportWidth = width
	g.viewportHeight = height
}

func (g *Game) GameOver() bool {
	return g.gameOver
}

func (g *Game) ToggleMute() bool {
	g.muted = !g.muted
	if g.onMuteChange != nil {
		g.onMuteChange(g.muted)
	}
	return g.muted
}

func (g *Game) Muted() bool {
	return g.muted
}

func (g *Game) PlayerHitPoints() int {
	return maxInt(0, g.playerHitPoints)
}

func (g *Game) MaxShipHitPoints() int {
	return g.maxShipHitPoints
}

func (g *Game) RepairFee() int {
	return shipRepairFee
}

func (g *Game) Port() Port {
	if len(g.ports) == 0 {
		return Port{}
	}
	return g.ports[0]
}

func (g *Game) Ports() []Port {
	ports := make([]Port, len(g.ports))
	copy(ports, g.ports)
	return ports
}

func (g *Game) Islands() []Island {
	islands := make([]Island, len(g.islands))
	copy(islands, g.islands)
	return islands
}

func (g *Game) CellIsIsland(x, y int) bool {
	for _, island := range g.islands {
		if island.ContainsCell(x, y) {
			return true
		}
	}
	return false
}

func (g *Game) CurrentPort() (Port, bool) {
	index := g.currentPortIndex()
	if index < 0 {
		return Port{}, false
	}
	return g.ports[index], true
}

func (g *Game) InPort() bool {
	return g.currentPortIndex() >= 0
}

func (g *Game) currentPortIndex() int {
	for i, port := range g.ports {
		if g.shipTouchesPort(port) {
			return i
		}
	}
	return -1
}

func (g *Game) shipTouchesPort(port Port) bool {
	bounds := portBounds(port, g.width, g.height)
	for _, cell := range shipFootprint(g.ship, g.heading) {
		if bounds.touches(cell) {
			return true
		}
	}
	return false
}

func (g *Game) Gold() int {
	return g.gold
}

func (g *Game) AddGold(amount int) {
	if g.gameOver || amount <= 0 {
		return
	}
	g.gold += amount
}

func (g *Game) HighScore() int {
	return maxInt(0, g.highScore)
}

func (g *Game) SetHighScore(score int) {
	g.highScore = maxInt(0, score)
}

func (g *Game) FinalizeScore() error {
	if g.scoreFinalized {
		return nil
	}

	if g.onScoreFinalized != nil {
		highScore, err := g.onScoreFinalized(g.Gold())
		if err != nil {
			return err
		}
		g.SetHighScore(highScore)
	} else if g.gold > g.highScore {
		g.highScore = g.gold
	}
	g.scoreFinalized = true
	return nil
}

func (g *Game) ScoreFinalized() bool {
	return g.scoreFinalized
}

func (g *Game) CargoCapacity() int {
	return g.cargoCapacity
}

func (g *Game) UpgradeCost() int {
	return upgradeCost
}

func (g *Game) CargoUsed() int {
	total := 0
	for _, count := range g.inventory {
		total += count
	}
	return total
}

func (g *Game) Inventory() [goodCount]int {
	return g.inventory
}

func (g *Game) InventoryFor(good Good) int {
	if !validGood(good) {
		return 0
	}
	return g.inventory[good]
}

func (g *Game) Price(good Good) int {
	if !validGood(good) {
		return 0
	}
	if port, ok := g.CurrentPort(); ok {
		return port.Prices[good]
	}
	return g.Port().Prices[good]
}

func (g *Game) SelectedTradeGood() Good {
	return g.selectedTradeGood
}

func (g *Game) SelectTradeGood(good Good) {
	if validGood(good) {
		g.selectedTradeGood = good
	}
}

func (g *Game) TradeQuantity() int {
	return g.tradeQuantity
}

func (g *Game) IncreaseTradeQuantity() {
	if g.tradeQuantity < g.cargoCapacity {
		g.tradeQuantity++
	}
}

func (g *Game) DecreaseTradeQuantity() {
	if g.tradeQuantity > 1 {
		g.tradeQuantity--
	}
}

func (g *Game) BuySelected() int {
	return g.Buy(g.selectedTradeGood, g.tradeQuantity)
}

func (g *Game) SellSelected() int {
	return g.Sell(g.selectedTradeGood, g.tradeQuantity)
}

func (g *Game) RepairShip() int {
	if g.gameOver || !g.InPort() || g.playerHitPoints >= g.maxShipHitPoints || g.gold < shipRepairFee {
		return 0
	}

	g.gold -= shipRepairFee
	g.playerHitPoints = g.maxShipHitPoints
	if g.onRepair != nil {
		g.onRepair()
	}
	return shipRepairFee
}

func (g *Game) BuyPortUpgrade() int {
	portIndex := g.currentPortIndex()
	if g.gameOver || portIndex < 0 || g.gold < upgradeCost {
		return 0
	}

	port := &g.ports[portIndex]
	if port.UpgradePurchased || !validUpgrade(port.Upgrade) {
		return 0
	}

	g.gold -= upgradeCost
	g.applyUpgrade(port.Upgrade)
	port.UpgradePurchased = true
	return upgradeCost
}

func (g *Game) applyUpgrade(upgrade UpgradeKind) {
	switch upgrade {
	case UpgradeHull:
		g.maxShipHitPoints += hullUpgradeHitPoints
		g.playerHitPoints += hullUpgradeHitPoints
	case UpgradeCannons:
		g.cannonCooldown -= cooldownUpgradeReduction
		if g.cannonCooldown < minimumCannonCooldown {
			g.cannonCooldown = minimumCannonCooldown
		}
		if g.cannonCooldownRemaining > g.cannonCooldown {
			g.cannonCooldownRemaining = g.cannonCooldown
		}
	case UpgradeCargo:
		g.cargoCapacity += cargoUpgradeCapacity
	case UpgradeAimLines:
		g.aimLinesEnabled = true
	}
}

func (g *Game) Buy(good Good, quantity int) int {
	port, inPort := g.CurrentPort()
	if !inPort || !validGood(good) || quantity <= 0 {
		return 0
	}

	price := port.Prices[good]
	if price <= 0 {
		return 0
	}

	maxByGold := g.gold / price
	maxByCapacity := g.cargoCapacity - g.CargoUsed()
	bought := minInt(quantity, minInt(maxByGold, maxByCapacity))
	if bought <= 0 {
		return 0
	}

	g.gold -= bought * price
	g.inventory[good] += bought
	if g.onTrade != nil {
		g.onTrade()
	}
	return bought
}

func (g *Game) Sell(good Good, quantity int) int {
	port, inPort := g.CurrentPort()
	if !inPort || !validGood(good) || quantity <= 0 {
		return 0
	}

	sold := minInt(quantity, g.inventory[good])
	if sold <= 0 {
		return 0
	}

	g.inventory[good] -= sold
	g.gold += sold * port.Prices[good]
	if g.onTrade != nil {
		g.onTrade()
	}
	return sold
}

func Goods() []Good {
	return []Good{GoodRum, GoodSugar, GoodTobacco}
}

func GoodName(good Good) string {
	switch good {
	case GoodRum:
		return "Rum"
	case GoodSugar:
		return "Sugar"
	case GoodTobacco:
		return "Tobacco"
	default:
		return "Unknown"
	}
}

func (g *Game) Update(dt time.Duration) {
	if dt <= 0 || g.gameOver {
		return
	}

	g.updateTurning(dt)
	g.updateEnemyCannonCooldown(dt)
	g.updateEnemyAI(dt)
	g.updateShots(dt)
	if g.gameOver {
		return
	}
	g.updateEnemyMovement(dt)
	g.updateCannonCooldown(dt)
	if g.gameOver {
		return
	}

	throttle := 0
	if g.controls[ControlForward] {
		throttle++
	}
	if g.controls[ControlBackward] {
		throttle--
	}
	if throttle == 0 {
		g.moveRemainder = 0
		g.updateEnemySpawning(dt)
		return
	}

	g.moveAlongHeading(throttle, dt)
	g.updateEnemySpawning(dt)
}

func (g *Game) SetBounds(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}

	if width == g.width && height == g.height {
		return
	}

	g.ship.X = scaleCoordinate(g.ship.X, g.width, width)
	g.ship.Y = scaleCoordinate(g.ship.Y, g.height, height)
	if !g.enemyDestroyed {
		g.enemy.Position.X = scaleCoordinate(g.enemy.Position.X, g.width, width)
		g.enemy.Position.Y = scaleCoordinate(g.enemy.Position.Y, g.height, height)
	}
	for i := range g.spawnedEnemies {
		g.spawnedEnemies[i].Position.X = scaleCoordinate(g.spawnedEnemies[i].Position.X, g.width, width)
		g.spawnedEnemies[i].Position.Y = scaleCoordinate(g.spawnedEnemies[i].Position.Y, g.height, height)
	}
	g.width = width
	g.height = height
	g.islands = defaultIslands(width, height, g.priceRNG)
	g.positionPorts()
	g.clampShip()
	g.clampEnemy()
	g.removeExpiredShots()
}

func (g *Game) Width() int {
	return g.width
}

func (g *Game) Height() int {
	return g.height
}

func (g *Game) Ship() Position {
	return g.ship
}

func (g *Game) Heading() Heading {
	return g.heading
}

func (g *Game) updateTurning(dt time.Duration) {
	turn := 0
	if g.controls[ControlTurnLeft] {
		turn--
	}
	if g.controls[ControlTurnRight] {
		turn++
	}
	if turn == 0 {
		return
	}

	g.turnElapsed += dt
	for g.turnElapsed >= g.turnInterval {
		g.rotate(turn)
		g.turnElapsed -= g.turnInterval
	}
}

func (g *Game) updateEnemyCannonCooldown(dt time.Duration) {
	decreaseCooldown(&g.enemyCannonCooldown, dt)
	for i := range g.spawnedEnemies {
		decreaseCooldown(&g.spawnedEnemies[i].cannonCooldown, dt)
	}
}

func (g *Game) updateEnemyAI(dt time.Duration) {
	if g.gameOver {
		return
	}
	if !g.enemyDestroyed {
		g.updateEnemyShipAI(&g.enemy, &g.enemyTurnElapsed, &g.enemyCannonCooldown, g.enemyCannonLoad, dt)
	}
	for i := range g.spawnedEnemies {
		load := g.spawnedEnemies[i].cannonLoad
		if !validCannonLoad(load) {
			load = LoadCannonballs
		}
		g.updateEnemyShipAI(&g.spawnedEnemies[i], &g.spawnedEnemies[i].turnElapsed, &g.spawnedEnemies[i].cannonCooldown, load, dt)
	}
}

func decreaseCooldown(remaining *time.Duration, dt time.Duration) {
	if *remaining <= 0 {
		return
	}
	*remaining -= dt
	if *remaining < 0 {
		*remaining = 0
	}
}

func (g *Game) updateEnemyMovement(dt time.Duration) {
	if g.gameOver || g.enemyShipSpeed <= 0 {
		return
	}

	if !g.enemyDestroyed && g.moveEnemyForward(&g.enemy, dt) {
		g.enemyDestroyed = true
	}

	kept := g.spawnedEnemies[:0]
	for i := range g.spawnedEnemies {
		enemy := g.spawnedEnemies[i]
		if g.moveEnemyForward(&enemy, dt) {
			continue
		}
		kept = append(kept, enemy)
	}
	g.spawnedEnemies = kept
}

func (g *Game) moveEnemyForward(enemy *EnemyShip, dt time.Duration) bool {
	if dt <= 0 || enemy.hitPoints <= 0 {
		return false
	}

	enemy.moveRemainder += g.enemyShipSpeed * dt.Seconds()
	steps := int(enemy.moveRemainder)
	if steps == 0 {
		return false
	}
	enemy.moveRemainder -= float64(steps)

	dx, dy := headingStep(enemy.Heading)
	for step := 0; step < steps; step++ {
		next := Position{X: enemy.Position.X + float64(dx), Y: enemy.Position.Y + float64(dy)}
		if g.shipIntersectsIsland(next, enemy.Heading) {
			enemy.Heading = rotatedHeading(enemy.Heading, 1)
			enemy.moveRemainder = 0
			return false
		}
		enemy.Position = next
		if g.shipTouchesMapEdge(enemy.Position, enemy.Heading) {
			return true
		}
	}
	return false
}

func (g *Game) shipTouchesMapEdge(position Position, heading Heading) bool {
	for _, cell := range shipFootprint(position, heading) {
		if cell.x <= 0 || cell.y <= 0 || cell.x >= g.width-1 || cell.y >= g.height-1 {
			return true
		}
	}
	return false
}

func (g *Game) updateEnemyShipAI(enemy *EnemyShip, turnElapsed *time.Duration, cannonCooldown *time.Duration, cannonLoad CannonLoad, dt time.Duration) {
	if !g.enemyCanEngage(enemy.Position) {
		*turnElapsed = 0
		return
	}

	shotHeading := headingToward(enemy.Position, g.ship)
	if g.enemyNeedsToClose(enemy.Position) {
		g.rotateEnemyToward(enemy, turnElapsed, shotHeading, dt)
		return
	}

	targetHeading, side := enemyAimFor(enemy.Heading, shotHeading)
	if !g.rotateEnemyToward(enemy, turnElapsed, targetHeading, dt) {
		return
	}

	if *cannonCooldown > 0 {
		return
	}
	g.fireCannonFrom(enemy.Position, enemy.Heading, g.enemyCannonLoadForTarget(enemy.Position, cannonLoad), side, ShotOwnerEnemy)
	*cannonCooldown = g.enemyCannonCooldownDuration
}

func (g *Game) enemyCanEngage(position Position) bool {
	return !g.InPort() && g.enemyAggroRange > 0 && g.enemyDistanceToPlayer(position) <= g.enemyAggroRange
}

func (g *Game) enemyNeedsToClose(position Position) bool {
	return g.cannonRange > 0 && g.enemyDistanceToPlayer(position) > g.cannonRange
}

func (g *Game) rotateEnemyToward(enemy *EnemyShip, turnElapsed *time.Duration, target Heading, dt time.Duration) bool {
	if enemy.Heading == target {
		*turnElapsed = 0
		return true
	}

	*turnElapsed += dt
	for *turnElapsed >= g.turnInterval && enemy.Heading != target {
		enemy.Heading = rotatedHeading(enemy.Heading, turnDirectionToward(enemy.Heading, target))
		*turnElapsed -= g.turnInterval
	}
	return enemy.Heading == target
}

func (g *Game) enemyCannonLoadForTarget(position Position, baseLoad CannonLoad) CannonLoad {
	grapeRange := g.enemyAggroRange / 3
	if grapeRange < 1 {
		grapeRange = 1
	}
	if g.enemyDistanceToPlayer(position) <= grapeRange {
		return LoadGrapeShot
	}
	return baseLoad
}

func (g *Game) enemyDistanceToPlayer(position Position) float64 {
	dx := g.ship.X - position.X
	dy := g.ship.Y - position.Y
	return math.Hypot(dx, dy)
}

func (g *Game) updateCannonCooldown(dt time.Duration) {
	if g.cannonCooldownRemaining <= 0 {
		return
	}
	g.cannonCooldownRemaining -= dt
	if g.cannonCooldownRemaining < 0 {
		g.cannonCooldownRemaining = 0
	}
}

func (g *Game) moveAlongHeading(throttle int, dt time.Duration) {
	g.moveRemainder += g.movementSpeed(throttle) * dt.Seconds()
	steps := int(g.moveRemainder)
	if steps == 0 {
		return
	}

	g.moveRemainder -= float64(steps)
	g.stepAlongHeading(throttle, steps)
}

func (g *Game) nudgeAlongHeading(throttle int, dt time.Duration) {
	if dt <= 0 {
		return
	}

	steps := int(g.movementSpeed(throttle) * dt.Seconds())
	if steps < 1 {
		steps = 1
	}
	g.stepAlongHeading(throttle, steps)
}

func (g *Game) movementSpeed(throttle int) float64 {
	if throttle < 0 {
		return g.shipSpeed / 2
	}
	return g.shipSpeed
}

func (g *Game) updateEnemySpawning(dt time.Duration) {
	if g.gameOver || g.enemySpawnInterval <= 0 || g.enemyCount() >= g.maxEnemyShips() {
		return
	}

	g.enemySpawnElapsed += dt
	for g.enemySpawnElapsed >= g.enemySpawnInterval && g.enemyCount() < g.maxEnemyShips() {
		g.enemySpawnElapsed -= g.enemySpawnInterval
		g.spawnEnemyShip()
	}
}

func (g *Game) spawnEnemyShip() bool {
	minXf, maxXf := shipBounds(g.width)
	minYf, maxYf := shipBounds(g.height)
	minX, maxX := int(math.Ceil(minXf)), int(math.Floor(maxXf))
	minY, maxY := int(math.Ceil(minYf)), int(math.Floor(maxYf))
	if maxX < minX || maxY < minY {
		return false
	}

	rng := ensureRNG(g.enemySpawnRNG)
	g.enemySpawnRNG = rng
	for attempt := 0; attempt < 200; attempt++ {
		enemy := EnemyShip{
			Position: Position{
				X: float64(minX + rng.Intn(maxX-minX+1)),
				Y: float64(minY + rng.Intn(maxY-minY+1)),
			},
			Heading:    Heading(rng.Intn(int(headingCount))),
			hitPoints:  shipHitPoints,
			cannonLoad: LoadCannonballs,
		}
		if g.validEnemySpawn(enemy) {
			g.spawnedEnemies = append(g.spawnedEnemies, enemy)
			return true
		}
	}
	return false
}

func (g *Game) validEnemySpawn(enemy EnemyShip) bool {
	if !g.inBounds(enemy.Position) {
		return false
	}
	if g.shipIntersectsIsland(enemy.Position, enemy.Heading) {
		return false
	}
	if g.enemyWithinCannonRange(enemy) {
		return false
	}
	if g.enemyVisible(enemy) {
		return false
	}
	if g.enemyTooCloseToEnemies(enemy) {
		return false
	}
	if g.enemyTooCloseToPorts(enemy) {
		return false
	}
	return true
}

func (g *Game) enemyTooCloseToEnemies(enemy EnemyShip) bool {
	if !g.enemyDestroyed && shipsTooClose(enemy.Position, enemy.Heading, g.enemy.Position, g.enemy.Heading, enemySpawnClearance) {
		return true
	}
	for _, existing := range g.spawnedEnemies {
		if shipsTooClose(enemy.Position, enemy.Heading, existing.Position, existing.Heading, enemySpawnClearance) {
			return true
		}
	}
	return false
}

func (g *Game) enemyTooCloseToPorts(enemy EnemyShip) bool {
	for _, port := range g.ports {
		bounds := portBounds(port, g.width, g.height)
		for _, cell := range shipFootprint(enemy.Position, enemy.Heading) {
			if bounds.containsWithin(cell, enemySpawnClearance) {
				return true
			}
		}
	}
	return false
}

func (g *Game) enemyWithinCannonRange(enemy EnemyShip) bool {
	if g.cannonRange <= 0 {
		return false
	}
	for _, cell := range shipFootprint(enemy.Position, enemy.Heading) {
		dx := float64(cell.x) - g.ship.X
		dy := float64(cell.y) - g.ship.Y
		if math.Hypot(dx, dy) <= g.cannonRange {
			return true
		}
	}
	return false
}

func (g *Game) enemyVisible(enemy EnemyShip) bool {
	bounds := g.visibleBounds()
	for _, cell := range shipFootprint(enemy.Position, enemy.Heading) {
		if cell.x >= bounds.left && cell.x <= bounds.right && cell.y >= bounds.top && cell.y <= bounds.bottom {
			return true
		}
	}
	return false
}

func (g *Game) visibleBounds() gridBounds {
	width, height := g.spawnViewportDimensions()
	maxX := maxInt(0, g.width-width)
	maxY := maxInt(0, g.height-height)
	left := clampInt(int(math.Round(g.ship.X))-width/2, 0, maxX)
	top := clampInt(int(math.Round(g.ship.Y))-height/2, 0, maxY)
	return gridBounds{left: left, top: top, right: left + width - 1, bottom: top + height - 1}
}

func (g *Game) spawnViewportDimensions() (int, int) {
	width := g.viewportWidth
	height := g.viewportHeight
	if width <= 0 {
		width = minInt(g.width, 80)
	}
	if height <= 0 {
		height = minInt(g.height, 24)
	}
	width = clampInt(width, 1, maxInt(1, g.width))
	height = clampInt(height, 1, maxInt(1, g.height))
	return width, height
}

func (g *Game) enemyCount() int {
	count := len(g.spawnedEnemies)
	if !g.enemyDestroyed {
		count++
	}
	return count
}

func (g *Game) maxEnemyShips() int {
	if g.enemyDensityCells <= 0 {
		return 1
	}
	return maxInt(1, g.width*g.height/g.enemyDensityCells)
}

func (g *Game) updatePortVisit() {
	current := g.currentPortIndex()
	if current < 0 {
		if g.dockedPort >= 0 && g.onPortStateChange != nil {
			g.onPortStateChange(false)
		}
		g.dockedPort = -1
		return
	}
	if current == g.dockedPort {
		return
	}
	wasAtSea := g.dockedPort < 0
	if g.lastVisitedPort >= 0 && g.lastVisitedPort != current {
		g.regenerateOtherPortPrices(current)
	}
	g.lastVisitedPort = current
	g.dockedPort = current
	if wasAtSea && g.onPortStateChange != nil {
		g.onPortStateChange(true)
	}
}

func (g *Game) regenerateOtherPortPrices(currentPort int) {
	averages := g.averagePortPrices()
	for i := range g.ports {
		if i == currentPort {
			continue
		}
		g.ports[i].Prices = nearbyPortPrices(g.ports[i].Prices, averages, g.priceRNG)
	}
}

func (g *Game) averagePortPrices() [goodCount]int {
	var averages [goodCount]int
	if len(g.ports) == 0 {
		return averages
	}

	for _, good := range Goods() {
		sum := 0
		for _, port := range g.ports {
			sum += port.Prices[good]
		}
		averages[good] = int(math.Round(float64(sum) / float64(len(g.ports))))
	}
	return averages
}

func (g *Game) stepAlongHeading(throttle int, steps int) {
	if steps <= 0 || throttle == 0 {
		return
	}

	dx, dy := headingStep(g.heading)
	if throttle < 0 {
		dx = -dx
		dy = -dy
	}

	before := g.ship
	minX, maxX := shipBounds(g.width)
	minY, maxY := shipBounds(g.height)
	for step := 0; step < steps; step++ {
		next := Position{
			X: clamp(g.ship.X+float64(dx), minX, maxX),
			Y: clamp(g.ship.Y+float64(dy), minY, maxY),
		}
		if next == g.ship || g.shipIntersectsIsland(next, g.heading) {
			g.moveRemainder = 0
			break
		}
		g.ship = next
	}
	g.updatePortVisit()

	if g.ship != before && g.onMove != nil {
		g.onMove(g.ship)
	}
}

func (g *Game) fireCannonFrom(position Position, shipHeading Heading, load CannonLoad, side CannonSide, owner ShotOwner) {
	shotHeading := cannonHeading(shipHeading, side)
	base := cannonMouthPosition(position, shotHeading)
	fired := true
	switch load {
	case LoadCannonballs:
		g.addShot(base, shotHeading, LoadCannonballs, owner)
	case LoadGrapeShot:
		g.addSpreadShot(base, rotatedHeading(shotHeading, -1), owner)
		g.addSpreadShot(base, shotHeading, owner)
		g.addSpreadShot(base, rotatedHeading(shotHeading, 1), owner)
	default:
		fired = false
	}
	if fired && g.onCannonFire != nil {
		g.onCannonFire()
	}
}

func cannonMouthPosition(position Position, shotHeading Heading) Position {
	dx, dy := headingStep(shotHeading)
	return Position{X: position.X + float64(dx*3), Y: position.Y + float64(dy*3)}
}

func cannonHeading(shipHeading Heading, side CannonSide) Heading {
	if side == CannonLeft {
		return rotatedHeading(shipHeading, -2)
	}
	return rotatedHeading(shipHeading, 2)
}

func (g *Game) addSpreadShot(base Position, heading Heading, owner ShotOwner) {
	g.addShot(spreadShotPosition(base, heading), heading, LoadGrapeShot, owner)
}

func spreadShotPosition(base Position, heading Heading) Position {
	dx, dy := headingStep(heading)
	return Position{X: base.X + float64(dx), Y: base.Y + float64(dy)}
}

func (g *Game) addShot(position Position, heading Heading, load CannonLoad, owner ShotOwner) {
	if !g.inBounds(position) || g.positionInIsland(position) {
		return
	}
	g.shots = append(g.shots, Shot{Position: position, Heading: heading, Load: load, Owner: owner, Range: g.cannonRange})
}

func (g *Game) updateShots(dt time.Duration) {
	if len(g.shots) == 0 {
		return
	}

	kept := g.shots[:0]
	for _, shot := range g.shots {
		shot.remainder += g.shotSpeed * dt.Seconds()
		steps := int(shot.remainder)
		if steps > 0 {
			shot.remainder -= float64(steps)
		}

		hit := g.applyShotHit(shot)
		blocked := g.positionInIsland(shot.Position)
		dx, dy := headingStep(shot.Heading)
		for step := 0; step < steps && !hit && !blocked && shot.Range > 0; step++ {
			shot.Position.X += float64(dx)
			shot.Position.Y += float64(dy)
			shot.Range--
			blocked = g.positionInIsland(shot.Position)
			if !blocked {
				hit = g.applyShotHit(shot)
			}
		}

		if hit || blocked || !g.inBounds(shot.Position) || shot.Range <= 0 {
			continue
		}
		kept = append(kept, shot)
	}
	g.shots = kept
}

func (g *Game) applyShotHit(shot Shot) bool {
	if shot.Owner == ShotOwnerEnemy {
		return g.applyPlayerShotHit(shot)
	}
	return g.applyEnemyShotHit(shot)
}

func (g *Game) applyEnemyShotHit(shot Shot) bool {
	if !g.enemyDestroyed && shotIntersectsShip(shot, g.enemy.Position, g.enemy.Heading) {
		g.applyPrimaryEnemyDamage(shot.Load)
		return true
	}

	for i := range g.spawnedEnemies {
		if shotIntersectsShip(shot, g.spawnedEnemies[i].Position, g.spawnedEnemies[i].Heading) {
			g.applySpawnedEnemyDamage(i, shot.Load)
			return true
		}
	}
	return false
}

func (g *Game) applyPrimaryEnemyDamage(load CannonLoad) {
	g.enemy.hitPoints -= shotDamage(load)
	g.notifyShipHit()
	if g.enemy.hitPoints <= 0 {
		g.enemyDestroyed = true
		g.awardEnemySunkReward()
	}
}

func (g *Game) applySpawnedEnemyDamage(index int, load CannonLoad) {
	if index < 0 || index >= len(g.spawnedEnemies) {
		return
	}
	g.spawnedEnemies[index].hitPoints -= shotDamage(load)
	g.notifyShipHit()
	if g.spawnedEnemies[index].hitPoints <= 0 {
		g.removeSpawnedEnemy(index)
		g.awardEnemySunkReward()
	}
}

func (g *Game) notifyShipHit() {
	if g.onShipHit != nil {
		g.onShipHit()
	}
}

func (g *Game) awardEnemySunkReward() {
	g.gold += enemySunkReward
	if g.onEnemySunk != nil {
		g.onEnemySunk()
	}
}

func (g *Game) removeSpawnedEnemy(index int) {
	copy(g.spawnedEnemies[index:], g.spawnedEnemies[index+1:])
	g.spawnedEnemies = g.spawnedEnemies[:len(g.spawnedEnemies)-1]
}

func (g *Game) applyPlayerShotHit(shot Shot) bool {
	if g.gameOver || !shotIntersectsShip(shot, g.ship, g.heading) {
		return false
	}

	g.playerHitPoints -= shotDamage(shot.Load)
	g.notifyShipHit()
	if g.playerHitPoints <= 0 {
		g.gameOver = true
	}
	return true
}

func shotDamage(load CannonLoad) int {
	switch load {
	case LoadCannonballs:
		return cannonballDamage
	case LoadGrapeShot:
		return grapeShotDamage
	default:
		return 0
	}
}

func shotIntersectsShip(shot Shot, position Position, heading Heading) bool {
	shotCell := gridCell{x: int(math.Round(shot.Position.X)), y: int(math.Round(shot.Position.Y))}
	for _, cell := range shipFootprint(position, heading) {
		if cell == shotCell {
			return true
		}
	}
	return shotIntersectsDiagonalShipHitbox(shotCell, position, heading)
}

func shotIntersectsDiagonalShipHitbox(shotCell gridCell, position Position, heading Heading) bool {
	if !headingIsDiagonal(heading) {
		return false
	}

	cx := int(math.Round(position.X))
	cy := int(math.Round(position.Y))
	dx, dy := headingStep(heading)
	leftDX, leftDY := dy, -dx
	rx := shotCell.x - cx
	ry := shotCell.y - cy
	forward2 := absInt(rx*dx + ry*dy)
	lateral2 := absInt(rx*leftDX + ry*leftDY)
	if forward2 > 6 || lateral2 > 4 {
		return false
	}

	switch {
	case forward2 <= 1:
		return lateral2 <= 4
	case forward2 <= 3:
		return lateral2 <= 3
	case forward2 <= 5:
		return lateral2 <= 1
	default:
		return lateral2 == 0
	}
}

func headingIsDiagonal(heading Heading) bool {
	return heading == HeadingNE || heading == HeadingSE || heading == HeadingSW || heading == HeadingNW
}

func (g *Game) removeExpiredShots() {
	kept := g.shots[:0]
	for _, shot := range g.shots {
		if g.inBounds(shot.Position) && !g.positionInIsland(shot.Position) {
			kept = append(kept, shot)
		}
	}
	g.shots = kept
}

func (g *Game) inBounds(position Position) bool {
	return position.X >= 0 && position.X < float64(g.width) && position.Y >= 0 && position.Y < float64(g.height)
}

func (g *Game) rotate(steps int) {
	g.heading = rotatedHeading(g.heading, steps)
}

func rotatedHeading(heading Heading, steps int) Heading {
	next := (int(heading) + steps) % int(headingCount)
	if next < 0 {
		next += int(headingCount)
	}
	return Heading(next)
}

func (g *Game) clampShip() {
	minX, maxX := shipBounds(g.width)
	minY, maxY := shipBounds(g.height)
	g.ship.X = clamp(g.ship.X, minX, maxX)
	g.ship.Y = clamp(g.ship.Y, minY, maxY)
	if g.shipIntersectsIsland(g.ship, g.heading) {
		g.ship = nearestOpenWater(g.ship, g.heading, g.islands, g.width, g.height)
	}
}

func (g *Game) clampEnemy() {
	minX, maxX := shipBounds(g.width)
	minY, maxY := shipBounds(g.height)
	if !g.enemyDestroyed {
		g.enemy.Position.X = clamp(g.enemy.Position.X, minX, maxX)
		g.enemy.Position.Y = clamp(g.enemy.Position.Y, minY, maxY)
	}
	for i := range g.spawnedEnemies {
		g.spawnedEnemies[i].Position.X = clamp(g.spawnedEnemies[i].Position.X, minX, maxX)
		g.spawnedEnemies[i].Position.Y = clamp(g.spawnedEnemies[i].Position.Y, minY, maxY)
	}
}

func defaultPlayerPosition(width, height int, islands []Island) Position {
	minX, maxX := shipBounds(width)
	minY, maxY := shipBounds(height)
	position := Position{X: float64(width-1) / 2, Y: float64(height-1) / 2}
	if len(islands) > 0 {
		island := islands[0]
		position.Y = island.Center.Y + float64(island.RadiusY) + 24
	}
	position.X = clamp(position.X, minX, maxX)
	position.Y = clamp(position.Y, minY, maxY)
	return nearestOpenWater(position, HeadingN, islands, width, height)
}

func defaultEnemyPosition(width, height int) Position {
	x := float64(width-1) * 0.75
	y := float64(height-1) / 2
	minX, maxX := shipBounds(width)
	minY, maxY := shipBounds(height)
	return Position{X: clamp(x, minX, maxX), Y: clamp(y, minY, maxY)}
}

func defaultShotRange(enemyAggroRange float64) float64 {
	if enemyAggroRange <= 0 {
		return 0
	}
	return enemyAggroRange * defaultCannonRange
}

func capShotRange(shotRange, enemyAggroRange float64) float64 {
	if shotRange < 0 {
		return 0
	}
	if enemyAggroRange <= 0 || shotRange < enemyAggroRange {
		return shotRange
	}
	return math.Nextafter(enemyAggroRange, 0)
}

func defaultEnemyAggroRange(width, height int) float64 {
	if width <= 0 || height <= 0 {
		return 0
	}
	smallest := minInt(width, height)
	if smallest < 12 {
		return 0
	}
	return math.Min(maxDefaultEnemyRange, float64(smallest)/5)
}

func headingToward(from Position, to Position) Heading {
	dx := to.X - from.X
	dy := to.Y - from.Y
	if math.Abs(dx) < 0.5 && math.Abs(dy) < 0.5 {
		return HeadingN
	}

	octant := int(math.Round(math.Atan2(dy, dx) / (math.Pi / 4)))
	switch (octant%8 + 8) % 8 {
	case 0:
		return HeadingE
	case 1:
		return HeadingSE
	case 2:
		return HeadingS
	case 3:
		return HeadingSW
	case 4:
		return HeadingW
	case 5:
		return HeadingNW
	case 6:
		return HeadingN
	default:
		return HeadingNE
	}
}

func enemyAimFor(current Heading, shotHeading Heading) (Heading, CannonSide) {
	leftHeading := rotatedHeading(shotHeading, 2)
	rightHeading := rotatedHeading(shotHeading, -2)
	if headingDistance(current, leftHeading) < headingDistance(current, rightHeading) {
		return leftHeading, CannonLeft
	}
	return rightHeading, CannonRight
}

func headingDistance(from Heading, to Heading) int {
	diff := absInt(int(to) - int(from))
	return minInt(diff, int(headingCount)-diff)
}

func turnDirectionToward(from Heading, to Heading) int {
	forward := (int(to) - int(from) + int(headingCount)) % int(headingCount)
	if forward == 0 {
		return 0
	}
	if forward <= int(headingCount)/2 {
		return 1
	}
	return -1
}

func (g *Game) positionPorts() {
	for i := range g.ports {
		if g.ports[i].OnIsland {
			if len(g.islands) > 0 {
				g.ports[i].Position = islandPortPosition(g.islands[0], g.width, g.height)
			}
			continue
		}
		g.ports[i].Position = cornerPortPosition(g.ports[i].Corner, g.width, g.height)
	}
}

func cornerPortPosition(corner MapCorner, width, height int) Position {
	switch corner {
	case CornerNW:
		return Position{X: 0, Y: 0}
	case CornerNE:
		return Position{X: float64(maxInt(0, width-portWidth)), Y: 0}
	case CornerSE:
		return Position{X: float64(maxInt(0, width-portWidth)), Y: float64(maxInt(0, height-portHeight))}
	case CornerSW:
		return Position{X: 0, Y: float64(maxInt(0, height-portHeight))}
	default:
		return Position{X: 0, Y: 0}
	}
}

func islandPortPosition(island Island, width, height int) Position {
	centerX := int(math.Round(island.Center.X))
	centerY := int(math.Round(island.Center.Y))
	southShore := centerY
	for y := centerY; y < height && island.ContainsCell(centerX, y); y++ {
		southShore = y
	}

	x := centerX - portWidth/2
	y := southShore - 1
	x = clampInt(x, 0, maxInt(0, width-portWidth))
	y = clampInt(y, 0, maxInt(0, height-portHeight))
	return Position{X: float64(x), Y: float64(y)}
}

func defaultIslands(width, height int, rng *rand.Rand) []Island {
	if width < 120 || height < 72 {
		return nil
	}

	rng = ensureRNG(rng)
	radiusX := clampInt(width/10, 12, 28)
	radiusY := clampInt(height/10, 8, 18)
	center := Position{X: float64(width-1) / 2, Y: float64(height-1) / 2}
	islands := []Island{{Center: center, RadiusX: radiusX, RadiusY: radiusY}}
	islands = append(islands, randomSmallIsland(width, height, rng, 0.18, 0.35, 0.18, 0.38))
	islands = append(islands, randomSmallIsland(width, height, rng, 0.72, 0.86, 0.68, 0.86))
	return islands
}

func randomSmallIsland(width, height int, rng *rand.Rand, minXRatio, maxXRatio, minYRatio, maxYRatio float64) Island {
	radiusX := clampInt(4+rng.Intn(5), 3, maxInt(3, width/20))
	radiusY := clampInt(3+rng.Intn(4), 2, maxInt(2, height/20))
	minX := clampInt(int(float64(width)*minXRatio), radiusX+shipMarginInt(), maxInt(radiusX+shipMarginInt(), width-radiusX-shipMarginInt()-1))
	maxX := clampInt(int(float64(width)*maxXRatio), minX, maxInt(minX, width-radiusX-shipMarginInt()-1))
	minY := clampInt(int(float64(height)*minYRatio), radiusY+shipMarginInt(), maxInt(radiusY+shipMarginInt(), height-radiusY-shipMarginInt()-1))
	maxY := clampInt(int(float64(height)*maxYRatio), minY, maxInt(minY, height-radiusY-shipMarginInt()-1))
	return Island{
		Center:  Position{X: float64(randomIntInRange(rng, minX, maxX)), Y: float64(randomIntInRange(rng, minY, maxY))},
		RadiusX: radiusX,
		RadiusY: radiusY,
	}
}

func randomIntInRange(rng *rand.Rand, minValue, maxValue int) int {
	if maxValue <= minValue {
		return minValue
	}
	return minValue + rng.Intn(maxValue-minValue+1)
}

func shipMarginInt() int {
	return int(math.Ceil(shipMargin))
}

func (g *Game) shipIntersectsIsland(position Position, heading Heading) bool {
	return shipIntersectsIslands(position, heading, g.islands)
}

func shipIntersectsIslands(position Position, heading Heading, islands []Island) bool {
	for _, cell := range shipFootprint(position, heading) {
		for _, island := range islands {
			if island.ContainsCell(cell.x, cell.y) {
				return true
			}
		}
	}
	return false
}

func (g *Game) positionInIsland(position Position) bool {
	return g.CellIsIsland(int(math.Round(position.X)), int(math.Round(position.Y)))
}

func nearestOpenWater(position Position, heading Heading, islands []Island, width, height int) Position {
	if !shipIntersectsIslands(position, heading, islands) {
		return position
	}

	minX, maxX := shipBounds(width)
	minY, maxY := shipBounds(height)
	for radius := 1; radius < maxInt(width, height); radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if absInt(dx) != radius && absInt(dy) != radius {
					continue
				}
				candidate := Position{
					X: clamp(position.X+float64(dx), minX, maxX),
					Y: clamp(position.Y+float64(dy), minY, maxY),
				}
				if !shipIntersectsIslands(candidate, heading, islands) {
					return candidate
				}
			}
		}
	}
	return position
}

func configuredPortCorners(configured []MapCorner, rng *rand.Rand) [2]MapCorner {
	if len(configured) >= 2 && validMapCorner(configured[0]) && validMapCorner(configured[1]) && configured[0] != configured[1] {
		return [2]MapCorner{configured[0], configured[1]}
	}

	rng = ensureRNG(rng)
	corners := []MapCorner{CornerNW, CornerNE, CornerSE, CornerSW}
	firstIndex := rng.Intn(len(corners))
	first := corners[firstIndex]
	corners = append(corners[:firstIndex], corners[firstIndex+1:]...)
	second := corners[rng.Intn(len(corners))]
	return [2]MapCorner{first, second}
}

func validMapCorner(corner MapCorner) bool {
	return corner == CornerNW || corner == CornerNE || corner == CornerSE || corner == CornerSW
}

func configuredPortUpgrades(portNames []string, configured map[string]UpgradeKind, rng *rand.Rand) []UpgradeKind {
	rng = ensureRNG(rng)
	upgrades := make([]UpgradeKind, len(portNames))
	used := make(map[UpgradeKind]bool)

	for i, portName := range portNames {
		upgrade, ok := configured[portName]
		if !ok || !validUpgrade(upgrade) || used[upgrade] {
			continue
		}
		upgrades[i] = upgrade
		used[upgrade] = true
	}

	for i := range upgrades {
		if validUpgrade(upgrades[i]) {
			continue
		}
		upgrades[i] = randomUnusedPortUpgrade(used, rng)
	}
	return upgrades
}

func randomUnusedPortUpgrade(used map[UpgradeKind]bool, rng *rand.Rand) UpgradeKind {
	choices := make([]UpgradeKind, 0, len(portUpgradeKinds()))
	for _, upgrade := range portUpgradeKinds() {
		if !used[upgrade] {
			choices = append(choices, upgrade)
		}
	}
	if len(choices) == 0 {
		return UpgradeNone
	}
	upgrade := choices[rng.Intn(len(choices))]
	used[upgrade] = true
	return upgrade
}

func portUpgradeKinds() []UpgradeKind {
	return []UpgradeKind{UpgradeHull, UpgradeCannons, UpgradeCargo, UpgradeAimLines}
}

func validUpgrade(upgrade UpgradeKind) bool {
	return upgrade == UpgradeHull || upgrade == UpgradeCannons || upgrade == UpgradeCargo || upgrade == UpgradeAimLines
}

func randomPortPrices(rng *rand.Rand) [goodCount]int {
	rng = ensureRNG(rng)
	return [goodCount]int{
		GoodRum:     8 + rng.Intn(9),
		GoodSugar:   5 + rng.Intn(8),
		GoodTobacco: 12 + rng.Intn(13),
	}
}

func nearbyPortPrices(previous [goodCount]int, averages [goodCount]int, rng *rand.Rand) [goodCount]int {
	rng = ensureRNG(rng)
	next := previous
	for _, good := range Goods() {
		next[good] = nearbyPrice(previous[good], averages[good], rng)
	}
	return next
}

func nearbyPrice(previous int, average int, rng *rand.Rand) int {
	if previous <= 0 {
		return 1
	}
	maxDelta := maxInt(1, previous/5)
	bias := marketBiasDelta(previous, average, maxDelta)
	randomWindow := maxInt(1, maxDelta/2)
	delta := bias + rng.Intn(randomWindow*2+1) - randomWindow
	delta = clampInt(delta, -maxDelta, maxDelta)
	if delta == 0 {
		delta = fallbackPriceDelta(previous, average, rng)
	}
	next := previous + delta
	if next < 1 {
		next = 1
	}
	if next == previous {
		next += fallbackPriceDelta(previous, average, rng)
		if next < 1 {
			next = 1
		}
	}
	return next
}

func marketBiasDelta(previous int, average int, maxDelta int) int {
	distance := average - previous
	if distance == 0 {
		return 0
	}
	magnitude := maxInt(1, absInt(distance)/4)
	magnitude = minInt(magnitude, maxDelta)
	if distance < 0 {
		return -magnitude
	}
	return magnitude
}

func fallbackPriceDelta(previous int, average int, rng *rand.Rand) int {
	if average > previous {
		return 1
	}
	if average < previous {
		return -1
	}
	if rng.Intn(2) == 0 {
		return -1
	}
	return 1
}

func ensureRNG(rng *rand.Rand) *rand.Rand {
	if rng != nil {
		return rng
	}
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func applyPortPrices(prices []int, overrides map[Good]int) {
	for good, price := range overrides {
		if validGood(good) && price > 0 {
			prices[good] = price
		}
	}
}

func shipBounds(size int) (float64, float64) {
	if size <= 1 {
		return 0, 0
	}

	margin := shipMargin
	maxMargin := float64(size-1) / 2
	if margin > maxMargin {
		margin = maxMargin
	}
	return margin, float64(size-1) - margin
}

type gridCell struct {
	x int
	y int
}

type gridBounds struct {
	left   int
	top    int
	right  int
	bottom int
}

func (b gridBounds) touches(cell gridCell) bool {
	return b.containsWithin(cell, 1)
}

func (b gridBounds) containsWithin(cell gridCell, margin int) bool {
	return cell.x >= b.left-margin && cell.x <= b.right+margin && cell.y >= b.top-margin && cell.y <= b.bottom+margin
}

func portBounds(port Port, width, height int) gridBounds {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	left := int(math.Round(port.Position.X))
	top := int(math.Round(port.Position.Y))
	left = clampInt(left, 0, width-1)
	top = clampInt(top, 0, height-1)
	right := minInt(width-1, left+portWidth-1)
	bottom := minInt(height-1, top+portHeight-1)
	return gridBounds{left: left, top: top, right: right, bottom: bottom}
}

func shipFootprint(position Position, heading Heading) []gridCell {
	cx := int(math.Round(position.X))
	cy := int(math.Round(position.Y))
	dx, dy := headingStep(heading)
	leftDX, leftDY := dy, -dx
	cell := func(forward, lateral int) gridCell {
		return gridCell{
			x: cx + forward*dx + lateral*leftDX,
			y: cy + forward*dy + lateral*leftDY,
		}
	}

	cells := make([]gridCell, 0, 15)
	for forward := -3; forward <= 3; forward++ {
		cells = append(cells, cell(forward, 0))
	}
	for forward := -1; forward <= 1; forward++ {
		cells = append(cells, cell(forward, -1), cell(forward, 1))
	}
	cells = append(cells, cell(0, -2), cell(0, 2))
	return cells
}

func shipsOverlap(a Position, aHeading Heading, b Position, bHeading Heading) bool {
	return shipsTooClose(a, aHeading, b, bHeading, 0)
}

func shipsTooClose(a Position, aHeading Heading, b Position, bHeading Heading, margin int) bool {
	for _, aCell := range shipFootprint(a, aHeading) {
		for _, bCell := range shipFootprint(b, bHeading) {
			if absInt(aCell.x-bCell.x) <= margin && absInt(aCell.y-bCell.y) <= margin {
				return true
			}
		}
	}
	return false
}

func scaleCoordinate(value float64, oldSize, newSize int) float64 {
	if oldSize <= 1 || newSize <= 1 {
		return 0
	}
	return value * float64(newSize-1) / float64(oldSize-1)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
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

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func headingStep(heading Heading) (int, int) {
	switch heading {
	case HeadingN:
		return 0, -1
	case HeadingNE:
		return 1, -1
	case HeadingE:
		return 1, 0
	case HeadingSE:
		return 1, 1
	case HeadingS:
		return 0, 1
	case HeadingSW:
		return -1, 1
	case HeadingW:
		return -1, 0
	case HeadingNW:
		return -1, -1
	default:
		return 0, -1
	}
}

func validControl(control Control) bool {
	return control >= ControlForward && control <= ControlTurnRight
}

func validCannonLoad(load CannonLoad) bool {
	return load == LoadCannonballs || load == LoadGrapeShot
}

func validCannonSide(side CannonSide) bool {
	return side == CannonLeft || side == CannonRight
}

func validGood(good Good) bool {
	return good >= GoodRum && good < goodCount
}
