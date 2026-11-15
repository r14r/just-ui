package main

import (
    "path/filepath"
    "testing"
)

func TestParseExample(t *testing.T) {
    cat, err := parseFile(filepath.Join("examples", "justfile"))
    if err != nil { t.Fatal(err) }
    if len(cat.Recipes) != 4 { t.Fatalf("got %d recipes",len(cat.Recipes)) }
    var found bool
    for _, r := range cat.Recipes {
        if r.Name == "build-book" {
            found = true
            if r.Group != "Books" || len(r.Params) != 2 || !r.Params[0].Required || r.Params[1].Type != "select" { t.Fatalf("bad recipe: %+v",r) }
        }
    }
    if !found { t.Fatal("missing build-book") }
}

func TestGroupAttribute(t *testing.T) {
    cat, err := parseFile(filepath.Join("examples", "justfile"))
    if err != nil { t.Fatal(err) }
    for _,r := range cat.Recipes { if r.Name=="build" && r.Group!="Build" { t.Fatalf("group: %s",r.Group) } }
}
