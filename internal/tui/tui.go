package tui

import (
	"fmt"
	"io"
	"os"
	"time"

	"pirates/internal/game"
)

const (
	frameInterval       = time.Second / 60
	legacyNudgeDuration = 100 * time.Millisecond
	keyboardModeEnable  = "\x1b[>10u"
	keyboardModeDisable = "\x1b[<u"
	enterScreen         = "\x1b[?1049h\x1b[?25l\x1b[?7l\x1b[2J"
	leaveScreen         = "\x1b[?7h\x1b[?25h\x1b[?1049l"
)

func Run(g *game.Game) error {
	input := int(os.Stdin.Fd())
	output := os.Stdout

	restoreRaw, err := enableRawMode(input)
	if err != nil {
		return fmt.Errorf("enable raw terminal mode: %w", err)
	}
	defer restoreRaw()

	if _, err := io.WriteString(output, enterScreen+keyboardModeEnable); err != nil {
		return err
	}
	defer io.WriteString(output, keyboardModeDisable+leaveScreen)

	parser := InputParser{}
	buf := make([]byte, 256)
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	lastFrame := time.Now()
	for now := range ticker.C {
		for {
			n, err := readAvailable(input, buf)
			if err != nil {
				return err
			}
			if n == 0 {
				break
			}

			for _, event := range parser.Feed(buf[:n]) {
				if handleEvent(g, event) {
					return nil
				}
			}
		}

		width, height := terminalDimensions(input)
		g.SetViewport(width, height)
		g.Update(now.Sub(lastFrame))
		lastFrame = now

		if _, err := io.WriteString(output, Render(g, width, height)); err != nil {
			return err
		}
	}

	return nil
}

func handleEvent(g *game.Game, event Event) bool {
	control := gameControlForKey(event.Control)
	switch event.Type {
	case EventControlPress:
		if event.Legacy {
			if !g.IsControlPressed(control) {
				g.NudgeControl(control, legacyNudgeDuration)
			}
		} else {
			g.PressControl(control)
		}
	case EventControlRepeat:
		return false
	case EventControlRelease:
		g.ReleaseControl(control)
	case EventLoadSelect:
		if g.InPort() {
			g.SelectTradeGood(gameGoodForLoadKey(event.Load))
		} else {
			g.SelectCannonLoad(gameLoadForKey(event.Load))
		}
	case EventFirePress:
		g.FireCannon(gameCannonSideForKey(event.Fire))
	case EventFireRepeat, EventFireRelease:
		return false
	case EventTradeGoodSelect:
		if g.InPort() {
			g.SelectTradeGood(gameGoodForTradeKey(event.TradeGood))
		}
	case EventTradeQuantityIncrease:
		if g.InPort() {
			g.IncreaseTradeQuantity()
		}
	case EventTradeQuantityDecrease:
		if g.InPort() {
			g.DecreaseTradeQuantity()
		}
	case EventTradeBuy:
		if g.InPort() {
			g.BuySelected()
		}
	case EventTradeSell:
		if g.InPort() {
			g.SellSelected()
		}
	case EventRepair:
		if g.InPort() {
			g.RepairShip()
		}
	case EventBuyUpgrade:
		if g.InPort() {
			g.BuyPortUpgrade()
		}
	case EventQuit:
		return true
	}

	return false
}

func gameLoadForKey(key LoadKey) game.CannonLoad {
	switch key {
	case KeyCannonballs:
		return game.LoadCannonballs
	case KeyGrapeShot:
		return game.LoadGrapeShot
	default:
		return game.LoadCannonballs
	}
}

func gameGoodForLoadKey(key LoadKey) game.Good {
	switch key {
	case KeyCannonballs:
		return game.GoodRum
	case KeyGrapeShot:
		return game.GoodSugar
	default:
		return game.GoodRum
	}
}

func gameGoodForTradeKey(key TradeGoodKey) game.Good {
	switch key {
	case KeyTradeRum:
		return game.GoodRum
	case KeyTradeSugar:
		return game.GoodSugar
	case KeyTradeTobacco:
		return game.GoodTobacco
	default:
		return game.GoodRum
	}
}

func gameCannonSideForKey(key FireKey) game.CannonSide {
	switch key {
	case KeyFireLeft:
		return game.CannonLeft
	case KeyFireRight:
		return game.CannonRight
	default:
		return game.CannonLeft
	}
}

func gameControlForKey(key ControlKey) game.Control {
	switch key {
	case KeyForward:
		return game.ControlForward
	case KeyBackward:
		return game.ControlBackward
	case KeyTurnLeft:
		return game.ControlTurnLeft
	case KeyTurnRight:
		return game.ControlTurnRight
	default:
		return game.ControlForward
	}
}
