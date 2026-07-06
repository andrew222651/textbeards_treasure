package tui

import (
	"strconv"
	"strings"
)

type EventType int

const (
	EventControlPress EventType = iota
	EventControlRepeat
	EventControlRelease
	EventLoadSelect
	EventFirePress
	EventFireRepeat
	EventFireRelease
	EventTradeGoodSelect
	EventTradeQuantityIncrease
	EventTradeQuantityDecrease
	EventTradeBuy
	EventTradeSell
	EventRepair
	EventBuyUpgrade
	EventMuteToggle
	EventGoldCheat
	EventQuit
)

type ControlKey int

const (
	KeyForward ControlKey = iota
	KeyBackward
	KeyTurnLeft
	KeyTurnRight
)

type LoadKey int

const (
	KeyCannonballs LoadKey = iota
	KeyGrapeShot
)

type FireKey int

const (
	KeyFireLeft FireKey = iota
	KeyFireRight
)

type TradeGoodKey int

const (
	KeyTradeRum TradeGoodKey = iota
	KeyTradeSugar
	KeyTradeTobacco
)

type Event struct {
	Type      EventType
	Control   ControlKey
	Load      LoadKey
	Fire      FireKey
	TradeGood TradeGoodKey
	Legacy    bool
}

type InputParser struct {
	buffer []byte
}

func (p *InputParser) Feed(data []byte) []Event {
	p.buffer = append(p.buffer, data...)

	var events []Event
	for len(p.buffer) > 0 {
		if p.buffer[0] != 0x1b {
			events = append(events, parseLegacyByte(p.buffer[0])...)
			p.buffer = p.buffer[1:]
			continue
		}

		if len(p.buffer) == 1 {
			break
		}

		if p.buffer[1] != 91 {
			events = append(events, Event{Type: EventQuit})
			p.buffer = p.buffer[1:]
			continue
		}

		end := csiEndIndex(p.buffer)
		if end == -1 {
			break
		}

		if p.buffer[end] == 117 {
			events = append(events, parseCSIU(string(p.buffer[2:end]))...)
		}
		p.buffer = p.buffer[end+1:]
	}

	return events
}

func parseLegacyByte(b byte) []Event {
	if control, ok := controlForByte(b); ok {
		return []Event{{Type: EventControlPress, Control: control, Legacy: true}}
	}
	if load, ok := loadForByte(b); ok {
		return []Event{{Type: EventLoadSelect, Load: load, Legacy: true}}
	}
	if fire, ok := fireForByte(b); ok {
		return []Event{{Type: EventFirePress, Fire: fire, Legacy: true}}
	}
	if tradeGood, ok := tradeGoodForByte(b); ok {
		return []Event{{Type: EventTradeGoodSelect, TradeGood: tradeGood, Legacy: true}}
	}

	switch b {
	case 91:
		return []Event{{Type: EventTradeQuantityDecrease, Legacy: true}}
	case 93:
		return []Event{{Type: EventTradeQuantityIncrease, Legacy: true}}
	case 98, 66:
		return []Event{{Type: EventTradeBuy, Legacy: true}}
	case 120, 88:
		return []Event{{Type: EventTradeSell, Legacy: true}}
	case 114, 82:
		return []Event{{Type: EventRepair, Legacy: true}}
	case 117, 85:
		return []Event{{Type: EventBuyUpgrade, Legacy: true}}
	case 109, 77:
		return []Event{{Type: EventMuteToggle, Legacy: true}}
	case 7:
		return []Event{{Type: EventGoldCheat, Legacy: true}}
	case 3:
		return []Event{{Type: EventQuit}}
	default:
		return nil
	}
}

