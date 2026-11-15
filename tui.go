package main

import (
 "bufio"
 "errors"
 "fmt"
 "io"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "strings"

 "github.com/charmbracelet/bubbles/list"
 "github.com/charmbracelet/bubbles/textinput"
 tea "github.com/charmbracelet/bubbletea"
)
type recipeItem struct{ recipe Recipe }
func(i recipeItem) Title()string{return i.recipe.Label+"  ["+i.recipe.Group+"]"}
func(i recipeItem) Description()string{return i.recipe.Description+"  ·  just "+i.recipe.Name}
func(i recipeItem) FilterValue()string{return i.recipe.Name+" "+i.recipe.Label+" "+i.recipe.Group+" "+i.recipe.Description}
type choiceItem string
func(i choiceItem) Title()string{return string(i)}
func(i choiceItem) Description()string{return ""}
func(i choiceItem) FilterValue()string{return string(i)}
type stage int
const(browsing stage=iota;inputStage;choiceStage;confirmationStage;runningStage)
type appModel struct{
 file string
 recipes list.Model
 choices list.Model
 entry textinput.Model
 stage stage
 selected Recipe
 argument int
 values []string
 err string
 chooser bool
 chosen string
}
type commandFinished struct{err error}
func newModel(file string,catalog Catalog,chooser bool)appModel{
 items:=make([]list.Item,0,len(catalog.Recipes))
 for _,r:=range catalog.Recipes{items=append(items,recipeItem{r})}
 l:=list.New(items,list.NewDefaultDelegate(),80,20)
 l.Title="just-ui · Choose a recipe"
 l.SetShowStatusBar(true);l.SetShowHelp(true);l.SetFilteringEnabled(true)
 l.DisableQuitKeybindings()
 entry:=textinput.New();entry.Prompt="> ";entry.Width=50
 choices:=list.New([]list.Item{},list.NewDefaultDelegate(),80,20)
 choices.DisableQuitKeybindings()
 return appModel{file:file,recipes:l,choices:choices,entry:entry,chooser:chooser}
}
func(m appModel) Init()tea.Cmd{return nil}
func(m appModel) Update(message tea.Msg)(tea.Model,tea.Cmd){
 switch msg:=message.(type){
 case tea.WindowSizeMsg:
  width:=msg.Width;if width<30{width=30};height:=msg.Height-3;if height<8{height=8}
  m.recipes.SetSize(width,height);m.choices.SetSize(width,height)
 case commandFinished:
  m.stage=browsing
  if msg.err!=nil{m.err="Command finished with error: "+msg.err.Error()}else{m.err="Command completed successfully"}
  return m,nil
 case tea.KeyMsg:
  key:=msg.String()
  if key=="ctrl+c"{return m,tea.Quit}
  switch m.stage{
  case browsing:
   if key=="q"&&m.recipes.FilterState()!=list.Filtering{return m,tea.Quit}
   if key=="enter"&&m.recipes.FilterState()!=list.Filtering{
    if selected,ok:=m.recipes.SelectedItem().(recipeItem);ok{
     m.selected=selected.recipe;m.values=make([]string,len(m.selected.Params));m.argument=0;m.err=""
     if m.chooser{m.chosen=m.selected.Name;return m,tea.Quit}
     return m,m.advance()
    }
   }
  case inputStage:
   if key=="esc"{return m,m.back()}
   if key=="enter"{
    value:=m.entry.Value()
    if err:=validate(m.selected.Params[m.argument],value);err!=nil{m.err=err.Error();return m,nil}
    m.err="";m.values[m.argument]=value;m.argument++
    return m,m.advance()
   }
   var cmd tea.Cmd;m.entry,cmd=m.entry.Update(msg);return m,cmd
  case choiceStage:
   if key=="esc"{return m,m.back()}
   if key=="enter"&&m.choices.FilterState()!=list.Filtering{
    if selected,ok:=m.choices.SelectedItem().(choiceItem);ok{
     m.values[m.argument]=string(selected);m.argument++;return m,m.advance()
    }
   }
  case confirmationStage:
   if key=="esc"||key=="n"{m.stage=browsing;return m,nil}
   if key=="y"||key=="enter"{return m,m.execute()}
   return m,nil
  case runningStage:return m,nil
  }
 }
 switch m.stage{
 case browsing:
  var cmd tea.Cmd;m.recipes,cmd=m.recipes.Update(message);return m,cmd
 case choiceStage:
  var cmd tea.Cmd;m.choices,cmd=m.choices.Update(message);return m,cmd
 case inputStage:
  var cmd tea.Cmd;m.entry,cmd=m.entry.Update(message);return m,cmd
 }
 return m,nil
}
func validate(p Param,value string)error{
 if p.Required&&strings.TrimSpace(value)==""{return fmt.Errorf("%s is required",p.Label)}
 if p.Type=="number"&&value!=""{if _,err:=strconv.ParseFloat(value,64);err!=nil{return fmt.Errorf("%s must be a number",p.Label)}}
 if p.Type=="select"&&len(p.Options)>0{
  for _,v:=range p.Options{if v==value{return nil}}
  return errors.New("value must be one of the available options")
 }
 return nil
}
func(m *appModel) advance()tea.Cmd{
 if m.argument>=len(m.selected.Params){
  if m.selected.Confirm{m.stage=confirmationStage;return nil}
  return m.execute()
 }
 p:=m.selected.Params[m.argument]
 if p.Type=="select"||p.Type=="checkbox"{
  opts:=p.Options;if p.Type=="checkbox"{opts=[]string{"true","false"}}
  if len(opts)>0{
   items:=make([]list.Item,0,len(opts));for _,o:=range opts{items=append(items,choiceItem(o))}
   m.choices=list.New(items,list.NewDefaultDelegate(),m.recipes.Width(),m.recipes.Height())
   m.choices.Title=p.Label+" · choose value";m.choices.DisableQuitKeybindings()
   for i,v:=range opts{if v==p.Default{m.choices.Select(i);break}}
   m.stage=choiceStage;return nil
  }
 }
 m.entry=textinput.New();m.entry.Width=50;m.entry.Prompt="> ";m.entry.Placeholder=p.Label
 if p.Type=="password"{m.entry.EchoMode=textinput.EchoPassword;m.entry.EchoCharacter='•'}
 value:=m.values[m.argument];if value==""{value=p.Default};m.entry.SetValue(value)
 m.stage=inputStage;return m.entry.Focus()
}
func(m *appModel) back()tea.Cmd{
 if m.argument==0{m.stage=browsing;return nil}
 m.argument--;return m.advance()
}
func(m *appModel) execute()tea.Cmd{
 args:=[]string{"--justfile",m.file,m.selected.Name};args=append(args,m.values...)
 executable,err:=os.Executable()
 if err!=nil{
  m.stage=browsing
  m.err="Cannot start command: "+err.Error()
  return nil
 }
 // Execute through our own child process so the command output remains on screen
 // until Enter is pressed. Bubble Tea suspends its alternate-screen UI.
 cmd:=exec.Command(executable,append([]string{"--exec-recipe"},args...)...)
 cmd.Dir=filepath.Dir(m.file)
 m.stage=runningStage
 return tea.ExecProcess(cmd,func(err error)tea.Msg{return commandFinished{err:err}})
}
// runRecipeAndWait runs a selected recipe with a connected terminal, then waits
// for Enter before Bubble Tea resumes its alternate-screen interface.
func runRecipeAndWait(file, recipe string, values []string, stdin io.Reader, stdout, stderr io.Writer) error {
 args:=append([]string{"--justfile",file,recipe},values...)
 cmd:=exec.Command("just",args...)
 cmd.Dir=filepath.Dir(file)
 cmd.Stdin=stdin
 cmd.Stdout=stdout
 cmd.Stderr=stderr
 err:=cmd.Run()
 if err!=nil {
  fmt.Fprintf(stderr,"\\nCommand failed: %v\\n",err)
 }
 fmt.Fprint(stdout,"\\nPress Enter to go back to just-ui")
 // If the command previously read from stdin, continue with the same stream.
 _,readErr:=bufio.NewReader(stdin).ReadString('\\n')
 fmt.Fprintln(stdout)
 if readErr!=nil{return fmt.Errorf("waiting for Enter: %w",readErr)}
 return err
}
func(m appModel) View()string{
 switch m.stage{
 case browsing:
  extra:="\n↑/↓ navigate  / filter  Enter select  q quit"
  if m.err!=""{extra="\n"+m.err+extra}
  return m.recipes.View()+extra
 case inputStage:
  p:=m.selected.Params[m.argument]
  label:=fmt.Sprintf("%s — %s (%d/%d)\n",m.selected.Label,p.Label,m.argument+1,len(m.selected.Params))
  if p.Required{label+="Required\n"}
  if m.err!=""{label+="Error: "+m.err+"\n"}
  return label+"\n"+m.entry.View()+"\n\nEnter next  Esc back  Ctrl+C quit"
 case choiceStage:return m.choices.View()+"\nEnter choose  / filter  Esc back"
 case confirmationStage:return "Confirm execution: just "+m.selected.Name+"\n\nPress y/Enter to run or n/Esc to cancel."
 case runningStage:return "Running: just "+m.selected.Name+"\n"
 }
 return ""
}
func runTUI(file string,catalog Catalog)error{
 if len(catalog.Recipes)==0{return errors.New("no recipes found")}
 if !isTerminal(){return errors.New("requires an interactive terminal; use --list for noninteractive output")}
 _,err:=tea.NewProgram(newModel(file,catalog,false),tea.WithAltScreen()).Run();return err
}
func isTerminal()bool{info,err:=os.Stdin.Stat();return err==nil&&(info.Mode()&os.ModeCharDevice)!=0}
func runChooser(catalog Catalog)error{
 if isTerminal(){return errors.New("--chooser expects recipe names on stdin (from just --choose)")}
 scanner:=bufio.NewScanner(os.Stdin);names:=map[string]bool{}
 for scanner.Scan(){names[strings.TrimSpace(scanner.Text())]=true}
 if err:=scanner.Err();err!=nil{return err}
 allowed:=[]Recipe{}
 for _,recipe:=range catalog.Recipes{if names[recipe.Name]{allowed=append(allowed,recipe)}}
 if len(allowed)==0{return errors.New("no matching recipes received from just --choose")}
 tty,err:=os.OpenFile("/dev/tty",os.O_RDWR,0);if err!=nil{return fmt.Errorf("chooser requires controlling terminal: %w",err)}
 defer tty.Close()
 final,err:=tea.NewProgram(newModel("",Catalog{Recipes:allowed},true),tea.WithInput(tty),tea.WithOutput(tty),tea.WithAltScreen()).Run()
 if err!=nil{return err}
 if choice:=final.(appModel).chosen;choice!=""{fmt.Println(choice)}
 return nil
}
