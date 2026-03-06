export namespace main {
	
	export class TablePinnedModel {
	    left: string[];
	    right: string[];
	
	    static createFrom(source: any = {}) {
	        return new TablePinnedModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.left = source["left"];
	        this.right = source["right"];
	    }
	}
	export class TablePrefs {
	    version: number;
	    pageSize: number;
	    density: string;
	    columnVisibility: Record<string, boolean>;
	    columnOrder: string[];
	    pinnedColumns: TablePinnedModel;
	
	    static createFrom(source: any = {}) {
	        return new TablePrefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.pageSize = source["pageSize"];
	        this.density = source["density"];
	        this.columnVisibility = source["columnVisibility"];
	        this.columnOrder = source["columnOrder"];
	        this.pinnedColumns = this.convertValues(source["pinnedColumns"], TablePinnedModel);
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
	export class UIPrefs {
	    themeMode: string;
	    palettePreset: string;
	    navCollapsed: boolean;
	    pinActionsColumnRight: boolean;
	    dataGridBorderRadius: number;
	
	    static createFrom(source: any = {}) {
	        return new UIPrefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.themeMode = source["themeMode"];
	        this.palettePreset = source["palettePreset"];
	        this.navCollapsed = source["navCollapsed"];
	        this.pinActionsColumnRight = source["pinActionsColumnRight"];
	        this.dataGridBorderRadius = source["dataGridBorderRadius"];
	    }
	}
	export class Settings {
	    apiBaseUrl: string;
	    authMode: string;
	    apiToken?: string;
	    refreshToken?: string;
	    requestTimeoutSeconds: number;
	    uiPrefs: UIPrefs;
	    tablePrefs: Record<string, TablePrefs>;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.authMode = source["authMode"];
	        this.apiToken = source["apiToken"];
	        this.refreshToken = source["refreshToken"];
	        this.requestTimeoutSeconds = source["requestTimeoutSeconds"];
	        this.uiPrefs = this.convertValues(source["uiPrefs"], UIPrefs);
	        this.tablePrefs = this.convertValues(source["tablePrefs"], TablePrefs, true);
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
	export class UIPrefsPatch {
	    themeMode?: string;
	    palettePreset?: string;
	    navCollapsed?: boolean;
	    pinActionsColumnRight?: boolean;
	    dataGridBorderRadius?: number;
	
	    static createFrom(source: any = {}) {
	        return new UIPrefsPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.themeMode = source["themeMode"];
	        this.palettePreset = source["palettePreset"];
	        this.navCollapsed = source["navCollapsed"];
	        this.pinActionsColumnRight = source["pinActionsColumnRight"];
	        this.dataGridBorderRadius = source["dataGridBorderRadius"];
	    }
	}
	export class SettingsPatch {
	    apiBaseUrl?: string;
	    authMode?: string;
	    apiToken?: string;
	    refreshToken?: string;
	    requestTimeoutSeconds?: number;
	    uiPrefs?: UIPrefsPatch;
	    tablePrefs?: Record<string, TablePrefs>;
	
	    static createFrom(source: any = {}) {
	        return new SettingsPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.authMode = source["authMode"];
	        this.apiToken = source["apiToken"];
	        this.refreshToken = source["refreshToken"];
	        this.requestTimeoutSeconds = source["requestTimeoutSeconds"];
	        this.uiPrefs = this.convertValues(source["uiPrefs"], UIPrefsPatch);
	        this.tablePrefs = this.convertValues(source["tablePrefs"], TablePrefs, true);
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

