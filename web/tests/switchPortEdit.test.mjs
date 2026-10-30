import test from 'node:test'
import assert from 'node:assert/strict'
import { applyEdit, initialEdit, releasedBonds } from '../src/lib/switchPortEdit.ts'

const state = {
  ports: [
    { name: 'ether1', type: 'ether', mode: 'access', pvid: 10, tagged: [] },
    { name: 'ether2', type: 'ether', mode: 'bond-member', bond: 'bond1', pvid: 0, tagged: [] },
    { name: 'ether3', type: 'ether', mode: 'bond-member', bond: 'bond1', pvid: 0, tagged: [] },
    { name: 'ether4', type: 'ether', mode: 'access', pvid: 20, tagged: [] },
    { name: 'bond1', type: 'bond', mode: 'trunk', pvid: 1, tagged: [10, 20] },
  ],
  bonds: [{ name: 'bond1', members: ['ether2', 'ether3'], mode: '802.3ad' }],
}

const definition = {
  model: 'crs',
  host: '10.0.0.2',
  ports: { ether1: { mode: 'access', vlan: 10 } },
  bonds: { bond1: { members: ['ether2', 'ether3'], port: { mode: 'trunk', tagged: [10, 20] } } },
}

test('a bond member opens on its bond', () => {
  const edit = initialEdit({ ports: ['ether2'], definition, state })
  assert.equal(edit.role, 'bond')
  assert.equal(edit.bondName, 'bond1')
  assert.deepEqual(edit.members, ['ether2', 'ether3'])
  assert.deepEqual(edit.port, { mode: 'trunk', tagged: [10, 20] })
})

test('turning a bond member into a port drops the bond in the same draft', () => {
  const context = { ports: ['ether2'], definition, state }
  const edit = { ...initialEdit(context), role: 'port', port: { mode: 'access', vlan: 30 } }
  const next = applyEdit(edit, context)
  assert.deepEqual(next.bonds, {})
  assert.deepEqual(next.ports.ether2, { mode: 'access', vlan: 30 })
  assert.deepEqual(releasedBonds(edit, context), [{ name: 'bond1', remaining: ['ether3'], recreated: false }])
})

test('changing members keeps the bond name and recreates it', () => {
  const context = { ports: ['ether2'], definition, state }
  const edit = { ...initialEdit(context), members: ['ether2', 'ether4'] }
  const next = applyEdit(edit, context)
  assert.deepEqual(next.bonds.bond1.members, ['ether2', 'ether4'])
  assert.deepEqual(releasedBonds(edit, context), [{ name: 'bond1', remaining: ['ether3'], recreated: true }])
})

test('editing the vlans of a bond keeps it', () => {
  const context = { ports: ['ether3'], definition, state }
  const edit = { ...initialEdit(context), port: { mode: 'access', vlan: 20 } }
  assert.deepEqual(releasedBonds(edit, context), [])
  assert.deepEqual(applyEdit(edit, context).bonds.bond1.port, { mode: 'access', vlan: 20 })
})

test('a new bond takes its members out of the ports', () => {
  const context = { ports: ['ether1', 'ether4'], definition, state }
  const edit = { ...initialEdit(context), role: 'bond' }
  const next = applyEdit(edit, context)
  assert.equal(edit.bondName, 'bond2')
  assert.deepEqual(next.ports, {})
  assert.deepEqual(next.bonds.bond2, { members: ['ether1', 'ether4'], mode: undefined, port: { mode: 'access', vlan: 10 } })
})
