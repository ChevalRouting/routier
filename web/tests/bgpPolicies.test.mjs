import test from 'node:test'
import assert from 'node:assert/strict'
import { migrateBGPPolicies } from '../src/lib/bgpPolicies.ts'

test('legacy policies become explicit neighbor filters without replacing overrides', () => {
  const bgp = { address_families: { 'ipv4-unicast': { route_map_in: 'EXTERNAL', route_map_out: 'EXTERNAL', maximum_paths: 2 } }, neighbors: [
    { address: '192.0.2.1' },
    { address: '192.0.2.2', address_families: { 'ipv4-unicast': { route_map_in: 'CUSTOM', disabled: true } } },
    { address: '2001:db8::1' },
  ] }
  const migrated = migrateBGPPolicies(bgp)
  assert.deepEqual(migrated.address_families['ipv4-unicast'], { maximum_paths: 2 })
  assert.deepEqual(migrated.neighbors[0].address_families['ipv4-unicast'], { route_map_in: 'EXTERNAL', route_map_out: 'EXTERNAL' })
  assert.deepEqual(migrated.neighbors[1].address_families['ipv4-unicast'], { route_map_in: 'CUSTOM', route_map_out: 'EXTERNAL', disabled: true })
  assert.deepEqual(migrated.neighbors[2].address_families['ipv6-unicast'], {})
  assert.equal(bgp.address_families['ipv4-unicast'].route_map_in, 'EXTERNAL')
  assert.equal(migrateBGPPolicies(migrated), migrated)
})

test('new configs and VPN import/export route maps remain unchanged', () => {
  const bgp = { address_families: { 'ipv4-unicast': { route_map_vpn_import: 'VPN-IN', route_map_vpn_export: 'VPN-OUT' } }, neighbors: [] }
  assert.equal(migrateBGPPolicies(bgp), bgp)
})
