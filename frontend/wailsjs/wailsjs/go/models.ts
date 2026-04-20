export namespace entity {
	
	export class CreateFolderInput {
	    parent_id?: string;
	    name: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateFolderInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.parent_id = source["parent_id"];
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class EnvVariableInput {
	    key: string;
	    value: string;
	    enabled: boolean;
	    sort_order: number;
	
	    static createFrom(source: any = {}) {
	        return new EnvVariableInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.enabled = source["enabled"];
	        this.sort_order = source["sort_order"];
	    }
	}
	export class EnvironmentVariableRow {
	    id: string;
	    key: string;
	    value: string;
	    enabled: boolean;
	    sort_order: number;
	
	    static createFrom(source: any = {}) {
	        return new EnvironmentVariableRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.value = source["value"];
	        this.enabled = source["enabled"];
	        this.sort_order = source["sort_order"];
	    }
	}
	export class EnvironmentFull {
	    id: string;
	    name: string;
	    description: string;
	    is_active: boolean;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    variables: EnvironmentVariableRow[];
	
	    static createFrom(source: any = {}) {
	        return new EnvironmentFull(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.variables = this.convertValues(source["variables"], EnvironmentVariableRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EnvironmentSummary {
	    id: string;
	    name: string;
	    description: string;
	    is_active: boolean;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new EnvironmentSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class FolderItem {
	    id: string;
	    parent_id?: string;
	    name: string;
	    description: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new FolderItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parent_id = source["parent_id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RequestAuth {
	    type?: string;
	    bearer_token?: string;
	    username?: string;
	    password?: string;
	    api_key?: string;
	    api_key_name?: string;
	    api_key_in?: string;
	
	    static createFrom(source: any = {}) {
	        return new RequestAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.bearer_token = source["bearer_token"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.api_key = source["api_key"];
	        this.api_key_name = source["api_key_name"];
	        this.api_key_in = source["api_key_in"];
	    }
	}
	export class MultipartPart {
	    key: string;
	    kind: string;
	    value?: string;
	    file_path?: string;
	
	    static createFrom(source: any = {}) {
	        return new MultipartPart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.kind = source["kind"];
	        this.value = source["value"];
	        this.file_path = source["file_path"];
	    }
	}
	export class KeyValue {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class HTTPExecuteInput {
	    method: string;
	    url: string;
	    headers?: KeyValue[];
	    query_params?: KeyValue[];
	    root_folder_id?: string;
	    request_id?: string;
	    body_mode?: string;
	    body?: string;
	    form_fields?: KeyValue[];
	    multipart_parts?: MultipartPart[];
	    auth?: RequestAuth;
	
	    static createFrom(source: any = {}) {
	        return new HTTPExecuteInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.url = source["url"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.query_params = this.convertValues(source["query_params"], KeyValue);
	        this.root_folder_id = source["root_folder_id"];
	        this.request_id = source["request_id"];
	        this.body_mode = source["body_mode"];
	        this.body = source["body"];
	        this.form_fields = this.convertValues(source["form_fields"], KeyValue);
	        this.multipart_parts = this.convertValues(source["multipart_parts"], MultipartPart);
	        this.auth = this.convertValues(source["auth"], RequestAuth);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HTTPExecuteResult {
	    status_code: number;
	    duration_ms: number;
	    response_size_bytes: number;
	    response_headers?: KeyValue[];
	    response_body: string;
	    body_truncated: boolean;
	    error_message?: string;
	    final_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new HTTPExecuteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status_code = source["status_code"];
	        this.duration_ms = source["duration_ms"];
	        this.response_size_bytes = source["response_size_bytes"];
	        this.response_headers = this.convertValues(source["response_headers"], KeyValue);
	        this.response_body = source["response_body"];
	        this.body_truncated = source["body_truncated"];
	        this.error_message = source["error_message"];
	        this.final_url = source["final_url"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HistoryItem {
	    id: string;
	    root_folder_id?: string;
	    request_id?: string;
	    method: string;
	    url: string;
	    status_code: number;
	    duration_ms?: number;
	    response_size_bytes?: number;
	    request_headers_json?: string;
	    response_headers_json?: string;
	    request_body?: string;
	    response_body?: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new HistoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.root_folder_id = source["root_folder_id"];
	        this.request_id = source["request_id"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.status_code = source["status_code"];
	        this.duration_ms = source["duration_ms"];
	        this.response_size_bytes = source["response_size_bytes"];
	        this.request_headers_json = source["request_headers_json"];
	        this.response_headers_json = source["response_headers_json"];
	        this.request_body = source["request_body"];
	        this.response_body = source["response_body"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HistorySummary {
	    id: string;
	    root_folder_id?: string;
	    request_id?: string;
	    method: string;
	    url: string;
	    status_code: number;
	    duration_ms?: number;
	    response_size_bytes?: number;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new HistorySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.root_folder_id = source["root_folder_id"];
	        this.request_id = source["request_id"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.status_code = source["status_code"];
	        this.duration_ms = source["duration_ms"];
	        this.response_size_bytes = source["response_size_bytes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportOptions {
	    create_environment: boolean;
	    activate_environment: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.create_environment = source["create_environment"];
	        this.activate_environment = source["activate_environment"];
	    }
	}
	export class ImportResult {
	    root_folder_id: string;
	    root_folder_name: string;
	    folders_created: number;
	    requests_created: number;
	    environment_id?: string;
	    environment_name?: string;
	    format_label: string;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root_folder_id = source["root_folder_id"];
	        this.root_folder_name = source["root_folder_name"];
	        this.folders_created = source["folders_created"];
	        this.requests_created = source["requests_created"];
	        this.environment_id = source["environment_id"];
	        this.environment_name = source["environment_name"];
	        this.format_label = source["format_label"];
	        this.warnings = source["warnings"];
	    }
	}
	export class ImportedRequest {
	    name: string;
	    method: string;
	    url: string;
	    body_mode?: string;
	    raw_body?: string;
	    headers?: KeyValue[];
	    query_params?: KeyValue[];
	    form_fields?: KeyValue[];
	    multipart_parts?: MultipartPart[];
	    auth?: RequestAuth;
	
	    static createFrom(source: any = {}) {
	        return new ImportedRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.body_mode = source["body_mode"];
	        this.raw_body = source["raw_body"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.query_params = this.convertValues(source["query_params"], KeyValue);
	        this.form_fields = this.convertValues(source["form_fields"], KeyValue);
	        this.multipart_parts = this.convertValues(source["multipart_parts"], MultipartPart);
	        this.auth = this.convertValues(source["auth"], RequestAuth);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportedFolder {
	    name: string;
	    description?: string;
	    items?: ImportedItem[];
	
	    static createFrom(source: any = {}) {
	        return new ImportedFolder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.items = this.convertValues(source["items"], ImportedItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportedItem {
	    folder?: ImportedFolder;
	    request?: ImportedRequest;
	
	    static createFrom(source: any = {}) {
	        return new ImportedItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = this.convertValues(source["folder"], ImportedFolder);
	        this.request = this.convertValues(source["request"], ImportedRequest);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportedVariable {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportedVariable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class ImportedCollection {
	    name: string;
	    description?: string;
	    variables?: ImportedVariable[];
	    root_items?: ImportedItem[];
	    format_label: string;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportedCollection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.variables = this.convertValues(source["variables"], ImportedVariable);
	        this.root_items = this.convertValues(source["root_items"], ImportedItem);
	        this.format_label = source["format_label"];
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	export class SavedRequestFull {
	    id: string;
	    folder_id: string;
	    name: string;
	    method: string;
	    url: string;
	    body_mode: string;
	    raw_body?: string;
	    headers?: KeyValue[];
	    query_params?: KeyValue[];
	    form_fields?: KeyValue[];
	    multipart_parts?: MultipartPart[];
	    auth?: RequestAuth;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new SavedRequestFull(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.folder_id = source["folder_id"];
	        this.name = source["name"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.body_mode = source["body_mode"];
	        this.raw_body = source["raw_body"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.query_params = this.convertValues(source["query_params"], KeyValue);
	        this.form_fields = this.convertValues(source["form_fields"], KeyValue);
	        this.multipart_parts = this.convertValues(source["multipart_parts"], MultipartPart);
	        this.auth = this.convertValues(source["auth"], RequestAuth);
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SavedRequestSummary {
	    id: string;
	    folder_id: string;
	    name: string;
	    method: string;
	    url: string;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new SavedRequestSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.folder_id = source["folder_id"];
	        this.name = source["name"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

