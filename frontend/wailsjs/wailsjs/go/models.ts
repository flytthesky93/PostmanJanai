export namespace entity {
	
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
	    body_mode?: string;
	    body?: string;
	    form_fields?: KeyValue[];
	    multipart_parts?: MultipartPart[];
	
	    static createFrom(source: any = {}) {
	        return new HTTPExecuteInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.url = source["url"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.query_params = this.convertValues(source["query_params"], KeyValue);
	        this.body_mode = source["body_mode"];
	        this.body = source["body"];
	        this.form_fields = this.convertValues(source["form_fields"], KeyValue);
	        this.multipart_parts = this.convertValues(source["multipart_parts"], MultipartPart);
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
	
	
	export class WorkspaceItem {
	    id: string;
	    workspace_name: string;
	    workspace_description: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workspace_name = source["workspace_name"];
	        this.workspace_description = source["workspace_description"];
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

}

