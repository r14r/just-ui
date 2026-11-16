package main

import (
 "bufio"
 "embed"
 "errors"
 "flag"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "sort"
 "strings"
)

//go:embed VERSION
var versionFS embed.FS

type Param struct {
 Name string `json:"name"`
 Label string `json:"label"`
 Description string `json:"description,omitempty"`
 Min string `json:"min,omitempty"`
 Max string `json:"max,omitempty"`
 Type string `json:"type"`
 Required bool `json:"required"`
 Default string `json:"default,omitempty"`
 Options []string `json:"options,omitempty"`
}
type Recipe struct {
 Name string `json:"name"`
 Label string `json:"label"`
 Group string `json:"group"`
 Description string `json:"description"`
 Confirm bool `json:"confirm"`
 Params []Param `json:"params"`
}
type Catalog struct { Recipes []Recipe `json:"recipes"` }
var header = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)(\s+[^:]*)?:\s*(?:#.*)?$`)
var argToken = regexp.MustCompile(`(?:"[^"]*"|'[^']*'|[^\s]+)`)
var groupAttr = regexp.MustCompile(`\bgroup\(['"]([^'"]+)['"]\)`)

// parseFile parses ordinary root-level recipes and nearby @ui annotations.
// For execution and syntax validation, just remains authoritative.
func parseFile(path string) (Catalog,error) {
 f,err:=os.Open(path);if err!=nil{return Catalog{},err};defer f.Close()
 recipes:=[]Recipe{};comments:=[]string{};attrs:=[]string{}
 sc:=bufio.NewScanner(f);sc.Buffer(make([]byte,4096),1024*1024)
 for sc.Scan(){
  line:=sc.Text();trim:=strings.TrimSpace(line)
  if strings.HasPrefix(line,"#"){comments=append(comments,strings.TrimSpace(strings.TrimPrefix(line,"#")));continue}
  if strings.HasPrefix(line,"[")&&strings.HasSuffix(trim,"]"){attrs=append(attrs,trim);continue}
  if trim==""{continue}
  if strings.HasPrefix(line," ")||strings.HasPrefix(line,"\t"){continue}
  m:=header.FindStringSubmatch(line)
  if m==nil{comments=nil;attrs=nil;continue}
  r:=Recipe{Name:m[1],Label:m[1],Group:"Other"}
  for _,a:=range attrs{if g:=groupAttr.FindStringSubmatch(a);len(g)>1{r.Group=g[1]}}
  configured:=map[string]Param{}
  for _,c:=range comments{
   if !strings.HasPrefix(c,"@ui "){if r.Description==""{r.Description=c};continue}
   meta:=strings.TrimPrefix(c,"@ui ");fields:=strings.Fields(meta)
   if len(fields)==0{continue}
   value:=func(prefix string)string{return strings.TrimSpace(strings.TrimPrefix(meta,prefix))}
   switch fields[0]{
   case "group":r.Group=value("group")
   case "label":r.Label=value("label")
   case "description":r.Description=value("description")
   case "confirm":r.Confirm=true
   case "param":
    if len(fields)<2{continue}
    p:=Param{Name:fields[1],Label:fields[1],Type:"text"}
    for _,f:=range fields[2:]{
     pair:=strings.SplitN(f,"=",2)
     if len(pair)==1{if f=="required"{p.Required=true};continue}
     val:=strings.Trim(pair[1],`"'`)
     switch pair[0]{
     case "type":p.Type=val
     case "label":p.Label=strings.ReplaceAll(val,"_"," ")
     case "description":p.Description=strings.ReplaceAll(val,"_"," ")
     case "min":p.Min=val
     case "max":p.Max=val
     case "default":p.Default=val
     case "options":p.Options=strings.Split(val,"|")
     }
    }
    configured[p.Name]=p
   }
  }
  for _,arg:=range argToken.FindAllString(strings.TrimSpace(m[2]),-1){
   if strings.HasPrefix(arg,"+")||strings.HasPrefix(arg,"*"){continue}
   name,def,hasDefault:=strings.Cut(arg,"=")
   if !hasDefault{name=arg}
   def=strings.Trim(def,`"'`)
   if name==""{continue}
   p,exists:=configured[name];if !exists{p=Param{Name:name,Label:name,Type:"text"}}
   p.Required=!hasDefault&&p.Default==""
   if p.Default==""{p.Default=def}
   r.Params=append(r.Params,p)
  }
  recipes=append(recipes,r);comments=nil;attrs=nil
 }
 if err:=sc.Err();err!=nil{return Catalog{},err}
 sort.SliceStable(recipes,func(i,j int)bool{if recipes[i].Group==recipes[j].Group{return recipes[i].Name<recipes[j].Name};return recipes[i].Group<recipes[j].Group})
 return Catalog{Recipes:recipes},nil
}
func main() {
 if err:=run(os.Args[1:]);err!=nil {fmt.Fprintln(os.Stderr,"just-ui:",err);os.Exit(1)}
}
func run(args []string) error {
 flags:=flag.NewFlagSet("just-ui",flag.ContinueOnError)
 justfile:=flags.String("justfile","","Path to Justfile (default: search current directory and parents)")
 versionFlag:=flags.Bool("version",false,"Print version")
 listFlag:=flags.Bool("list",false,"List recipes without TUI")
 chooser:=flags.Bool("chooser",false,"Chooser mode for JUST_CHOOSER (stdin names, stdout selected recipe)")
 if err:=flags.Parse(args);err!=nil{if errors.Is(err,flag.ErrHelp){return nil};return err}
 if *versionFlag {b,_:=versionFS.ReadFile("VERSION");fmt.Println(strings.TrimSpace(string(b)));return nil}
 if _,err:=exec.LookPath("just");err!=nil{return errors.New("just executable missing from PATH")}
 file,err:=resolveJustfile(*justfile);if err!=nil{return err}
 if output,err:=exec.Command("just","--justfile",file,"--summary").CombinedOutput();err!=nil{return fmt.Errorf("invalid Justfile: %w\n%s",err,output)}
 catalog,err:=parseFile(file);if err!=nil{return err}
 if *listFlag{for _,r:=range catalog.Recipes{fmt.Printf("%-20s %-25s %s\n",r.Group,r.Name,r.Description)};return nil}
 if *chooser{return runChooser(catalog)}
 return runTUI(file,catalog)
}
func resolveJustfile(given string)(string,error){
 if given!=""{abs,err:=filepath.Abs(given);if err!=nil{return "",err};if _,err=os.Stat(abs);err!=nil{return "",err};return abs,nil}
 current,err:=os.Getwd();if err!=nil{return "",err}
 for {
  for _,name:=range []string{"justfile","Justfile",".justfile"}{
   path:=filepath.Join(current,name);if info,err:=os.Stat(path);err==nil&&!info.IsDir(){return path,nil}
  }
  parent:=filepath.Dir(current);if parent==current{break};current=parent
 }
 return "",errors.New("no Justfile found; use --justfile PATH")
}
