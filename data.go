package myelin

import (
	"time"

	"github.com/neuroplastio/myelin/myelinkey"
)

// The data types: exported fields, msgpack tags and msgp's generated codec,
// and no other methods. Each encodes as a msgpack map keyed by its fields'
// tags. What works on them is functions, in the package's other files.

// The generator is pinned in mise.toml, not in go.mod: myelin is imported,
// and a tool directive would put msgp's own dependencies in every importer's
// module graph. Times are msgpack timestamps (extension −1), read back in
// UTC. A key (myelinkey.Key) is a msgpack string: its canonical name.
//
//go:generate msgp -marshal=false -tests=false
//msgp:newtime
//msgp:timezone utc
//msgp:replace myelinkey.Key with:string

// Schema is what a program declares once, for [NewEnv]: what can hold the
// keyboard, and the keymap's layers.
type Schema struct {
	// Nodes are the path's vocabulary: the kinds of node that can hold
	// the keyboard, outermost first.
	Nodes []NodeType `msg:"nodes"`
	// Layers are the keymap's layers, lowest first: ["defaults", "user"],
	// or ["extensions", "defaults", "user"].
	Layers []string `msg:"layers"`
	// Leader is the leader key, written short, when no Doc names one:
	// "ctrl+x", "space". Empty means no leader.
	Leader string `msg:"leader,omitempty"`
	// Timeout is how long a buffered sequence waits for its next key. Half
	// a second suits vim-like sequences; a menu people read wants seconds.
	Timeout time.Duration `msg:"timeout"`
}

// NodeType is one kind of node on the path.
type NodeType struct {
	// Name is the node's name in contexts: a CEL identifier, not "focus".
	Name string `msg:"name"`
	// Attrs are the node's attributes and their types, which contexts read
	// as name.attr.
	Attrs map[string]Type `msg:"attrs,omitempty"`
	// Text is set on a node that takes typed text, such as a text field:
	// while one is on the path, a key that types text is the node's unless
	// a binding at least as deep takes it.
	Text bool `msg:"text,omitempty"`
}

// Type is an attribute's CEL type: "string", "bool", "int", "uint",
// "double", "bytes", "dyn", "list<T>" or "map<K,V>", such as
// "map<string,string>".
type Type string

// Doc is one layer's keymap: the tree a program builds from its own files,
// such as its defaults or the user's config.
type Doc struct {
	// Layer is one of the schema's layers. Several documents may share one.
	Layer string `msg:"layer"`
	// Leader is the leader key, written short; the highest layer's wins
	// over the schema's.
	Leader string `msg:"leader,omitempty"`
	// Drops are the lower layers this one leaves out, such as a user's
	// keymap that starts from nothing dropping ["defaults"].
	Drops []string `msg:"drops,omitempty"`
	// Binds are in order: at equal depth and layer the later one wins.
	Binds []Bind `msg:"binds,omitempty"`
	// Groups label key prefixes for the menus.
	Groups []Group `msg:"groups,omitempty"`
	// Source is where Leader and Drops were written.
	Source string `msg:"source,omitempty"`
}

// Bind binds a key sequence to an action, or with Unbind takes a binding
// away. Binds and unbinds keep one order.
type Bind struct {
	// Keys are the sequence as written: "g g", "<leader> n".
	Keys string `msg:"keys"`
	// Action is what the keys run, as the program names it. It is empty
	// for an unbind.
	Action string `msg:"action,omitempty"`
	// Unbind binds the keys to nothing, taking them from the bindings it
	// wins over.
	Unbind bool `msg:"unbind,omitempty"`
	// Args are the action's values. Absent (nil or empty) means none, and
	// [""] means one empty value; a program can tell them apart.
	Args []string `msg:"args,omitempty"`
	// Label is how menus show the binding; empty leaves it to the program.
	Label string `msg:"label,omitempty"`
	// When is the bind's context, one CEL expression; empty holds
	// everywhere.
	When string `msg:"when,omitempty"`
	// Source is where the bind was written, as the program words it:
	// "config.kdl:12", an extension's manifest.
	Source string `msg:"source,omitempty"`
	// Props are the program's own properties, such as a flag its check
	// reads. myelin passes them back and never reads them.
	Props map[string]string `msg:"props,omitempty"`
}

