import type { PageLoad } from '$lib/types'
import { scansApi } from '$lib/api/api'

interface Params {
    id: string
}

export const load: PageLoad = async ({ params }: { params: Params }) => {
    const scan = await scansApi.getScan(parseInt(params.id))
    
    return {
        scan,
        status: scan.status
    }
}
