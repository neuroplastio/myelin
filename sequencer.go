package myelin

import (
	"slices"
	"time"

	"github.com/neuroplastio/myelin/myelinkey"
)

// The sequencer is functions over Pending, a value: a key, the state and
// the time go in, and the new state and a Result come out. myelin reads no
// clock and starts no goroutine. A program keeps its Pending, and when a
// Result has a Deadline it wakes then in its own loop (a timer in a select,
// Bubble Tea's tea.Tick, a page's setTimeout) and calls Expire.
//
// What happens to the keys a Result drops is the program's: one forwards
// them to the terminal it fronts, another types them into its field, or
// feeds the last one again on its own. Feed never retries a key by itself.

// Feed is one key typed on a path at now.
//
//   - A key a longer binding continues, at least as deep as any binding it
//     finishes, is buffered: the Result's Deadline is now plus the schema's
//     Timeout, and so is the new state's.
//   - A key that finishes a binding, with nothing as deep continuing it,
//     runs it.
//   - A key that does neither drops the keys pending with it, and the state
//     is cleared.
//
// With nothing pending, a key no binding starts with is one map lookup.
func Feed(km *Keymap, st Pending, k myelinkey.Key, p Path, now time.Time) (Pending, Result) {
	if len(st.Seq) == 0 && !km.Starts(k) {
		return Pending{}, Result{Dropped: []myelinkey.Key{k}}
	}
	seq := append(slices.Clone(st.Seq), k)
	m := km.Match(seq, p)
	switch {
	case m.Prefix:
		deadline := now.Add(km.env.schema.Timeout)
		return Pending{Seq: seq, Path: p, Deadline: deadline}, Result{Deadline: deadline}
	case m.Exact != nil:
		return Pending{}, Result{Run: m.Exact}
	}
	return Pending{}, Result{Dropped: seq}
}

// Expire is the pending keys timing out. Before the state's deadline it
// does nothing, so a stale timer is harmless. After it, the binding the
// pending keys finish runs, on the path they were typed on; with none they
// drop. Either way the state is cleared.
func Expire(km *Keymap, st Pending, now time.Time) (Pending, Result) {
	if len(st.Seq) == 0 || now.Before(st.Deadline) {
		return st, Result{}
	}
	if m := km.Match(st.Seq, st.Path); m.Exact != nil {
		return Pending{}, Result{Run: m.Exact}
	}
	return Pending{}, Result{Dropped: st.Seq}
}

// Arm is the leader as if it had been typed on the path at now: a tap, or a
// phone's button where there is no ctrl to press. With no leader it is the
// empty state.
func Arm(km *Keymap, p Path, now time.Time) Pending {
	if !km.hasLeader {
		return Pending{}
	}
	return Pending{Seq: []myelinkey.Key{km.leader}, Path: p, Deadline: now.Add(km.env.schema.Timeout)}
}
