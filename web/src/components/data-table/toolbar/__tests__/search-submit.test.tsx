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
import type { Table } from '@tanstack/react-table'
import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { Input } from '@/components/ui/input'

import { DataTableToolbar } from '../toolbar'

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

function AdditionalSearchFixture(props: {
  table: Table<unknown>
  onSearch: () => void
  onSearchedModel: (value: string) => void
}) {
  const [modelValue, setModelValue] = useState('')
  return (
    <DataTableToolbar
      table={props.table}
      searchPlaceholder='Filter channels'
      onSearch={() => {
        props.onSearch()
        props.onSearchedModel(modelValue)
      }}
      additionalSearch={
        <Input
          aria-label='Model filter'
          value={modelValue}
          onChange={(event) => setModelValue(event.target.value)}
        />
      }
      hideViewOptions
    />
  )
}

describe('DataTableToolbar search submission', () => {
  it('pressing Enter in the primary input commits the draft and searches', async () => {
    const user = userEvent.setup()
    const { table, committedValues } = createTable()
    const onSearch = vi.fn()
    render(
      <DataTableToolbar
        table={table}
        searchPlaceholder='Filter channels'
        onSearch={onSearch}
        hideViewOptions
      />
    )
    const input = screen.getByPlaceholderText('Filter channels')

    await user.type(input, 'openai')
    expect(committedValues).toEqual([])

    await user.type(input, '{Enter}')
    expect(committedValues).toEqual(['openai'])
    expect(onSearch).toHaveBeenCalledTimes(1)
  })

  it('pressing Enter in an additional input invokes the same search action', async () => {
    const user = userEvent.setup()
    const { table, committedValues } = createTable()
    const onSearch = vi.fn()
    const searchedModels: string[] = []
    render(
      <AdditionalSearchFixture
        table={table}
        onSearch={onSearch}
        onSearchedModel={(value) => searchedModels.push(value)}
      />
    )
    const input = screen.getByLabelText('Model filter')

    await user.type(input, 'gpt-5')
    await user.type(input, '{Enter}')

    expect(searchedModels).toEqual(['gpt-5'])
    expect(onSearch).toHaveBeenCalledTimes(1)
  })

  it('pressing Enter during IME composition does not search', async () => {
    const { table, committedValues } = createTable()
    const onSearch = vi.fn()
    render(
      <DataTableToolbar
        table={table}
        searchPlaceholder='Filter channels'
        onSearch={onSearch}
        hideViewOptions
      />
    )
    const input = screen.getByPlaceholderText('Filter channels')

    fireEvent.change(input, { target: { value: '渠道' } })
    fireEvent.keyDown(input, { key: 'Enter', isComposing: true })

    expect(committedValues).toEqual([])
    expect(onSearch).not.toHaveBeenCalled()
  })
})