// Group labels a key prefix, for menus: "<leader> f" is "+file".
type Group struct {
	// Keys are the prefix as written.
	Keys string `msg:"keys"`
	// Label is how menus show the group.
	Label string `msg:"label"`
	// When is the group's context; where several groups of the same keys
	// hold, the label follows the bindings' precedence.
	When string `msg:"when,omitempty"`
	// Source is where the group was written.
	Source string `msg:"source,omitempty"`
}

// Path is what holds the keyboard, outermost first. The last step is the
// focus.
type Path []Step

// Step is one node on the path.
type Step struct {
	// Name is one of the schema's node types.
	Name string `msg:"name"`
	// Attrs are values of the attributes the node's type declares, by
	// name. One a step doesn't give is absent: a context that reads it
	// doesn't hold, and [Context.MayHold] treats it as unknown. An
	// attribute declared "int" takes an integer of any Go kind.
	Attrs map[string]any `msg:"attrs,omitempty"`
}

// Binding is a [Bind] as the keymap holds it: its sequence resolved and its
// layer named.
type Binding struct {
	Bind `msg:"bind"`
	// Seq is the sequence with <leader> resolved.
	Seq []myelinkey.Key `msg:"seq"`
	// Layer is the layer of the document the bind came from.
	Layer string `msg:"layer"`
}

// Match is what a sequence matches on a path.
type Match struct {
	// Exact is the binding that wins at the sequence; absent when nothing
	// binds it or an unbind won.
	Exact *Binding `msg:"exact,omitempty"`
	// Prefix is set when a longer binding, at least as deep as Exact,
	// continues the sequence: wait for the next key.
	Prefix bool `msg:"prefix,omitempty"`
}

// Candidate is one binding [Keymap.Explain] considered for a sequence.
type Candidate struct {
	// Binding is the binding considered: one of the sequence's, or of a
	// longer sequence it begins.
	Binding Binding `msg:"binding"`
	// Holds is set when the binding's context holds on the path.
	Holds bool `msg:"holds"`
	// Depth is how deep into the path its context reaches; -1 when it
	// doesn't hold.
	Depth int `msg:"depth"`
	// Wins is set on the binding that runs: the one [Keymap.Match] returns.
	Wins bool `msg:"wins"`
}

// Problem is something wrong with a keymap, where it was written.
type Problem struct {
	// Source is the Source of what the problem is about.
	Source string `msg:"source"`
	// Text says what is wrong, and how to write it instead where there is
	// one way.
	Text string `msg:"text"`
	// Warning is set on a problem the keymap loads with.
	Warning bool `msg:"warning,omitempty"`
}

// Menu is a which-key level, or with nested items every level under a
// sequence.
type Menu struct {
	// Seq are the keys typed so far, the leader included.
	Seq []myelinkey.Key `msg:"seq"`
	// Label is the group's label: "+file".
	Label string `msg:"label,omitempty"`
	// Items are the keys that can follow, in the order the keymap first
	// binds each.
	Items []Item `msg:"items"`
}

// Item is one key that can follow a menu's sequence: a group, or the
// binding that wins there.
type Item struct {
	// Key is the key that follows the menu's sequence.
	Key myelinkey.Key `msg:"key"`
	// Label is the group's or the binding's. A binding with none is the
	// program's to word.
	Label string `msg:"label,omitempty"`
	// Group is set when more keys follow.
	Group bool `msg:"group,omitempty"`
	// Binding is the binding that wins at the key; nil for a group.
	Binding *Binding `msg:"binding,omitempty"`
	// Enabled is the program's verdict on a leaf; a group is enabled when
	// anything under it is.
	Enabled bool `msg:"enabled"`
	// Items are a group's own, in a tree.
	Items []Item `msg:"items,omitempty"`
}

// Pending is the sequencer's state: the keys typed so far, where, and until
// when.
type Pending struct {
	// Seq are the keys typed so far.
	Seq []myelinkey.Key `msg:"seq,omitempty"`
	// Path is where they were typed: an expiry runs what they finish
	// there.
	Path Path `msg:"path,omitempty"`
	// Deadline is when the buffered keys expire; zero when nothing is
	// buffered.
	Deadline time.Time `msg:"deadline,omitempty"`
}

// Result is what one key, or an expiry, did. At most one field is set.
type Result struct {
	// Run is the binding to run.
	Run *Binding `msg:"run,omitempty"`
	// Deadline is set when keys are buffered: wake then and call [Expire].
	Deadline time.Time `msg:"deadline,omitempty"`
	// Dropped are keys that matched nothing: the program types, forwards
	// or swallows them.
	Dropped []myelinkey.Key `msg:"dropped,omitempty"`
}
