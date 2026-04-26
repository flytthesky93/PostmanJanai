export namespace entity {
	
	export class AssertionResult {
	    name: string;
	    source: string;
	    expression: string;
	    operator: string;
	    expected: string;
	    actual: string;
	    passed: boolean;
	    error_message?: string;
	
	    static createFrom(source: any = {}) {
	        return new AssertionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.source = source["source"];
	        this.expression = source["expression"];
	        this.operator = source["operator"];
	        this.expected = source["expected"];
	        this.actual = source["actual"];
	        this.passed = source["passed"];
	        this.error_message = source["error_message"];
	    }
	}
	export class CaptureResult {
	    name: string;
	    target_scope: string;
	    target_variable: string;
	    source: string;
	    expression: string;
	    value: string;
	    captured: boolean;
	    error_message?: string;
	
	    static createFrom(source: any = {}) {
	        return new CaptureResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.target_scope = source["target_scope"];
	        this.target_variable = source["target_variable"];
	        this.source = source["source"];
	        this.expression = source["expression"];
	        this.value = source["value"];
	        this.captured = source["captured"];
	        this.error_message = source["error_message"];
	    }
	}
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
	    kind: string;
	    enabled: boolean;
	    sort_order: number;
	
	    static createFrom(source: any = {}) {
	        return new EnvVariableInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.kind = source["kind"];
	        this.enabled = source["enabled"];
	        this.sort_order = source["sort_order"];
	    }
	}
	export class EnvironmentVariableRow {
	    id: string;
	    key: string;
	    value: string;
	    kind: string;
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
	        this.kind = source["kind"];
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
	    sort_order: number;
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
	        this.sort_order = source["sort_order"];
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
	    insecure_skip_verify?: boolean;
	
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
	        this.insecure_skip_verify = source["insecure_skip_verify"];
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
	    captures?: CaptureResult[];
	    assertions?: AssertionResult[];
	
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
	        this.captures = this.convertValues(source["captures"], CaptureResult);
	        this.assertions = this.convertValues(source["assertions"], AssertionResult);
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
	    insecure_tls?: boolean;
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
	        this.insecure_tls = source["insecure_tls"];
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
	    insecure_tls?: boolean;
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
	        this.insecure_tls = source["insecure_tls"];
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
	
	
	
	
	
	
	export class ProxySettings {
	    mode: string;
	    url: string;
	    username: string;
	    password: string;
	    no_proxy: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.url = source["url"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.no_proxy = source["no_proxy"];
	    }
	}
	export class ProxyTestResult {
	    ok: boolean;
	    status_code: number;
	    duration_ms: number;
	    error_message?: string;
	    final_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.status_code = source["status_code"];
	        this.duration_ms = source["duration_ms"];
	        this.error_message = source["error_message"];
	        this.final_url = source["final_url"];
	    }
	}
	export class RequestAssertionInput {
	    name: string;
	    source: string;
	    expression: string;
	    operator: string;
	    expected: string;
	    enabled: boolean;
	    sort_order: number;
	
	    static createFrom(source: any = {}) {
	        return new RequestAssertionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.source = source["source"];
	        this.expression = source["expression"];
	        this.operator = source["operator"];
	        this.expected = source["expected"];
	        this.enabled = source["enabled"];
	        this.sort_order = source["sort_order"];
	    }
	}
	export class RequestAssertionRow {
	    id: string;
	    name: string;
	    source: string;
	    expression: string;
	    operator: string;
	    expected: string;
	    enabled: boolean;
	    sort_order: number;
	
	    static createFrom(source: any = {}) {
	        return new RequestAssertionRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.source = source["source"];
	        this.expression = source["expression"];
	        this.operator = source["operator"];
	        this.expected = source["expected"];
	        this.enabled = source["enabled"];
	        this.sort_order = source["sort_order"];
	    }
	}
	
	export class RequestCaptureInput {
	    name: string;
	    source: string;
	    expression: string;
	    target_scope: string;
	    target_variable: string;
	    enabled: boolean;
	    sort_order: number;
	
	    static createFrom(source: any = {}) {
	        return new RequestCaptureInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.source = source["source"];
	        this.expression = source["expression"];
	        this.target_scope = source["target_scope"];
	        this.target_variable = source["target_variable"];
	        this.enabled = source["enabled"];
	        this.sort_order = source["sort_order"];
	    }
	}
	export class RequestCaptureRow {
	    id: string;
	    name: string;
	    source: string;
	    expression: string;
	    target_scope: string;
	    target_variable: string;
	    enabled: boolean;
	    sort_order: number;
	
	    static createFrom(source: any = {}) {
	        return new RequestCaptureRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.source = source["source"];
	        this.expression = source["expression"];
	        this.target_scope = source["target_scope"];
	        this.target_variable = source["target_variable"];
	        this.enabled = source["enabled"];
	        this.sort_order = source["sort_order"];
	    }
	}
	export class RunFolderInput {
	    folder_id: string;
	    environment_id?: string;
	    stop_on_fail?: boolean;
	    notes?: string;
	
	    static createFrom(source: any = {}) {
	        return new RunFolderInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder_id = source["folder_id"];
	        this.environment_id = source["environment_id"];
	        this.stop_on_fail = source["stop_on_fail"];
	        this.notes = source["notes"];
	    }
	}
	export class RunnerRunRequestRow {
	    id: string;
	    run_id: string;
	    request_id?: string;
	    request_name: string;
	    method: string;
	    url: string;
	    status: string;
	    status_code: number;
	    duration_ms: number;
	    response_size_bytes: number;
	    error_message?: string;
	    request_headers_json?: string;
	    response_headers_json?: string;
	    request_body?: string;
	    response_body?: string;
	    body_truncated?: boolean;
	    assertions?: AssertionResult[];
	    captures?: CaptureResult[];
	    sort_order: number;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new RunnerRunRequestRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.run_id = source["run_id"];
	        this.request_id = source["request_id"];
	        this.request_name = source["request_name"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.status = source["status"];
	        this.status_code = source["status_code"];
	        this.duration_ms = source["duration_ms"];
	        this.response_size_bytes = source["response_size_bytes"];
	        this.error_message = source["error_message"];
	        this.request_headers_json = source["request_headers_json"];
	        this.response_headers_json = source["response_headers_json"];
	        this.request_body = source["request_body"];
	        this.response_body = source["response_body"];
	        this.body_truncated = source["body_truncated"];
	        this.assertions = this.convertValues(source["assertions"], AssertionResult);
	        this.captures = this.convertValues(source["captures"], CaptureResult);
	        this.sort_order = source["sort_order"];
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
	export class RunnerRunDetail {
	    id: string;
	    folder_id?: string;
	    folder_name: string;
	    environment_id?: string;
	    environment_name: string;
	    status: string;
	    total_count: number;
	    passed_count: number;
	    failed_count: number;
	    error_count: number;
	    duration_ms: number;
	    // Go type: time
	    started_at: any;
	    // Go type: time
	    finished_at?: any;
	    requests: RunnerRunRequestRow[];
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new RunnerRunDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.folder_id = source["folder_id"];
	        this.folder_name = source["folder_name"];
	        this.environment_id = source["environment_id"];
	        this.environment_name = source["environment_name"];
	        this.status = source["status"];
	        this.total_count = source["total_count"];
	        this.passed_count = source["passed_count"];
	        this.failed_count = source["failed_count"];
	        this.error_count = source["error_count"];
	        this.duration_ms = source["duration_ms"];
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.finished_at = this.convertValues(source["finished_at"], null);
	        this.requests = this.convertValues(source["requests"], RunnerRunRequestRow);
	        this.notes = source["notes"];
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
	
	export class RunnerRunSummary {
	    id: string;
	    folder_id?: string;
	    folder_name: string;
	    environment_id?: string;
	    environment_name: string;
	    status: string;
	    total_count: number;
	    passed_count: number;
	    failed_count: number;
	    error_count: number;
	    duration_ms: number;
	    // Go type: time
	    started_at: any;
	    // Go type: time
	    finished_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new RunnerRunSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.folder_id = source["folder_id"];
	        this.folder_name = source["folder_name"];
	        this.environment_id = source["environment_id"];
	        this.environment_name = source["environment_name"];
	        this.status = source["status"];
	        this.total_count = source["total_count"];
	        this.passed_count = source["passed_count"];
	        this.failed_count = source["failed_count"];
	        this.error_count = source["error_count"];
	        this.duration_ms = source["duration_ms"];
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.finished_at = this.convertValues(source["finished_at"], null);
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
	    insecure_skip_verify?: boolean;
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
	        this.insecure_skip_verify = source["insecure_skip_verify"];
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
	export class SearchFolderHit {
	    id: string;
	    name: string;
	    parent_id?: string;
	    root_id: string;
	    path: string[];
	    ancestor_ids: string[];
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchFolderHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.parent_id = source["parent_id"];
	        this.root_id = source["root_id"];
	        this.path = source["path"];
	        this.ancestor_ids = source["ancestor_ids"];
	        this.description = source["description"];
	    }
	}
	export class SearchRequestHit {
	    id: string;
	    folder_id: string;
	    root_id: string;
	    name: string;
	    method: string;
	    url: string;
	    path: string[];
	    ancestor_ids: string[];
	
	    static createFrom(source: any = {}) {
	        return new SearchRequestHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.folder_id = source["folder_id"];
	        this.root_id = source["root_id"];
	        this.name = source["name"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.path = source["path"];
	        this.ancestor_ids = source["ancestor_ids"];
	    }
	}
	export class SearchResults {
	    query: string;
	    folders: SearchFolderHit[];
	    requests: SearchRequestHit[];
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchResults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.folders = this.convertValues(source["folders"], SearchFolderHit);
	        this.requests = this.convertValues(source["requests"], SearchRequestHit);
	        this.truncated = source["truncated"];
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
	export class TrustedCASummary {
	    id: string;
	    label: string;
	    enabled: boolean;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new TrustedCASummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.enabled = source["enabled"];
	        this.created_at = source["created_at"];
	    }
	}

}

