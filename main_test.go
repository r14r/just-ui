package main

import (
    "path/filepath"
    "testing"
    "os"
    "os/exec"
    "strings"
    "errors"
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

func TestParameterValidation(t *testing.T) {
 cases := []struct {
  param Param
  value string
  wantError bool
 }{
  {Param{Label:"Book",Required:true},"",true},
  {Param{Label:"Book",Required:true},"26",false},
  {Param{Label:"Count",Type:"number"},"twenty",true},
  {Param{Label:"Count",Type:"number"},"12.5",false},
  {Param{Label:"Layout",Type:"select",Options:[]string{"a","b"}},"c",true},
  {Param{Label:"Layout",Type:"select",Options:[]string{"a","b"}},"b",false},
 }
 for _,tc:=range cases {
  got:=validate(tc.param,tc.value)
  if (got!=nil)!=tc.wantError{t.Errorf("validate(%q,%q) error=%v; wantError=%v",tc.param.Label,tc.value,got,tc.wantError)}
 }
}

func TestModelStartsWithRecipeList(t *testing.T) {
 catalog,err:=parseFile(filepath.Join("examples","justfile"))
 if err!=nil{t.Fatal(err)}
 model:=newModel("examples/justfile",catalog,false)
 if model.stage!=browsing{t.Fatal("expected browsing stage")}
 if len(model.recipes.Items())!=len(catalog.Recipes){t.Fatal("missing recipes in TUI")}
}

func TestResolveJustfileExplicit(t *testing.T) {
 filename,err:=resolveJustfile(filepath.Join("examples","justfile"))
 if err!=nil{t.Fatal(err)}
 if !filepath.IsAbs(filename){t.Fatal("Justfile must resolve to absolute path")}
}

func TestExampleJustfileValid(t *testing.T) {
 if _,err:=exec.LookPath("just");err!=nil{t.Skip("just not installed")}
 cmd:=exec.Command("just","--justfile","examples/justfile","--summary")
 if output,err:=cmd.CombinedOutput();err!=nil{t.Fatalf("example Justfile invalid: %v\n%s",err,output)}
}

func TestExecutionWaitsForEnterAndPreservesStatus(t *testing.T) {
 directory:=t.TempDir()
 fake:=filepath.Join(directory,"just")
 script:="#!/bin/sh\nprintf 'recipe output\\n'\nexit 7\n"
 if err:=os.WriteFile(fake,[]byte(script),0755);err!=nil{t.Fatal(err)}
 t.Setenv("PATH",directory+string(os.PathListSeparator)+os.Getenv("PATH"))
 cmd:=executionCommand(filepath.Join(directory,"justfile"),"test",[]string{"input"})
 cmd.Stdin=strings.NewReader("\n")
 output,err:=cmd.CombinedOutput()
 if !strings.Contains(string(output),"recipe output")||!strings.Contains(string(output),"Press Enter to go back to just-ui"){t.Fatalf("missing output or prompt: %q",output)}
 var exit *exec.ExitError
 if !errors.As(err,&exit)||exit.ExitCode()!=7{t.Fatalf("exit status not preserved: %v",err)}
}
