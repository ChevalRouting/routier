import{c as a,aF as c}from"./index-DKMmAMRG.js";import{j as n}from"./vendor-xyflow-BJ9fLDTq.js";/**
 * @license lucide-react v0.447.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const f=a("Pencil",[["path",{d:"M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z",key:"1a8usu"}],["path",{d:"m15 5 4 4",key:"1mk7zo"}]]);function u(e){return n.jsx(c,{...e,hideNullValues:!0})}const i=["version","hostname","interfaces","tunnels","routing","wireguard","nftables","sysctl","dns","users","services","logging"];function l(e){return e?[...i.filter(s=>s in e),...Object.keys(e).filter(s=>!i.includes(s))].map(s=>[s,e[s]]):[]}function p({data:e,defaultOpen:s=!0}){const r=l(e);return r.length===0?n.jsx("div",{className:"text-sm text-muted-foreground",children:"empty"}):n.jsx("div",{className:"md:columns-2 [column-gap:0.75rem]",children:r.map(([t,o])=>n.jsx("div",{className:"mb-3 break-inside-avoid",children:n.jsx(u,{name:t,value:o,defaultOpen:s})},t))})}export{p as C,f as P,u as S,l as o};
