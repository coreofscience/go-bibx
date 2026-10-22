package search_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenize(t *testing.T) {
	t.Run("lowercases and splits by whitespace", func(t *testing.T) {
		tokens := search.Tokenize("Hello World")
		assert.Equal(t, []string{"hello", "world"}, tokens)
	})

	t.Run("strips punctuation", func(t *testing.T) {
		tokens := search.Tokenize("Graph-Neural Networks (GNNs), a survey!")
		assert.Equal(t, []string{"graph", "neural", "networks", "gnns", "a", "survey"}, tokens)
	})

	t.Run("handles multiple spaces, tabs, and newlines", func(t *testing.T) {
		tokens := search.Tokenize("  multiple \t whitespace \n tokens   ")
		assert.Equal(t, []string{"multiple", "whitespace", "tokens"}, tokens)
	})

	t.Run("empty string returns empty slice", func(t *testing.T) {
		assert.Empty(t, search.Tokenize(""))
		assert.Empty(t, search.Tokenize("   "))
		assert.Empty(t, search.Tokenize("!@#$%^&*()"))
	})
}

func TestBM25Search_ExactTitleMatch(t *testing.T) {
	docs := []search.Document{
		{
			ID:    "doc1",
			Title: "Graph Neural Networks for Knowledge Graphs",
		},
		{
			ID:       "doc2",
			Title:    "Convolutional Networks in Computer Vision",
			Abstract: "We mention graph briefly in the conclusion.",
		},
	}

	results := search.BM25Search(docs, "Graph Neural Networks", 10)
	require.Len(t, results, 2)
	assert.Equal(t, "doc1", results[0].ID)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestBM25Search_MultiTermQuery(t *testing.T) {
	docs := []search.Document{
		{
			ID:       "doc_both",
			Title:    "Introduction to Graph Neural Networks",
			Abstract: "Overview of graph representation and neural architectures.",
		},
		{
			ID:       "doc_one",
			Title:    "Graph Algorithms in Python",
			Abstract: "A book on breadth first search and shortest paths.",
		},
		{
			ID:       "doc_none",
			Title:    "Quantum Computing Principles",
			Abstract: "Qubits and entanglement.",
		},
	}

	results := search.BM25Search(docs, "graph neural", 10)
	require.Len(t, results, 2)
	assert.Equal(t, "doc_both", results[0].ID)
	assert.Equal(t, "doc_one", results[1].ID)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestBM25Search_IDFWeighting(t *testing.T) {
	// Term "qubit" appears only in doc1 (rare)
	// Term "study" appears in all 5 documents (common)
	docs := []search.Document{
		{
			ID:       "rare_term_doc",
			Title:    "A study of qubit decoherence",
			Abstract: "Detailed qubit analysis in quantum computers.",
		},
		{
			ID:       "common_doc_1",
			Title:    "A study on deep learning",
			Abstract: "Comprehensive study on transformers.",
		},
		{
			ID:       "common_doc_2",
			Title:    "A study of database indexing",
			Abstract: "Another study on B-trees.",
		},
		{
			ID:       "common_doc_3",
			Title:    "A study regarding compilers",
			Abstract: "Empirical study on LLVM optimizations.",
		},
		{
			ID:       "common_doc_4",
			Title:    "A study in network routing",
			Abstract: "Study of packet latency.",
		},
	}

	// Query has both rare "qubit" and common "study"
	results := search.BM25Search(docs, "study qubit", 10)
	require.NotEmpty(t, results)
	// The document matching "qubit" must rank highest due to higher IDF
	assert.Equal(t, "rare_term_doc", results[0].ID)
}

func TestBM25Search_FieldWeighting(t *testing.T) {
	// Doc Title has the term "attention" in Title (weight 3.0)
	// Doc Abstract has the term "attention" in Abstract (weight 1.0)
	// Both documents have equal length to isolate field weight effects
	docs := []search.Document{
		{
			ID:       "doc_title",
			Title:    "Attention mechanism",
			Abstract: "General overview",
		},
		{
			ID:       "doc_abstract",
			Title:    "General overview",
			Abstract: "Attention mechanism",
		},
	}

	// Disable phrase boost to isolate field weights
	results := search.BM25Search(docs, "attention", 10, search.WithPhraseBoost(0.0))
	require.Len(t, results, 2)
	assert.Equal(t, "doc_title", results[0].ID)
	assert.Equal(t, "doc_abstract", results[1].ID)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestBM25Search_PhraseBoost(t *testing.T) {
	docs := []search.Document{
		{
			ID:    "phrase_match",
			Title: "Deep Learning for Natural Language Processing",
		},
		{
			ID:    "terms_separated",
			Title: "Learning with Deep Hierarchical Representations",
		},
	}

	results := search.BM25Search(docs, "deep learning", 10, search.WithPhraseBoost(2.0))
	require.Len(t, results, 2)
	assert.Equal(t, "phrase_match", results[0].ID)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestBM25Search_AuthorAndKeywordMatching(t *testing.T) {
	docs := []search.Document{
		{
			ID:      "author_match",
			Title:   "Modern Information Retrieval",
			Authors: []string{"Gerard Salton", "Ricardo Baeza-Yates"},
		},
		{
			ID:       "keyword_match",
			Title:    "Search Architectures",
			Keywords: []string{"pagerank", "salton model"},
		},
	}

	results := search.BM25Search(docs, "salton", 10)
	require.Len(t, results, 2)
	// Both matched on "salton" (author and keyword weights are 2.0)
	assert.Greater(t, results[0].Score, 0.0)
	assert.Greater(t, results[1].Score, 0.0)
}

func TestBM25Search_CaseInsensitivity(t *testing.T) {
	docs := []search.Document{
		{
			ID:    "doc1",
			Title: "GRAPH NEURAL NETWORKS",
		},
	}

	resultsLower := search.BM25Search(docs, "graph neural", 10)
	resultsUpper := search.BM25Search(docs, "GRAPH NEURAL", 10)
	resultsMixed := search.BM25Search(docs, "GrApH nEuRaL", 10)

	require.Len(t, resultsLower, 1)
	require.Len(t, resultsUpper, 1)
	require.Len(t, resultsMixed, 1)

	assert.Equal(t, resultsLower[0].Score, resultsUpper[0].Score)
	assert.Equal(t, resultsLower[0].Score, resultsMixed[0].Score)
}

func TestBM25Search_EdgeCases(t *testing.T) {
	docs := []search.Document{
		{
			ID:    "doc1",
			Title: "Introduction to Computer Science",
		},
	}

	t.Run("empty docs returns nil", func(t *testing.T) {
		assert.Nil(t, search.BM25Search(nil, "computer", 10))
		assert.Nil(t, search.BM25Search([]search.Document{}, "computer", 10))
	})

	t.Run("empty query returns nil", func(t *testing.T) {
		assert.Nil(t, search.BM25Search(docs, "", 10))
		assert.Nil(t, search.BM25Search(docs, "   ", 10))
	})

	t.Run("query with only punctuation returns nil", func(t *testing.T) {
		assert.Nil(t, search.BM25Search(docs, "!@#$ %^&*()", 10))
	})

	t.Run("zero matches returns empty slice", func(t *testing.T) {
		results := search.BM25Search(docs, "nonexistenttermxyz", 10)
		assert.Empty(t, results)
	})

	t.Run("limit handling", func(t *testing.T) {
		multiDocs := []search.Document{
			{ID: "d1", Title: "Network Science Paper One"},
			{ID: "d2", Title: "Network Science Paper Two"},
			{ID: "d3", Title: "Network Science Paper Three"},
		}

		assert.Nil(t, search.BM25Search(multiDocs, "network", 0))
		assert.Nil(t, search.BM25Search(multiDocs, "network", -5))

		res1 := search.BM25Search(multiDocs, "network", 1)
		assert.Len(t, res1, 1)

		resAll := search.BM25Search(multiDocs, "network", 100)
		assert.Len(t, resAll, 3)
	})
}
