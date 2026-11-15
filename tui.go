package main

import (
 "bufio"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "strings"

 "github.com/charmbracelet/bubbles/list"
 "github.com/charmbracelet/bubbles/textinput"
 tea "github.com/charmbracelet/bubbletea"
 "github.com/charmbracelet/lipgloss"
)

type recipeItem struct{ recipe Recipe }
func(i recipeItem) Title() string {return i.recipe.Label}
func(i recipeItem) Description() string {return i.recipe.Group+" · "+i.recipe.Name}
func(i recipeItem) FilterValue() string {return i.recipe.Name+" "+i.recipe.Label+" "+i.recipe.Group+" "+i.recipe.Description}
type choiceItem string
func(i choiceItem) Title() string {return string(i)}
func(i choiceItem) Description() string{return ""}
func(i choiceItem) FilterValue() string{return string(i)}

type stage int
const (browsing stage=iota; inputStage; choiceStage; confirmationStage; runningStage)
type appModel struct {
 file string
 recipes list.Model
 choices list.Model
 entry textinput.Model
 stage stage
 selected Recipe
 hasSelection bool
 argument int
 values []string
 err string
 chooser bool
 chosen string
 width int
 height int
}
type commandFinished struct{err error}

func newModel(file string,cat Catalog,chooser bool) appModel {
 items:=make([]list.Item,0,len(cat.Recipes))
 for _,r:=range cat.Recipes {items=append(items,recipeItem{r})}
 l:=list.New(items,list.NewDefaultDelegate(),30,18)
 l.Title="Commands"
 l.SetShowStatusBar(true);l.SetShowHelp(true);l.SetFilteringEnabled(true);l.DisableQuitKeybindings()
 choices:=list.New([]list.Item{},list.NewDefaultDelegate(),50,12)
 choices.DisableQuitKeybindings()
 entry:=textinput.New();entry.Prompt="> ";entry.Width=45
 return appModel{file:file,recipes:l,choices:choices,entry:entry,chooser:chooser,width:110,height:28}
}
func(m appModel) Init() tea.Cmd {return nil}

