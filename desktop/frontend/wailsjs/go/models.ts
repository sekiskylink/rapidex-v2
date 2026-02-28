export namespace main {
	
	export class Settings {
	    apiBaseUrl: string;
	    authMode: string;
	    apiToken?: string;
	    requestTimeoutSeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.authMode = source["authMode"];
	        this.apiToken = source["apiToken"];
	        this.requestTimeoutSeconds = source["requestTimeoutSeconds"];
	    }
	}
	export class SettingsPatch {
	    apiBaseUrl?: string;
	    authMode?: string;
	    apiToken?: string;
	    requestTimeoutSeconds?: number;
	
	    static createFrom(source: any = {}) {
	        return new SettingsPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.authMode = source["authMode"];
	        this.apiToken = source["apiToken"];
	        this.requestTimeoutSeconds = source["requestTimeoutSeconds"];
	    }
	}

}

