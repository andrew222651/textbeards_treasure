package tui

import "testing"

func TestInputParserParsesKittyControlPressRepeatAndRelease(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("\x1b[119;1:1u\x1b[119;1:2u\x1b[119;1:3u"))

	assertEvents(t, events, []Event{
		{Type: EventControlPress, Control: KeyForward},
		{Type: EventControlRepeat, Control: KeyForward},
		{Type: EventControlRelease, Control: KeyForward},
	})
}

func TestInputParserParsesAllControlKeys(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("\x1b[119;1:1u\x1b[97;1:1u\x1b[115;1:1u\x1b[100;1:1u"))

	assertEvents(t, events, []Event{
		{Type: EventControlPress, Control: KeyForward},
		{Type: EventControlPress, Control: KeyTurnLeft},
		{Type: EventControlPress, Control: KeyBackward},
		{Type: EventControlPress, Control: KeyTurnRight},
	})
}

func TestInputParserKeepsPartialCSIUSequence(t *testing.T) {
	var parser InputParser

	if events := parser.Feed([]byte("\x1b[119;")); len(events) != 0 {
		t.Fatalf("expected no events for partial CSI-u sequence, got %#v", events)
	}

	events := parser.Feed([]byte("1:1u"))
	assertEvents(t, events, []Event{{Type: EventControlPress, Control: KeyForward}})
}

func TestInputParserParsesLegacyControlKeysAsNudges(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("wAsD"))

	assertEvents(t, events, []Event{
		{Type: EventControlPress, Control: KeyForward, Legacy: true},
		{Type: EventControlPress, Control: KeyTurnLeft, Legacy: true},
		{Type: EventControlPress, Control: KeyBackward, Legacy: true},
		{Type: EventControlPress, Control: KeyTurnRight, Legacy: true},
	})
}

func TestInputParserParsesLoadAndFireKeys(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("\x1b[49;1:1u\x1b[50;1:1u\x1b[113;1:1u\x1b[113;1:2u\x1b[113;1:3u\x1b[101;1:1u"))

	assertEvents(t, events, []Event{
		{Type: EventLoadSelect, Load: KeyCannonballs},
		{Type: EventLoadSelect, Load: KeyGrapeShot},
		{Type: EventFirePress, Fire: KeyFireLeft},
		{Type: EventFireRepeat, Fire: KeyFireLeft},
		{Type: EventFireRelease, Fire: KeyFireLeft},
		{Type: EventFirePress, Fire: KeyFireRight},
	})
}

func TestInputParserParsesLegacyLoadAndFireKeys(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("12qE "))

	assertEvents(t, events, []Event{
		{Type: EventLoadSelect, Load: KeyCannonballs, Legacy: true},
		{Type: EventLoadSelect, Load: KeyGrapeShot, Legacy: true},
		{Type: EventFirePress, Fire: KeyFireLeft, Legacy: true},
		{Type: EventFirePress, Fire: KeyFireRight, Legacy: true},
	})
}

func TestInputParserParsesTradeKeys(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("\x1b[51;1:1u\x1b[93;1:1u\x1b[91;1:1u\x1b[98;1:1u\x1b[120;1:1u\x1b[114;1:1u\x1b[117;1:1u"))

	assertEvents(t, events, []Event{
		{Type: EventTradeGoodSelect, TradeGood: KeyTradeTobacco},
		{Type: EventTradeQuantityIncrease},
		{Type: EventTradeQuantityDecrease},
		{Type: EventTradeBuy},
		{Type: EventTradeSell},
		{Type: EventRepair},
		{Type: EventBuyUpgrade},
	})
}

func TestInputParserParsesLegacyTradeKeys(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("3[]bXrU"))

	assertEvents(t, events, []Event{
		{Type: EventTradeGoodSelect, TradeGood: KeyTradeTobacco, Legacy: true},
		{Type: EventTradeQuantityDecrease, Legacy: true},
		{Type: EventTradeQuantityIncrease, Legacy: true},
		{Type: EventTradeBuy, Legacy: true},
		{Type: EventTradeSell, Legacy: true},
		{Type: EventRepair, Legacy: true},
		{Type: EventBuyUpgrade, Legacy: true},
	})
}

func TestInputParserParsesMuteToggleKey(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("mM\x1b[109;1:1u\x1b[109;1:2u\x1b[109;1:3u\x1b[77;2:1u"))

	assertEvents(t, events, []Event{
		{Type: EventMuteToggle, Legacy: true},
		{Type: EventMuteToggle, Legacy: true},
		{Type: EventMuteToggle},
		{Type: EventMuteToggle},
	})
}

func TestInputParserParsesGoldCheat(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("\x07\x1b[103;5:1u\x1b[103;5:2u\x1b[103;5:3u\x1b[71;5:1u"))

	assertEvents(t, events, []Event{
		{Type: EventGoldCheat, Legacy: true},
		{Type: EventGoldCheat},
		{Type: EventGoldCheat},
	})
}

func TestInputParserIgnoresSpacebarForFiring(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte(" \x1b[32;1:1u\x1b[32;1:2u\x1b[32;1:3u"))
	assertEvents(t, events, nil)
}

func TestInputParserParsesQuitInputs(t *testing.T) {
	var parser InputParser

	events := parser.Feed([]byte("\x03\x1b[27;1:1u\x1b[99;5:1u"))

	assertEvents(t, events, []Event{
		{Type: EventQuit},
		{Type: EventQuit},
		{Type: EventQuit},
	})
}

func assertEvents(t *testing.T, got, want []Event) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d events, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event %d: expected %#v, got %#v", i, want[i], got[i])
		}
	}
}