func(m *appModel) panelWidths()(int,int){
 w:=m.width
 if w<60 {w=60}
 left:=w*30/100
 if left<22{left=22}
 right:=w-left
 return left,right
}
func(m *appModel) resize(){
 left,right:=m.panelWidths()
 h:=m.height-5
 if h<8{h=8}
 m.recipes.SetSize(left-2,h)
 m.choices.SetSize(right-6,h-11)
 m.entry.Width=right-10
 if m.entry.Width<15{m.entry.Width=15}
}
func(m appModel) Update(message tea.Msg)(tea.Model,tea.Cmd){
 switch msg:=message.(type) {
 case tea.WindowSizeMsg:
  m.width=msg.Width;m.height=msg.Height;m.resize()
 case commandFinished:
  m.stage=browsing
  if msg.err!=nil {m.err="Command failed: "+msg.err.Error()}else{m.err="Command completed successfully"}
  return m,nil
 case tea.KeyMsg:
  key:=msg.String()
  if key=="ctrl+c"{return m,tea.Quit}
  switch m.stage{
  case browsing:
   if key=="q"&&m.recipes.FilterState()!=list.Filtering{return m,tea.Quit}
   if key=="enter"&&m.recipes.FilterState()!=list.Filtering{
    if item,ok:=m.recipes.SelectedItem().(recipeItem);ok{
     if m.chooser {m.chosen=item.recipe.Name;return m,tea.Quit}
     m.selected=item.recipe;m.hasSelection=true;m.values=make([]string,len(m.selected.Params))
     m.argument=0;m.err=""
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
    if chosen,ok:=m.choices.SelectedItem().(choiceItem);ok{
     m.values[m.argument]=string(chosen);m.argument++
     return m,m.advance()
    }
   }
  case confirmationStage:
   if key=="esc"||key=="n"{m.stage=browsing;return m,nil}
   if key=="e"&&len(m.values)>0 {
    m.argument=0;return m,m.advance()
   }
   if key=="enter"||key=="y"{return m,m.execute()}
  case runningStage:return m,nil
  }
 }
 switch m.stage{
 case browsing:
  var cmd tea.Cmd;m.recipes,cmd=m.recipes.Update(message);return m,cmd
 case inputStage:
  var cmd tea.Cmd;m.entry,cmd=m.entry.Update(message);return m,cmd
 case choiceStage:
  var cmd tea.Cmd;m.choices,cmd=m.choices.Update(message);return m,cmd
 }
 return m,nil
}
func validate(p Param,v string)error {
 if p.Required&&strings.TrimSpace(v)==""{return fmt.Errorf("%s is required",p.Label)}
 if p.Type=="number"&&v!=""{if _,err:=strconv.ParseFloat(v,64);err!=nil{return fmt.Errorf("%s must be a number",p.Label)}}
 if p.Type=="select"&&len(p.Options)>0 {
  for _,opt:=range p.Options{if opt==v{return nil}}
  return errors.New("invalid select option")
 }
 return nil
}
func(m *appModel) advance() tea.Cmd {
 if m.argument>=len(m.selected.Params) {
  m.stage=confirmationStage
  return nil
 }
 p:=m.selected.Params[m.argument]
 if p.Type=="select"||p.Type=="checkbox" {
  opts:=p.Options
  if p.Type=="checkbox"{opts=[]string{"true","false"}}
  if len(opts)>0{
   items:=make([]list.Item,0,len(opts));for _,opt:=range opts{items=append(items,choiceItem(opt))}
   _,right:=m.panelWidths()
   m.choices=list.New(items,list.NewDefaultDelegate(),right-6,m.height-14)
   m.choices.Title=p.Label+" · choose value"
   m.choices.DisableQuitKeybindings()
   for i,opt:=range opts{if opt==p.Default||opt==m.values[m.argument]{m.choices.Select(i);break}}
   m.stage=choiceStage;return nil
  }
 }
 m.entry=textinput.New();m.entry.Prompt="> ";m.entry.Placeholder=p.Label
 _,right:=m.panelWidths();m.entry.Width=right-10
 if m.entry.Width<15{m.entry.Width=15}
 if p.Type=="password"{m.entry.EchoMode=textinput.EchoPassword;m.entry.EchoCharacter='•'}
 value:=m.values[m.argument];if value==""{value=p.Default}
 m.entry.SetValue(value);m.stage=inputStage
 return m.entry.Focus()
}
func(m *appModel) back()tea.Cmd{
 if m.argument==0 {m.stage=browsing;return nil}
 m.argument--;return m.advance()
}

// shellQuote makes the displayed command safe to copy into a POSIX shell.
func shellQuote(arg string)string{
 if arg!=""&&strings.IndexFunc(arg,func(r rune)bool{
  return !(r>='a'&&r<='z'||r>='A'&&r<='Z'||r>='0'&&r<='9'||strings.ContainsRune("_@%+=:,./-",r))
 })==-1{return arg}
 return "'"+strings.ReplaceAll(arg,"'","'\\''")+"'"
}
func commandLine(file,recipe string,values []string)string{
 args:=append([]string{"just","--justfile",file,recipe},values...)
 quoted:=make([]string,len(args));for i,arg:=range args{quoted[i]=shellQuote(arg)}
 return strings.Join(quoted," ")
}
func(m appModel) displayedCommand()string{
 args:=append([]string(nil),m.values...)
 if m.stage==inputStage&&m.argument<len(args){args[m.argument]=m.entry.Value()}
 // Password values are not echoed into screen contents or terminal scrollback.
 for i,p:=range m.selected.Params{
  if p.Type=="password"&&i<len(args){args[i]="<hidden>"}
 }
 return commandLine(m.file,m.selected.Name,args)
}
// executionCommand prints the command line, then hands the terminal to just.
// It waits for Enter even on a nonzero exit; the exit status is preserved.
func executionCommand(file,recipe string,values []string)*exec.Cmd{
 preview:=commandLine(file,recipe,values)
 script:="printf '%s\\n\\n' \"$1\"; shift; just \"$@\"; status=$?; printf '\\nPress Enter to go back to just-ui'; IFS= read -r _ || true; printf '\\n'; exit \"$status\""
 args:=append([]string{"-c",script,"just-ui",preview,"--justfile",file,recipe},values...)
 cmd:=exec.Command("sh",args...);cmd.Dir=filepath.Dir(file)
 return cmd
}
func(m *appModel) execute()tea.Cmd{
 // Avoid showing secrets in the printed command line, without altering argv.
 cmd:=executionCommand(m.file,m.selected.Name,m.values)
 for i,p:=range m.selected.Params{
  if p.Type=="password"&&i<len(m.values){
   cmd.Args[4]=m.displayedCommand()
   break
  }
 }
 m.stage=runningStage
 return tea.ExecProcess(cmd,func(err error)tea.Msg{return commandFinished{err:err}})
}
func(m appModel) rightPanel()string{
 if !m.hasSelection{
  if item,ok:=m.recipes.SelectedItem().(recipeItem);ok{
   return "Select a command\\n\\n"+item.recipe.Label+"\\n"+item.recipe.Description+"\\n\\nPress Enter to configure its parameters."
  }
  return "No commands available."
 }
 var b strings.Builder
 fmt.Fprintf(&b,"%s\\n%s\\n\\n",m.selected.Label,m.selected.Description)
 if len(m.selected.Params)==0 {b.WriteString("No parameters required.\\n")}
 for i,p:=range m.selected.Params{
  val:=p.Default
  if i<len(m.values)&&m.values[i]!="" {val=m.values[i]}
  if m.stage==inputStage&&i==m.argument {val=m.entry.Value()}
  if p.Type=="password"&&val!=""{val="<hidden>"}
  if val==""{val="(not set)"}
  marker:="  ";if i==m.argument&&(m.stage==inputStage||m.stage==choiceStage){marker="> "}
  fmt.Fprintf(&b,"%s%s: %s\\n",marker,p.Label,val)
 }
 b.WriteString("\\nCommand line:\\n")
 b.WriteString(m.displayedCommand()+"\\n\\n")
 switch m.stage{
 case inputStage:
  p:=m.selected.Params[m.argument]
  fmt.Fprintf(&b,"Parameter %d/%d: %s\\n",m.argument+1,len(m.selected.Params),p.Label)
  b.WriteString(m.entry.View()+"\\n\\nEnter next · Esc back")
 case choiceStage:
  b.WriteString(m.choices.View()+"\\nEnter select · Esc back")
 case confirmationStage:
  b.WriteString("Ready to run.\\nEnter / y execute · e edit · Esc cancel")
 case runningStage:
  b.WriteString("Running command...")
 default:
  b.WriteString("Select another command or press Enter to configure.")
 }
 if m.err!=""{b.WriteString("\\n\\n"+m.err)}
 return b.String()
}
func(m appModel) View()string{
 leftWidth,rightWidth:=m.panelWidths()
 left:=lipgloss.NewStyle().Width(leftWidth).MaxWidth(leftWidth).Render(m.recipes.View())
 rightStyle:=lipgloss.NewStyle().Width(rightWidth-4).MaxWidth(rightWidth-4).Padding(1,1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63"))
 right:=rightStyle.Render(m.rightPanel())
 footer:="↑/↓ browse · / search · Enter select/run · Esc back · q quit"
 if m.chooser {footer="Choose recipe for just --choose · Enter select · q quit"}
 return lipgloss.JoinHorizontal(lipgloss.Top,left,right)+"\\n"+footer
}
func runTUI(file string,catalog Catalog)error{
 if len(catalog.Recipes)==0{return errors.New("no recipes found")}
 if !isTerminal(){return errors.New("requires interactive terminal; use --list")}
 _,err:=tea.NewProgram(newModel(file,catalog,false),tea.WithAltScreen()).Run()
 return err
}
func isTerminal()bool{info,err:=os.Stdin.Stat();return err==nil&&(info.Mode()&os.ModeCharDevice)!=0}
func runChooser(catalog Catalog)error{
 if isTerminal(){return errors.New("--chooser expects recipe names from stdin (just --choose)")}
 scanner:=bufio.NewScanner(os.Stdin);names:=map[string]bool{}
 for scanner.Scan(){names[strings.TrimSpace(scanner.Text())]=true}
 if err:=scanner.Err();err!=nil{return err}
 allowed:=[]Recipe{};for _,recipe:=range catalog.Recipes{if names[recipe.Name]{allowed=append(allowed,recipe)}}
 if len(allowed)==0{return errors.New("no matching recipes received from just --choose")}
 tty,err:=os.OpenFile("/dev/tty",os.O_RDWR,0)
 if err!=nil{return fmt.Errorf("chooser requires a controlling terminal: %w",err)}
 defer tty.Close()
 final,err:=tea.NewProgram(newModel("",Catalog{Recipes:allowed},true),tea.WithInput(tty),tea.WithOutput(tty),tea.WithAltScreen()).Run()
 if err!=nil{return err}
 if choice:=final.(appModel).chosen;choice!=""{fmt.Println(choice)}
 return nil
}
