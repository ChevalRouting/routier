import { readdir, readFile } from 'node:fs/promises'
import { join, relative } from 'node:path'
import parser from '@typescript-eslint/parser'
import { commentViolations } from './rules.mjs'

async function sourceFiles(directory) {
  const files = []
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) {
      if (relative('src', path) !== 'api') files.push(...await sourceFiles(path))
    } else if (/\.(ts|tsx)$/.test(path)) {
      files.push(path)
    }
  }
  return files
}

let failed = false
for (const path of await sourceFiles('src')) {
  const source = parser.parse(await readFile(path, 'utf8'), { filePath: path, comment: true, loc: true, ecmaFeatures: { jsx: true } })
  for (const comment of commentViolations(source.comments)) {
    console.error(`${path}:${comment.loc.start.line}: maco/no-comments: Use names and structure instead of prose comments.`)
    failed = true
  }
}
process.exitCode = failed ? 1 : 0
