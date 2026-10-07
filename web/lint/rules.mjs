function rule(messages, create) {
  return { meta: { type: 'suggestion', schema: [], messages }, create }
}

function attribute(node, name) {
  return node.attributes.find((entry) => entry.type === 'JSXAttribute' && entry.name.name === name)
}

function attributeValue(node, name) {
  const value = attribute(node, name)?.value
  return value?.type === 'Literal' ? value.value : value?.expression?.value
}

function jsxName(node) {
  return node.name.type === 'JSXIdentifier' ? node.name.name : ''
}

function insideTable(node) {
  for (let parent = node.parent; parent; parent = parent.parent) {
    if (parent.type === 'JSXElement' && ['table', 'thead', 'tbody', 'tr'].includes(jsxName(parent.openingElement))) return true
  }
  return false
}

function namedType(node) {
  return node.parent.type === 'TSTypeAliasDeclaration'
}

function requiredDirective(text) {
  return /^\/\s*<reference\b/.test(text.trim()) || /^\s*@ts-expect-error\b/.test(text) || /^\s*eslint-disable-(next-line|line)\b/.test(text)
}

export function commentViolations(comments) {
  return comments.filter((comment) => !requiredDirective(comment.value))
}

function svgArtwork(node) {
  if (node.parent?.type !== 'JSXAttribute' || !['fill', 'stroke'].includes(node.parent.name.name)) return false
  for (let parent = node.parent; parent; parent = parent.parent) {
    if (parent.type === 'JSXElement' && jsxName(parent.openingElement) === 'svg') return true
  }
  return false
}

function checkColor(context, node, text) {
  if (svgArtwork(node)) return
  if (/(?:^|[\s:'"`])(?:bg|text|border|ring|fill|stroke|accent)-(?:red|green|blue|yellow|amber|orange|emerald|rose|slate|gray|zinc|white|black)(?:-\d+)?(?=$|[\s/'"`])/.test(text) || /(?:#(?:[a-f\d]{6}|[a-f\d]{3})\b|\b(?:rgb|rgba|hsl|hsla)\()/i.test(text)) {
    context.report({ node, messageId: 'palette' })
  }
}

const noComments = rule({ comment: 'Use names and structure instead of prose comments.' }, (context) => ({
  Program() {
    for (const comment of commentViolations(context.sourceCode.getAllComments())) {
      context.report({ loc: comment.loc, messageId: 'comment' })
    }
  },
}))

const sharedControls = rule({
  select: 'Use the cheval-ui Select primitives.',
  input: 'Use the shared cheval-ui Input.',
  textarea: 'Use the shared cheval-ui Textarea.',
  checkbox: 'Use a shared cheval-ui checkbox or SwitchRow.',
  number: 'Use IntegerEntryRow so raw input and validation stay consistent.',
}, (context) => ({
  JSXOpeningElement(node) {
    const name = jsxName(node)
    if (name === 'select') context.report({ node, messageId: 'select' })
    if (name === 'textarea') context.report({ node, messageId: 'textarea' })
    if (name !== 'input') return
    const type = attributeValue(node, 'type')
    if (type === 'checkbox') context.report({ node, messageId: 'checkbox' })
    else if (type === 'number') context.report({ node, messageId: 'number' })
    else if (type !== 'file' && type !== 'hidden') context.report({ node, messageId: 'input' })
  },
}))

const semanticPalette = rule({ palette: 'Use semantic palette tokens instead of custom UI colors.' }, (context) => ({
  Literal(node) {
    if (typeof node.value === 'string') checkColor(context, node, node.value)
  },
  TemplateElement(node) {
    checkColor(context, node, node.value.cooked || node.value.raw)
  },
}))

const neutralTableActions = rule({ neutral: 'Keep table actions neutral; use destructive styling in AlertDialog confirmation.' }, (context) => ({
  JSXOpeningElement(node) {
    if (jsxName(node) === 'Button' && ['suggested', 'destructive'].includes(attributeValue(node, 'variant')) && insideTable(node)) {
      context.report({ node, messageId: 'neutral' })
    }
  },
}))

const namedHandlers = rule({ handler: 'Move multi-step actions to a named component handler.' }, (context) => ({
  JSXAttribute(node) {
    if (!/^on[A-Z]/.test(node.name.name)) return
    const value = node.value?.expression
    if (!['ArrowFunctionExpression', 'FunctionExpression'].includes(value?.type)) return
    if (value.body.type === 'BlockStatement' && value.body.body.length > 2) context.report({ node, messageId: 'handler' })
  },
}))

const namedTypes = rule({ type: 'Declare a named interface or type instead of an inline object type.' }, (context) => ({
  TSTypeLiteral(node) {
    if (!namedType(node)) context.report({ node, messageId: 'type' })
  },
}))

const accessibleIconActions = rule({ label: 'Icon actions need an aria-label and a title tooltip.' }, (context) => {
  const icons = new Set()
  return {
    ImportDeclaration(node) {
      if (node.source.value === 'lucide-react') {
        for (const specifier of node.specifiers) icons.add(specifier.local.name)
      }
    },
    JSXElement(node) {
      if (!['Button', 'button'].includes(jsxName(node.openingElement))) return
      const children = node.children.filter((child) => child.type !== 'JSXText' || child.value.trim())
      if (children.length !== 1 || children[0].type !== 'JSXElement' || !icons.has(jsxName(children[0].openingElement))) return
      const opening = node.openingElement
      if (!attribute(opening, 'aria-label') || !attribute(opening, 'title')) context.report({ node: opening, messageId: 'label' })
    },
  }
})

export default {
  rules: {
    'no-comments': noComments,
    'shared-controls': sharedControls,
    'semantic-palette': semanticPalette,
    'neutral-table-actions': neutralTableActions,
    'named-handlers': namedHandlers,
    'named-types': namedTypes,
    'accessible-icon-actions': accessibleIconActions,
  },
}
