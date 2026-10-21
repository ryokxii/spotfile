export namespace activity {
	
	export class Opened {
	    path: string;
	    // Go type: time
	    at: any;
	
	    static createFrom(source: any = {}) {
	        return new Opened(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.at = this.convertValues(source["at"], null);
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
	export class Search {
	    query: string;
	    // Go type: time
	    at: any;
	
	    static createFrom(source: any = {}) {
	        return new Search(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.at = this.convertValues(source["at"], null);
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
	export class Update {
	    path: string;
	    // Go type: time
	    at: any;
	
	    static createFrom(source: any = {}) {
	        return new Update(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.at = this.convertValues(source["at"], null);
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

export namespace library {
	
	export class File {
	    path: string;
	    bytes: number;
	    // Go type: time
	    modified: any;
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.bytes = source["bytes"];
	        this.modified = this.convertValues(source["modified"], null);
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
	export class TypeCount {
	    type: string;
	    files: number;
	
	    static createFrom(source: any = {}) {
	        return new TypeCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.files = source["files"];
	    }
	}

}

export namespace main {
	
	export class FolderStat {
	    path: string;
	    files: number;
	
	    static createFrom(source: any = {}) {
	        return new FolderStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.files = source["files"];
	    }
	}
	export class Dashboard {
	    folders: FolderStat[];
	    indexBytes: number;
	    files: number;
	    chunks: number;
	    types: library.TypeCount[];
	    lastUpdate?: activity.Update;
	    recentSearches: activity.Search[];
	    recentFiles: activity.Opened[];
	    stale: library.File[];
	    staleBytes: number;
	    history: boolean;
	    staleDays: number;
	
	    static createFrom(source: any = {}) {
	        return new Dashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folders = this.convertValues(source["folders"], FolderStat);
	        this.indexBytes = source["indexBytes"];
	        this.files = source["files"];
	        this.chunks = source["chunks"];
	        this.types = this.convertValues(source["types"], library.TypeCount);
	        this.lastUpdate = this.convertValues(source["lastUpdate"], activity.Update);
	        this.recentSearches = this.convertValues(source["recentSearches"], activity.Search);
	        this.recentFiles = this.convertValues(source["recentFiles"], activity.Opened);
	        this.stale = this.convertValues(source["stale"], library.File);
	        this.staleBytes = source["staleBytes"];
	        this.history = source["history"];
	        this.staleDays = source["staleDays"];
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
	export class EngineConfig {
	    modelPath: string;
	    serverPath: string;
	
	    static createFrom(source: any = {}) {
	        return new EngineConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modelPath = source["modelPath"];
	        this.serverPath = source["serverPath"];
	    }
	}
	export class EngineStatus {
	    ready: boolean;
	    error: string;
	    documents: number;
	    chunks: number;
	    folders: FolderStat[];
	    indexBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new EngineStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.error = source["error"];
	        this.documents = source["documents"];
	        this.chunks = source["chunks"];
	        this.folders = this.convertValues(source["folders"], FolderStat);
	        this.indexBytes = source["indexBytes"];
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

export namespace settings {
	
	export class Settings {
	    theme: string;
	    motion: string;
	    folders: string[];
	    name: string;
	    history: boolean;
	    staleDays: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.motion = source["motion"];
	        this.folders = source["folders"];
	        this.name = source["name"];
	        this.history = source["history"];
	        this.staleDays = source["staleDays"];
	    }
	}

}

export namespace vectorstore {
	
	export class SearchResult {
	    docPath: string;
	    chunkIdx: number;
	    pageNum: number;
	    text: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.docPath = source["docPath"];
	        this.chunkIdx = source["chunkIdx"];
	        this.pageNum = source["pageNum"];
	        this.text = source["text"];
	        this.score = source["score"];
	    }
	}

}

