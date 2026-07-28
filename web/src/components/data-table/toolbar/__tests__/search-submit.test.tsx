/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { after, describe, test } from 'node:test'

import type { Table } from '@tanstack/react-table'
import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLInputElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'KeyboardEvent',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const i18next = (await import('i18next')).default
const { initReactI18next } = await import('react-i18next')
await i18next.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        Search: 'Search',
        Reset: 'Reset',
      },
    },
  },
})
const { DataTableToolbar } = await import('../toolbar')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type RenderedToolbar = {
  container: HTMLDivElement
  root: ReturnType<typeof createRoot>
}

function createTable() {
  let globalFilter = ''
  const committedValues: string[] = []
  const table = {
    getState: () => ({ columnFilters: [], globalFilter }),
    getColumn: () => undefined,
    setGlobalFilter: (value: string) => {
      globalFilter = value
      committedValues.push(value)
    },
    resetColumnFilters: () => undefined,
  } as unknown as Table<unknown>

  return { table, committedValues }
}

async function renderToolbar(
  table: Table<unknown>,
  onSearch: () => void,
  additionalSearch?: React.ReactNode
): Promise<RenderedToolbar> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <DataTableToolbar
        table={table}
        searchPlaceholder='Filter channels'
        onSearch={onSearch}
        additionalSearch={additionalSearch}
        hideViewOptions
      />
    )
  })

  return { container, root }
}

async function setInputValue(input: HTMLInputElement, value: string) {
  await act(async () => {
    const valueSetter = Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      'value'
    )?.set
    valueSetter?.call(input, value)
    input.dispatchEvent(new Event('input', { bubbles: true }))
  })
}

async function pressEnter(input: HTMLInputElement, isComposing = false) {
  const event = new KeyboardEvent('keydown', {
    key: 'Enter',
    bubbles: true,
    cancelable: true,
  })
  if (isComposing) {
    Object.defineProperty(event, 'isComposing', { value: true })
  }
  await act(async () => input.dispatchEvent(event))
  return event
}

async function unmountToolbar(rendered: RenderedToolbar) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

describe('DataTableToolbar search submission', () => {
  after(() => {
    domWindow.close()
  })

  test('pressing Enter in the primary input commits the draft and searches', async () => {
    const { table, committedValues } = createTable()
    let searchCount = 0
    const rendered = await renderToolbar(table, () => {
      searchCount += 1
    })
    const input = rendered.container.querySelector<HTMLInputElement>(
      'input[placeholder="Filter channels"]'
    )

    assert.ok(input)
    await setInputValue(input, 'openai')
    const event = await pressEnter(input)

    assert.equal(event.defaultPrevented, true)
    assert.deepEqual(committedValues, ['openai'])
    assert.equal(searchCount, 1)
    await unmountToolbar(rendered)
  })

  test('pressing Enter in an additional input invokes the same search action', async () => {
    const { table } = createTable()
    let modelValue = ''
    const searchedModels: string[] = []
    const rendered = await renderToolbar(
      table,
      () => searchedModels.push(modelValue),
      <input
        aria-label='Model filter'
        onInput={(event) => {
          modelValue = event.currentTarget.value
        }}
      />
    )
    const input = rendered.container.querySelector<HTMLInputElement>(
      'input[aria-label="Model filter"]'
    )

    assert.ok(input)
    await setInputValue(input, 'gpt-5')
    await pressEnter(input)

    assert.deepEqual(searchedModels, ['gpt-5'])
    await unmountToolbar(rendered)
  })

  test('pressing Enter during IME composition does not search', async () => {
    const { table, committedValues } = createTable()
    let searchCount = 0
    const rendered = await renderToolbar(table, () => {
      searchCount += 1
    })
    const input = rendered.container.querySelector<HTMLInputElement>(
      'input[placeholder="Filter channels"]'
    )

    assert.ok(input)
    await setInputValue(input, '渠道')
    const event = await pressEnter(input, true)

    assert.equal(event.defaultPrevented, false)
    assert.deepEqual(committedValues, [])
    assert.equal(searchCount, 0)
    await unmountToolbar(rendered)
  })
})
