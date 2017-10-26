// Package term provides a data structure to store terms that can be matched in a parse
package term

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/HedgeChart/go-patricia/patricia"
)

type TermData struct {
	exactTrie     *patricia.Trie
	substringTrie *patricia.Trie
}

type (
	Prefix      []byte
	Item        interface{}
	VisitorFunc func(prefix Prefix, item Item) error
)

const minimumLengthForSubstringSearch = 0

func NewTermData() *TermData {
	td := new(TermData)
	td.exactTrie = patricia.NewTrie()
	td.substringTrie = patricia.NewTrie()

	return td
}

func (td *TermData) Lookup(key string) interface{} {
	lookupKey := getLookupKey(key)
	exact := td.exactTrie.Get(lookupKey)

	return exact
}

func (td *TermData) LookupSubstring(key string) interface{} {
	lookupKey := getLookupKey(key)

	// Find an exact match on the exact trie
	exact := td.exactTrie.Get(lookupKey)

	if exact != nil {
		return exact
	}

	// Find an exact match on the substring trie
	substring := td.substringTrie.Get(lookupKey)
	if substring != nil {
		return substring
	}

	//fmt.Printf("Visit Subtree\n")
	// Now visit subnodes to figure out if anything matches
	var found Item
	f := func(prefix Prefix, item Item) error {
		//fmt.Printf("prefix=%s, item=%q\n", prefix, item)
		found = item
		return nil
	}
	td.VisitSubtreeExact(string(lookupKey), f)

	if found == nil {
		td.VisitSubtreeSubstring(string(lookupKey), f)
	}
	//fmt.Printf("Found=%s\n", found)

	return found
}

func (td *TermData) Insert(key string, value interface{}) {
	lookupKey := getLookupKey(key)
	td.exactTrie.Insert(lookupKey, value)

	for i := 1; i <= 5; i++ {
		substring := dropWords(i, key)
		if len(substring) > minimumLengthForSubstringSearch && substring[0] != '{' {
			td.substringTrie.Insert(getLookupKey(substring), value)
		}
	}
}

func (td *TermData) MatchSubtree(key string) bool {
	lookupKey := getLookupKey(key)

	return td.exactTrie.MatchSubtree(lookupKey)
}

func (td *TermData) Match(key string) bool {
	lookupKey := getLookupKey(key)

	return td.exactTrie.Match(lookupKey)
}

func (td *TermData) VisitSubtreeExact(prefix string, visitor VisitorFunc) {
	patPrefix := getLookupKey(prefix)
	td.exactTrie.VisitSubtree(patPrefix, createVisitorFunc(visitor))
}

func (td *TermData) VisitSubtreeSubstring(prefix string, visitor VisitorFunc) {
	patPrefix := getLookupKey(prefix)
	td.substringTrie.VisitSubtree(patPrefix, createVisitorFunc(visitor))
}

func (td *TermData) Visit(visitor VisitorFunc) {
	td.exactTrie.Visit(createVisitorFunc(visitor))
}

func (td *TermData) PrintPrefix(key string) {
	lookupKey := getLookupKey(key)
	printSubTrie(lookupKey, td.exactTrie)
}

func (td *TermData) PrintAll() {
	fmt.Printf("--------- Exact Matches ------------- \n")
	printTrie(td.exactTrie)
	fmt.Printf("--------- Substring Matches ------------- \n")
	printTrie(td.substringTrie)
	fmt.Printf("--------- End Matches ------------- \n")
}

func createVisitorFunc(visitor VisitorFunc) patricia.VisitorFunc {
	return func(prefix patricia.Prefix, item patricia.Item) error {
		return visitor(Prefix(prefix), Item(item))
	}
}

func getLookupKey(key string) patricia.Prefix {
	lookupKey := patricia.Prefix(strings.Trim(strings.ToLower(key), " "))
	return lookupKey
}

func dropWords(n int, s string) string {
	sArr := strings.Split(s, " ")
	output := ""

	if 0 < n && n < len(sArr) {
		output = strings.Join(sArr[n:], " ") + " " + strings.Join(sArr[:n], " ")
	}

	return strings.Trim(output, " ")
}

func printTrie(trie *patricia.Trie) {
	printItem := func(prefix patricia.Prefix, item patricia.Item) error {
		m, _ := json.Marshal(item)
		fmt.Printf("%q: %s\n", prefix, m)
		return nil
	}

	trie.Visit(printItem)
}

func printSubTrie(prefix patricia.Prefix, trie *patricia.Trie) {
	printItem := func(prefix patricia.Prefix, item patricia.Item) error {
		m, _ := json.Marshal(item)
		fmt.Printf("%q: %s\n", prefix, m)
		return nil
	}

	trie.VisitSubtree(prefix, printItem)
}
