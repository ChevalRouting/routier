import test from 'node:test'
import assert from 'node:assert/strict'
import { applyFailureFromResult, applyFailureFromError } from '../src/lib/applyFailure.ts'

test('artifact validation failure retains the exact run and file diagnostics', () => {
  const errors = [{ tool: 'nft', dest: '/etc/nftables/routier.nft', line: 12, message: 'syntax error' }]
  const failure = applyFailureFromResult({ status: 'validation_failed', bundleID: 'run-1', logID: 'run-1', errors })
  assert.equal(failure.message, 'syntax error')
  assert.equal(failure.logID, 'run-1')
  assert.equal(failure.bundleID, 'run-1')
  assert.deepEqual(failure.errors, errors)
})

test('runtime HTTP failure preserves diagnostics and the original error', () => {
  const failure = applyFailureFromError({ message: 'reload kea-dhcp4: exited 1', body: { result: {
    status: 'apply_failed', bundleID: 'run-2', logID: 'run-2', errors: [{ line: 0, message: 'reload kea-dhcp4: exited 1' }],
  } } })
  assert.equal(failure.message, 'reload kea-dhcp4: exited 1')
  assert.equal(failure.bundleID, 'run-2')
  assert.equal(failure.logID, 'run-2')
})

test('a failure without preserved artifacts still links to its log', () => {
  const failure = applyFailureFromResult({ status: 'apply_failed', logID: 'run-3' })
  assert.equal(failure.bundleID, undefined)
  assert.equal(failure.logID, 'run-3')
})

test('network and preflight failures never select an unrelated historical run', () => {
  assert.deepEqual(applyFailureFromError(new Error('connection dropped')), { message: 'connection dropped' })
})

test('only an applied status clears the failure', () => {
  assert.equal(applyFailureFromResult({ status: 'applied', snapID: 'snap' }), null)
  assert.ok(applyFailureFromResult({ status: 'unexpected' }))
})
