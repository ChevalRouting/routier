import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { RuleTester } from 'eslint'
import parser from '@typescript-eslint/parser'
import plugin, { commentViolations } from './rules.mjs'

RuleTester.describe = describe
RuleTester.it = it
RuleTester.itOnly = it.only

const tester = new RuleTester({
  languageOptions: { parser, ecmaVersion: 'latest', sourceType: 'module', parserOptions: { ecmaFeatures: { jsx: true } } },
})

for (const [name, cases] of Object.entries({
  'no-comments': {
    valid: ["const url = 'https://example.com'", "/// <reference types=\"vite/client\" />\nconst ready = true", '// @ts-expect-error required fixture\nconst ready = true', '// eslint-disable-next-line no-unused-vars\nconst ready = true'],
    invalid: [
      { code: '// prose\nconst ready = true', errors: [{ messageId: 'comment' }] },
      { code: 'const view = <div>{/* prose */}</div>', errors: [{ messageId: 'comment' }] },
    ],
  },
  'shared-controls': {
    valid: ['const view = <Select />', 'const view = <IntegerEntryRow />', 'const view = <input type="file" />', 'const view = <input type="hidden" />'],
    invalid: [
      { code: 'const view = <select />', errors: [{ messageId: 'select' }] },
      { code: 'const view = <input />', errors: [{ messageId: 'input' }] },
      { code: 'const view = <textarea />', errors: [{ messageId: 'textarea' }] },
      { code: 'const view = <input type="checkbox" />', errors: [{ messageId: 'checkbox' }] },
      { code: "const view = <input type={'number'} />", errors: [{ messageId: 'number' }] },
    ],
  },
  'semantic-palette': {
    valid: ['const view = <div className="bg-background text-success" />', 'const view = <svg><path fill="#16181d" /></svg>'],
    invalid: [
      { code: 'const view = <div className="dark:text-green-400" />', errors: [{ messageId: 'palette' }] },
      { code: 'const view = <div className="bg-white" />', errors: [{ messageId: 'palette' }] },
      { code: 'const view = <div className={`bg-[#101216]`} />', errors: [{ messageId: 'palette' }] },
      { code: "const theme = { background: '#101216' }", errors: [{ messageId: 'palette' }] },
    ],
  },
  'neutral-table-actions': {
    valid: ['const view = <table><tbody><tr><td><Button>Delete</Button></td></tr></tbody></table>', 'const view = <Dialog><Button variant="destructive">Confirm</Button></Dialog>'],
    invalid: [{ code: 'const view = <table><tbody><tr><td><Button variant="destructive">Delete</Button></td></tr></tbody></table>', errors: [{ messageId: 'neutral' }] }],
  },
  'named-handlers': {
    valid: ['const view = <Button onClick={submit} />', 'const view = <Button onClick={() => { setOpen(false); setError("") }} />'],
    invalid: [{ code: 'const view = <Button onClick={async () => { if (busy) return; await save(); setOpen(false) }} />', errors: [{ messageId: 'handler' }] }],
  },
  'named-types': {
    valid: ['interface Request { name: string }', 'type Request = { name: string }', 'function submit(value: Request) {}'],
    invalid: [
      { code: 'function submit(value: { name: string }) {}', errors: [{ messageId: 'type' }] },
      { code: 'type Requests = Array<{ name: string }>', errors: [{ messageId: 'type' }] },
    ],
  },
  'accessible-icon-actions': {
    valid: ['import { X } from "lucide-react"; const view = <Button title="Close" aria-label="Close"><X /></Button>', 'import { X } from "lucide-react"; const view = <Button><X />Close</Button>'],
    invalid: [
      { code: 'import { X } from "lucide-react"; const view = <Button><X /></Button>', errors: [{ messageId: 'label' }] },
      { code: 'import { X as Close } from "lucide-react"; const view = <button aria-label="Close"><Close /></button>', errors: [{ messageId: 'label' }] },
    ],
  },
})) {
  tester.run(name, plugin.rules[name], cases)
}

it('blanket disable comments cannot suppress the independent comment check', () => {
  const source = parser.parse('/* eslint-disable */\nconst ready = true', { comment: true, loc: true })
  assert.equal(commentViolations(source.comments).length, 1)
})
