import{aY as l,l as h}from"./index-C4pV3ML2.js";class w extends Error{constructor(t){super(`Stream failed (HTTP ${t})`),this.status=t}}function p(e){return e.split(`
`).filter(t=>t.startsWith("data:")).map(t=>t.slice(5).trimStart()).join(`
`)}async function S(e,t,c,a){const r=await fetch(`${l()}${e}`,{headers:{Authorization:`Bearer ${h()??""}`},signal:t});if(!r.ok||!r.body)throw new w(r.status);a==null||a();const n=r.body.getReader(),d=new TextDecoder;let o="";for(;!t.aborted;){const{done:f,value:m}=await n.read();if(f)return;o+=d.decode(m,{stream:!0});const s=o.split(`

`);o=s.pop()??"";for(const u of s){const i=p(u);i&&c(i)}}}export{S as r};
