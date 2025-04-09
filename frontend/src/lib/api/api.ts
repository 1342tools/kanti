import type {
	RootDomain,
	Subdomain,
	Endpoint,
	Technology,
	Parameter,
	RequestResponse,
	Scan,
	ApiError,
	GraphData,
	GraphNode,
	GraphLink,
	// Import new types
	Organization,
	ScanTemplate // Add ScanTemplate type
} from '../types';

export const API_BASE_URL = 'http://localhost:8080/api'; // Export the base URL

// Custom Error class to hold status and response data
export class HttpError extends Error {
    status: number;
    data: any; // The parsed JSON error response

    constructor(message: string, status: number, data: any) {
        super(message);
        this.name = 'HttpError';
        this.status = status;
        this.data = data;
        // Set the prototype explicitly.
        Object.setPrototypeOf(this, HttpError.prototype);
    }
}


async function fetchApi<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        }
    });

    let data: any;
    try {
        // Try to parse JSON, but handle cases where the body might be empty or not JSON
        const text = await response.text();
        if (text) {
            data = JSON.parse(text);
        } else {
            data = { detail: response.statusText || 'Error' }; // Use status text if no body
        }
    } catch (e) {
        // If JSON parsing fails, use the status text as the error detail
        data = { detail: response.statusText || `HTTP error ${response.status}` };
    }


    if (!response.ok) {
        // Throw the custom HttpError with status and data
        const errorMessage = (data as ApiError)?.detail || `HTTP error ${response.status}`;
        throw new HttpError(errorMessage, response.status, data);
    }

    // If response is ok, but data wasn't parsed successfully earlier (e.g., empty 204 response)
    // return an empty object or handle as appropriate for your API design.
    // Here, we assume successful responses always have parseable JSON.
    return data as T;
}

// Function specifically for FormData uploads (doesn't set Content-Type)
async function postFormData<T>(endpoint: string, formData: FormData, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        method: 'POST', // Default to POST for uploads
        ...options,
        body: formData,
        headers: {
            // DO NOT set Content-Type here; browser handles it for FormData
            ...options.headers
        }
    });

    let data: any;
    try {
        const text = await response.text();
        if (text) {
            data = JSON.parse(text);
        } else {
            data = { detail: response.statusText || 'Success' }; // Assume success if no body
        }
    } catch (e) {
        data = { detail: response.statusText || `HTTP error ${response.status}` };
    }

    if (!response.ok) {
        const errorMessage = (data as ApiError)?.detail || `HTTP error ${response.status}`;
        throw new HttpError(errorMessage, response.status, data);
    }

    return data as T;
}


// Domains API
export const domainsApi = {
	getDomains: () => fetchApi<RootDomain[]>('/domains'),
	getDomain: (id: number) => fetchApi<RootDomain>(`/domains/${id}`),
	// Update createDomain to accept organization_id
	createDomain: (domainData: { domain: string; organization_id: number }) =>
		fetchApi<RootDomain>('/domains', {
			method: 'POST',
			body: JSON.stringify(domainData)
		}),
	// Update scanDomain to accept optional template ID
	scanDomain: (id: number, scanTemplateId?: number) => {
		const body = scanTemplateId ? JSON.stringify({ scan_template_id: scanTemplateId }) : undefined;
		return fetchApi<{ message: string, scan_id: number }>(`/domains/${id}/scan`, {
			method: 'POST',
			body: body // Send body only if template ID is provided
		});
	}
};

// Subdomains API
export const subdomainsApi = {
    getSubdomains: (domainId?: number) => {
        const query = domainId ? `?domain_id=${domainId}` : '';
        return fetchApi<Subdomain[]>(`/subdomains${query}`);
    },
    getSubdomain: (id: number) => fetchApi<Subdomain>(`/subdomains/${id}`),
    getSubdomainEndpoints: (id: number) => fetchApi<Endpoint[]>(`/subdomains/${id}/endpoints`)
};

// Endpoints API
export const endpointsApi = {
    getEndpoints: (subdomainId?: number) => {
        const query = subdomainId ? `?subdomain_id=${subdomainId}` : '';
        return fetchApi<Endpoint[]>(`/endpoints${query}`);
    },
    getEndpoint: (id: number) => fetchApi<Endpoint>(`/endpoints/${id}`),
    getEndpointParameters: (id: number) => fetchApi<Parameter[]>(`/endpoints/${id}/parameters`),
    getEndpointRequestResponses: (id: number) => fetchApi<RequestResponse[]>(`/endpoints/${id}/request-responses`)
};

