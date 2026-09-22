import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { parse as parseSFC } from '@vue/compiler-sfc'
import { baseParse } from '@vue/compiler-dom'
import { parse } from '@babel/parser'
import { baseCompile } from '@intlify/message-compiler'

const root = new URL('../src/', import.meta.url)
const read = file => fs.readFileSync(file, 'utf8')
const flatten = (object, prefix = '') => Object.fromEntries(Object.entries(object).flatMap(([key, value]) => typeof value === 'string' ? [[prefix + key, value]] : Object.entries(flatten(value, prefix + key + '.'))))
const en = flatten(JSON.parse(read(new URL('locales/en.json', root))))
const zh = flatten(JSON.parse(read(new URL('locales/zh-CN.json', root))))
const errors = []
const parameters = text => [...text.matchAll(/\{([a-zA-Z]\w*)\}/g)].map(m => m[1]).sort().join(',')
if (JSON.stringify(Object.keys(en).sort()) !== JSON.stringify(Object.keys(zh).sort())) errors.push('Catalogue keys differ')
for (const key of Object.keys(en)) {
  if (parameters(en[key]) !== parameters(zh[key] || '')) errors.push('Parameters differ: ' + key)
  for (const [lang, catalogue] of Object.entries({ en, zh })) baseCompile(catalogue[key] || '', { onError: error => errors.push(`${lang}:${key}: ${error.message}`) })
}
const han = /[\u3400-\u9fff]/
// Protocol names, units and identifiers are deliberately not translated.
const technicalText = /^(?:[\s\d.,:;()×°/%+−–—·#<>\-]*|(?:CUPS|cups-web|PDF|PPD|USB|DPI|dpi|API|URI|ID|GS|HTTP|HTTPS|PNG|JPEG|MB|KB|B|mm|px|A[0-9]|Letter|Legal|auto|www\.[\w.]+):?|(?:A[0-9]|Letter|Legal) \([\d.×]+(?:mm|in)\)|\d+ dpi|dpi \/)$/
function checkJS(source, file) {
  const ast = parse(source, { sourceType: 'module', plugins: ['typescript'] })
  function visit(node, parent) {
    if (!node || typeof node !== 'object') return
    if (node.type === 'StringLiteral' && han.test(node.value)) errors.push(`${file}:${node.loc.start.line}: untranslated string ${node.value}`)
    if (node.type === 'TemplateElement' && han.test(node.value.cooked || '')) errors.push(`${file}: untranslated template literal`)
    if (node.type === 'ObjectProperty' && ['label', 'header', 'title', 'description'].includes(node.key.name) && node.value.type === 'StringLiteral' && /[A-Za-z]/.test(node.value.value) && !technicalText.test(node.value.value)) errors.push(`${file}: untranslated label ${node.value.value}`)
    for (const [key,value] of Object.entries(node)) {
      if (['comments','leadingComments','trailingComments','loc','tokens'].includes(key)) continue
      if (Array.isArray(value)) value.forEach(child=>visit(child,node))
      else if (value && typeof value==='object') visit(value,node)
    }
  }
  visit(ast)
}
function walk(directory) {
  for (const item of fs.readdirSync(directory, { withFileTypes: true })) {
    const file = path.join(directory,item.name)
    if (item.isDirectory()) { if (item.name !== 'locales') walk(file); continue }
    if (!/\.(vue|js)$/.test(file) || file.endsWith('.test.js')) continue
    const source = read(file)
    for (const match of source.matchAll(/\bt\('([^']+)'/g)) if (!(match[1] in en)) errors.push(`${file}: unknown key ${match[1]}`)
    if (file.endsWith('.js')) { checkJS(source,file); continue }
    const { descriptor } = parseSFC(source)
    if (descriptor.scriptSetup) checkJS(descriptor.scriptSetup.content,file)
    const template = baseParse(descriptor.template?.content || '')
    function inspect(node) {
      // Code examples contain literal command and protocol syntax.
      if (node.type===1 && node.tag==='code') return
      if (node.type===2 && /[A-Za-z\u3400-\u9fff]/.test(node.content) && !technicalText.test(node.content.trim())) errors.push(`${file}: untranslated text ${node.content.trim()}`)
      for (const prop of node.props || []) {
        if (prop.type===6 && ['label','title','placeholder','aria-label','alt'].includes(prop.name) && prop.value && /[A-Za-z\u3400-\u9fff]/.test(prop.value.content) && !technicalText.test(prop.value.content)) errors.push(`${file}: untranslated ${prop.name}: ${prop.value.content}`)
        if (prop.type===7 && prop.exp && han.test(prop.exp.content)) errors.push(`${file}: untranslated template expression`)
      }
      if(node.type===5 && han.test(node.content.content)) errors.push(`${file}: untranslated interpolation`)
      for(const child of node.children||[]) inspect(child)
    }
    inspect(template)
  }
}
walk(fileURLToPath(root))
if (errors.length) { console.error(errors.join('\n')); process.exit(1) }
console.log(`Checked ${Object.keys(en).length} matching translations and application source`)
