package myelinkey

import "strings"

// physicalNames are the physical keys: the W3C UI Events code values
// (https://www.w3.org/TR/uievents-code/), in snake case, as Ghostty names
// them, section by section.
var physicalNames = []string{
	// Writing system keys.
	"backquote", "backslash", "bracket_left", "bracket_right", "comma",
	"digit_0", "digit_1", "digit_2", "digit_3", "digit_4",
	"digit_5", "digit_6", "digit_7", "digit_8", "digit_9",
	"equal", "intl_backslash", "intl_ro", "intl_yen",
	"key_a", "key_b", "key_c", "key_d", "key_e", "key_f", "key_g", "key_h", "key_i",
	"key_j", "key_k", "key_l", "key_m", "key_n", "key_o", "key_p", "key_q", "key_r",
	"key_s", "key_t", "key_u", "key_v", "key_w", "key_x", "key_y", "key_z",
	"minus", "period", "quote", "semicolon", "slash",
	// Functional keys.
	"alt_left", "alt_right", "backspace", "caps_lock", "context_menu",
	"control_left", "control_right", "enter", "meta_left", "meta_right",
	"shift_left", "shift_right", "space", "tab", "convert", "kana_mode", "non_convert",
	// The control pad.
	"delete", "end", "help", "home", "insert", "page_down", "page_up",
	// The arrow pad.
	"arrow_down", "arrow_left", "arrow_right", "arrow_up",
	// The numpad.
	"num_lock", "numpad_0", "numpad_1", "numpad_2", "numpad_3", "numpad_4",
	"numpad_5", "numpad_6", "numpad_7", "numpad_8", "numpad_9",
	"numpad_add", "numpad_backspace", "numpad_clear", "numpad_clear_entry",
	"numpad_comma", "numpad_decimal", "numpad_divide", "numpad_enter", "numpad_equal",
	"numpad_memory_add", "numpad_memory_clear", "numpad_memory_recall",
	"numpad_memory_store", "numpad_memory_subtract", "numpad_multiply",
	"numpad_paren_left", "numpad_paren_right", "numpad_subtract",
	"numpad_separator", "numpad_up", "numpad_down", "numpad_right", "numpad_left",
	"numpad_begin", "numpad_home", "numpad_end", "numpad_insert", "numpad_delete",
	"numpad_page_up", "numpad_page_down",
	// The function section.
	"escape", "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12",
	"f13", "f14", "f15", "f16", "f17", "f18", "f19", "f20", "f21", "f22", "f23", "f24", "f25",
	"fn", "fn_lock", "print_screen", "scroll_lock", "pause",
	// Media keys.
	"browser_back", "browser_favorites", "browser_forward", "browser_home",
	"browser_refresh", "browser_search", "browser_stop", "eject",
	"launch_app_1", "launch_app_2", "launch_mail", "media_play_pause", "media_select",
	"media_stop", "media_track_next", "media_track_previous", "power", "sleep",
	"audio_volume_down", "audio_volume_mute", "audio_volume_up", "wake_up",
	// Legacy keys.
	"copy", "cut", "paste",
}

// isPhysical is physicalNames as a set.
var isPhysical = func() map[string]bool {
	m := make(map[string]bool, len(physicalNames))
	for _, n := range physicalNames {
		m[n] = true
	}
	return m
}()

// otherNames are names other keymaps give physical keys (VS Code, Zed,
// Helix, older Ghostty), which the default reading takes too.
var otherNames = map[string]string{
	"up": "arrow_up", "down": "arrow_down", "left": "arrow_left", "right": "arrow_right",
	"pageup": "page_up", "pagedown": "page_down", "pgup": "page_up", "pgdn": "page_down",
	"esc": "escape",
}

// physicalOf reads a physical key's name as the default reading takes it:
// its own name in any case ("page_down", "Escape"), a W3C code value
// ("PageDown", "KeyA", "Digit1"), or another keymap's name ("pgdn", "up").
func physicalOf(s string) (string, bool) {
	lower := strings.ToLower(s)
	if isPhysical[lower] {
		return lower, true
	}
	if n, ok := otherNames[lower]; ok {
		return n, true
	}
	// A W3C code value, as Ghostty reads one: each capital or digit after
	// the first starts a word.
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case 'a' <= c && c <= 'z':
			b.WriteByte(c)
		case 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteByte(c | 0x20)
		default:
			return "", false
		}
	}
	if n := b.String(); isPhysical[n] {
		return n, true
	}
	return "", false
}
