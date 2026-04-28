package install

import "encoding/json"

// jsonMarshal is a thin alias kept in its own file so we can swap to a
// canonical-form encoder later (for hashing/signing) without touching the
// install logic.
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
