
const $=id=>document.getElementById(id);let items=[],editing=-1,dirty=false;let groups=["Nicht zugeordnet"];
function status(msg,kind='ok'){const el=$('status');el.textContent=msg;el.className='status '+kind;}
function markDirty(){dirty=true;$('save').textContent='Änderungen speichern *';}
function suggestedAddresses(){const entries=items.filter(x=>x.kind==='entry'&&x.ipv4);if(!entries.length)return {ipv4:'',ipv6:''};const counts={};for(const it of entries){const parts=it.ipv4.split('.');if(parts.length===4&&parts.every(x=>/^\d+$/.test(x))){const prefix=parts.slice(0,3).join('.');counts[prefix]=Math.max(counts[prefix]||0,Number(parts[3]));}}const prefix=Object.keys(counts).sort((a,b)=>counts[b]-counts[a])[0];if(!prefix)return {ipv4:'',ipv6:''};const octet=Math.min(254,counts[prefix]+1);const ipv4=prefix+'.'+octet;const matching=entries.find(x=>x.ipv4.startsWith(prefix+'.')&&x.ipv6);let ipv6='';if(matching){const m=matching.ipv6.match(/^(.*:)(\d+)$/);if(m)ipv6=m[1]+octet;}return {ipv4,ipv6};}
function showModal(index=-1){editing=index;const suggested=suggestedAddresses();const it=index>=0?items[index]:{mac:'',ipv4:suggested.ipv4,ipv6:suggested.ipv6,hostname:'',lease:'infinite',group:$('groupFilter').value||'Nicht zugeordnet'};$('modalTitle').textContent=(uiText[langSelect.value]||uiText.de)[index>=0?'editTitle':'newTitle'];for(const k of ['mac','ipv4','ipv6','hostname','lease'])$(k).value=it[k]||'';$('group').value=it.group||'Nicht zugeordnet';$('modal').classList.add('open');$('mac').focus();applyUILanguage(langSelect.value);}
function hideModal(){$('modal').classList.remove('open');}
function esc(s){return String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}
function groupLabel(g){const l=langSelect?.value||'de';return g==='Nicht zugeordnet'?(l==='en'?'Unassigned':l==='es'?'Sin asignar':g):g;}
function refreshGroups(){const found=items.filter(x=>x.kind==='entry').map(x=>x.group||'Nicht zugeordnet');groups=[...new Set([...groups,'Nicht zugeordnet',...found])].filter(Boolean);for(const id of ['groupFilter','group']){const el=$(id),old=el.value;el.innerHTML=(id==='groupFilter'?'<option value="">Alle Gruppen</option>':'')+groups.map(g=>'<option value="'+esc(g)+'">'+esc(groupLabel(g))+'</option>').join('');if([...el.options].some(o=>o.value===old))el.value=old;}}
function render(){
 // Duplikate sichtbar markieren, damit sie nicht nur in einem Dialog erscheinen.
 const duplicateIndexes=new Set();
 for(const d of findDuplicateEntries()){for(const it of d.matches){const idx=items.indexOf(it);if(idx>=0)duplicateIndexes.add(idx);}}
 // Die Gruppenliste ist die kanonische Reihenfolge. Geräte werden in ihren
 // bestehenden Array-Slots nach dieser Reihenfolge sortiert, damit Aktionen
 // und Kommentare ihre gültigen Indizes behalten.
 refreshGroups();
 const entrySlots=[],entries=[],groupOrder=new Map(groups.map((g,i)=>[g,i]));
 items.forEach((it,i)=>{if(it.kind==='entry'){entrySlots.push(i);entries.push(it);}});
 entries.sort((a,b)=>(groupOrder.get(a.group||'Nicht zugeordnet')??9999)-(groupOrder.get(b.group||'Nicht zugeordnet')??9999));
 entrySlots.forEach((slot,i)=>{items[slot]=entries[i];});
 const q=$('search').value.toLowerCase(),filter=$('groupFilter').value,tbody=$('rows');tbody.innerHTML='';let visible=0,lastRenderedGroup=null;items.forEach((it,i)=>{if(it.kind==='entry'){it.group=it.group||'Nicht zugeordnet';const hay=[it.mac,it.ipv4,it.ipv6,it.hostname,it.lease,it.group].join(' ').toLowerCase();if((q&&!hay.includes(q))||(filter&&it.group!==filter))return;visible++;if(it.group!==lastRenderedGroup){const heading=document.createElement('tr');heading.className='group-heading';heading.draggable=true;heading.dataset.groupName=it.group;heading.innerHTML='<td colspan="7"><div class="group-title"><span class="group-pill">'+esc(it.group)+'</span><div class="group-order-actions"><button type="button" class="btn group-move" data-move-group-up="'+esc(it.group)+'" title="">↑</button><button type="button" class="btn group-move" data-move-group-down="'+esc(it.group)+'" title="">↓</button><span class="group-drag-hint" aria-hidden="true">⋮⋮</span></div></div></td>';tbody.appendChild(heading);lastRenderedGroup=it.group;}const tr=document.createElement('tr');tr.className='group-row'+(duplicateIndexes.has(i)?' duplicate-entry':'');tr.dataset.itemIndex=i;tr.dataset.entryGroup=it.group;tr.draggable=true;if(duplicateIndexes.has(i))tr.title=duplicateRowTitle(i);tr.innerHTML='<td>'+esc(it.mac)+'</td><td>'+esc(it.ipv4)+'</td><td>'+esc(it.ipv6)+'</td><td>'+esc(it.hostname)+'</td><td>'+esc(it.lease)+'</td><td class="group-cell"><select aria-label="Gruppe ändern" data-group="'+i+'">'+groups.map(g=>'<option '+(g===it.group?'selected':'')+' value="'+esc(g)+'">'+esc(groupLabel(g))+'</option>').join('')+'</select></td><td class="actions entry-actions"><button class="btn entry-move" data-move-entry-up="'+i+'" title="" aria-label="">↑</button><button class="btn entry-move" data-move-entry-down="'+i+'" title="" aria-label="">↓</button><span class="entry-drag-hint" aria-hidden="true">⋮⋮</span><button class="btn" data-edit="'+i+'">Bearbeiten</button><button class="btn danger" data-del="'+i+'">Löschen</button></td>';tbody.appendChild(tr);}else{const raw=it.raw||'';if(it.kind==='blank')return;if(it.kind==='comment'&&!$('showComments').checked)return;if(it.kind==='comment'&&raw.trim().startsWith('# LeaseEditor-Gruppe:'))return;if(it.kind==='comment'&&/^[#\s=_-]+$/.test(raw.trim())&&raw.trim().length>=5)return;if(q&&!raw.toLowerCase().includes(q))return;const tr=document.createElement('tr');tr.className=it.kind==='comment'?'comment-row':'section';tr.dataset.itemIndex=i;if(it.kind==='comment')tr.draggable=true;const label=it.kind==='blank'?'':esc(raw);const controls=it.kind==='comment'?'<td><div class="comment-actions"><button class="btn comment-move" data-move-comment-up="'+i+'" title="Kommentar nach oben verschieben" aria-label="Kommentar nach oben verschieben">↑</button><button class="btn comment-move" data-move-comment-down="'+i+'" title="Kommentar nach unten verschieben" aria-label="Kommentar nach unten verschieben">↓</button><button class="btn" data-edit-comment="'+i+'">Bearbeiten</button><button class="btn danger" data-del-comment="'+i+'">Kommentar löschen</button></div></td>':'';tr.innerHTML='<td colspan="'+(it.kind==='comment'?'6':'7')+'">'+label+(it.kind==='raw'?(rowText[langSelect?.value||'de']||rowText.de).unknown:'')+'</td>'+controls;tbody.appendChild(tr);}});$('empty').hidden=visible!==0;const ct=uiText[langSelect?.value||'de']||uiText.de;$('count').textContent=visible+' '+ct.count+' · '+groups.length+' '+ct.groups;applyDynamicLabels();}
function normalizeDuplicateValue(field,value){
 let v=String(value??'').trim().toLowerCase();
 if(field==='mac')v=v.replace(/[-.\s]/g,':').replace(/:{2,}/g,':');
 if(field==='ipv4')v=v.replace(/\s+/g,'');
 if(field==='ipv6')v=v.replace(/\s+/g,'');
 return v;
}
function findDuplicateEntries(candidate=null,skipIndex=-1){
 const lang=langSelect?.value||'de';
 const hostnameLabel=lang==='en'?'Hostname':lang==='es'?'Nombre de host':'Hostname';
 const fields=[['mac','MAC'],['ipv4','IPv4'],['ipv6','IPv6'],['hostname',hostnameLabel]];
 const all=items.map((it,index)=>({it,index})).filter(x=>x.it&&x.it.kind==='entry');
 const found=[];
 for(const [field,label] of fields){
  if(candidate){
   const value=normalizeDuplicateValue(field,candidate[field]);
   if(!value)continue;
   const matches=all.filter(x=>x.index!==skipIndex&&normalizeDuplicateValue(field,x.it[field])===value);
   if(matches.length)found.push({field,label,value:String(candidate[field]??'').trim(),matches:matches.map(x=>x.it)});
  }else{
   const seen=new Map();
   for(const x of all){
    const value=normalizeDuplicateValue(field,x.it[field]);if(!value)continue;
    if(seen.has(value)){
     let duplicate=found.find(d=>d.field===field&&normalizeDuplicateValue(field,d.value)===value);
     if(!duplicate){duplicate={field,label,value:String(x.it[field]??'').trim(),matches:[seen.get(value),x.it]};found.push(duplicate);}
     else if(!duplicate.matches.includes(x.it))duplicate.matches.push(x.it);
    }else seen.set(value,x.it);
   }
  }
 }
 return found;
}
function duplicateRowTitle(index){
 const entry=items[index];if(!entry)return '';
 const lang=langSelect?.value||'de';
 const hostnameLabel=lang==='en'?'Hostname':lang==='es'?'Nombre de host':'Hostname';
 const fields=[['mac','MAC'],['ipv4','IPv4'],['ipv6','IPv6'],['hostname',hostnameLabel]];
 const dup=fields.filter(([field])=>{const v=normalizeDuplicateValue(field,entry[field]);return v&&items.some((it,i)=>i!==index&&it.kind==='entry'&&normalizeDuplicateValue(field,it[field])===v);}).map(([field,label])=>label+': '+entry[field]);
 const prefix=lang==='en'?'Duplicate value: ':lang==='es'?'Valor duplicado: ':'Doppelter Wert: ';
 return dup.length?prefix+dup.join(', '):'';
}

function duplicateMessage(duplicates){const lang=langSelect?.value||'de';const lines=duplicates.map(d=>`${d.label}: ${d.value}`);const intro=lang==='en'?'Duplicate entry detected. The following value is already assigned:':lang==='es'?'Se ha detectado una entrada duplicada. El siguiente valor ya está asignado:':'Doppelter Eintrag gefunden. Folgender Wert ist bereits vergeben:';return intro+'\n'+lines.join('\n');}
async function load(){if(dirty&&!await askConfirm(dt('discard')))return;try{const r=await fetch('/api/reservations');if(!r.ok)throw new Error(await r.text());const d=await r.json();items=d.items||[];groups=[...new Set(['Nicht zugeordnet',...items.filter(x=>x.kind==='entry').map(x=>x.group||'Nicht zugeordnet')])];dirty=false;$('save').textContent=(uiText[langSelect?.value||'de']||uiText.de).save;render();const duplicates=findDuplicateEntries();if(duplicates.length){status((langSelect.value==='en'?'Duplicate entries found: ':langSelect.value==='es'?'Se encontraron entradas duplicadas: ':'Doppelte Einträge gefunden: ')+duplicates.map(d=>d.label+' '+d.value).join(' · '),'err');await inform(duplicateMessage(duplicates));}else{status((uiText[langSelect.value]||uiText.de).loaded,'ok');}}catch(e){status((uiText[langSelect.value]||uiText.de).loadFailed+e.message,'err');}}
let draggedGroupName=null,groupDropTarget=null;
function reorderGroup(name,delta){const from=groups.indexOf(name),to=from+delta;if(from<0||to<0||to>=groups.length)return;const [g]=groups.splice(from,1);groups.splice(to,0,g);markDirty();render();}
$('rows').addEventListener('dragstart',e=>{
 const heading=e.target.closest('tr.group-heading[draggable="true"]');
 if(heading){draggedGroupName=heading.dataset.groupName;groupDropTarget=null;e.dataTransfer.effectAllowed='move';e.dataTransfer.setData('text/plain','group:'+draggedGroupName);heading.classList.add('dragging-group');return;}
});
$('rows').addEventListener('dragover',e=>{
 if(draggedGroupName===null)return;
 const target=e.target.closest('tr.group-heading[data-group-name]');
 if(!target||target.dataset.groupName===draggedGroupName)return;
 e.preventDefault();e.dataTransfer.dropEffect='move';groupDropTarget=target.dataset.groupName;
 document.querySelectorAll('.group-heading.group-drop-before,.group-heading.group-drop-after').forEach(r=>r.classList.remove('group-drop-before','group-drop-after'));
 const rect=target.getBoundingClientRect();target.classList.add(e.clientY<rect.top+rect.height/2?'group-drop-before':'group-drop-after');
});
$('rows').addEventListener('drop',e=>{
 if(draggedGroupName===null)return;
 const target=e.target.closest('tr.group-heading[data-group-name]');e.preventDefault();
 const from=groups.indexOf(draggedGroupName),to=target?groups.indexOf(target.dataset.groupName):-1;
 const before=target&&e.clientY<target.getBoundingClientRect().top+target.getBoundingClientRect().height/2;
 document.querySelectorAll('.dragging-group,.group-drop-before,.group-drop-after').forEach(r=>r.classList.remove('dragging-group','group-drop-before','group-drop-after'));
 draggedGroupName=null;groupDropTarget=null;
 if(from<0||to<0||from===to)return;
 const [g]=groups.splice(from,1);let at=groups.indexOf(target.dataset.groupName);if(!before)at++;groups.splice(Math.max(0,Math.min(groups.length,at)),0,g);markDirty();render();
});
$('rows').addEventListener('dragend',()=>{draggedGroupName=null;groupDropTarget=null;document.querySelectorAll('.dragging-group,.group-drop-before,.group-drop-after').forEach(r=>r.classList.remove('dragging-group','group-drop-before','group-drop-after'));});
// Geräte innerhalb ihrer aktuellen Gruppe sortieren. Die Geräte werden auf den
// vorhandenen Eintrags-Slots neu angeordnet, damit Kommentare/sonstige Zeilen
// und ihre Positionen in der Datei nicht versehentlich mitverschoben werden.
function entrySlotsForGroup(group){return items.map((it,index)=>({it,index})).filter(x=>x.it&&x.it.kind==='entry'&&(x.it.group||'Nicht zugeordnet')===group).map(x=>x.index);}
function moveEntryToPosition(fromIndex,targetIndex,before=true){
 const source=items[fromIndex],target=items[targetIndex];
 if(!source||source.kind!=='entry'||!target||target.kind!=='entry')return false;
 const group=source.group||'Nicht zugeordnet';
 if((target.group||'Nicht zugeordnet')!==group)return false;
 const slots=entrySlotsForGroup(group);
 const ordered=slots.map(i=>items[i]);
 const from=ordered.indexOf(source),to=ordered.indexOf(target);
 if(from<0||to<0||from===to)return false;
 ordered.splice(from,1);
 let at=ordered.indexOf(target);
 if(!before)at++;
 ordered.splice(Math.max(0,Math.min(ordered.length,at)),0,source);
 slots.forEach((slot,i)=>{items[slot]=ordered[i];});
 markDirty();render();return true;
}
function moveEntryByDelta(index,delta){
 const it=items[index];if(!it||it.kind!=='entry')return;
 const group=it.group||'Nicht zugeordnet',slots=entrySlotsForGroup(group),pos=slots.indexOf(index),next=pos+delta;
 if(pos<0||next<0||next>=slots.length)return;
 const a=items[slots[pos]],b=items[slots[next]];items[slots[pos]]=b;items[slots[next]]=a;markDirty();render();
}
let draggedEntryIndex=null,entryDropTargetIndex=null,entryDropBefore=true;
$('rows').addEventListener('dragstart',e=>{
 const row=e.target.closest('tr.group-row[draggable="true"]');if(!row)return;
 // Avoid starting a row drag when interacting with a control inside it.
 if(e.target.closest('button,select,input,a')){e.preventDefault();return;}
 draggedEntryIndex=Number(row.dataset.itemIndex);entryDropTargetIndex=null;
 e.dataTransfer.effectAllowed='move';e.dataTransfer.setData('text/plain','entry:'+draggedEntryIndex);row.classList.add('dragging-entry');
});
$('rows').addEventListener('dragover',e=>{
 if(draggedEntryIndex===null)return;
 const target=e.target.closest('tr.group-row[data-item-index]');
 if(!target||Number(target.dataset.itemIndex)===draggedEntryIndex||target.dataset.entryGroup!==items[draggedEntryIndex]?.group){
  document.querySelectorAll('.group-row.entry-drop-before,.group-row.entry-drop-after').forEach(r=>r.classList.remove('entry-drop-before','entry-drop-after'));entryDropTargetIndex=null;return;
 }
 e.preventDefault();e.dataTransfer.dropEffect='move';entryDropTargetIndex=Number(target.dataset.itemIndex);
 const rect=target.getBoundingClientRect();entryDropBefore=e.clientY<rect.top+rect.height/2;
 document.querySelectorAll('.group-row.entry-drop-before,.group-row.entry-drop-after').forEach(r=>r.classList.remove('entry-drop-before','entry-drop-after'));
 target.classList.add(entryDropBefore?'entry-drop-before':'entry-drop-after');
});
$('rows').addEventListener('drop',e=>{
 if(draggedEntryIndex===null)return;
 e.preventDefault();const from=draggedEntryIndex,to=entryDropTargetIndex,before=entryDropBefore;
 document.querySelectorAll('.dragging-entry,.entry-drop-before,.entry-drop-after').forEach(r=>r.classList.remove('dragging-entry','entry-drop-before','entry-drop-after'));
 draggedEntryIndex=null;entryDropTargetIndex=null;
 if(to!==null)moveEntryToPosition(from,to,before);
});
$('rows').addEventListener('dragend',()=>{draggedEntryIndex=null;entryDropTargetIndex=null;document.querySelectorAll('.dragging-entry,.entry-drop-before,.entry-drop-after').forEach(r=>r.classList.remove('dragging-entry','entry-drop-before','entry-drop-after'));});

let draggedCommentIndex=null,dropPlaceholder=null,dropBefore=true,dropTargetIndex=null;
function clearDropMarker(removePlaceholder=true){
  if(removePlaceholder&&dropPlaceholder){dropPlaceholder.remove();dropPlaceholder=null;}
  document.querySelectorAll('.drop-before,.drop-after').forEach(r=>r.classList.remove('drop-before','drop-after'));
}
function makeDropPlaceholder(){
  const row=document.createElement('tr');
  row.className='comment-drop-placeholder';
  row.innerHTML='<td colspan="7"></td>';
  return row;
}
$('rows').addEventListener('dragstart',e=>{
  const row=e.target.closest('tr.comment-row[draggable="true"]');
  if(!row)return;
  draggedCommentIndex=Number(row.dataset.itemIndex);
  dropTargetIndex=null;
  e.dataTransfer.effectAllowed='move';
  e.dataTransfer.setData('text/plain',String(draggedCommentIndex));
  row.classList.add('dragging-comment');
  dropPlaceholder=makeDropPlaceholder();
});
$('rows').addEventListener('dragover',e=>{
  if(draggedCommentIndex===null)return;
  e.preventDefault();
  e.dataTransfer.dropEffect='move';
  const target=e.target.closest('tr[data-item-index]');
  if(!target||target.classList.contains('dragging-comment')){
    dropTargetIndex=null;
    clearDropMarker(false);
    if(dropPlaceholder&&dropPlaceholder.parentNode)dropPlaceholder.remove();
    return;
  }
  const targetIndex=Number(target.dataset.itemIndex);
  if(!Number.isInteger(targetIndex)){dropTargetIndex=null;return;}
  dropTargetIndex=targetIndex;
  const rect=target.getBoundingClientRect();
  dropBefore=e.clientY<rect.top+rect.height/2;
  clearDropMarker(false);
  if(!dropPlaceholder)dropPlaceholder=makeDropPlaceholder();
  if(dropBefore){
    target.parentNode.insertBefore(dropPlaceholder,target);
    target.classList.add('drop-before');
  }else{
    target.parentNode.insertBefore(dropPlaceholder,target.nextSibling);
    target.classList.add('drop-after');
  }
});
$('rows').addEventListener('drop',e=>{
  if(draggedCommentIndex===null)return;
  e.preventDefault();
  const from=draggedCommentIndex,to=dropTargetIndex,before=dropBefore;
  clearDropMarker();
  draggedCommentIndex=null;
  dropTargetIndex=null;
  document.querySelectorAll('.dragging-comment').forEach(row=>row.classList.remove('dragging-comment'));
  if(to===null||from===to||!items[from]||items[from].kind!=='comment'){render();return;}
  const [moved]=items.splice(from,1);
  let insertAt=to;
  if(from<to)insertAt--;
  if(!before)insertAt++;
  insertAt=Math.max(0,Math.min(items.length,insertAt));
  items.splice(insertAt,0,moved);
  markDirty();
  render();
});
$('rows').addEventListener('dragend',()=>{
  draggedCommentIndex=null;
  dropTargetIndex=null;
  clearDropMarker();
  document.querySelectorAll('.dragging-comment').forEach(row=>row.classList.remove('dragging-comment'));
});
$('rows').addEventListener('change',e=>{const sel=e.target.closest('[data-group]');if(sel){items[Number(sel.dataset.group)].group=sel.value;markDirty();render();}});$('rows').addEventListener('click',async e=>{const gu=e.target.closest('[data-move-group-up]'),gd=e.target.closest('[data-move-group-down]');if(gu||gd){reorderGroup((gu||gd).dataset[gu?'moveGroupUp':'moveGroupDown'],gu?-1:1);return;}const eu=e.target.closest('[data-move-entry-up]'),edn=e.target.closest('[data-move-entry-down]');if(eu||edn){moveEntryByDelta(Number((eu||edn).dataset[eu?'moveEntryUp':'moveEntryDown']),eu?-1:1);return;}const ed=e.target.closest('[data-edit]'),del=e.target.closest('[data-del]'),ec=e.target.closest('[data-edit-comment]'),dc=e.target.closest('[data-del-comment]'),mu=e.target.closest('[data-move-comment-up]'),md=e.target.closest('[data-move-comment-down]');if(mu||md){const i=Number((mu||md).dataset[mu?'moveCommentUp':'moveCommentDown']),to=i+(mu?-1:1);if(to>=0&&to<items.length){const [moved]=items.splice(i,1);items.splice(to,0,moved);markDirty();render();}}if(ed)showModal(Number(ed.dataset.edit));if(del){const i=Number(del.dataset.del);if(await askConfirm(dt('removeReservation'))){items.splice(i,1);markDirty();render();}}if(ec){const i=Number(ec.dataset.editComment),it=items[i];const value=await askPrompt(dt('editComment'),it.raw||'#');if(value===null)return;const v=value.trim();if(!v.startsWith('#')){await inform(dt('commentHash'));return;}it.raw=v;markDirty();render();}if(dc){const i=Number(dc.dataset.delComment);if(await askConfirm(dt('removeComment'))){items.splice(i,1);markDirty();render();}}});
$('addComment').onclick=async()=>{const value=await askPrompt(dt('newComment'),'# -');if(value===null)return;let v=value.trim();if(!v.startsWith('#')){await inform(dt('commentHash'));return;}if(v==='#')v='# ';items.unshift({kind:'comment',raw:v});markDirty();render();status((uiText[langSelect.value]||uiText.de).commentAdded,'ok');};
$('form').addEventListener('submit',async e=>{e.preventDefault();const it={kind:'entry',mac:$('mac').value.trim(),ipv4:$('ipv4').value.trim(),ipv6:$('ipv6').value.trim(),hostname:$('hostname').value.trim(),lease:$('lease').value.trim(),group:$('group').value||'Nicht zugeordnet'};const duplicates=findDuplicateEntries(it,editing);if(duplicates.length){await inform(duplicateMessage(duplicates));return;}if(editing>=0)items[editing]=it;else items.push(it);markDirty();hideModal();render();});
$('addGroup').onclick=async()=>{const name=await askPrompt(dt('newGroup'));if(!name||!name.trim())return;const g=name.trim();if(groups.includes(g)){await inform(dt('groupExists'));return;}groups.push(g);$('groupFilter').value=g;markDirty();render();};$('renameGroup').onclick=async()=>{const old=$('groupFilter').value;if(!old){await inform(dt('chooseGroup'));return;}if(old==='Nicht zugeordnet'){await inform(dt('cannotRenameUnassigned'));return;}const name=await askPrompt(dt('renameGroup'),old);if(!name||!name.trim())return;const g=name.trim();if(groups.includes(g)){await inform(dt('groupExists'));return;}items.forEach(it=>{if(it.kind==='entry'&&(it.group||'Nicht zugeordnet')===old)it.group=g;});groups=groups.map(x=>x===old?g:x);$('groupFilter').value=g;markDirty();render();};$('deleteGroup').onclick=async()=>{const old=$('groupFilter').value;if(!old){await inform(dt('chooseGroup'));return;}if(old==='Nicht zugeordnet'){await inform(dt('cannotDeleteUnassigned'));return;}if(!await askConfirm(dt('deleteGroup').replace('{group}',old)))return;items.forEach(it=>{if(it.kind==='entry'&&it.group===old)it.group='Nicht zugeordnet';});groups=groups.filter(x=>x!==old);$('groupFilter').value='';markDirty();render();};$('groupFilter').onchange=render;$('cancel').onclick=hideModal;$('modal').addEventListener('click',e=>{if(e.target===$('modal'))hideModal()});$('add').onclick=()=>showModal();$('reload').onclick=load;$('search').oninput=render;$('showComments').addEventListener('change',render);
$('restartDnsmasq').onclick=async()=>{if(!await askConfirm(langSelect.value==='en'?'Restart the dnsmasq service now?':langSelect.value==='es'?'¿Reiniciar el servicio dnsmasq ahora?':'Den dnsmasq-Dienst jetzt neu starten?'))return;const b=$('restartDnsmasq');b.disabled=true;try{const r=await fetch('/api/dnsmasq/restart',{method:'POST'});const d=await r.json();if(!r.ok)throw new Error(d.error||'Restart failed');status(langSelect.value==='en'?'dnsmasq restarted successfully.':langSelect.value==='es'?'dnsmasq se reinició correctamente.':'dnsmasq wurde erfolgreich neu gestartet.','ok');}catch(e){status(e.message||'dnsmasq restart failed','err')}finally{b.disabled=false;}};
$('save').onclick=async()=>{const duplicates=findDuplicateEntries();if(duplicates.length&&!await askConfirm(dt('duplicateSave')+'\n\n'+duplicateMessage(duplicates)))return;if(!await askConfirm(dt('overwrite')))return;try{$('save').disabled=true;const r=await fetch('/api/reservations',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({items})});if(!r.ok){let msg=await r.text();try{msg=JSON.parse(msg).error||msg}catch{}throw new Error(msg)}dirty=false;$('save').textContent=(uiText[langSelect.value]||uiText.de).save;status((uiText[langSelect.value]||uiText.de).saved,'ok');await load();}catch(e){status('Speichern fehlgeschlagen: '+e.message,'err')}finally{$('save').disabled=false;}};

const dialogText={
 de:{ok:'OK',cancel:'Abbrechen',removeReservation:'Diese Reservierung entfernen?',editComment:'Kommentar bearbeiten:',newComment:'Neuen Kommentar eingeben (mit # am Anfang):',commentHash:'Ein Kommentar muss mit # beginnen.',removeComment:'Diesen Kommentar wirklich löschen?',newGroup:'Name der neuen Gruppe:',groupExists:'Diese Gruppe existiert bereits.',chooseGroup:'Bitte zuerst eine konkrete Gruppe auswählen.',cannotRenameUnassigned:'Die Gruppe „Nicht zugeordnet“ kann nicht umbenannt werden.',renameGroup:'Neuer Gruppenname:',cannotDeleteUnassigned:'Diese Gruppe kann nicht gelöscht werden.',deleteGroup:'Gruppe „{group}“ löschen? Zugeordnete Geräte werden „Nicht zugeordnet“.',discard:'Ungespeicherte Änderungen verwerfen und Datei neu laden?',duplicateSave:'Es sind doppelte Einträge vorhanden. Trotzdem speichern?',overwrite:'Die bestehende dnsmasq.dhcphosts-Datei jetzt überschreiben? Eine .bak-Sicherung wird angelegt.',invalidTitle:'Hinweis',confirmTitle:'Bitte bestätigen',promptTitle:'Eingabe'},
 en:{ok:'OK',cancel:'Cancel',removeReservation:'Remove this reservation?',editComment:'Edit comment:',newComment:'Enter a new comment (must start with #):',commentHash:'A comment must start with #.',removeComment:'Really delete this comment?',newGroup:'Name of the new group:',groupExists:'This group already exists.',chooseGroup:'Please select a specific group first.',cannotRenameUnassigned:'The “Unassigned” group cannot be renamed.',renameGroup:'New group name:',cannotDeleteUnassigned:'This group cannot be deleted.',deleteGroup:'Delete group “{group}”? Assigned devices will become “Unassigned”.',discard:'Discard unsaved changes and reload the file?',duplicateSave:'Duplicate entries exist. Save anyway?',overwrite:'Overwrite the existing dnsmasq.dhcphosts file now? A .bak backup will be created.',invalidTitle:'Notice',confirmTitle:'Please confirm',promptTitle:'Input'},
 es:{ok:'Aceptar',cancel:'Cancelar',removeReservation:'¿Eliminar esta reserva?',editComment:'Editar comentario:',newComment:'Introduce un comentario nuevo (debe empezar por #):',commentHash:'El comentario debe empezar por #.',removeComment:'¿Eliminar este comentario?',newGroup:'Nombre del nuevo grupo:',groupExists:'Este grupo ya existe.',chooseGroup:'Selecciona primero un grupo concreto.',cannotRenameUnassigned:'No se puede cambiar el nombre del grupo «Sin asignar».',renameGroup:'Nuevo nombre del grupo:',cannotDeleteUnassigned:'No se puede eliminar este grupo.',deleteGroup:'¿Eliminar el grupo «{group}»? Los dispositivos asignados pasarán a «Sin asignar».',discard:'¿Descartar los cambios sin guardar y volver a cargar el archivo?',duplicateSave:'Hay entradas duplicadas. ¿Guardar de todos modos?',overwrite:'¿Sobrescribir ahora el archivo dnsmasq.dhcphosts existente? Se creará una copia de seguridad .bak.',invalidTitle:'Aviso',confirmTitle:'Confirmar',promptTitle:'Entrada'}
};
function dt(key){return (dialogText[langSelect?.value||'de']||dialogText.de)[key]||key;}
function appDialog({title='',message='',value='',mode='alert',placeholder=''}){return new Promise(resolve=>{const root=$('appDialog'),form=$('appDialogForm'),input=$('appDialogInput');$('appDialogTitle').textContent=title;$('appDialogMessage').textContent=message;input.value=value;input.placeholder=placeholder;input.hidden=mode!=='prompt';$('appDialogCancel').hidden=mode==='alert';$('appDialogOK').textContent=dt('ok');$('appDialogCancel').textContent=dt('cancel');root.classList.add('open');const finish=result=>{root.classList.remove('open');form.onsubmit=null;$('appDialogCancel').onclick=null;root.onclick=null;resolve(result);};form.onsubmit=e=>{e.preventDefault();finish(mode==='prompt'?input.value:true);};$('appDialogCancel').onclick=()=>finish(mode==='prompt'?null:false);root.onclick=e=>{if(e.target===root)finish(mode==='prompt'?null:false);};if(mode==='prompt')input.focus();else $('appDialogOK').focus();});}
const askConfirm=message=>appDialog({title:dt('confirmTitle'),message,mode:'confirm'});
const askPrompt=(message,value='')=>appDialog({title:dt('promptTitle'),message,value,mode:'prompt'});
const inform=message=>appDialog({title:dt('invalidTitle'),message,mode:'alert'});

window.addEventListener('beforeunload',e=>{if(dirty){e.preventDefault();e.returnValue='';}});load();

// Translate only application-owned UI labels. Never walk all text nodes:
// DHCP comments, hostnames, log/output text and user data must remain byte-for-byte visible.
const rowText={
 de:{edit:'Bearbeiten',delete:'Löschen',deleteComment:'Kommentar löschen',changeGroup:'Gruppe ändern',moveUp:'Kommentar nach oben verschieben',moveDown:'Kommentar nach unten verschieben',groupUp:'Gruppe nach oben verschieben',groupDown:'Gruppe nach unten verschieben',entryUp:'Gerät innerhalb der Gruppe nach oben verschieben',entryDown:'Gerät innerhalb der Gruppe nach unten verschieben',unknown:' (unbekannte Zeile – wird unverändert erhalten)',count:'Reservierung(en)',groups:'Gruppe(n)'},
 en:{edit:'Edit',delete:'Delete',deleteComment:'Delete comment',changeGroup:'Change group',moveUp:'Move comment up',moveDown:'Move comment down',groupUp:'Move group up',groupDown:'Move group down',entryUp:'Move device up within group',entryDown:'Move device down within group',unknown:' (unknown line – preserved unchanged)',count:'reservation(s)',groups:'group(s)'},
 es:{edit:'Editar',delete:'Eliminar',deleteComment:'Eliminar comentario',changeGroup:'Cambiar grupo',moveUp:'Mover comentario hacia arriba',moveDown:'Mover comentario hacia abajo',groupUp:'Mover grupo hacia arriba',groupDown:'Mover grupo hacia abajo',entryUp:'Mover dispositivo hacia arriba dentro del grupo',entryDown:'Mover dispositivo hacia abajo dentro del grupo',unknown:' (línea desconocida; se conserva sin cambios)',count:'reserva(s)',groups:'grupo(s)'}
};
function applyDynamicLabels(){const lang=langSelect.value||'de',t=rowText[lang]||rowText.de;document.querySelectorAll('[data-edit]').forEach(b=>b.textContent=t.edit);document.querySelectorAll('[data-del]').forEach(b=>b.textContent=t.delete);document.querySelectorAll('[data-edit-comment]').forEach(b=>b.textContent=t.edit);document.querySelectorAll('[data-del-comment]').forEach(b=>b.textContent=t.deleteComment);document.querySelectorAll('[data-group]').forEach(s=>s.setAttribute('aria-label',t.changeGroup));document.querySelectorAll('[data-move-comment-up]').forEach(b=>{b.title=t.moveUp;b.setAttribute('aria-label',t.moveUp)});document.querySelectorAll('[data-move-comment-down]').forEach(b=>{b.title=t.moveDown;b.setAttribute('aria-label',t.moveDown)});document.querySelectorAll('[data-move-group-up]').forEach(b=>{b.title=t.groupUp;b.setAttribute('aria-label',t.groupUp)});document.querySelectorAll('[data-move-group-down]').forEach(b=>{b.title=t.groupDown;b.setAttribute('aria-label',t.groupDown)});document.querySelectorAll('[data-move-entry-up]').forEach(b=>{b.title=t.entryUp;b.setAttribute('aria-label',t.entryUp)});document.querySelectorAll('[data-move-entry-down]').forEach(b=>{b.title=t.entryDown;b.setAttribute('aria-label',t.entryDown)});const count=$('count');if(count){const m=count.textContent.match(/^(\d+) .+ · (\d+) .+$/);if(m){const u=uiText[lang]||uiText.de;count.textContent=`${m[1]} ${u.count} · ${m[2]} ${u.groups}`;}}}
const langSelect=$('languageSelect'),themeBtn=$('themeToggle');
const uiText={
 de:{count:'Reservierung(en)',groups:'Gruppe(n)',editTitle:'Reservierung bearbeiten',newTitle:'Neue Reservierung',macLabel:'MAC-Adresse *',ipv4Label:'IPv4-Adresse *',ipv6Label:'IPv6-Adresse (optional)',hostnameLabel:'Hostname',groupField:'Gruppe',leaseLabel:'Lease-Dauer (leer lassen, wenn nicht gesetzt)',empty:'Keine passenden Einträge gefunden.',search:'Suche nach MAC, IP, Hostname oder Kommentar…',reload:'Neu laden',add:'+ Reservierung',addGroup:'+ Gruppe',addComment:'+ Kommentar',showComments:'Kommentare anzeigen',save:'Änderungen speichern',restart:'dnsmasq neu starten',groupLabel:'Gruppe anzeigen:',allGroups:'Alle Gruppen',rename:'Gruppe umbenennen',deleteGroup:'Gruppe löschen',cancel:'Abbrechen',apply:'Übernehmen',loaded:'Datei geladen.',headers:['MAC-Adresse','IPv4-Adresse','IPv6-Adresse','Hostname','Lease','Gruppe','Aktionen'],warning:'Direktes Speichern: Änderungen schreiben die Datei auf dem Server. Vor jedem Speichern wird eine Sicherung als .bak erstellt. Verwende das Tool nur in einem vertrauenswürdigen Netzwerk.',tagline:'DHCP-Leases · Geräteerkennung · Hersteller · Netzwerkstatus',fileLabel:'Datei:',footer:'LeaseEditor · Änderungen werden erst mit „Änderungen speichern“ auf die Datei geschrieben. Kommentare und unbekannte Zeilen werden beim Speichern beibehalten.',loaded:'Datei geladen.',loading:'Lade Datei …',commentAdded:'Kommentar eingefügt. Zum dauerhaften Übernehmen bitte Änderungen speichern.',saved:'Datei erfolgreich gespeichert. Prüfe anschließend, ob dnsmasq die Änderungen neu laden muss.',loadFailed:'Laden fehlgeschlagen: ',saveFailed:'Speichern fehlgeschlagen: '},
 en:{count:'reservation(s)',groups:'group(s)',editTitle:'Edit reservation',newTitle:'New reservation',macLabel:'MAC address *',ipv4Label:'IPv4 address *',ipv6Label:'IPv6 address (optional)',hostnameLabel:'Hostname',groupField:'Group',leaseLabel:'Lease duration (leave blank if unset)',empty:'No matching entries found.',search:'Search MAC, IP, hostname or comment…',reload:'Reload',add:'+ Reservation',addGroup:'+ Group',addComment:'+ Comment',showComments:'Show comments',save:'Save changes',restart:'Restart dnsmasq',groupLabel:'Show group:',allGroups:'All groups',rename:'Rename group',deleteGroup:'Delete group',cancel:'Cancel',apply:'Apply',loaded:'File loaded.',headers:['MAC address','IPv4 address','IPv6 address','Hostname','Lease','Group','Actions'],warning:'Direct saving: changes write the file on the server. A .bak backup is created before each save. Use this tool only on a trusted network.',tagline:'DHCP leases · Device discovery · Vendor · Network status',fileLabel:'File:',footer:'LeaseEditor · Changes are written to the file only when you select “Save changes”. Comments and unknown lines are preserved when saving.',loaded:'File loaded.',loading:'Loading file …',commentAdded:'Comment added. Select Save changes to write it permanently.',saved:'File saved successfully. Check whether dnsmasq needs to reload the changes.',loadFailed:'Loading failed: ',saveFailed:'Saving failed: '},
 es:{count:'reserva(s)',groups:'grupo(s)',editTitle:'Editar reserva',newTitle:'Nueva reserva',macLabel:'Dirección MAC *',ipv4Label:'Dirección IPv4 *',ipv6Label:'Dirección IPv6 (opcional)',hostnameLabel:'Nombre de host',groupField:'Grupo',leaseLabel:'Duración de concesión (dejar vacío si no se establece)',empty:'No se encontraron entradas coincidentes.',search:'Buscar MAC, IP, nombre de host o comentario…',reload:'Recargar',add:'+ Reserva',addGroup:'+ Grupo',addComment:'+ Comentario',showComments:'Mostrar comentarios',save:'Guardar cambios',restart:'Reiniciar dnsmasq',groupLabel:'Mostrar grupo:',allGroups:'Todos los grupos',rename:'Renombrar grupo',deleteGroup:'Eliminar grupo',cancel:'Cancelar',apply:'Aceptar',loaded:'Archivo cargado.',headers:['Dirección MAC','Dirección IPv4','Dirección IPv6','Nombre de host','Concesión','Grupo','Acciones'],warning:'Guardado directo: los cambios escriben el archivo en el servidor. Antes de cada guardado se crea una copia .bak. Usa esta herramienta solo en una red de confianza.',tagline:'Concesiones DHCP · Detección de dispositivos · Fabricante · Estado de la red',fileLabel:'Archivo:',footer:'LeaseEditor · Los cambios se escriben en el archivo al seleccionar «Guardar cambios». Los comentarios y las líneas desconocidas se conservan al guardar.',loaded:'Archivo cargado.',loading:'Cargando archivo …',commentAdded:'Comentario añadido. Selecciona Guardar cambios para guardarlo permanentemente.',saved:'Archivo guardado correctamente. Comprueba si dnsmasq necesita recargar los cambios.',loadFailed:'Error al cargar: ',saveFailed:'Error al guardar: '}
};
function applyUILanguage(lang){const t=uiText[lang]||uiText.de;document.documentElement.lang=lang;const tagline=document.querySelector('.tagline');if(tagline)tagline.textContent=t.tagline;const meta=document.querySelector('.meta');if(meta&&meta.lastChild&&meta.lastChild.nodeType===3)meta.lastChild.textContent=' · '+t.fileLabel+' ';const foot=document.querySelector('.foot');if(foot){const link=foot.querySelector('a');foot.textContent=t.footer+' · ';if(link)foot.appendChild(link);}if($('count').textContent==='Lade Datei …'||$('count').textContent==='Loading file …'||$('count').textContent==='Cargando archivo …')$('count').textContent=t.loading;$('modalTitle').textContent=editing>=0?t.editTitle:t.newTitle;const fieldLabels={'mac':t.macLabel,'ipv4':t.ipv4Label,'ipv6':t.ipv6Label,'hostname':t.hostnameLabel,'group':t.groupField,'lease':t.leaseLabel};for(const [id,txt] of Object.entries(fieldLabels)){const label=document.querySelector('label[for="'+id+'"]');if(label)label.textContent=txt;}$('empty').textContent=t.empty;const group=$('group');if(group.options.length===0){}$('search').placeholder=t.search;$('reload').textContent=t.reload;$('add').textContent=t.add;$('addGroup').textContent=t.addGroup;$('addComment').textContent=t.addComment;$('showCommentsLabel').textContent=t.showComments;$('restartDnsmasq').textContent=t.restart;if(!$('save').textContent.includes('*'))$('save').textContent=t.save;document.querySelector('label[for="groupFilter"]').textContent=t.groupLabel;$('groupFilter').options[0].textContent=t.allGroups;$('renameGroup').textContent=t.rename;$('deleteGroup').textContent=t.deleteGroup;$('cancel').textContent=t.cancel;document.querySelector('#form button[type="submit"]').textContent=t.apply;document.querySelectorAll('thead th').forEach((th,i)=>{if(t.headers[i])th.textContent=t.headers[i]});const warning=document.querySelector('.warning');if(warning){warning.replaceChildren();const strong=document.createElement('strong');strong.textContent=lang==='en'?'Direct saving:':lang==='es'?'Guardado directo:':'Direktes Speichern:';warning.append(strong,document.createTextNode(t.warning.slice(strong.textContent.length)));}const s=$('status').textContent;if(['Datei geladen.','File loaded.','Archivo cargado.'].includes(s))status(t.loaded,'ok');document.querySelectorAll('#group option').forEach(o=>{if(o.value==='Nicht zugeordnet')o.textContent=lang==='en'?'Unassigned':lang==='es'?'Sin asignar':'Nicht zugeordnet';});applyDynamicLabels();themeBtn.setAttribute('aria-label',lang==='en'?'Switch theme':lang==='es'?'Cambiar tema':'Design umschalten');}
langSelect.value=localStorage.getItem('leaseeditor-language')||'de';langSelect.addEventListener('change',()=>{localStorage.setItem('leaseeditor-language',langSelect.value);applyUILanguage(langSelect.value);render();});
function setUITTheme(v){document.body.classList.toggle('light-theme',v==='light');themeBtn.textContent=v==='light'?'☾':'☼';const lang=langSelect.value||'de';themeBtn.setAttribute('aria-label',lang==='en'?(v==='light'?'Switch to dark theme':'Switch to light theme'):lang==='es'?(v==='light'?'Cambiar al tema oscuro':'Cambiar al tema claro'):(v==='light'?'Zum dunklen Design wechseln':'Zum hellen Design wechseln'));localStorage.setItem('leaseeditor-theme',v);}setUITTheme(localStorage.getItem('leaseeditor-theme')||'dark');themeBtn.addEventListener('click',()=>setUITTheme(document.body.classList.contains('light-theme')?'dark':'light'));applyUILanguage(langSelect.value);