// Technologies API
export const technologiesApi = {
    getTechnologies: () => fetchApi<Technology[]>('/technologies'),
    getTechnology: (id: number) => fetchApi<Technology>(`/technologies/${id}`),
    getDomainsWithTechnology: (id: number) => fetchApi<RootDomain[]>(`/technologies/${id}/domains`),
    getSubdomainsWithTechnology: (id: number) => fetchApi<Subdomain[]>(`/technologies/${id}/subdomains`),
    getEndpointsWithTechnology: (id: number) => fetchApi<Endpoint[]>(`/technologies/${id}/endpoints`)
};

// Scans API
export const scansApi = {
	getScans: (domainId?: number) => {
        // Corrected: Only one declaration of query
        const query = domainId ? `?root_domain_id=${domainId}` : ''; // Use root_domain_id for filtering
        return fetchApi<Scan[]>(`/scans${query}`);
    },
    getScan: (id: number) => fetchApi<Scan>(`/scans/${id}`),
    // New function to start scans (root or subdomain)
    startScan: (scanData: { rootDomainId: number; subdomainId?: number; scanTemplateId?: number }) => {
        const body = {
            root_domain_id: scanData.rootDomainId,
            ...(scanData.subdomainId && { subdomain_id: scanData.subdomainId }),
            ...(scanData.scanTemplateId && { scan_template_id: scanData.scanTemplateId })
        };
        return fetchApi<{ message: string, scan_id: number }>('/scans', {
            method: 'POST',
            body: JSON.stringify(body)
        });
    }
};

// Add Organizations API
export const organizationsApi = {
	getOrganizations: () => fetchApi<Organization[]>('/organizations'),
	getOrganization: (id: number) => fetchApi<Organization>(`/organizations/${id}`), // Use Organization (now includes counts)
	createOrganization: (orgData: { name: string }) =>
		fetchApi<Organization>('/organizations', {
			method: 'POST',
			body: JSON.stringify(orgData)
		})
	// Add getOrganizationDomains if needed separately later
};

// --- Import API ---
export const importApi = {
	// Update to accept organizationId and include it in the URL
	uploadUrls: (organizationId: number, formData: FormData) =>
		postFormData<{ message: string }>(`/organizations/${organizationId}/import/urls`, formData)
};

// --- Scan Templates API ---
export const scanTemplatesApi = {
	getScanTemplates: () => fetchApi<ScanTemplate[]>('/scan-templates'),
	getScanTemplate: (id: number) => fetchApi<ScanTemplate>(`/scan-templates/${id}`),
	createScanTemplate: (templateData: Omit<ScanTemplate, 'id' | 'created_at' | 'updated_at'>) =>
		fetchApi<ScanTemplate>('/scan-templates', {
			method: 'POST',
			body: JSON.stringify(templateData)
		}),
	updateScanTemplate: (id: number, templateData: Partial<Omit<ScanTemplate, 'id' | 'created_at' | 'updated_at'>>) =>
		fetchApi<ScanTemplate>(`/scan-templates/${id}`, {
			method: 'PUT',
			body: JSON.stringify(templateData)
		}),
	deleteScanTemplate: (id: number) =>
		fetchApi<{ message: string }>(`/scan-templates/${id}`, {
			method: 'DELETE'
		})
};


// Define the expected structure from the backend graph endpoint
interface ApiNode {
  id: string;
  label: string;
  type: string;
  size: number;
  color: string;
  x: number;
  y: number;
}

interface ApiLink {
  from: string;
  to: string;
}

interface ApiGraphDataResponse {
  nodes: ApiNode[];
  links: ApiLink[];
}


// Graph API
export const graphApi = {
	// Fetch the pre-processed graph data directly from the backend
	getGraphData: () => fetchApi<ApiGraphDataResponse>('/graph')
};

// Settings API
export const settingsApi = {
	getSettings: () => fetchApi<{ [key: string]: string }>('/settings'),
	saveSettings: (settingsData: { [key: string]: string }) =>
		fetchApi<{ message: string }>('/settings', {
			method: 'POST',
			body: JSON.stringify(settingsData)
		})
};