func parseCSIU(params string) []Event {
	fields := strings.Split(params, ";")
	if len(fields) == 0 {
		return nil
	}

	keyCode, err := strconv.Atoi(firstSubfield(fields[0]))
	if err != nil {
		return nil
	}

	modifiers := 0
	eventType := 1
	if len(fields) > 1 && fields[1] != "" {
		modifierParts := strings.Split(fields[1], ":")
		encodedModifiers, err := strconv.Atoi(modifierParts[0])
		if err == nil && encodedModifiers > 0 {
			modifiers = encodedModifiers - 1
		}
		if len(modifierParts) > 1 {
			parsedEventType, err := strconv.Atoi(modifierParts[1])
			if err == nil {
				eventType = parsedEventType
			}
		}
	}

	if control, ok := controlForKeyCode(keyCode); ok {
		switch eventType {
		case 2:
			return []Event{{Type: EventControlRepeat, Control: control}}
		case 3:
			return []Event{{Type: EventControlRelease, Control: control}}
		default:
			return []Event{{Type: EventControlPress, Control: control}}
		}
	}

	if load, ok := loadForKeyCode(keyCode); ok {
		if eventType == 3 {
			return nil
		}
		return []Event{{Type: EventLoadSelect, Load: load}}
	}

	if fire, ok := fireForKeyCode(keyCode); ok {
		switch eventType {
		case 2:
			return []Event{{Type: EventFireRepeat, Fire: fire}}
		case 3:
			return []Event{{Type: EventFireRelease, Fire: fire}}
		default:
			return []Event{{Type: EventFirePress, Fire: fire}}
		}
	}

	if tradeGood, ok := tradeGoodForKeyCode(keyCode); ok {
		if eventType == 3 {
			return nil
		}
		return []Event{{Type: EventTradeGoodSelect, TradeGood: tradeGood}}
	}

	switch {
	case keyCode == 91 && eventType != 3:
		return []Event{{Type: EventTradeQuantityDecrease}}
	case keyCode == 93 && eventType != 3:
		return []Event{{Type: EventTradeQuantityIncrease}}
	case (keyCode == 98 || keyCode == 66) && eventType != 3:
		return []Event{{Type: EventTradeBuy}}
	case (keyCode == 120 || keyCode == 88) && eventType != 3:
		return []Event{{Type: EventTradeSell}}
	case (keyCode == 114 || keyCode == 82) && eventType != 3:
		return []Event{{Type: EventRepair}}
	case (keyCode == 117 || keyCode == 85) && eventType != 3:
		return []Event{{Type: EventBuyUpgrade}}
	case (keyCode == 109 || keyCode == 77) && eventType == 1:
		return []Event{{Type: EventMuteToggle}}
	case (keyCode == 103 || keyCode == 71) && modifiers&4 != 0 && eventType == 1:
		return []Event{{Type: EventGoldCheat}}
	case keyCode == 27 && eventType != 3:
		return []Event{{Type: EventQuit}}
	case keyCode == 99 && modifiers&4 != 0 && eventType != 3:
		return []Event{{Type: EventQuit}}
	default:
		return nil
	}
}

func controlForByte(b byte) (ControlKey, bool) {
	switch b {
	case 119, 87:
		return KeyForward, true
	case 115, 83:
		return KeyBackward, true
	case 97, 65:
		return KeyTurnLeft, true
	case 100, 68:
		return KeyTurnRight, true
	default:
		return 0, false
	}
}

func loadForByte(b byte) (LoadKey, bool) {
	switch b {
	case 49:
		return KeyCannonballs, true
	case 50:
		return KeyGrapeShot, true
	default:
		return 0, false
	}
}

func fireForByte(b byte) (FireKey, bool) {
	switch b {
	case 113, 81:
		return KeyFireLeft, true
	case 101, 69:
		return KeyFireRight, true
	default:
		return 0, false
	}
}

func tradeGoodForByte(b byte) (TradeGoodKey, bool) {
	switch b {
	case 51:
		return KeyTradeTobacco, true
	default:
		return 0, false
	}
}

func controlForKeyCode(keyCode int) (ControlKey, bool) {
	if keyCode >= 65 && keyCode <= 90 {
		keyCode += 32
	}
	return controlForByte(byte(keyCode))
}

func loadForKeyCode(keyCode int) (LoadKey, bool) {
	return loadForByte(byte(keyCode))
}

func fireForKeyCode(keyCode int) (FireKey, bool) {
	return fireForByte(byte(keyCode))
}

func tradeGoodForKeyCode(keyCode int) (TradeGoodKey, bool) {
	return tradeGoodForByte(byte(keyCode))
}

func firstSubfield(field string) string {
	if before, _, ok := strings.Cut(field, ":"); ok {
		return before
	}
	return field
}

func csiEndIndex(data []byte) int {
	for i := 2; i < len(data); i++ {
		b := data[i]
		if b >= 0x40 && b <= 0x7e {
			return i
		}
	}
	return -1
}
