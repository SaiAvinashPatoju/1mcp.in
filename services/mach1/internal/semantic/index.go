// Package semantic implements a lightweight tool-ranking index used by the
// supervisor to surface the most relevant tools for an agent query.
//
// Why TF-IDF and not ONNX/LanceDB:
//
//   For tool selection across ~100 tools the dominant signal is keyword
//   overlap (tool name, MCP id, description). A 90MB embedding model adds
//   binary bloat for marginal recall gain at this scale. A pure-Go in-memory
//   TF-IDF index ranks the entire corpus in << 1ms with zero dependencies.
//
// Phase 6.5 swaps the implementation behind the same Index interface for
// LanceDB + ONNX MiniLM once the corpus crosses ~1k tools (rare in practice).
package semantic

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Doc is one indexed item.
type Doc struct {
	ID   string // opaque key returned in results (e.g. "fs__read_file")
	Text string // searchable text (id + name + description + tool descriptions)
}

// Index is a goroutine-safe TF-IDF index. Build once, query many.
type Index struct {
	docIDs    []string
	docVecs   []map[string]float64 // term -> tf-idf weight
	idf       map[string]float64
	docLength int
}

// Build constructs the index from docs. Empty corpus returns an empty index
// that always returns no results.
func Build(docs []Doc) *Index {
	idx := &Index{
		docIDs:    make([]string, 0, len(docs)),
		docVecs:   make([]map[string]float64, 0, len(docs)),
		idf:       make(map[string]float64),
		docLength: len(docs),
	}
	if len(docs) == 0 {
		return idx
	}
	tokenized := make([][]string, len(docs))
	df := make(map[string]int) // doc-frequency
	for i, d := range docs {
		toks := tokenize(d.Text)
		tokenized[i] = toks
		seen := make(map[string]struct{}, len(toks))
		for _, t := range toks {
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			df[t]++
		}
	}
	// Smoothed IDF: log((N+1)/(df+1)) + 1 (sklearn-style) avoids zero IDF
	// for terms appearing in every doc and undefined for unseen terms.
	N := float64(len(docs))
	for term, count := range df {
		idx.idf[term] = math.Log((N+1)/(float64(count)+1)) + 1
	}
	for i, toks := range tokenized {
		tf := make(map[string]float64, len(toks))
		for _, t := range toks {
			tf[t]++
		}
		vec := make(map[string]float64, len(tf))
		for t, f := range tf {
			vec[t] = (1 + math.Log(f)) * idx.idf[t]
		}
		normalize(vec)
		idx.docIDs = append(idx.docIDs, docs[i].ID)
		idx.docVecs = append(idx.docVecs, vec)
	}
	return idx
}

// Result is one ranked hit.
type Result struct {
	ID    string
	Score float64
}

// Rank returns the top-k docs by cosine similarity to query. k <= 0 returns
// all positive-scoring docs.
func (i *Index) Rank(query string, k int) []Result {
	if i == nil || i.docLength == 0 {
		return nil
	}
	toks := tokenize(query)
	if len(toks) == 0 {
		return nil
	}
	qtf := make(map[string]float64, len(toks))
	for _, t := range toks {
		qtf[t]++
	}
	qvec := make(map[string]float64, len(qtf))
	for t, f := range qtf {
		idf, ok := i.idf[t]
		if !ok {
			continue // unseen term contributes nothing
		}
		qvec[t] = (1 + math.Log(f)) * idf
	}
	if len(qvec) == 0 {
		return nil
	}
	normalize(qvec)

	out := make([]Result, 0, len(i.docVecs))
	for j, dv := range i.docVecs {
		s := dot(qvec, dv)
		if s > 0 {
			out = append(out, Result{ID: i.docIDs[j], Score: s})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Score > out[b].Score })
	if k > 0 && len(out) > k {
		out = out[:k]
	}
	return out
}

// tokenize lowercases, splits on non-alphanumeric, drops single-char tokens
// and a small English stopword list. Deliberately simple — good enough for
// short tool descriptions; swap for a real tokenizer if recall ever lags.
func tokenize(s string) []string {
	if s == "" {
		return nil
	}
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) <= 1 {
			continue
		}
		if _, stop := stopwords[f]; stop {
			continue
		}
		out = append(out, f)
	}
	return out
}

func normalize(v map[string]float64) {
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	if sum == 0 {
		return
	}
	n := math.Sqrt(sum)
	for k := range v {
		v[k] /= n
	}
}

func dot(a, b map[string]float64) float64 {
	if len(a) > len(b) {
		a, b = b, a
	}
	var s float64
	for k, v := range a {
		if w, ok := b[k]; ok {
			s += v * w
		}
	}
	return s
}

var stopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "with": {}, "from": {}, "this": {},
	"that": {}, "are": {}, "was": {}, "you": {}, "your": {}, "into": {},
	"its": {}, "use": {}, "via": {}, "all": {}, "any": {}, "but": {},
	"can": {}, "has": {}, "had": {}, "have": {}, "not": {}, "out": {},
	"per": {}, "than": {}, "then": {}, "they": {}, "them": {}, "their": {},
	"will": {}, "would": {}, "could": {}, "should": {}, "been": {},
}
