import type { Scan } from '$lib/types'

export interface PageData {
    scan: Scan
    status: Scan['status']
}
