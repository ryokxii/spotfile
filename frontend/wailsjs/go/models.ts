export namespace main {
	
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
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.motion = source["motion"];
	        this.folders = source["folders"];
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

