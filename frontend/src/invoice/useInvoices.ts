import { useEffect, useState } from 'react'
import { api } from '../auth/api'
import type { Invoice } from './Invoice'


interface UseInvoicesResult {
    invoices: Invoice[]
    fetching: boolean
    fetchResult: Error | null
    refetchInvoices: () => void
}

export function useInvoices(): UseInvoicesResult {
    const [invoices, setInvoices] = useState<Invoice[]>([])
    const [fetching, setFetching] = useState(true)
    const [fetchResult, setFetchResult] = useState<Error | null>(null)
    const [refetchFlag, setRefetchFlag] = useState(0)

    useEffect(() => {
          let cancelled = false
    
          setFetching(true)
          setFetchResult(null)
    
          async function loadInvoices() {
              try {
                  const response = await api<Invoice[]>('/api/invoices', { method: 'GET' })
                  if (!cancelled) {
                      setInvoices(response)
                      setFetchResult(null)
                  }
              } catch (error) {
                  if (!cancelled) {
                      setFetchResult(new Error(error instanceof Error ? error.message : String(error)))
                  }
              } finally {
                  if (!cancelled) {
                      setFetching(false)
                  }
              }
          }
    
          loadInvoices()
    
          return () => { cancelled = true }
      }, [refetchFlag])
    
      function refetchInvoices() {
          setRefetchFlag(f => f + 1)
      }
    
      return { invoices, fetching, fetchResult, refetchInvoices }
    }