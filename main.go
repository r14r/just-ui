package main

import (
 "bufio"
 "context"
 "crypto/rand"
 "embed"
 "encoding/hex"
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "html/template"
 "log"
 "net"
 "net/http"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "sort"
 "strconv"
 "strings"
 "sync"
 "time"
)

//go:embed VERSION
var versionFS embed.FS

type Param struct {
 Name string `json:"name"`
 Label string `json:"label"`
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
type Job struct {
 ID string `json:"id"`
 Recipe string `json:"recipe"`
 Args []string `json:"args"`
 Output string `json:"output"`
 Finished bool `json:"finished"`
 ExitCode int `json:"exitCode"`
 Started time.Time `json:"started"`
}
type App struct {
 File string
 Catalog Catalog
 Mu sync.Mutex
 Jobs map[string]*Job
 Token string
}
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
func sendJSON(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json; charset=utf-8");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
func(a *App) findRecipe(name string)(Recipe,bool){for _,r:=range a.Catalog.Recipes{if r.Name==name{return r,true}};return Recipe{},false}
func(a *App) handler(w http.ResponseWriter,r *http.Request){
 if r.Method!="GET"&&(r.Header.Get("Origin")!="http://"+r.Host||r.Header.Get("X-Just-UI-Token")!=a.Token){http.Error(w,"invalid origin or token",403);return}
 switch{
 case r.URL.Path=="/"&&r.Method=="GET":
  w.Header().Set("Content-Type","text/html; charset=utf-8");_ = page.Execute(w,map[string]string{"Token":a.Token})
 case r.URL.Path=="/api/recipes"&&r.Method=="GET":sendJSON(w,200,a.Catalog)
 case r.URL.Path=="/api/run"&&r.Method=="POST":
  var request struct{Recipe string `json:"recipe"`;Values map[string]string `json:"values"`;Confirm bool `json:"confirm"`}
  r.Body=http.MaxBytesReader(w,r.Body,64<<10)
  if err:=json.NewDecoder(r.Body).Decode(&request);err!=nil{http.Error(w,"invalid JSON",400);return}
  recipe,ok:=a.findRecipe(request.Recipe);if !ok{http.Error(w,"unknown recipe",404);return}
  if recipe.Confirm&&!request.Confirm{http.Error(w,"confirmation required",400);return}
  args:=[]string{"--justfile",a.File,recipe.Name}
  for _,p:=range recipe.Params{
   v,exists:=request.Values[p.Name];if !exists{v=p.Default}
   if p.Required&&strings.TrimSpace(v)==""{http.Error(w,"missing: "+p.Name,400);return}
   if p.Type=="select"&&len(p.Options)>0{
    match:=false;for _,opt:=range p.Options{if v==opt{match=true;break}}
    if !match{http.Error(w,"invalid option: "+p.Name,400);return}
   }
   if p.Type=="number"&&v!=""{if _,err:=strconv.ParseFloat(v,64);err!=nil{http.Error(w,"invalid number",400);return}}
   args=append(args,v)
  }
  id:=randomID();job:=&Job{ID:id,Recipe:recipe.Name,Args:args,Started:time.Now()}
  a.Mu.Lock();a.Jobs[id]=job;a.Mu.Unlock()
  go a.execute(job);sendJSON(w,202,map[string]string{"id":id})
 case strings.HasPrefix(r.URL.Path,"/api/jobs/")&&r.Method=="GET":
  id:=strings.TrimPrefix(r.URL.Path,"/api/jobs/")
  a.Mu.Lock();job,ok:=a.Jobs[id]
  if !ok{a.Mu.Unlock();http.NotFound(w,r);return}
  snapshot:=*job;a.Mu.Unlock();sendJSON(w,200,snapshot)
 default:http.NotFound(w,r)
 }
}
func(a *App) execute(job *Job){
 ctx,cancel:=context.WithTimeout(context.Background(),30*time.Minute);defer cancel()
 cmd:=exec.CommandContext(ctx,"just",job.Args...);cmd.Dir=filepath.Dir(a.File)
 out,err:=cmd.CombinedOutput();status:=0
 if err!=nil{
  status=1;var ee *exec.ExitError
  if errors.As(err,&ee){status=ee.ExitCode()}
  out=append(out,[]byte("\nERROR: "+err.Error())...)
 }
 if len(out)>1<<20{out=append(out[:1<<20],[]byte("\n... output truncated")...)}
 a.Mu.Lock();job.Output=string(out);job.ExitCode=status;job.Finished=true;a.Mu.Unlock()
}
func randomID()string{var b [12]byte;_,_=rand.Read(b[:]);return hex.EncodeToString(b[:])}
func main(){
 file:=flag.String("justfile","justfile","Path to Justfile")
 listen:=flag.String("listen","127.0.0.1:0","Local web interface address")
 noBrowser:=flag.Bool("no-browser",false,"Do not open browser")
 versionFlag:=flag.Bool("version",false,"Print version and exit")
 flag.Parse()
 if *versionFlag{b,_:=versionFS.ReadFile("VERSION");fmt.Println(strings.TrimSpace(string(b)));return}
 abs,err:=filepath.Abs(*file);if err!=nil{log.Fatal(err)}
 if _,err:=exec.LookPath("just");err!=nil{log.Fatal("install just first: ",err)}
 catalog,err:=parseFile(abs);if err!=nil{log.Fatal(err)}
 validate:=exec.Command("just","--justfile",abs,"--summary")
 if b,err:=validate.CombinedOutput();err!=nil{log.Fatalf("invalid justfile: %v\n%s",err,b)}
 host,_,err:=net.SplitHostPort(*listen);if err!=nil{log.Fatal(err)}
 if host!="127.0.0.1"&&host!="localhost"&&host!="::1"{log.Fatal("only loopback addresses are allowed")}
 ln,err:=net.Listen("tcp",*listen);if err!=nil{log.Fatal(err)}
 app:=&App{File:abs,Catalog:catalog,Jobs:map[string]*Job{},Token:randomID()}
 url:="http://"+ln.Addr().String();fmt.Println("just-ui "+url)
 if !*noBrowser{
  go func(){
   time.Sleep(250*time.Millisecond)
   opener:="xdg-open";if _,err:=os.Stat("/usr/bin/open");err==nil{opener="open"}
   _=exec.Command(opener,url).Run()
  }()
 }
 mux:=http.NewServeMux();mux.HandleFunc("/",app.handler)
 log.Fatal((&http.Server{Handler:mux,ReadHeaderTimeout:5*time.Second}).Serve(ln))
}
var page=template.Must(template.New("page").Parse(indexHTML))
