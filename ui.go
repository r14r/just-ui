package main

const indexHTML = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>just-ui</title>
<style>
:root{font-family:system-ui,-apple-system,sans-serif;color-scheme:dark;background:#111720;color:#eef2fa}
*{box-sizing:border-box}body{margin:0}
header{padding:18px 24px;border-bottom:1px solid #30394a;font-size:22px;font-weight:700}
small{color:#acbad0;font-size:13px}main{display:grid;grid-template-columns:270px minmax(0,1fr);min-height:calc(100vh - 65px)}
aside{padding:18px;border-right:1px solid #30394a}section{padding:30px;max-width:1000px;width:100%}
h2{margin:0 0 8px}.group{margin:18px 0 8px;color:#93b7ff;font-size:12px;text-transform:uppercase;letter-spacing:.1em}
.item{display:block;width:100%;padding:10px 12px;margin:3px 0;text-align:left;background:transparent;color:inherit;border:0;border-radius:8px;cursor:pointer}
.item.active,.item:hover{background:#26354d}input,select{background:#1c2739;border:1px solid #445571;border-radius:7px;padding:10px;color:inherit;width:100%;font:inherit}
label{display:block;margin:18px 0 4px;font-weight:650}
.actions{display:flex;align-items:center;gap:14px;margin-top:24px;flex-wrap:wrap}
button.run{background:#6797ef;border:0;border-radius:8px;padding:12px 20px;color:#071329;font-weight:750;cursor:pointer}
button.run:disabled{opacity:.45}pre{background:#0a0e16;padding:20px;border:1px solid #33415a;white-space:pre-wrap;overflow-wrap:anywhere;border-radius:9px;min-height:150px}
code{color:#b7d4ff;overflow-wrap:anywhere}.muted{color:#acbad0}#search{margin-bottom:12px}
@media(max-width:750px){main{display:block}aside{border-right:0;border-bottom:1px solid #30394a}section{padding:20px}}
</style></head>
<body>
<header>just-ui <small>· Justfile command runner</small></header>
<main>
<aside><input id="search" placeholder="Search recipes..." aria-label="Search recipes"><div id="menu"></div></aside>
<section>
<h2 id="title">Select a command</h2><p id="description" class="muted">Choose a recipe on the left.</p>
<form id="form"><div id="fields"></div><div class="actions"><button class="run" id="run" type="submit" disabled>Run command</button><code id="preview"></code></div></form>
<p id="status" aria-live="polite"></p><pre id="output">Output will appear here after execution.</pre>
</section>
</main>
<script>
const token='{{.Token}}';
let recipes=[],chosen=null;
const el=id=>document.getElementById(id);
function element(tag,value,cls){const e=document.createElement(tag);e.textContent=value;if(cls)e.className=cls;return e}
async function load(){
 const resp=await fetch('/api/recipes');
 if(!resp.ok)throw Error(await resp.text());
 recipes=(await resp.json()).recipes||[];renderMenu();
}
function renderMenu(){
 const query=el('search').value.toLowerCase(),menu=el('menu');menu.replaceChildren();
 let group='';
 for(const recipe of recipes.filter(r=>(r.name+' '+r.label+' '+r.group+' '+r.description).toLowerCase().includes(query))){
  if(recipe.group!==group){group=recipe.group;menu.append(element('div',group,'group'))}
  const button=element('button',recipe.label,'item'+(chosen&&chosen.name===recipe.name?' active':''));
  button.type='button';button.onclick=()=>selectRecipe(recipe);menu.append(button);
 }
}
function selectRecipe(recipe){
 chosen=recipe;renderMenu();el('title').textContent=recipe.label;
 el('description').textContent=recipe.description||recipe.name;
 const container=el('fields');container.replaceChildren();
 for(const param of recipe.params){
  const label=element('label',param.label+(param.required?' *':''));label.htmlFor='param-'+param.name;container.append(label);
  let control;
  if(param.type==='select'){
   control=document.createElement('select');
   for(const option of param.options||[]){const o=document.createElement('option');o.value=option;o.textContent=option;control.append(o)}
  }else{
   control=document.createElement('input');
   control.type=({number:'number',password:'password',checkbox:'checkbox',path:'text'}[param.type]||'text');
   if(param.type==='checkbox')control.style.width='auto';
  }
  control.id='param-'+param.name;control.required=param.required;
  if(param.type==='checkbox')control.checked=param.default==='true';
  else control.value=param.default||'';
  control.addEventListener('input',updatePreview);
  control.addEventListener('change',updatePreview);
  container.append(control);
 }
 el('run').disabled=false;el('output').textContent='Output will appear here after execution.';
 el('status').textContent='';updatePreview();
}
function values(){
 const result={};for(const param of chosen.params){
  const control=el('param-'+param.name);
  result[param.name]=param.type==='checkbox'?(control.checked?'true':'false'):control.value;
 }return result;
}
function updatePreview(){
 if(!chosen)return;
 const v=values();
 el('preview').textContent='just '+chosen.name+' '+chosen.params.map(p=>p.type==='password'?'••••':JSON.stringify(v[p.name])).join(' ');
}
el('search').addEventListener('input',renderMenu);
el('form').addEventListener('submit',async event=>{
 event.preventDefault();if(!chosen)return;
 const recipe=chosen;let confirmed=false;
 if(recipe.confirm){confirmed=confirm('Run '+recipe.label+'? This command requires confirmation.');if(!confirmed)return}
 el('run').disabled=true;el('status').textContent='Running…';el('output').textContent='';
 try{
  const response=await fetch('/api/run',{method:'POST',headers:{'Content-Type':'application/json','X-Just-UI-Token':token},body:JSON.stringify({recipe:recipe.name,values:values(),confirm:confirmed})});
  if(!response.ok)throw Error(await response.text());
  const result=await response.json();
  while(true){
   await new Promise(resolve=>setTimeout(resolve,650));
   const statusResponse=await fetch('/api/jobs/'+encodeURIComponent(result.id));
   if(!statusResponse.ok)throw Error(await statusResponse.text());
   const job=await statusResponse.json();
   if(job.finished){el('output').textContent=job.output||'(no output)';el('status').textContent='Exit code: '+job.exitCode;break}
  }
 }catch(err){el('status').textContent='Error: '+err.message}
 finally{el('run').disabled=false}
});
load().catch(err=>el('status').textContent='Error: '+err.message);
</script>
</body></html>`
